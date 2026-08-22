//go:build !darwin && !linux

package app

import (
	"fmt"
	"os/exec"
	"time"
)

func configureProcessGroup(_ *exec.Cmd) {}

func ownedProcessIDs(_ string) ([]int, error) {
	return nil, fmt.Errorf("process ownership is unsupported")
}

func terminateOwnedProcesses(_ string, _ time.Duration, _ int) error {
	return fmt.Errorf("process ownership is unsupported")
}

func enableProcessBoundary() error { return fmt.Errorf("process ownership is unsupported") }
