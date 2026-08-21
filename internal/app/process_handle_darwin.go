//go:build darwin && !cgo

package app

import (
	"fmt"
	"syscall"
)

type stableProcess struct {
	identity processIdentity
}

func openStableProcess(identity processIdentity) (*stableProcess, error) {
	return nil, fmt.Errorf("stable Darwin process signaling requires cgo audit-token support")
}

func (process *stableProcess) signal(signal syscall.Signal) error {
	return fmt.Errorf("stable Darwin process signaling requires cgo audit-token support")
}

func (process *stableProcess) close() error { return nil }
