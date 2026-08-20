//go:build !darwin && !linux

package app

import (
	"fmt"
	"os/exec"
	"time"
)

func configureProcessGroup(_ *exec.Cmd) {}

func processGroupExists(pgid int) (bool, error) {
	return false, fmt.Errorf("process-group ownership is unsupported for process group %d", pgid)
}

func terminateProcessGroup(pgid int, _ time.Duration) error {
	return fmt.Errorf("process-group termination is unsupported for process group %d", pgid)
}
