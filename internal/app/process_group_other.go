//go:build !darwin && !linux

package app

import (
	"fmt"
	"os/exec"
	"time"
)

func configureProcessGroup(_ *exec.Cmd) {}

func configureTrackedProcessLaunch(command *exec.Cmd) { configureProcessGroup(command) }

func waitForTrackedProcessLaunch(_ int, _ time.Time) error {
	return fmt.Errorf("tracked process launch is unsupported")
}

func releaseTrackedProcessLaunch(_ int) error {
	return fmt.Errorf("tracked process launch is unsupported")
}

func ownedProcessIDs(_ string) ([]int, error) {
	return nil, fmt.Errorf("process ownership is unsupported")
}

func terminateOwnedProcesses(_ string, _ time.Duration, _ int) error {
	return fmt.Errorf("process ownership is unsupported")
}

func enableProcessBoundary() error { return fmt.Errorf("process ownership is unsupported") }

func terminateProcessGroup(_ int) error { return fmt.Errorf("process ownership is unsupported") }
