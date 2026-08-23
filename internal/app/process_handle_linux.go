//go:build linux

package app

import (
	"fmt"
	"syscall"
)

const (
	sysPIDFDOpen       = 434
	sysPIDFDSendSignal = 424
)

type stableProcess struct {
	identity processIdentity
	pidfd    int
}

func openStableProcess(identity processIdentity) (*stableProcess, error) {
	matches, err := identityMatches(identity)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errStaleProcessIdentity
	}
	fd, _, errno := syscall.Syscall(sysPIDFDOpen, uintptr(identity.PID), 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return &stableProcess{identity: identity, pidfd: int(fd)}, nil
}

func (process *stableProcess) signal(signal syscall.Signal) error {
	_, _, errno := syscall.Syscall6(sysPIDFDSendSignal, uintptr(process.pidfd), uintptr(signal), 0, 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("signal stable pidfd for %d: %w", process.identity.PID, errno)
	}
	return nil
}

func (process *stableProcess) close() error { return syscall.Close(process.pidfd) }
