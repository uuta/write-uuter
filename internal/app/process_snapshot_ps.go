//go:build darwin && !cgo

package app

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func psProcessSnapshot() (map[int]processIdentity, error) {
	command := exec.Command("/bin/ps", "-axo", "pid=,ppid=,lstart=")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list process identities: %w", err)
	}
	result := make(map[int]processIdentity)
	psPID := 0
	if command.Process != nil {
		psPID = command.Process.Pid
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 7 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		parent, parentErr := strconv.Atoi(fields[1])
		if pidErr != nil || parentErr != nil || pid <= 0 || pid == psPID {
			continue
		}
		result[pid] = processIdentity{PID: pid, ParentPID: parent, Started: strings.Join(fields[2:], " ")}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func nativeProcessIdentity(pid int) (processIdentity, bool, error) {
	snapshot, err := psProcessSnapshot()
	if err != nil {
		return processIdentity{}, false, err
	}
	identity, found := snapshot[pid]
	return identity, found, nil
}

func childProcessIdentities(parentPID int) ([]processIdentity, error) {
	snapshot, err := psProcessSnapshot()
	if err != nil {
		return nil, err
	}
	var children []processIdentity
	for _, identity := range snapshot {
		if identity.ParentPID == parentPID {
			children = append(children, identity)
		}
	}
	return children, nil
}
