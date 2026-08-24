# Concurrency Review (Round 2)

## Summary

The new preflight has bounded output, bounded waiting, clean race-detector results, and correct ordinary process-group setup, but MF5 is only partially fixed because a descendant that creates a new session/process group escapes termination. The post-`Wait` cleanup also signals a numeric process-group ID after the leader has been reaped, leaving a narrow but destructive PID-reuse race.

## Regression validation of round-1 fixes

- **MF1 — NOT IN MY LENS.** Mixed-provider smoke artifact completeness and profile/audit binding belong to requirements/API-contract review, not concurrency/process-tree signalling.
- **MF2 — NOT IN MY LENS.** Credential-directory assertion quality belongs to requirements/security review.
- **MF3 — NOT IN MY LENS.** Documentation accuracy for `HOME` and sandbox behavior belongs to requirements/security review.
- **MF4 — NOT IN MY LENS.** Darwin sandbox read grants belong to security review.
- **MF5 — PARTIALLY FIXED.** `internal/app/claude_auth.go:59-103` applies a finite context deadline, finite `WaitDelay`, bounded stdout capture, one `Run`/`Wait` lifecycle, and process-group cancellation. `internal/app/process_group_unix.go:40-42` sets `Setpgid` before `Start`; with the default zero `Pgid`, the successfully started child becomes leader of the group identified by its positive PID, so the normal path cannot pass 0 or 1 to `kill(-pgid)`, and it cannot target the controller's existing group. `internal/app/claude_auth_test.go:61-121` verifies an ordinary stdout-holding child, an ordinary child on timeout, and oversized output; `internal/app/model_policy_blackbox_test.go:354-389` verifies failures occur before run creation. These tests do not create a descendant that calls `setsid(2)` or `setpgid(2)`, and the implementation has no ancestry/ownership tracker around the preflight, so the promised full-tree cleanup remains incomplete (Finding 1). The unconditional sweep after `Run` also uses the reaped leader's PID as a group ID (Finding 2).

Verification evidence:

- `rtk go test ./internal/app -race -run '^(TestClaudeMaxPreflightBoundsDescendantHoldingStdoutPastExit|TestClaudeMaxPreflightTerminatesProcessTreeOnTimeout|TestClaudeMaxPreflightRejectsOversizedOutput|TestBoundedCaptureNeverExceedsItsLimit|TestProcessTracker.*|TestTerminate.*|Test.*Ownership.*)$' -count=1 -timeout 40m` — PASS (`Go test: 7 passed in 1 packages`).
- `rtk go test ./internal/app -race -run 'ClaudeMaxPreflight|BoundedCapture|ProcessTracker|Terminate.*Process|Ownership' -count=1 -timeout 40m` — PASS (`Go test: 18 passed in 1 packages`).
- `rtk go test ./internal/app -race -run '^TestBlackBoxClaudeMaxPreflightRunsBeforeRunCreation$' -count=1 -timeout 40m` — PASS (all 11 subtests; `Go test: 11 passed in 1 packages`).

No new goroutine/channel race, double-`Wait`, unbounded pipe-drain goroutine, or reviewer-order/PM-liveness/ownership-tracker behavioral drift was found in the changed source.

## Findings

### 1. A detached preflight descendant escapes the claimed full-tree deadline

- **Severity**: High
- **Location**: `internal/app/claude_auth.go:75`
- **Description**: Cancellation and the post-run sweep signal only the process group whose ID is the probe leader's PID. A probe descendant can call `setsid(2)` or `setpgid(2)`, leave that group, and continue after both `terminateProcessGroup` calls. `WaitDelay` bounds how long inherited pipes delay return, but closing those pipes does not terminate the escaped process. This is reachable for a wrong/broken configured executable or a CLI/helper that daemonizes, and contradicts MF5's requirement that the full probe tree and orphans be bounded before run initialization.
- **Evidence**: `internal/app/claude_auth.go:79-88` has only `configureProcessGroup`, `Cancel = terminateProcessGroup(command.Process.Pid)`, and the same group sweep after `Run`; `internal/app/process_group_unix.go:331-338` implements that operation solely as `syscall.Kill(-pid, SIGKILL)`. By POSIX process-group semantics, a descendant in a new session/group is not selected by that negative-PID signal. The tests at `internal/app/claude_auth_test.go:61-100` launch plain background `sleep` processes, which inherit the leader's group, so they cannot detect this escape.
- **Suggestion**: Put the preflight under a real ancestry/ownership boundary that tracks descendants independently of process-group membership (including detached descendants), then terminate and verify that owned set before returning. Add a deterministic helper-process test that creates a new session/group, records its PID, and proves it is gone after both timeout and leader-success paths; retain the existing output and editorial validations unchanged.

### 2. The post-`Run` sweep can signal an unrelated reused process group

- **Severity**: Medium
- **Location**: `internal/app/claude_auth.go:85`
- **Description**: `command.Run()` waits for and reaps the probe leader before the unconditional cleanup at lines 86-88 uses its former PID as a process-group ID. If the probe left no member in that group, the ID is no longer owned by this invocation and can be reused before `kill(-pid, SIGKILL)`; under PID pressure, this can kill an unrelated process group. The positive-PID guard prevents `kill(0, ...)`, but does not protect against post-reap identity reuse.
- **Evidence**: `internal/app/claude_auth.go:85-88` performs the sweep only after `Run` returns, while `internal/app/process_group_unix.go:331-338` validates only `pid > 0` and signals by numeric ID without a stable process identity or ownership check. In contrast, the long-lived invocation cleanup uses start-time identities and stable handles in `internal/app/process_group_unix.go:239-286`, demonstrating that numeric PID reuse is already treated as a real hazard elsewhere in this codebase.
- **Suggestion**: Avoid signalling a bare leader PID after it has been reaped. Integrate the preflight with stable ownership tracking and signal verified live identities, or restructure waiting/cleanup so the group is terminated and verified while its leader/group identity is still owned; add a regression test around stale/reused identity handling.
