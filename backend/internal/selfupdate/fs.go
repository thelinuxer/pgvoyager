package selfupdate

import (
	"os"
	"path/filepath"
)

// exePath returns the absolute, symlink-resolved path of the running binary.
func exePath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved, nil
	}
	return p, nil
}

// Writable reports whether the running executable's directory can be written
// by the current user (a precondition for in-place self-replace).
func Writable() bool {
	exe, err := exePath()
	if err != nil {
		return false
	}
	return writableDir(filepath.Dir(exe))
}

func writableDir(dir string) bool {
	f, err := os.CreateTemp(dir, ".pgvoyager-wtest-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
