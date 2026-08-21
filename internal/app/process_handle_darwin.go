//go:build darwin

package app

import (
	"errors"
	"fmt"
	"syscall"
)

type stableProcess struct {
	identity processIdentity
	kqueue   int
}

func openStableProcess(identity processIdentity) (*stableProcess, error) {
	matches, err := identityMatches(identity)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errStaleProcessIdentity
	}
	kqueue, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	event := syscall.Kevent_t{Ident: uint64(identity.PID), Filter: syscall.EVFILT_PROC, Flags: syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR, Fflags: syscall.NOTE_EXIT}
	if _, err := syscall.Kevent(kqueue, []syscall.Kevent_t{event}, nil, nil); err != nil {
		_ = syscall.Close(kqueue)
		return nil, err
	}
	matches, err = identityMatches(identity)
	if err != nil || !matches {
		_ = syscall.Close(kqueue)
		if err != nil {
			return nil, err
		}
		return nil, errStaleProcessIdentity
	}
	return &stableProcess{identity: identity, kqueue: kqueue}, nil
}

func (process *stableProcess) signal(signal syscall.Signal) error {
	events := make([]syscall.Kevent_t, 1)
	timeout := syscall.Timespec{}
	count, err := syscall.Kevent(process.kqueue, nil, events, &timeout)
	if err != nil {
		return err
	}
	if count != 0 {
		return errStaleProcessIdentity
	}
	matches, err := identityMatches(process.identity)
	if err != nil {
		return err
	}
	if !matches {
		return errStaleProcessIdentity
	}
	if err := syscall.Kill(process.identity.PID, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("signal stable pid %d: %w", process.identity.PID, err)
	}
	return nil
}

func (process *stableProcess) close() error { return syscall.Close(process.kqueue) }
