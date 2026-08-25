package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// claudeAuthTimeout bounds the pre-run authentication probe, including every
// descendant it starts and every descriptor those descendants inherited. The
// probe is a local status read, so a hang is a failure rather than something
// to wait out.
const claudeAuthTimeout = 30 * time.Second

// claudeAuthOutputLimit bounds how much of the probe's standard output is kept
// in memory. A status document is a few hundred bytes; anything approaching
// this limit is a broken or wrong executable, not a status response.
const claudeAuthOutputLimit = 64 << 10

// requiredClaudeAuth is the only accepted Claude authentication state: a
// logged-in claude.ai (OAuth) session on a Max subscription. An API-key,
// Bedrock, Vertex, or Foundry session is rejected so a run can never move to
// API billing or another provider.
var requiredClaudeAuth = struct {
	authMethod       string
	subscriptionType string
}{authMethod: "claude.ai", subscriptionType: "max"}

// verifyClaudeMaxSubscription runs the sanitized `claude auth status` probe and
// accepts only a logged-in claude.ai Max session. It runs before the run
// directory is created, uses the same credential/provider environment
// filtering as Claude invocations, and never returns or records the raw
// response, the account identity, or any credential value.
func verifyClaudeMaxSubscription(executable string) error {
	return verifyClaudeMaxSubscriptionWithin(executable, claudeAuthTimeout)
}

// claudeAuthWaitDelay is the slice of the deadline reserved for tearing the
// probe down: killing the process group and closing descriptors a descendant
// inherited. It is subtracted from the deadline rather than added to it, so
// the whole probe - process tree and inherited pipes included - returns within
// the deadline.
func claudeAuthWaitDelay(timeout time.Duration) time.Duration {
	delay := timeout / 4
	if delay > time.Second {
		delay = time.Second
	}
	if delay <= 0 {
		delay = time.Millisecond
	}
	return delay
}

func verifyClaudeMaxSubscriptionWithin(executable string, timeout time.Duration) error {
	teardown := claudeAuthWaitDelay(timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout-teardown)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "auth", "status")
	// Hard-limited capture: a runaway executable cannot grow the controller's
	// memory, and the overflow is reported rather than silently truncated into
	// a malformed-status error.
	capture := &boundedCapture{limit: claudeAuthOutputLimit}
	command.Stdout = capture
	command.Stderr = io.Discard
	command.Stdin = nil
	// The same allowlist the invocations use. ANTHROPIC_API_KEY and every
	// alternative provider-selection credential are absent by construction, so
	// the probe reports the same session the run will use.
	command.Env = providerBaseEnvironment()
	// The probe leads its own process group, so the deadline bounds the whole
	// tree rather than the immediate child: on timeout the group is signalled
	// before the leader is reaped, which is the only point where a descendant
	// is still reachable through it.
	configureProcessGroup(command)
	command.Cancel = func() error { return terminateProcessGroup(command.Process.Pid) }
	// A descendant that inherited the probe's stdout keeps the read end open
	// after the leader exits. Without a finite delay that descendant, not the
	// deadline, would decide when Wait returns.
	command.WaitDelay = teardown
	runErr := command.Run()
	if command.Process != nil {
		// Sweep any descendant that outlived a probe which exited on its own.
		_ = terminateProcessGroup(command.Process.Pid)
	}
	if ctx.Err() != nil {
		return fmt.Errorf("Claude Max preflight: `claude auth status` did not finish within %s; check the Claude CLI installation", timeout)
	}
	if capture.truncated {
		return fmt.Errorf("Claude Max preflight: `claude auth status` produced more than %d bytes of output instead of a status document; check that --claude points at the Claude CLI", claudeAuthOutputLimit)
	}
	// ErrWaitDelay means only that a descendant still held an inherited
	// descriptor; the probe's own exit status and output are still decisive,
	// and a descriptor held past the delay can only shorten what was captured,
	// which the strict status validation below rejects.
	if runErr != nil && !errors.Is(runErr, exec.ErrWaitDelay) {
		return fmt.Errorf("Claude Max preflight: `claude auth status` failed (%v); run `claude auth login` and retry", runErr)
	}
	return validateClaudeAuthStatus(capture.data)
}

// boundedCapture keeps at most limit bytes and records that more arrived. It
// never returns a short write or an error, so the reader keeps draining and
// the probe is bounded by the deadline rather than blocking on a full pipe.
type boundedCapture struct {
	limit     int
	data      []byte
	truncated bool
}

func (capture *boundedCapture) Write(chunk []byte) (int, error) {
	remaining := capture.limit - len(capture.data)
	switch {
	case remaining <= 0:
		if len(chunk) != 0 {
			capture.truncated = true
		}
	case len(chunk) > remaining:
		capture.data = append(capture.data, chunk[:remaining]...)
		capture.truncated = true
	default:
		capture.data = append(capture.data, chunk...)
	}
	return len(chunk), nil
}

// validateClaudeAuthStatus parses the status document strictly. Only the three
// decision fields are read; every other field, including the account identity,
// is ignored and never propagated.
func validateClaudeAuthStatus(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := rejectDuplicateKeys(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return fmt.Errorf("Claude Max preflight: `claude auth status` returned malformed JSON: %w", err)
	}
	var status struct {
		LoggedIn         *bool   `json:"loggedIn"`
		AuthMethod       *string `json:"authMethod"`
		SubscriptionType *string `json:"subscriptionType"`
	}
	if err := decoder.Decode(&status); err != nil {
		return fmt.Errorf("Claude Max preflight: `claude auth status` did not return a JSON object: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("Claude Max preflight: `claude auth status` returned more than one JSON document")
	}
	if status.LoggedIn == nil || status.AuthMethod == nil || status.SubscriptionType == nil {
		return fmt.Errorf("Claude Max preflight: `claude auth status` omitted loggedIn, authMethod, or subscriptionType")
	}
	if !*status.LoggedIn {
		return fmt.Errorf("Claude Max preflight: the Claude CLI is logged out; run `claude auth login` with your Max account")
	}
	if *status.AuthMethod != requiredClaudeAuth.authMethod {
		return fmt.Errorf("Claude Max preflight: authMethod is %q, but this policy requires a %q subscription session; API-key and third-party provider sessions are refused",
			*status.AuthMethod, requiredClaudeAuth.authMethod)
	}
	if *status.SubscriptionType != requiredClaudeAuth.subscriptionType {
		return fmt.Errorf("Claude Max preflight: subscriptionType is %q, but this policy requires %q",
			*status.SubscriptionType, requiredClaudeAuth.subscriptionType)
	}
	return nil
}
