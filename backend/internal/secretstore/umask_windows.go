//go:build windows

package secretstore

// Windows has no umask — POSIX file permissions don't apply. WithSecretUmask
// is a no-op on this platform; ACLs are the right primitive there but
// PgVoyager's threat model on Windows assumes per-user installs.

const secretUmask = 0

func setUmask(mask int) int { return 0 }
