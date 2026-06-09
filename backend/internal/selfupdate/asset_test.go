package selfupdate

import (
	"runtime"
	"testing"
)

func TestAssetNameCurrentPlatform(t *testing.T) {
	name, err := AssetName()
	if err != nil {
		t.Fatalf("AssetName() error: %v", err)
	}
	want := "pgvoyager-desktop-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if name != want {
		t.Fatalf("AssetName() = %q, want %q", name, want)
	}
}

func TestAssetNameForExplicit(t *testing.T) {
	got, err := assetNameFor("linux", "amd64")
	if err != nil || got != "pgvoyager-desktop-linux-amd64" {
		t.Fatalf("assetNameFor(linux,amd64) = %q, %v", got, err)
	}
	got, err = assetNameFor("windows", "amd64")
	if err != nil || got != "pgvoyager-desktop-windows-amd64.exe" {
		t.Fatalf("assetNameFor(windows,amd64) = %q, %v", got, err)
	}
	if _, err := assetNameFor("plan9", "amd64"); err == nil {
		t.Fatalf("assetNameFor(plan9) expected error, got nil")
	}
	if _, err := assetNameFor("linux", "mips"); err == nil {
		t.Fatalf("assetNameFor(mips) expected error, got nil")
	}
}
