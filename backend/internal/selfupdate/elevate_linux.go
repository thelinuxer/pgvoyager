//go:build linux

package selfupdate

import (
	"fmt"
	"os/exec"
	"strings"
)

// canElevate reports whether a GUI privilege-escalation tool is available.
func canElevate() bool {
	_, err := exec.LookPath("pkexec")
	return err == nil
}

// elevatedReplace copies staged over exe as root via pkexec (polkit shows a
// graphical auth dialog) and sets it executable. Returns an error if the user
// cancels or authentication fails.
func elevatedReplace(staged, exe string) error {
	script := fmt.Sprintf("cp -f %s %s && chmod 755 %s",
		shellQuote(staged), shellQuote(exe), shellQuote(exe))
	cmd := exec.Command("pkexec", "/bin/sh", "-c", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("selfupdate: elevated replace failed: %w: %s", err, out)
	}
	return nil
}

// shellQuote single-quotes s for safe use in a /bin/sh -c command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
