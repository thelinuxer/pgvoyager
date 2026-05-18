package chromelaunch

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindHonorsEnvOverride(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-browser")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("PGVOYAGER_BROWSER", fake)

	got, err := Find()
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != fake {
		t.Errorf("Find = %q, want %q", got, fake)
	}
}

func TestFindRejectsMissingOverride(t *testing.T) {
	t.Setenv("PGVOYAGER_BROWSER", filepath.Join(t.TempDir(), "does-not-exist"))
	if _, err := Find(); err == nil {
		t.Errorf("Find should fail when PGVOYAGER_BROWSER points at a missing path")
	}
}

func TestOptionsAppClassIncluded(t *testing.T) {
	// Sanity: AppClass is honored on Linux. On other OSes the field is
	// silently ignored by Chrome; this test just guards against the
	// struct field being dropped.
	if runtime.GOOS != "linux" {
		t.Skip("AppClass is X11-specific")
	}
	opt := Options{AppClass: "PgVoyager"}
	if opt.AppClass != "PgVoyager" {
		t.Errorf("AppClass round-trip broken")
	}
}
