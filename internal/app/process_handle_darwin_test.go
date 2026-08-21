//go:build darwin && cgo

package app

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestDarwinStableSignalCannotCrossFinalExitBoundary(t *testing.T) {
	target := exec.Command("/bin/sleep", "30")
	if err := target.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = target.Process.Kill()
		_, _ = target.Process.Wait()
	})
	sentinel := exec.Command("/bin/sleep", "30")
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sentinel.Process.Kill()
		_, _ = sentinel.Process.Wait()
	})

	identity, found, err := currentProcessIdentity(target.Process.Pid)
	if err != nil || !found {
		t.Fatalf("capture target identity: found=%v err=%v", found, err)
	}
	handle, err := openStableProcess(identity)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.close()
	stableSignalTestHook = func() {
		_ = target.Process.Kill()
		_, _ = target.Process.Wait()
	}
	defer func() { stableSignalTestHook = nil }()
	if err := handle.signal(syscall.SIGKILL); !errors.Is(err, errStaleProcessIdentity) {
		t.Fatalf("stable signal did not fail closed after target exit: %v", err)
	}
	if err := sentinel.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("unrelated sentinel was signaled across final identity boundary: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
}
