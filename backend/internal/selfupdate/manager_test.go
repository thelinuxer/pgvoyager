package selfupdate

import (
	"context"
	"errors"
	"testing"
)

func newTestManager() *Manager {
	m := NewManager("1.0.0")
	return m
}

func TestManagerNoUpdate(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v1.0.0", "url", nil }
	m.cycle(context.Background())
	if got := m.Status().Status; got != StatusIdle {
		t.Fatalf("status = %q, want idle", got)
	}
}

func TestManagerDownloadsWhenNewer(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "url", nil }
	m.writable = func() bool { return true }
	m.downloadFn = func(context.Context, string, func(int)) (string, error) { return "/tmp/staged", nil }
	m.cycle(context.Background())
	st := m.Status()
	if st.Status != StatusReady {
		t.Fatalf("status = %q, want ready", st.Status)
	}
	if st.LatestVersion != "2.0.0" {
		t.Fatalf("latest = %q, want 2.0.0", st.LatestVersion)
	}
	if m.staged != "/tmp/staged" {
		t.Fatalf("staged = %q", m.staged)
	}
	if st.NeedsElevation {
		t.Fatalf("NeedsElevation = true for writable install, want false")
	}
}

func TestManagerManualWhenNotWritableAndNoElevation(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "rel-url", nil }
	m.writable = func() bool { return false }
	m.canElevate = func() bool { return false }
	called := false
	m.downloadFn = func(context.Context, string, func(int)) (string, error) { called = true; return "", nil }
	m.cycle(context.Background())
	if m.Status().Status != StatusManual {
		t.Fatalf("status = %q, want manual", m.Status().Status)
	}
	if called {
		t.Fatalf("downloadFn must not run when not writable and not elevatable")
	}
}

func TestManagerElevatedWhenNotWritable(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "url", nil }
	m.writable = func() bool { return false }
	m.canElevate = func() bool { return true }
	m.downloadFn = func(context.Context, string, func(int)) (string, error) { return "/tmp/staged", nil }
	m.cycle(context.Background())
	st := m.Status()
	if st.Status != StatusReady {
		t.Fatalf("status = %q, want ready", st.Status)
	}
	if !st.NeedsElevation {
		t.Fatalf("NeedsElevation = false, want true")
	}
}

func TestManagerErrorOnDownloadFailure(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "url", nil }
	m.writable = func() bool { return true }
	m.downloadFn = func(context.Context, string, func(int)) (string, error) { return "", errors.New("boom") }
	m.cycle(context.Background())
	if m.Status().Status != StatusError {
		t.Fatalf("status = %q, want error", m.Status().Status)
	}
}

func TestManagerRestartGuards(t *testing.T) {
	m := newTestManager()
	if err := m.Restart(); err == nil {
		t.Fatalf("Restart with no staged update should error")
	}
}

func TestManagerRestartCallsApply(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "url", nil }
	m.writable = func() bool { return true }
	m.downloadFn = func(context.Context, string, func(int)) (string, error) { return "/tmp/staged", nil }
	m.cycle(context.Background())

	var applied string
	m.applyFn = func(p string) error { applied = p; return nil }
	if err := m.Restart(); err != nil {
		t.Fatalf("Restart error: %v", err)
	}
	if applied != "/tmp/staged" {
		t.Fatalf("applyFn called with %q, want /tmp/staged", applied)
	}
}

func TestManagerRestartSetsErrorOnApplyFailure(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "url", nil }
	m.writable = func() bool { return true }
	m.downloadFn = func(context.Context, string, func(int)) (string, error) { return "/tmp/staged", nil }
	m.cycle(context.Background())

	m.applyFn = func(string) error { return errors.New("rename failed") }
	if err := m.Restart(); err == nil {
		t.Fatalf("expected error from Restart")
	}
	if m.Status().Status != StatusError {
		t.Fatalf("status = %q, want error", m.Status().Status)
	}
}
