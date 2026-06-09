package selfupdate

import "strings"

// cleanEnv returns the current environment with PGVOYAGER_PORT removed so the
// relaunched process binds a fresh port instead of inheriting the old one.
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
