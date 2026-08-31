//go:build darwin || linux

package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestTerminateOwnedProcessesUsesOneAbsoluteDeadline(t *testing.T) {
	command := exec.Command("/bin/sleep", "30")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	}()
	// A stopped process cannot act on SIGTERM. SIGKILL makes it an unreaped
	// zombie, which deliberately remains observable for the hard phase without
	// spawning or leaking another process.
	if err := command.Process.Signal(syscall.SIGSTOP); err != nil {
		t.Fatal(err)
	}
	identity, found, err := currentProcessIdentity(command.Process.Pid)
	if err != nil || !found {
		t.Fatalf("child identity = %+v, %v, %v", identity, found, err)
	}
	manifest := filepath.Join(t.TempDir(), "ownership.json")
	if err := writeProcessManifest(manifest, map[int]processIdentity{identity.PID: identity}); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_ = terminateOwnedProcesses(manifest, 400*time.Millisecond, os.Getpid())
	elapsed := time.Since(started)
	// The prior implementation spent 100ms on the soft phase and then
	// restarted a fresh 400ms hard phase while the unreaped child remained
	// observable. One deadline keeps the whole operation below that 500ms sum.
	if elapsed > 460*time.Millisecond {
		t.Fatalf("termination exceeded its single 400ms budget: %s", elapsed)
	}
}

func TestStableProcessOwnershipRejectsReusedIdentitySentinel(t *testing.T) {
	command := exec.Command("/bin/sleep", "30")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	})

	var identity processIdentity
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, found, err := currentProcessIdentity(command.Process.Pid)
		if err != nil {
			t.Fatal(err)
		}
		if found {
			identity = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if identity.PID == 0 {
		t.Fatal("child identity was not observable")
	}
	identity.Started += "-reused-sentinel"
	manifest := filepath.Join(t.TempDir(), "ownership.json")
	if err := writeProcessManifest(manifest, map[int]processIdentity{identity.PID: identity}); err != nil {
		t.Fatal(err)
	}
	if err := signalOwnedProcesses(manifest, syscall.SIGKILL, 0); err != nil {
		t.Fatal(err)
	}
	if err := command.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("stale identity signaled unrelated live process: %v", err)
	}
}
