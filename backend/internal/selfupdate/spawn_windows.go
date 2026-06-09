//go:build windows

package selfupdate

import (
	"os/exec"
	"strings"
)

func realSpawnDetached(exe string) error {
	cmd := exec.Command(exe)
	cmd.Env = cleanEnv()
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
