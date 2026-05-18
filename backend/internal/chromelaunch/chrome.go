// Package chromelaunch finds a Chromium-family browser on the host and
// launches it in `--app` mode pointing at a given URL. Lorca was the
// obvious dependency for this, but it hardcodes `--enable-automation`
// which makes Chrome show the "Chrome is being controlled by automated
// test software" infobar. We don't need any of lorca's DevTools / JS-
// bridge features — just a window — so a ~50-line launcher avoids the
// banner and the dep.
package chromelaunch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// candidateBinaries lists the executable names we try, in order, on each
// platform. Names are matched against PATH via exec.LookPath; absolute
// paths are also accepted directly.
var candidateBinaries = map[string][]string{
	"linux": {
		"google-chrome", "google-chrome-stable",
		"chromium", "chromium-browser",
		"microsoft-edge", "microsoft-edge-stable",
		"brave-browser",
	},
	"darwin": {
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	},
	"windows": {
		// LookPath handles these via PATHEXT.
		"chrome",
		"msedge",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	},
}

// Find returns the absolute path to a usable Chromium-family browser,
// honoring the PGVOYAGER_BROWSER env override before falling back to the
// per-OS candidate list. Returns an empty string + error if nothing is
// installed.
func Find() (string, error) {
	if override := os.Getenv("PGVOYAGER_BROWSER"); override != "" {
		if _, err := os.Stat(override); err == nil {
			return override, nil
		}
		if path, err := exec.LookPath(override); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("PGVOYAGER_BROWSER=%s: not found", override)
	}
	for _, name := range candidateBinaries[runtime.GOOS] {
		if filepath.IsAbs(name) {
			if _, err := os.Stat(name); err == nil {
				return name, nil
			}
			continue
		}
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no Chromium-family browser found (install Chrome, Chromium, Edge, or Brave; or set PGVOYAGER_BROWSER)")
}

// Options configure a single `--app` window launch.
type Options struct {
	// URL is the page the window should open against.
	URL string
	// Width and Height seed the initial window size.
	Width, Height int
	// AppClass becomes the X11 WM_CLASS on Linux so a matching
	// .desktop entry's StartupWMClass field can resolve the dock icon.
	// Ignored elsewhere.
	AppClass string
	// Extra is appended verbatim to the Chrome command line.
	Extra []string
}

// Run launches Chrome in `--app` mode and blocks until the window closes
// or ctx is cancelled. The Chrome process inherits Stdout/Stderr so its
// occasional warnings reach the operator's terminal.
//
// We deliberately do NOT pass `--enable-automation`; that flag is what
// triggers the "Chrome is being controlled by automated test software"
// banner. We also drop `--no-default-browser-check` and similar test-
// only flags lorca hardcoded.
func Run(ctx context.Context, chromePath string, opt Options) error {
	profile, err := os.MkdirTemp("", "pgvoyager-chrome-*")
	if err != nil {
		return fmt.Errorf("temp profile dir: %w", err)
	}
	defer os.RemoveAll(profile)

	args := []string{
		"--app=" + opt.URL,
		"--user-data-dir=" + profile,
		fmt.Sprintf("--window-size=%d,%d", opt.Width, opt.Height),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-default-apps",
		"--disable-translate",
		"--disable-features=TranslateUI,InfoBars",
		"--disable-popup-blocking",
	}
	if opt.AppClass != "" {
		// On X11, --class sets WM_CLASS, which is what the
		// installed .desktop entry's StartupWMClass field matches
		// against to attach the correct icon. On Wayland, --class
		// is ignored (the app_id is derived from the .desktop
		// filename), so we also force Chrome to run under XWayland
		// via --ozone-platform=x11. Without this, Ubuntu 24.04 +
		// GNOME Shell shows the dock entry under Chrome's icon
		// instead of the PgVoyager elephant.
		if runtime.GOOS == "linux" {
			args = append(args, "--ozone-platform=x11")
		}
		args = append(args, "--class="+opt.AppClass)
	}
	args = append(args, opt.Extra...)

	cmd := exec.CommandContext(ctx, chromePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start chrome: %w", err)
	}
	// Wait returns when the user closes the window OR ctx is cancelled
	// (CommandContext kills the process on cancel).
	if err := cmd.Wait(); err != nil {
		// Ignore "signal: killed" — that's the ctx-cancel path.
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	return nil
}
