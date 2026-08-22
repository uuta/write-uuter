//go:build linux

package app

import (
	"fmt"
	"syscall"
)

// Linux subreaper adoption is the kernel boundary for descendants which call
// setsid/setpgid and outlive their immediate parent. Adopted descendants remain
// children of the runner and are therefore observable by the ownership tracker;
// this avoids relying on a sampling window around parent exit.
func enableProcessBoundary() error {
	const prSetChildSubreaper = 36
	if _, _, errno := syscall.Syscall6(syscall.SYS_PRCTL, prSetChildSubreaper, 1, 0, 0, 0, 0); errno != 0 {
		return fmt.Errorf("enable child-subreaper boundary: %w", errno)
	}
	return nil
}
