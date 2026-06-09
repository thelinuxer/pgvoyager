package selfupdate

import (
	"fmt"
	"os"
)

// Injectable seams for testing.
var (
	exePathFn     = exePath
	spawnDetached = realSpawnDetached
	signalSelf    = realSignalSelf
)

// osEnviron is a seam for the platform spawn files (testable).
var osEnviron = os.Environ

// Apply replaces the running executable with the staged binary and relaunches
// it. It renames the staged file over the current executable (atomic on the
// same filesystem; permitted over a running binary on Linux/macOS), spawns the
// new binary detached, and only then signals the current process to exit. If
// the spawn fails the current process is left running so the user can retry.
func Apply(stagedPath string) error {
	exe, err := exePathFn()
	if err != nil {
		return err
	}
	if err := os.Rename(stagedPath, exe); err != nil {
		return fmt.Errorf("selfupdate: swap binary: %w", err)
	}
	if err := spawnDetached(exe); err != nil {
		return fmt.Errorf("selfupdate: relaunch: %w", err)
	}
	return signalSelf()
}

func realSignalSelf() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(os.Interrupt) // desktop main bridges Interrupt/SIGTERM → shutdown
}
