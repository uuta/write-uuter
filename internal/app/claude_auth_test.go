//go:build darwin || linux

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// writeProbeScript installs a fake `claude` whose `auth status` behaviour is
// supplied by the caller. The pid file path is baked into the script because
// the probe environment is a fixed allowlist and cannot carry a test variable.
func writeProbeScript(t *testing.T, body string) (string, string) {
	t.Helper()
	directory := t.TempDir()
	pidPath := filepath.Join(directory, "descendant.pid")
	script := filepath.Join(directory, "claude")
	source := "#!/bin/sh\nPIDFILE=" + strconv.Quote(pidPath) + "\n" + body
	if err := os.WriteFile(script, []byte(source), 0o700); err != nil {
		t.Fatal(err)
	}
	return script, pidPath
}

func readDescendantPID(t *testing.T, pidPath string) int {
	t.Helper()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("probe descendant never recorded its pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("probe descendant recorded an unusable pid %q: %v", data, err)
	}
	return pid
}

func assertProcessReaped(t *testing.T, pid int, label string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatalf("%s (pid %d) outlived the Claude Max preflight", label, pid)
}

// A descendant that inherited the probe's standard output must not be able to
// keep the preflight alive after the probe itself exits, and must not survive
// it. Without a bounded teardown the inherited descriptor, not the deadline,
// would decide when the preflight returns.
func TestClaudeMaxPreflightBoundsDescendantHoldingStdoutPastExit(t *testing.T) {
	script, pidPath := writeProbeScript(t, `
sleep 30 &
echo $! > "$PIDFILE"
printf '%s\n' '{"loggedIn":true,"authMethod":"claude.ai","subscriptionType":"max"}'
exit 0
`)
	const timeout = 3 * time.Second
	started := time.Now()
	err := verifyClaudeMaxSubscriptionWithin(script, timeout)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("valid status rejected because a descendant held stdout: %v", err)
	}
	if elapsed >= timeout {
		t.Fatalf("preflight ran for %s, past its %s deadline", elapsed, timeout)
	}
	assertProcessReaped(t, readDescendantPID(t, pidPath), "stdout-holding descendant")
}

// The deadline bounds the whole process tree: a probe that hangs is killed
// together with every descendant it started, and neither is left orphaned.
func TestClaudeMaxPreflightTerminatesProcessTreeOnTimeout(t *testing.T) {
	script, pidPath := writeProbeScript(t, `
sleep 30 &
echo $! > "$PIDFILE"
sleep 30
`)
	const timeout = 2 * time.Second
	started := time.Now()
	err := verifyClaudeMaxSubscriptionWithin(script, timeout)
	elapsed := time.Since(started)
	if err == nil || !strings.Contains(err.Error(), "did not finish within") {
		t.Fatalf("hung probe was not reported as a timeout: %v", err)
	}
	if elapsed >= timeout {
		t.Fatalf("preflight ran for %s, past its %s deadline", elapsed, timeout)
	}
	assertProcessReaped(t, readDescendantPID(t, pidPath), "descendant of the hung probe")
}

// Standard output is captured through a hard-limited writer, so a wrong or
// broken executable cannot grow the controller's memory, and the overflow is
// reported as an actionable error rather than as a malformed status document.
func TestClaudeMaxPreflightRejectsOversizedOutput(t *testing.T) {
	script, _ := writeProbeScript(t, fmt.Sprintf(`
i=0
while [ $i -lt %d ]; do
  printf '%%s\n' 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  i=$((i+1))
done
exit 0
`, (claudeAuthOutputLimit/65)*3))
	err := verifyClaudeMaxSubscriptionWithin(script, 20*time.Second)
	if err == nil || !strings.Contains(err.Error(), "produced more than") {
		t.Fatalf("oversized probe output was not reported: %v", err)
	}
	if !strings.Contains(err.Error(), "--claude") {
		t.Errorf("oversized-output error is not actionable: %v", err)
	}
}

func TestBoundedCaptureNeverExceedsItsLimit(t *testing.T) {
	capture := &boundedCapture{limit: 8}
	for range 100 {
		if written, err := capture.Write([]byte("0123456789")); written != 10 || err != nil {
			t.Fatalf("bounded capture short-wrote: %d, %v", written, err)
		}
	}
	if len(capture.data) != 8 || !capture.truncated {
		t.Fatalf("bounded capture kept %d bytes (truncated=%v)", len(capture.data), capture.truncated)
	}
}
