//go:build linux

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func boundaryProcessIdentities(_ string) ([]processIdentity, error) { return nil, nil }

func nativeProcessIdentity(pid int) (processIdentity, bool, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if errors.Is(err, os.ErrNotExist) {
		return processIdentity{}, false, nil
	}
	if err != nil {
		return processIdentity{}, false, err
	}
	closing := strings.LastIndexByte(string(data), ')')
	if closing < 0 {
		return processIdentity{}, false, fmt.Errorf("invalid /proc stat for pid %d", pid)
	}
	fields := strings.Fields(string(data[closing+1:]))
	if len(fields) < 20 {
		return processIdentity{}, false, fmt.Errorf("short /proc stat for pid %d", pid)
	}
	parent, err := strconv.Atoi(fields[1])
	if err != nil {
		return processIdentity{}, false, err
	}
	return processIdentity{PID: pid, ParentPID: parent, Started: fields[19]}, true, nil
}

func childProcessIdentities(parentPID int) ([]processIdentity, error) {
	path := filepath.Join("/proc", strconv.Itoa(parentPID), "task", strconv.Itoa(parentPID), "children")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var children []processIdentity
	for _, field := range strings.Fields(string(data)) {
		pid, parseErr := strconv.Atoi(field)
		if parseErr != nil {
			return nil, parseErr
		}
		identity, found, identityErr := nativeProcessIdentity(pid)
		if identityErr != nil {
			return nil, identityErr
		}
		if found {
			children = append(children, identity)
		}
	}
	return children, nil
}
