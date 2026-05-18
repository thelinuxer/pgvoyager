// Package secretstore centralizes filesystem locations and permissions for
// PgVoyager's at-rest secrets: the SQLite DB containing plaintext connection
// passwords, saved queries, and any other config the user wouldn't want
// another local account to read.
package secretstore

import (
	"fmt"
	"os"
	"path/filepath"
)

// DirPerm is the mode every PgVoyager config directory must use. 0700
// prevents `other` and `group` accounts on a multi-user host from reading
// the credential DB. Previously these dirs were created 0755 — anyone with
// shell on the box could open them.
const DirPerm os.FileMode = 0o700

// FilePerm is the mode every secret file (SQLite DB, queries.json) must use.
// SQLite previously inherited the default umask (0644 = world-readable).
const FilePerm os.FileMode = 0o600

// Path returns the absolute config directory PgVoyager should use, honoring
// the PGVOYAGER_CONFIG_DIR override (used by E2E tests) before falling back
// to os.UserConfigDir + "pgvoyager", and finally os.TempDir if neither is
// available.
func Path() string {
	if d := os.Getenv("PGVOYAGER_CONFIG_DIR"); d != "" {
		return d
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "pgvoyager")
	}
	return filepath.Join(os.TempDir(), "pgvoyager")
}

// Ensure creates the config directory if missing and tightens its mode to
// DirPerm. Returns the directory path on success. Idempotent: re-applies
// the chmod on every call so an upgraded install gets the tightened perms
// even if the dir already existed under the old default.
func Ensure() (string, error) {
	dir := Path()
	if err := os.MkdirAll(dir, DirPerm); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.Chmod(dir, DirPerm); err != nil {
		return "", fmt.Errorf("chmod config dir: %w", err)
	}
	return dir, nil
}

// SecureFile applies FilePerm to path. Use after creating any file that
// holds a secret (the SQLite DB, MCP-config temp files, queries.json).
// Ignores ErrNotExist so callers can call it unconditionally.
func SecureFile(path string) error {
	err := os.Chmod(path, FilePerm)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}

// WithSecretUmask installs an `umask 0077` for the duration of fn and
// restores the previous value afterwards. Use around code paths that
// create secret files (sqlite DB + its WAL/journal sidecars, temp
// credential files) so they're born 0600 instead of needing a post-
// creation chmod. Post-creation chmod is racy with libsqlite's inode-
// movement detection — fresh files trigger SQLITE_READONLY_DBMOVED
// on the next write under modernc.org/sqlite.
func WithSecretUmask(fn func() error) error {
	old := setUmask(secretUmask)
	defer setUmask(old)
	return fn()
}

// ShredAndRemove best-effort overwrites a file with zeros before deleting
// it. Used for the connections.json migration backup, which contained
// plaintext passwords and was previously left on disk indefinitely.
// Failures are non-fatal — the caller still removes the file.
func ShredAndRemove(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode().IsRegular() && info.Size() > 0 {
		// Best-effort overwrite. On copy-on-write filesystems this
		// doesn't actually clear the blocks, but it's better than
		// leaving the plaintext readable.
		zeros := make([]byte, info.Size())
		_ = os.WriteFile(path, zeros, FilePerm)
	}
	return os.Remove(path)
}
