# Correctness Review

## Summary

The profile lookup, strict JSON parsing, policy digest, argv/audit binding, and audit-before-readiness ordering are internally consistent, but the Claude preflight is not bound to the executable bytes later staged. Verification: `go build ./...`, `go vet ./...`, and `gofmt -l .` passed/returned clean; `go test ./... -count=1` reported 93 passed and 2 timeout failures, while rerunning those two failures alone passed (`go test ./internal/app -run 'TestBlackBoxFinalCommitBoundaryRechecksCompleteAudit|TestBlackBoxFinalPublicationNeverDisturbsCompetingArticle' -count=1 -v`).

## Findings

### 1. Claude Max preflight is not bound to the client that is staged and launched

- **Severity**: High
- **Location**: `internal/app/app.go:126`
- **Description**: The controller resolves a pathname and executes it for `claude auth status`, but it does not retain that file identity or its bytes. After run initialization, `newTmuxRuntime` resolves and opens the pathname again, so an atomic replacement between preflight and staging causes a different executable to be launched despite the comment and contract requiring the exact preflighted client.
- **Evidence**: `verifyClaudeMaxSubscription(claudePath)` runs at `internal/app/app.go:132`; staging later assigns the same pathname to `sourcePath` and reopens it via `installPrivateRunner(sourcePath, stagedPath)` at `internal/app/tmux.go:137-140` (the actual `os.Open(source)` is at `internal/app/tmux.go:217`). No file descriptor, `os.FileInfo` identity check, or digest connects those operations.
- **Suggestion**: Create the private staged Claude copy before preflight, run `auth status` through that staged copy, and retain/reuse that exact copy for all per-invocation staging. Clean the private staging root on every pre-run failure so the run directory still remains uncreated.

### 2. Credential-cleanup black-box assertions check a directory that production no longer uses

- **Severity**: Medium
- **Location**: `internal/app/blackbox_test.go:1150`
- **Description**: Four cleanup assertions still test for the absence of `codex-homes`, while production renamed the credential root to `provider-homes`. These assertions now pass even if the real credential directory survives, so the affected failure-path tests no longer prove credential cleanup.
- **Evidence**: Production creates and removes `provider-homes` at `internal/app/tmux.go:114` and `internal/app/tmux.go:783-786`. `rg -n "codex-homes" internal/app/blackbox_test.go` finds stale checks at lines 1150, 1331, 1429, and 1463. The focused command `go test ./internal/app -run 'TestBlackBox(PersistentCleanupFailurePreservesOwnershipButDeletesCredentials|UnexpectedTmuxProbeFailureIsNotAbsence)$' -count=1 -v` passed both tests while they were checking only that obsolete path.
- **Suggestion**: Change all four checks to `provider-homes` and, where the private root is intentionally retained, assert that the actual staged `auth.json`/`installation_id` paths are absent as well.

### 3. The audit/argv binding helper does not assert the audit role

- **Severity**: Low
- **Location**: `internal/app/model_policy_blackbox_test.go:140`
- **Description**: `assertLaunchedProfile` claims to verify audit identity but compares only lens and candidate after checking the profile fields. It never compares `audit.Role` with the launched role, so a regression that records a wrong non-reviewer role can pass the binding tests.
- **Evidence**: Lines 134-142 compare provider/model/effort, digest, lens, and candidate; `rg -n "audit.Role|Role !=|Role ==" internal/app/model_policy_blackbox_test.go` returns no role assertion. Reviewer-role coverage at lines 240-242 only filters records for the separate lens test and does not cover PM, Researcher, Story Editor, or Writer role identity.
- **Suggestion**: Derive the expected audit role (`reviewer` for reviewer invocations, otherwise `record.Role`) and assert it in `assertLaunchedProfile`.
