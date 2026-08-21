//go:build darwin || linux

package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func configureProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func ownedProcessIDs(token string) ([]int, error) {
	if token == "" {
		return nil, fmt.Errorf("empty process ownership token")
	}
	command := exec.Command("/bin/ps", "eww", "-axo", "pid=,command=")
	command.Env = environmentWithoutOwnershipToken()
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list owned processes: %w", err)
	}
	var pids []int
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, token) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pid, parseErr := strconv.Atoi(fields[0])
		if parseErr == nil && pid > 1 && pid != os.Getpid() {
			pids = append(pids, pid)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return pids, nil
}

func processHasOwnership(pid int, token string) (bool, error) {
	command := exec.Command("/bin/ps", "eww", "-p", strconv.Itoa(pid), "-o", "command=")
	command.Env = environmentWithoutOwnershipToken()
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(output), token), nil
}

func environmentWithoutOwnershipToken() []string {
	environment := os.Environ()
	filtered := environment[:0]
	for _, item := range environment {
		if !strings.HasPrefix(item, "WRITE_UUTER_OWNERSHIP_TOKEN=") {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func signalOwnedProcesses(token string, signal syscall.Signal) error {
	pids, err := ownedProcessIDs(token)
	if err != nil {
		return err
	}
	var failures []error
	for _, pid := range pids {
		owned, verifyErr := processHasOwnership(pid, token)
		if verifyErr != nil {
			failures = append(failures, verifyErr)
			continue
		}
		if !owned {
			continue
		}
		if err := syscall.Kill(pid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
			failures = append(failures, fmt.Errorf("signal owned pid %d: %w", pid, err))
		}
	}
	return errors.Join(failures...)
}

func terminateOwnedProcesses(token string, timeout time.Duration) error {
	if err := signalOwnedProcesses(token, syscall.SIGTERM); err != nil {
		return err
	}
	softDeadline := time.Now().Add(timeout / 4)
	for time.Now().Before(softDeadline) {
		pids, err := ownedProcessIDs(token)
		if err != nil || len(pids) == 0 {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := signalOwnedProcesses(token, syscall.SIGKILL); err != nil {
			return err
		}
		pids, err := ownedProcessIDs(token)
		if err != nil || len(pids) == 0 {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	pids, err := ownedProcessIDs(token)
	if err != nil {
		return err
	}
	return fmt.Errorf("owned processes remain after termination: %v", pids)
}
