package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWritableDirTrue(t *testing.T) {
	dir := t.TempDir()
	if !writableDir(dir) {
		t.Fatalf("writableDir(%s) = false, want true", dir)
	}
}

func TestWritableDirFalse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based read-only dir not reliable on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	dir := t.TempDir()
	ro := filepath.Join(dir, "ro")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatal(err)
	}
	if writableDir(ro) {
		t.Fatalf("writableDir(read-only) = true, want false")
	}
}
