//go:build darwin && !cgo

package app

import (
	"fmt"
	"syscall"
)

// A Darwin production binary must use libproc audit tokens for stable process
// signaling. Keep the build failure explicit instead of producing a binary
// that starts runs but can never complete cleanup safely.
var _ = writeUuterDarwinBuildRequiresCGO

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
