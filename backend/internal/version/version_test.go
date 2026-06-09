package version

import "testing"

func TestIsDesktopDefault(t *testing.T) {
	// Default build (no ldflag) is not the desktop edition.
	if IsDesktop() {
		t.Fatalf("IsDesktop() = true for default build, want false")
	}
}

func TestIsDesktopWhenSet(t *testing.T) {
	orig := Edition
	t.Cleanup(func() { Edition = orig })
	Edition = "desktop"
	if !IsDesktop() {
		t.Fatalf("IsDesktop() = false when Edition=desktop, want true")
	}
}
