# Correctness Review (Round 2)

## Summary

MF2 is fixed and mutation-resistant: all four former vacuous checks now exercise the production `provider-homes` cleanup boundary and concrete credential filenames. No new correctness issues were found in the requested source and cross-file integration review.

## Regression validation of round-1 fixes

MF1 — **NOT IN MY LENS**. Successful authenticated smoke-run artifact validation belongs to the requirements/manager lens.

MF2 — **FIXED**. Production creates `provider-homes` at `internal/app/tmux.go:114` and stages `auth.json` / `installation_id` at `internal/app/tmux.go:276-277`. All four former assertion sites now call `assertNoStagedProviderCredentials`: `internal/app/blackbox_test.go:1151`, `internal/app/blackbox_test.go:1330`, `internal/app/blackbox_test.go:1426`, and `internal/app/blackbox_test.go:1458`. The helper at `internal/app/blackbox_test.go:1967-1991` intentionally accepts an existing private root whose `provider-homes` child is absent, rejects a present-but-empty `provider-homes` root, and independently rejects any surviving `auth.json` or `installation_id` anywhere below the private root. Exact mutation experiment: from `/tmp/write-uuter-mf2-r2`, `rtk go test -v .` against an exact scratch copy of the helper returned `Go test: 4 passed in 1 packages`; its subprocess cases proved missing root accepted, present-empty root rejected, and credentials under a renamed staging root rejected. Exact repository grep `rtk rg -n 'codex-homes' .` found the literal only in retained pre-fix review/diff/PM scratch artifacts (`docs/review/*` and `tmp/*`), never in current product or test source. Targeted call-site command `rtk go test ./internal/app -run 'TestBlackBox(UnexpectedTmuxProbeFailureIsNotAbsence|PersistentCleanupFailurePreservesOwnershipButDeletesCredentials|AmbiguousTmuxLaunchIsReconciledAndCleaned|ReadyPublicationTimeoutCleansOwnedRunner)$' -count=1 -timeout 40m -v` returned `Go test: 6 passed in 1 packages`.

MF3 — **NOT IN MY LENS**. Documentation/security-boundary wording belongs to the requirements/security lenses.

MF4 — **NOT IN MY LENS**. Admin-managed-settings sandbox exposure belongs to the security lens.

MF5 — **NOT IN MY LENS**. Lifecycle-bound regression ownership belongs to the resilience lens; I nevertheless checked the changed code for correctness interactions. `internal/app/claude_auth.go:59-103` gives timeout precedence over oversize, oversize precedence over exit/JSON parsing, treats `exec.ErrWaitDelay` narrowly, and validates status only after process-group teardown. `internal/app/claude_auth_test.go:61-132` is non-vacuous: the scripts must execute to publish descendant PIDs or output, timeout/oversize error text is asserted, and survivor PIDs are checked. Exact targeted command `rtk go test ./internal/app -run 'Test(ClaudeMaxPreflightBoundsDescendantHoldingStdoutPastExit|ClaudeMaxPreflightTerminatesProcessTreeOnTimeout|ClaudeMaxPreflightRejectsOversizedOutput|BoundedCaptureNeverExceedsItsLimit|ClaudeIsolationNeverGrantsAdminManagedPolicy|BlackBoxClaudeMaxPreflightRunsBeforeRunCreation|BlackBoxEveryRoleLaunchesItsDeclaredProfile|BlackBoxRevisionsPreserveDeclaredProfiles|BlackBoxReviewerLensesSelectDistinctProfiles)$' -count=1 -timeout 40m -v` returned `Go test: 19 passed in 1 packages`.

Cross-file verification: reviewer lookup still constructs `reviewer_` + lens at `internal/app/app.go:638`; `runWorker` obtains one `roleProfile` and passes it into the invocation at `internal/app/app.go:1204-1215`; argv uses `inv.Profile` at `internal/app/tmux.go:388-403`; audit publication uses that same `inv.Profile` at `internal/app/app.go:1613-1622`. The eight policy keys in `internal/app/policy.go:21-30` match `prompts/models.json`. Exact commands `rtk gofmt -l .`, `rtk go build ./...`, and `rtk go vet ./...` all exited 0, with no output. `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 rtk go build ./...` also exited 0. `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 rtk go build ./...` exposed a pre-existing baseline gap (undefined ownership types/functions also absent on `main`), not a regression from the new `terminateProcessGroup` shim; its signature is consistent with the Unix implementation and caller.

## Findings

No new issues found.
