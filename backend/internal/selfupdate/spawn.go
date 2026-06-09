package selfupdate

import "strings"

// strippedEnvPrefixes lists environment variable key prefixes (in KEY= form)
// that must not be inherited by the relaunched process. PGVOYAGER_PORT is
// dropped so the new process binds a fresh port. The dynamic-linker injection
// vars are dropped so a planted preload cannot survive a privileged relaunch.
var strippedEnvPrefixes = []string{
	"PGVOYAGER_PORT=",
	"LD_PRELOAD=",
	"LD_LIBRARY_PATH=",
	"DYLD_INSERT_LIBRARIES=",
	"DYLD_LIBRARY_PATH=",
}

// cleanEnv returns the current environment with strippedEnvPrefixes removed.
func cleanEnv() []string {
	out := make([]string, 0, len(osEnviron()))
	for _, kv := range osEnviron() {
		stripped := false
		for _, prefix := range strippedEnvPrefixes {
			if strings.HasPrefix(kv, prefix) {
				stripped = true
				break
			}
		}
		if !stripped {
			out = append(out, kv)
		}
	}
	return out
}
