//go:build !windows

package secretstore

import "syscall"

// secretUmask masks group + other entirely. Combined with secretstore's
// DirPerm (0700) this means any file PgVoyager creates while
// WithSecretUmask is active is born 0600 / 0700.
const secretUmask = 0o077

func setUmask(mask int) int {
	return syscall.Umask(mask)
}
