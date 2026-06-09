//go:build linux

package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// pkexecPath returns the absolute path to pkexec, preferring known fixed
// locations over $PATH to prevent a user-writable PATH entry from shadowing it.
func pkexecPath() (string, error) {
	for _, p := range []string{"/usr/bin/pkexec", "/bin/pkexec"} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}
	return exec.LookPath("pkexec")
}

// canElevate reports whether a GUI privilege-escalation path is available:
// pkexec (polkit) plus coreutils `install` for the destination-safe copy.
func canElevate() bool {
	if _, err := pkexecPath(); err != nil {
		return false
	}
	if _, err := exec.LookPath("install"); err != nil {
		return false
	}
	return true
}

// elevatedReplace installs the staged binary over exe as root via pkexec
// (polkit shows a graphical auth dialog). It first refuses anything but a
// regular file owned by the current user. staged lives in a 0700 user-owned
// cache dir, so no other local user can write or swap it — this closes the
// cross-user symlink/swap (TOCTOU) window against the root-level copy. The
// staged bytes were SHA256-verified against the release SHA256SUMS at download
// time. `install` is used instead of a shell `cp -f` + `chmod`: it does not
// follow a symlink at the destination, sets owner/mode atomically in one step,
// and avoids a shell entirely (no injection surface).
//
// A same-uid swap of the staged file is not a privilege boundary (that user
// already controls what they run and is the one authenticating); defending
// against a compromised-but-authenticating user account would require a signed
// payload re-verified inside a root helper, which is tracked as future work.
func elevatedReplace(staged, exe string) error {
	fi, err := os.Lstat(staged)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return fmt.Errorf("selfupdate: staged path is not a regular file")
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Geteuid() {
		return fmt.Errorf("selfupdate: staged file not owned by current user")
	}

	installPath, err := exec.LookPath("install")
	if err != nil {
		return err
	}
	pk, err := pkexecPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(pk, installPath, "-m", "0755", "-o", "root", "-g", "root", staged, exe)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("selfupdate: elevated replace failed: %w: %s", err, out)
	}
	return nil
}
