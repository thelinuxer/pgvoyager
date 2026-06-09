//go:build windows

package selfupdate

import (
	"os/exec"
)

func realSpawnDetached(exe string) error {
	cmd := exec.Command(exe)
	cmd.Env = cleanEnv()
	return cmd.Start()
}
