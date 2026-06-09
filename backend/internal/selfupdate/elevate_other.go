//go:build !linux

package selfupdate

import "fmt"

func canElevate() bool { return false }

func elevatedReplace(staged, exe string) error {
	return fmt.Errorf("selfupdate: privileged update not supported on this platform yet")
}
