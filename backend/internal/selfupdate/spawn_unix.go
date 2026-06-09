//go:build !windows

package selfupdate

import (
	"os/exec"
	"strings"
	"syscall"
)

// realSpawnDetached starts the new binary in its own process group so it
// survives the current process exiting, with a clean port so it binds fresh.
func realSpawnDetached(exe string) error {
	cmd := exec.Command(exe)
	cmd.Env = cleanEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

func cleanEnv() []string {
	out := make([]string, 0, len(osEnviron()))
	for _, kv := range osEnviron() {
		if strings.HasPrefix(kv, "PGVOYAGER_PORT=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
