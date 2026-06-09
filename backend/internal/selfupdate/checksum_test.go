package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
)

const sums = "" +
	"aaa111  pgvoyager-desktop-linux-arm64\n" +
	"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  pgvoyager-desktop-linux-amd64\n"

func TestSHA256FromSums(t *testing.T) {
	got, err := sha256FromSums(sums, "pgvoyager-desktop-linux-amd64")
	if err != nil || got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("sha256FromSums = %q, %v", got, err)
	}
	if _, err := sha256FromSums(sums, "missing"); err == nil {
		t.Fatalf("expected error for missing asset")
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "blob")
	if err := os.WriteFile(p, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(p, "pgvoyager-desktop-linux-amd64", sums); err != nil {
		t.Fatalf("VerifySHA256 (good) error: %v", err)
	}
	if err := os.WriteFile(p, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(p, "pgvoyager-desktop-linux-amd64", sums); err == nil {
		t.Fatalf("VerifySHA256 (bad) expected mismatch error")
	}
}
