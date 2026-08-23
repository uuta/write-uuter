package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// unstartableClient is present and executable but cannot be exec'd, so the
// tmux client fails before any request can reach the tmux server.
func unstartableClient(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tmux")
	if err := os.WriteFile(path, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// failingClient starts normally and then exits non-zero, which is the
// ambiguous case: tmux may already have applied the request.
func failingClient(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tmux")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho refused >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func reviewerRuntime(t *testing.T, executable string) (*tmuxRuntime, invocation) {
	t.Helper()
	inv := invocation{
		ID: "005-reviewer-evidence", Role: "reviewer_evidence", Lens: "evidence",
		Window:         "005-reviewer-evidence",
		PromptRelative: filepath.Join("prompts", "005-reviewer-evidence.md"),
		LogRelative:    filepath.Join("logs", "005-reviewer-evidence.log"),
		ExitRelative:   filepath.Join("exits", "005-reviewer-evidence.exit"),
	}
	control, err := openArtifactStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = control.Close() })
	return &tmuxRuntime{
		executable: executable, session: "write-uuter-test", commandTimeout: 2 * time.Second,
		controlStore: control, invocations: []invocation{inv},
	}, inv
}

func TestRunClientSeparatesPreStartFailureFromPostStartFailure(t *testing.T) {
	unstartable, _ := reviewerRuntime(t, unstartableClient(t))
	if _, started, err := unstartable.runClient("new-window"); err == nil || started {
		t.Fatalf("a client that never executed was reported as started: started=%v err=%v", started, err)
	}
	failing, _ := reviewerRuntime(t, failingClient(t))
	output, started, err := failing.runClient("new-window")
	if err == nil {
		t.Fatal("a refusing client was reported as successful")
	}
	if !started {
		t.Fatal("a client that ran and then failed was reported as never started")
	}
	if len(output) == 0 {
		t.Fatal("client diagnostics were dropped")
	}
}

// A local exec failure cannot have launched a reviewer, so it must claim no
// ownership, demand no audit files, and leave nothing for the attempt counter
// at app.go to count.
func TestPreStartLaunchFailureClaimsNoReviewerInvocation(t *testing.T) {
	runtime, inv := reviewerRuntime(t, unstartableClient(t))

	if err := runtime.requestLaunch(inv, "start reviewer_evidence worker", "new-window"); err == nil {
		t.Fatal("unstartable tmux client was accepted as a launch")
	}
	if runtime.launchAttempted(inv.ID) {
		t.Fatal("a reviewer attempt was counted for a client that never executed")
	}
	record := runtime.invocationRecord(inv.ID)
	if record == nil || record.started {
		t.Fatalf("an unlaunched reviewer was marked started: %+v", record)
	}

	// archiveAll must not demand prompt/log/exit files from a reviewer process
	// that never existed.
	destination, err := openArtifactStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	if err := runtime.archiveAll(destination); err != nil {
		t.Fatalf("archive demanded audit files from a nonexistent reviewer: %v", err)
	}
}

// Once the client has run, the outcome is ambiguous and ownership must be
// claimed conservatively even though the command failed.
func TestPostStartLaunchFailureKeepsConservativeOwnership(t *testing.T) {
	runtime, inv := reviewerRuntime(t, failingClient(t))

	if err := runtime.requestLaunch(inv, "start reviewer_evidence worker", "new-window"); err == nil {
		t.Fatal("a refused launch was accepted")
	}
	if !runtime.launchAttempted(inv.ID) {
		t.Fatal("an ambiguous launch was not counted conservatively")
	}
	record := runtime.invocationRecord(inv.ID)
	if record == nil || !record.started {
		t.Fatalf("an ambiguous launch did not retain ownership: %+v", record)
	}
}
