//go:build darwin || linux

package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// A staged credential copy must outlive every process that could still read
// it. The argv/private-path scan alone cannot prove that: a tracked process
// whose command line never mentions the private root is invisible to it.
func TestCloseCredentialsWaitsForOwnedIdentityExit(t *testing.T) {
	privateRoot := t.TempDir()
	controlDir := filepath.Join(privateRoot, "control")
	providerHomesDir := filepath.Join(privateRoot, "provider-homes")
	authPath := filepath.Join(providerHomesDir, "001-pm", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte("{\"staged\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("sleep", "30")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
	})
	identity, found, err := currentProcessIdentity(command.Process.Pid)
	if err != nil || !found {
		t.Fatalf("capture owned identity: %v (found=%v)", err, found)
	}
	ownershipRelative := filepath.Join("ownership", "001-pm.json")
	if err := writeProcessManifest(filepath.Join(controlDir, ownershipRelative), map[int]processIdentity{identity.PID: identity}); err != nil {
		t.Fatal(err)
	}

	runtime := &tmuxRuntime{
		privateRoot: privateRoot, controlDir: controlDir, providerHomesDir: providerHomesDir,
		commandTimeout: 200 * time.Millisecond,
		invocations: []invocation{{
			ID: "001-pm", Role: "pm", OwnershipRelative: ownershipRelative, started: true,
		}},
	}
	if err := runtime.auditPrivateProcesses(); err != nil {
		t.Fatalf("argv scan unexpectedly matched the tracked process: %v", err)
	}
	if err := runtime.closeCredentials(); err == nil {
		t.Fatal("staged credentials were removed while an owned identity was live")
	}
	if _, err := os.Lstat(authPath); err != nil {
		t.Fatalf("staged credential was deleted before the owned process exited: %v", err)
	}

	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = command.Process.Wait()
	stopped = true
	if err := runtime.closeCredentials(); err != nil {
		t.Fatalf("credential cleanup blocked after every owned identity exited: %v", err)
	}
	if _, err := os.Lstat(providerHomesDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("credential root survived verified cleanup: %v", err)
	}
}
