# Resilience Review
## Summary
The provider-neutral agent path preserves controller-owned logs, readiness ordering, timeout handling, and owned-process cleanup for both providers. The Claude authentication preflight, however, does not apply equivalent output and descendant-process bounds.
## Findings
### 1. Claude auth preflight can exceed its timeout and leave descendants running
- **Severity**: High
- **Location**: `internal/app/claude_auth.go:34`
- **Description**: The preflight uses `exec.CommandContext`, which kills only the direct `claude` process when the context expires; it does not create or terminate a process group. If `claude auth status` starts a helper that survives its parent, the helper can remain after preflight failure. If that helper inherits stdout, `command.Run()` can also remain blocked waiting for the output pipe to close even after the 30-second context expires, so the claimed timeout is not a hard lifecycle bound. This path runs before `tmuxRuntime` and therefore does not benefit from `startProcessTracker` / `terminateOwnedProcesses`, unlike normal Claude invocations.
- **Evidence**: `verifyClaudeMaxSubscription` constructs the probe with `exec.CommandContext(ctx, executable, "auth", "status")` and immediately calls `command.Run()` (`internal/app/claude_auth.go:34-43`), with no `configureProcessGroup`, ownership tracker, custom `Cancel`, or `WaitDelay`. By contrast, the launched-agent path calls `configureProcessGroup(command)` and tracks/terminates owned processes (`internal/app/agent_runner.go:113-136`). Focused verification command `rtk go test ./internal/app -run 'TestBlackBoxClaudeMaxPreflightRunsBeforeRunCreation|TestBlackBox.*(Timeout|Provider|Audit)|Test.*Claude.*Auth' -count=1` passed 22 tests, but the enumerated auth scenarios cover status/JSON failures and not a descendant that retains the output pipe.
- **Suggestion**: Give the preflight the same process-group lifecycle boundary as agent invocations: start it explicitly, place it in a dedicated process group, and on timeout/cancellation terminate and reap the whole group. Also set a bounded pipe-wait policy (for example a finite `WaitDelay`) so inherited descriptors cannot make `Run` outlive the deadline.

### 2. Claude auth status output is accumulated without a size limit
- **Severity**: Medium
- **Location**: `internal/app/claude_auth.go:35`
- **Description**: All stdout from the external Claude executable is appended to a `bytes.Buffer` until process exit. A broken or hostile executable can emit arbitrarily large output within the 30-second window, causing avoidable memory growth or OOM before strict JSON validation runs. This violates the preflight requirement to handle huge output cleanly before run initialization.
- **Evidence**: `var stdout bytes.Buffer` is assigned directly to `command.Stdout`, and `validateClaudeAuthStatus(stdout.Bytes())` is called only after `command.Run()` completes (`internal/app/claude_auth.go:35-50`). There is no `io.LimitedReader`, capped writer, or overflow detection on this path.
- **Suggestion**: Capture through a small hard-limited writer sized for the expected status document, detect overflow, terminate/reap the probe, and return an actionable malformed/oversized-status error without retaining the excess bytes.
