package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplySwapsAndSpawns(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pgvoyager-desktop")
	if err := os.WriteFile(exe, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, ".staged")
	if err := os.WriteFile(staged, []byte("NEW"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExe, origSpawn, origSignal := exePathFn, spawnDetached, signalSelf
	t.Cleanup(func() { exePathFn, spawnDetached, signalSelf = origExe, origSpawn, origSignal })
	exePathFn = func() (string, error) { return exe, nil }
	var spawned string
	spawnDetached = func(p string) error { spawned = p; return nil }
	signaled := false
	signalSelf = func() error { signaled = true; return nil }

	if err := Apply(staged); err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "NEW" {
		t.Fatalf("exe content = %q, want NEW", got)
	}
	if spawned != exe {
		t.Fatalf("spawned %q, want %q", spawned, exe)
	}
	if !signaled {
		t.Fatalf("signalSelf not called")
	}
}

func TestApplyDoesNotSignalWhenSpawnFails(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pgvoyager-desktop")
	_ = os.WriteFile(exe, []byte("OLD"), 0o755)
	staged := filepath.Join(dir, ".staged")
	_ = os.WriteFile(staged, []byte("NEW"), 0o755)

	origExe, origSpawn, origSignal := exePathFn, spawnDetached, signalSelf
	t.Cleanup(func() { exePathFn, spawnDetached, signalSelf = origExe, origSpawn, origSignal })
	exePathFn = func() (string, error) { return exe, nil }
	spawnDetached = func(p string) error { return os.ErrPermission }
	signaled := false
	signalSelf = func() error { signaled = true; return nil }

	if err := Apply(staged); err == nil {
		t.Fatalf("expected error when spawn fails")
	}
	if signaled {
		t.Fatalf("signalSelf must not run when spawn fails")
	}
}
