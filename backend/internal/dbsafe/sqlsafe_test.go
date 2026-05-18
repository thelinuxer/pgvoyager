package dbsafe

import (
	"errors"
	"strings"
	"testing"
)

func TestQuoteIdentBasic(t *testing.T) {
	q, err := QuoteIdent("users")
	if err != nil || q != `"users"` {
		t.Errorf(`QuoteIdent("users") = %q, %v; want "users", nil`, q, err)
	}
}

func TestQuoteIdentDoublesEmbeddedQuote(t *testing.T) {
	q, err := QuoteIdent(`weird"name`)
	if err != nil || q != `"weird""name"` {
		t.Errorf(`QuoteIdent(weird"name) = %q, %v; want "weird""name", nil`, q, err)
	}
}

func TestQuoteIdentRejectsNUL(t *testing.T) {
	_, err := QuoteIdent("users\x00; DROP TABLE x;--")
	if !errors.Is(err, ErrInvalidIdentifier) {
		t.Errorf("QuoteIdent with NUL returned err=%v, want ErrInvalidIdentifier", err)
	}
}

func TestQuoteStringEscapesQuotesAndBackslashes(t *testing.T) {
	q, err := QuoteString(`O'Reilly\nope`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := `E'O''Reilly\\nope'`
	if q != want {
		t.Errorf("QuoteString = %q, want %q", q, want)
	}
}

func TestQuoteStringRejectsNUL(t *testing.T) {
	_, err := QuoteString("ok\x00bad")
	if err == nil {
		t.Errorf("QuoteString with NUL returned nil err, want error")
	}
}

func TestCanonicalFKAction(t *testing.T) {
	cases := map[string]string{
		"cascade":     "CASCADE",
		"NO ACTION":   "NO ACTION",
		"  set  null": "SET NULL",
		"set default": "SET DEFAULT",
		"restrict":    "RESTRICT",
	}
	for in, want := range cases {
		got, err := CanonicalFKAction(in)
		if err != nil || got != want {
			t.Errorf("CanonicalFKAction(%q) = %q, %v; want %q, nil", in, got, err, want)
		}
	}
}

func TestCanonicalFKActionRejectsInjection(t *testing.T) {
	bad := []string{
		"NO ACTION; DROP TABLE users",
		"CASCADE--",
		"CASCADE/*",
		"DROP",
		"",
	}
	for _, s := range bad {
		if _, err := CanonicalFKAction(s); err == nil {
			t.Errorf("CanonicalFKAction(%q) accepted hostile input", s)
		}
	}
}

func TestValidEncoding(t *testing.T) {
	for _, ok := range []string{"UTF8", "utf8", "Latin1", "win1252", "  SQL_ASCII  "} {
		if !ValidEncoding(ok) {
			t.Errorf("ValidEncoding(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "UTF9", "'; DROP", "../etc/passwd"} {
		if ValidEncoding(bad) {
			t.Errorf("ValidEncoding(%q) = true, want false", bad)
		}
	}
}

func TestAssertNoStatementBreakout(t *testing.T) {
	for _, ok := range []string{"1", "CURRENT_TIMESTAMP", "now()", "(id > 0)", "x = 'a'"} {
		if err := AssertNoStatementBreakout(ok); err != nil {
			t.Errorf("AssertNoStatementBreakout(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{
		"1; DROP TABLE x",
		"1--comment",
		"x /* hi */",
		"x */ ok",
		"x\x00",
	} {
		if err := AssertNoStatementBreakout(bad); err == nil {
			t.Errorf("AssertNoStatementBreakout(%q) accepted hostile input", bad)
		}
	}
}

func TestValidColumnType(t *testing.T) {
	for _, ok := range []string{
		"int", "INTEGER", "varchar(255)", "numeric(10,2)",
		"text[]", "int[][]", "timestamp with time zone",
	} {
		if !ValidColumnType(ok) {
			t.Errorf("ValidColumnType(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{
		"",
		"int); DROP TABLE x; --",
		`varchar(255) DEFAULT 'x' /*`,
		"int" + strings.Repeat("a", 80),
		"int\\backslash",
		"int'",
		`int"`,
	} {
		if ValidColumnType(bad) {
			t.Errorf("ValidColumnType(%q) = true, want false", bad)
		}
	}
}

func TestSafeErrorMessageRedactsConnString(t *testing.T) {
	err := errors.New(`failed to dial: cannot parse postgres://user:supersecret@db.example.com:5432/app sslmode=disable`)
	msg := SafeErrorMessage(err)
	if strings.Contains(msg, "supersecret") {
		t.Errorf("SafeErrorMessage leaked password: %s", msg)
	}
	if !strings.Contains(msg, "[redacted-connstring]") {
		t.Errorf("SafeErrorMessage did not redact: %s", msg)
	}
}

func TestSafeErrorMessageNilAndEmpty(t *testing.T) {
	if SafeErrorMessage(nil) != "" {
		t.Errorf("nil err should give empty string")
	}
	if SafeErrorMessage(errors.New("   ")) != "database error" {
		t.Errorf("whitespace err should give fallback")
	}
}
