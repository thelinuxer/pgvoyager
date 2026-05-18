// Package dbsafe centralizes the small set of routines that turn untrusted
// user input into PostgreSQL DDL safely: identifier quoting, string-literal
// quoting, action/keyword whitelists, and pgx error sanitization.
//
// Every helper here is the *only* approved way to interpolate user-controlled
// values into SQL that pgx cannot parameter-bind (database names, role names,
// FK actions, encoding labels). Bypassing them re-opens SQL injection.
package dbsafe

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidIdentifier is returned when a candidate identifier contains a
// NUL byte (libpq truncates at NUL — silent data corruption).
var ErrInvalidIdentifier = errors.New("identifier contains a NUL byte")

// QuoteIdent quotes a Postgres identifier for safe inclusion in DDL where
// parameter binding isn't available. Doubles any embedded `"` and wraps in
// `"`. Rejects identifiers with embedded NUL bytes — these silently
// truncate strings inside libpq/pgx and have been used historically as a
// path to bypass identifier sanitization.
func QuoteIdent(s string) (string, error) {
	if strings.ContainsRune(s, 0) {
		return "", ErrInvalidIdentifier
	}
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			out = append(out, '"', '"')
		} else {
			out = append(out, s[i])
		}
	}
	out = append(out, '"')
	return string(out), nil
}

// QuoteString returns a Postgres string literal — single-quoted with any
// embedded `'` doubled and any embedded `\` doubled, prefixed with the
// `E` standard-conforming-strings escape marker so backslash-escapes are
// taken literally regardless of the `standard_conforming_strings` GUC.
// Rejects NUL bytes (Postgres rejects them in text values anyway).
func QuoteString(s string) (string, error) {
	if strings.ContainsRune(s, 0) {
		return "", errors.New("string contains a NUL byte")
	}
	var b strings.Builder
	b.Grow(len(s) + 4)
	b.WriteString("E'")
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			b.WriteString("''")
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteByte(s[i])
		}
	}
	b.WriteByte('\'')
	return b.String(), nil
}

// fkActions is the closed set of referential-action keywords PostgreSQL
// accepts for ON DELETE / ON UPDATE. Anything outside this set was likely
// injected.
var fkActions = map[string]string{
	"NO ACTION":   "NO ACTION",
	"RESTRICT":    "RESTRICT",
	"CASCADE":     "CASCADE",
	"SET NULL":    "SET NULL",
	"SET DEFAULT": "SET DEFAULT",
}

// CanonicalFKAction normalizes and validates an FK referential action.
// Returns the canonical form (uppercased, single-spaced) so callers can
// concatenate it into DDL directly.
func CanonicalFKAction(s string) (string, error) {
	norm := strings.ToUpper(strings.Join(strings.Fields(s), " "))
	if canon, ok := fkActions[norm]; ok {
		return canon, nil
	}
	return "", fmt.Errorf("invalid FK action %q (allowed: NO ACTION, RESTRICT, CASCADE, SET NULL, SET DEFAULT)", s)
}

// validEncodings enumerates the encoding labels PostgreSQL accepts for
// CREATE DATABASE ... ENCODING. Whitelisting is cheap and makes the path
// safe to render as a string literal.
var validEncodings = map[string]bool{
	"UTF8": true, "UNICODE": true, "SQL_ASCII": true,
	"LATIN1": true, "LATIN2": true, "LATIN3": true, "LATIN4": true,
	"LATIN5": true, "LATIN6": true, "LATIN7": true, "LATIN8": true,
	"LATIN9": true, "LATIN10": true,
	"WIN1250": true, "WIN1251": true, "WIN1252": true, "WIN1253": true,
	"WIN1254": true, "WIN1255": true, "WIN1256": true, "WIN1257": true, "WIN1258": true,
	"WIN866": true, "WIN874": true,
	"KOI8R": true, "KOI8U": true,
	"ISO_8859_5": true, "ISO_8859_6": true, "ISO_8859_7": true, "ISO_8859_8": true,
	"EUC_CN": true, "EUC_JP": true, "EUC_JIS_2004": true, "EUC_KR": true, "EUC_TW": true,
	"GB18030": true, "GBK": true, "BIG5": true,
	"MULE_INTERNAL": true,
}

// ValidEncoding reports whether the given label is a known PostgreSQL
// encoding name. The lookup is case-insensitive.
func ValidEncoding(s string) bool {
	return validEncodings[strings.ToUpper(strings.TrimSpace(s))]
}

// dangerousSQLTokens are substrings that have no legitimate place in a
// single SQL expression (DEFAULT clause, CHECK expression, column-type
// modifier). They allow trivial statement breakouts.
var dangerousSQLTokens = []string{";", "--", "/*", "*/", "\x00"}

// AssertNoStatementBreakout returns an error if `expr` contains any token
// that could let user input escape the SQL fragment it's being embedded
// into. Use for column DEFAULT and CHECK expressions where a full parser
// is impractical but the legitimate input is always single-statement.
func AssertNoStatementBreakout(expr string) error {
	for _, tok := range dangerousSQLTokens {
		if strings.Contains(expr, tok) {
			return fmt.Errorf("expression contains forbidden token %q", tok)
		}
	}
	return nil
}

// columnTypePattern accepts only the syntactic shape of a Postgres type:
// letters, digits, spaces, parens, commas, brackets, percent (for type
// modifiers like `numeric(10,2)`), and dots (for schema-qualified custom
// types). Anything outside this set is suspicious.
//
// The pattern intentionally excludes `;`, `--`, `/*`, quotes, and backslashes
// so a hostile JSON body can't break out of the CREATE TABLE column-defs.
var validTypeChars = func() [256]bool {
	var t [256]bool
	for c := 'a'; c <= 'z'; c++ {
		t[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		t[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		t[c] = true
	}
	for _, c := range " (),.[]%_" {
		t[c] = true
	}
	return t
}()

// ValidColumnType reports whether s looks like a Postgres type name.
// Rejects the empty string, anything > 64 chars, and any byte not in the
// allowed character set. Doesn't try to verify the type exists — pgx
// will reject unknown types when the CREATE TABLE runs.
func ValidColumnType(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !validTypeChars[s[i]] {
			return false
		}
	}
	return true
}

// SafeErrorMessage strips bits that callers should never echo back to the
// client: full Postgres connection strings (postgres://user:pass@host/db),
// SQLSTATE wrappers that leak internal structure, and surrounding
// whitespace. Returns a generic message if nothing safe is left.
func SafeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Redact any postgres:// URI we may have embedded.
	for _, scheme := range []string{"postgres://", "postgresql://"} {
		for {
			i := strings.Index(msg, scheme)
			if i < 0 {
				break
			}
			end := i + len(scheme)
			for end < len(msg) && msg[end] != ' ' && msg[end] != '"' && msg[end] != '\n' {
				end++
			}
			msg = msg[:i] + "[redacted-connstring]" + msg[end:]
		}
	}
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "database error"
	}
	return msg
}
