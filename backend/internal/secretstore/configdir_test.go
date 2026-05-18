package secretstore

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureCreatesDirWith0700(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX perm bits not meaningful on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "pgvoyager")
	t.Setenv("PGVOYAGER_CONFIG_DIR", target)

	got, err := Ensure()
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if got != target {
		t.Errorf("Ensure returned %q, want %q", got, target)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != DirPerm {
		t.Errorf("dir perm = %#o, want %#o (must not be readable by other accounts)", perm, DirPerm)
	}
}

func TestEnsureTightensExistingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX perm bits not meaningful on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "pgvoyager")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("PGVOYAGER_CONFIG_DIR", target)

	if _, err := Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	info, _ := os.Stat(target)
	if perm := info.Mode().Perm(); perm != DirPerm {
		t.Errorf("Ensure did not tighten existing dir: perm = %#o, want %#o", perm, DirPerm)
	}
}

func TestSecureFileAppliesFilePerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX perm bits not meaningful on Windows")
	}
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("password123"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := SecureFile(path); err != nil {
		t.Fatalf("SecureFile: %v", err)
	}
	info, _ := os.Stat(path)
	if perm := info.Mode().Perm(); perm != FilePerm {
		t.Errorf("file perm = %#o, want %#o", perm, FilePerm)
	}
}

func TestSecureFileMissingIsNoError(t *testing.T) {
	if err := SecureFile(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Errorf("SecureFile on missing path returned %v, want nil", err)
	}
}

func TestShredAndRemoveOverwritesAndDeletes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "creds.json")
	plaintext := []byte(`{"password":"supersecret"}`)
	if err := os.WriteFile(path, plaintext, 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := ShredAndRemove(path); err != nil {
		t.Fatalf("ShredAndRemove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file still exists after shred")
	}
}

func TestShredAndRemoveMissingIsNoError(t *testing.T) {
	if err := ShredAndRemove(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Errorf("ShredAndRemove on missing path returned %v", err)
	}
}
