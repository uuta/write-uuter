//go:build darwin || linux

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

const processManifestVersion = 1

type processIdentity struct {
	PID       int    `json:"pid"`
	ParentPID int    `json:"parent_pid"`
	Started   string `json:"started"`
}

type processManifest struct {
	SchemaVersion int               `json:"schema_version"`
	Processes     []processIdentity `json:"processes"`
}

type processTracker struct {
	path       string
	rootPID    int
	boundaries []string
	mu         sync.Mutex
	processes  map[int]processIdentity
	stop       chan struct{}
	done       chan struct{}
}

func configureProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// configureTrackedProcessLaunch makes the new process stop at exec before any
// runner code executes. The controller can therefore establish stable root
// ownership before the runner can fork, detach, or exit.
func configureTrackedProcessLaunch(command *exec.Cmd) {
	configureProcessGroup(command)
	command.SysProcAttr.Ptrace = true
}

func waitForTrackedProcessLaunch(pid int, deadline time.Time) error {
	for time.Now().Before(deadline) {
		var status syscall.WaitStatus
		waited, err := syscall.Wait4(pid, &status, syscall.WNOHANG|syscall.WUNTRACED, nil)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if err != nil {
			return fmt.Errorf("wait for tracked process launch: %w", err)
		}
		if waited == 0 {
			time.Sleep(time.Millisecond)
			continue
		}
		if waited != pid {
			continue
		}
		if status.Stopped() {
			return nil
		}
		return fmt.Errorf("tracked process exited before its launch boundary")
	}
	return fmt.Errorf("timed out waiting for tracked process launch boundary")
}

func releaseTrackedProcessLaunch(pid int) error {
	if err := syscall.PtraceDetach(pid); err != nil {
		return fmt.Errorf("release tracked process launch boundary: %w", err)
	}
	return nil
}

func startProcessTracker(path string, rootPID int, boundaries ...string) (*processTracker, error) {
	if err := enableProcessBoundary(); err != nil {
		return nil, err
	}
	tracker := &processTracker{
		path: path, rootPID: rootPID, boundaries: append([]string(nil), boundaries...), processes: make(map[int]processIdentity),
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	if err := tracker.refresh(); err != nil {
		return nil, err
	}
	go func() {
		defer close(tracker.done)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-tracker.stop:
				return
			case <-ticker.C:
				_ = tracker.refresh()
			}
		}
	}()
	return tracker, nil
}

func (tracker *processTracker) close() {
	select {
	case <-tracker.stop:
	default:
		close(tracker.stop)
	}
	<-tracker.done
}

func (tracker *processTracker) refresh() error {
	tracker.mu.Lock()
	dirty := false
	if len(tracker.processes) == 0 {
		root, found, err := currentProcessIdentity(tracker.rootPID)
		if err != nil {
			tracker.mu.Unlock()
			return err
		}
		if !found {
			tracker.mu.Unlock()
			return fmt.Errorf("ownership root pid %d disappeared", tracker.rootPID)
		}
		tracker.processes[root.PID] = root
		dirty = true
	}
	if dirty {
		if err := writeProcessManifest(tracker.path, tracker.processes); err != nil {
			tracker.mu.Unlock()
			return err
		}
	}
	tracker.mu.Unlock()
	return tracker.refreshDescendants()
}

func (tracker *processTracker) refreshBoundaries() error {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	dirty := false
	// A private open-file boundary is inherited by ordinary runner children and
	// remains discoverable after reparenting or setsid. Scan it at teardown,
	// rather than in the millisecond ancestry poll: proc_listpidspath is a
	// system-wide query and cannot be allowed to delay the hard runner deadline.
	for _, boundary := range tracker.boundaries {
		identities, err := boundaryProcessIdentities(boundary)
		if err != nil {
			return err
		}
		for _, identity := range identities {
			if identity.PID == os.Getpid() {
				continue
			}
			if previous, found := tracker.processes[identity.PID]; !found || previous.Started != identity.Started {
				tracker.processes[identity.PID] = identity
				dirty = true
			}
		}
	}
	if dirty {
		return writeProcessManifest(tracker.path, tracker.processes)
	}
	return nil
}

func (tracker *processTracker) refreshDescendants() error {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	dirty := false
	owned := make(map[int]bool, len(tracker.processes))
	for pid, identity := range tracker.processes {
		current, found, err := currentProcessIdentity(pid)
		if err != nil {
			return err
		}
		if found && current.Started == identity.Started {
			owned[pid] = true
		}
	}
	queue := make([]int, 0, len(owned))
	for pid := range owned {
		queue = append(queue, pid)
	}
	for len(queue) != 0 {
		parent := queue[0]
		queue = queue[1:]
		children, err := childProcessIdentities(parent)
		if err != nil {
			return err
		}
		for _, identity := range children {
			if owned[identity.PID] {
				continue
			}
			owned[identity.PID] = true
			if previous, found := tracker.processes[identity.PID]; !found || previous.Started != identity.Started {
				tracker.processes[identity.PID] = identity
				dirty = true
			}
			queue = append(queue, identity.PID)
		}
	}
	if dirty {
		return writeProcessManifest(tracker.path, tracker.processes)
	}
	return nil
}

func (tracker *processTracker) waitFor(pid int, deadline time.Time) (processIdentity, error) {
	for time.Now().Before(deadline) {
		if err := tracker.refresh(); err != nil {
			return processIdentity{}, err
		}
		tracker.mu.Lock()
		identity, found := tracker.processes[pid]
		tracker.mu.Unlock()
		if found {
			return identity, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return processIdentity{}, fmt.Errorf("pid %d was not captured by ownership tracker", pid)
}

func (tracker *processTracker) terminate(timeout time.Duration) error {
	return tracker.terminateUntil(time.Now().Add(timeout))
}

func (tracker *processTracker) terminateUntil(deadline time.Time) error {
	timeout := time.Until(deadline)
	if timeout < 0 {
		timeout = 0
	}
	tracker.close()
	if err := tracker.refreshBoundaries(); err != nil {
		return err
	}
	// Drain the kernel-adopted child boundary before signaling. A detached
	// setsid child can be reparented to the runner exactly as its original
	// parent exits; one final refresh is otherwise a lossy race.
	stableWindow := timeout / 4
	if stableWindow > 100*time.Millisecond {
		stableWindow = 100 * time.Millisecond
	}
	stableUntil := time.Now().Add(stableWindow)
	if stableUntil.After(deadline) {
		stableUntil = deadline
	}
	for time.Now().Before(stableUntil) {
		if err := tracker.refreshDescendants(); err != nil {
			return err
		}
		sleepUntil(stableUntil, 5*time.Millisecond)
	}
	if err := tracker.refreshBoundaries(); err != nil {
		return err
	}
	return terminateOwnedProcessesUntil(tracker.path, deadline, os.Getpid())
}

func currentProcessIdentity(pid int) (processIdentity, bool, error) {
	return nativeProcessIdentity(pid)
}

func identityMatches(expected processIdentity) (bool, error) {
	current, found, err := currentProcessIdentity(expected.PID)
	return found && current.Started == expected.Started, err
}

func writeProcessManifest(path string, processes map[int]processIdentity) error {
	identities := make([]processIdentity, 0, len(processes))
	for _, identity := range processes {
		identities = append(identities, identity)
	}
	sort.Slice(identities, func(left, right int) bool { return identities[left].PID < identities[right].PID })
	data, err := json.Marshal(processManifest{SchemaVersion: processManifestVersion, Processes: identities})
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".ownership-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(name)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	keep = true
	return nil
}

func readProcessManifest(path string) (processManifest, error) {
	var manifest processManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, err
	}
	if err := decodeStrictJSON(data, &manifest); err != nil {
		return manifest, err
	}
	if manifest.SchemaVersion != processManifestVersion || len(manifest.Processes) == 0 {
		return manifest, fmt.Errorf("invalid process ownership manifest")
	}
	return manifest, nil
}

func ownedProcessIDs(path string) ([]int, error) {
	manifest, err := readProcessManifest(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, identity := range manifest.Processes {
		matches, matchErr := identityMatches(identity)
		if matchErr != nil {
			return nil, matchErr
		}
		if matches {
			pids = append(pids, identity.PID)
		}
	}
	return pids, nil
}

func signalOwnedProcesses(path string, signal syscall.Signal, exceptPID int) error {
	manifest, err := readProcessManifest(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var failures []error
	for index := len(manifest.Processes) - 1; index >= 0; index-- {
		identity := manifest.Processes[index]
		if identity.PID == exceptPID {
			continue
		}
		handle, handleErr := openStableProcess(identity)
		if errors.Is(handleErr, errStaleProcessIdentity) || errors.Is(handleErr, syscall.ESRCH) {
			continue
		}
		if handleErr != nil {
			failures = append(failures, handleErr)
			continue
		}
		if signalErr := handle.signal(signal); signalErr != nil && !errors.Is(signalErr, syscall.ESRCH) && !errors.Is(signalErr, errStaleProcessIdentity) {
			failures = append(failures, signalErr)
		}
		_ = handle.close()
	}
	return errors.Join(failures...)
}

func terminateOwnedProcesses(path string, timeout time.Duration, exceptPID int) error {
	return terminateOwnedProcessesUntil(path, time.Now().Add(timeout), exceptPID)
}

func terminateOwnedProcessesUntil(path string, deadline time.Time, exceptPID int) error {
	if err := signalOwnedProcesses(path, syscall.SIGTERM, exceptPID); err != nil {
		return err
	}
	remaining := time.Until(deadline)
	softDeadline := time.Now().Add(remaining / 4)
	if softDeadline.After(deadline) {
		softDeadline = deadline
	}
	for time.Now().Before(softDeadline) {
		pids, err := ownedProcessIDs(path)
		pids = withoutPID(pids, exceptPID)
		if err != nil || len(pids) == 0 {
			return err
		}
		sleepUntil(softDeadline, 20*time.Millisecond)
	}
	for time.Now().Before(deadline) {
		if err := signalOwnedProcesses(path, syscall.SIGKILL, exceptPID); err != nil {
			return err
		}
		pids, err := ownedProcessIDs(path)
		pids = withoutPID(pids, exceptPID)
		if err != nil || len(pids) == 0 {
			return err
		}
		sleepUntil(deadline, 20*time.Millisecond)
	}
	pids, err := ownedProcessIDs(path)
	if err != nil {
		return err
	}
	pids = withoutPID(pids, exceptPID)
	if len(pids) == 0 {
		return nil
	}
	return fmt.Errorf("owned processes remain after termination: %v", pids)
}

func sleepUntil(deadline time.Time, interval time.Duration) {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return
	}
	if interval > remaining {
		interval = remaining
	}
	time.Sleep(interval)
}

func withoutPID(pids []int, excluded int) []int {
	result := pids[:0]
	for _, pid := range pids {
		if pid != excluded {
			result = append(result, pid)
		}
	}
	return result
}

// terminateProcessGroup kills a process group started by configureProcessGroup,
// where the leader's pid is also the group id. It is used for short-lived
// controller probes whose descendants must not outlive them; longer-lived
// invocations are owned by the ownership tracker instead.
func terminateProcessGroup(pid int) error {
	if pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
