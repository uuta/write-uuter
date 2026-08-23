//go:build darwin || linux

package app

import (
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

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
