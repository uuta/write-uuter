//go:build darwin || linux

package app

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

func configureProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func processGroupExists(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, fmt.Errorf("invalid process group %d", pgid)
	}
	err := syscall.Kill(-pgid, 0)
	switch {
	case err == nil, errors.Is(err, syscall.EPERM):
		return true, nil
	case errors.Is(err, syscall.ESRCH):
		return false, nil
	default:
		return false, err
	}
}

func terminateProcessGroup(pgid int, timeout time.Duration) error {
	exists, err := processGroupExists(pgid)
	if err != nil || !exists {
		return err
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	softDeadline := time.Now().Add(timeout / 4)
	for time.Now().Before(softDeadline) {
		exists, err = processGroupExists(pgid)
		if err != nil || !exists {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		exists, err = processGroupExists(pgid)
		if err != nil || !exists {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("process group %d still exists after termination", pgid)
}
