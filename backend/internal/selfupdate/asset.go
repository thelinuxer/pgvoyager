// Package selfupdate implements desktop-edition in-place updates: resolving
// the current platform's release asset, verifying its SHA256, and atomically
// replacing the running executable.
package selfupdate

import (
	"fmt"
	"runtime"
)

// AssetName returns the GitHub release asset filename for the desktop binary
// on the current platform (e.g. "pgvoyager-desktop-linux-amd64").
func AssetName() (string, error) {
	return assetNameFor(runtime.GOOS, runtime.GOARCH)
}

func assetNameFor(goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return "", fmt.Errorf("selfupdate: unsupported OS %q", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("selfupdate: unsupported architecture %q", goarch)
	}
	name := fmt.Sprintf("pgvoyager-desktop-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name, nil
}
