# Requirements Compliance and Scope Control Review (Round 2)

## Summary
Both must-fix items I own are fixed and independently verified: `smoke-runs/mixed-20260825-2/`
is a genuine, internally cross-verifiable successful authenticated mixed-provider run covering
all eight role/lens identities with the exact contracted profiles, and every README /
`docs/roles.md` sentence about HOME, `CLAUDE_CODE_TMPDIR`, `~/.claude.json`, Keychain,
spawned-tool denial and `--safe-mode` now matches `internal/app/agent_runner.go` and
`internal/app/isolation_darwin.go`. The contract sweep found no scope creep, no weakened test or
validation, and all eleven required black-box items still covered and passing; one Low,
uncertain lifecycle observation outside the smoke run is recorded.

## Verification commands run

```
$ go build ./...                                       # exit 0
$ go vet ./...                                         # exit 0
$ gofmt -l .                                           # no output
$ go test ./internal/app -count=1 -timeout 40m -run 'TestBlackBox(EveryRoleLaunchesItsDeclaredProfile|RevisionsPreserveDeclaredProfiles|InvalidPolicyFailsBeforeRunCreation|ClaudeMaxPreflightRunsBeforeRunCreation|ProviderExecutableSelectionIsIndependent|PolicyMutationAfterValidationCannotChangeLaunchedProfiles|UnavailableModelBlocksWithoutFallback|ProviderProcessesNeverReceiveExternalCredentials|ClaudeKeychainAccessIsProcessScoped|ExplicitEmptyExecutableOverridesFailBeforeRunInitialization)$'
ok  	github.com/uuta/write-uuter/internal/app	55.222s
$ go test ./internal/app -count=1 -timeout 40m -run 'TestBlackBox(TimeoutBlocksAndCleansProcesses|DetachedDescendantsAreKilledOnTerminalPaths|PersistentCleanupFailurePreservesOwnershipButDeletesCredentials)$'
ok  	github.com/uuta/write-uuter/internal/app	15.026s
```

## Regression validation of round-1 fixes

### MF1 — no successful authenticated mixed-provider smoke run — **FIXED**

**Run status and internal consistency.** `smoke-runs/mixed-20260825-2/workflow.json:2-6,26`
reports `"status": "succeeded"`, `"phase": "complete"`, `"current_candidate": 3`,
`"review_attempt_count": 9`. Every dependent artifact agrees:

| Check | Result |
| --- | --- |
| `article.md` present | yes (`-rw-r--r--`) |
| `drafts/` | `article-001.md`, `article-002.md`, `article-003.md` |
| `reviews/` | `article-001`: evidence, story; `article-002`: evidence, story, clarity; `article-003`: evidence, story, clarity, copy = **9** review directories, matching `review_attempt_count: 9` |
| `pm-decisions/` | `article-001.md` (2 lenses), `article-002.md` (3), `article-003.md` (4) — one entry per executed review |
| `.control/{invocations,prompts,logs,exits}` | 15 files each, same 15 IDs |

**Published revision digest.** The digest is over the accepted draft, which is copied
byte-for-byte to `article.md` (`internal/app/app.go:1358-1371` `succeed()` → `revisionFor`,
`internal/app/files.go:505-508`):

```
$ shasum -a 256 smoke-runs/mixed-20260825-2/article.md smoke-runs/mixed-20260825-2/drafts/*.md
1f134fc8c2c77dbaf95f81638d75a3e7834880a0a942c32cfb5e5fdf050c9844  article.md
a4adaf177675f0fd9aba0587033d70f57e473f8f6eba503b88551d5d808c3272  drafts/article-001.md
de5cf22227acee3d99de7f8fe6ee985a0fe473d3843a49a66b71d5e076d5b67d  drafts/article-002.md
1f134fc8c2c77dbaf95f81638d75a3e7834880a0a942c32cfb5e5fdf050c9844  drafts/article-003.md
```
`workflow.json:5` `current_revision` = `sha256:1f134fc8…0c9844` = `article.md` = `drafts/article-003.md`. **Match.**

**All 15 audit entries, all eight identities, exact contract profiles.** Enumerated from
`smoke-runs/mixed-20260825-2/.control/invocations/`:

| # | invocation | role | lens | cand | provider | model | effort | contract table |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 001-pm | pm | – | – | codex | gpt-5.6-sol | high | ✔ |
| 2 | 002-researcher | researcher | – | – | claude_code | claude-sonnet-5 | medium | ✔ |
| 3 | 003-story-editor | story_editor | – | – | claude_code | claude-opus-5 | high | ✔ |
| 4 | 004-writer | writer | – | 1 | claude_code | claude-opus-5 | medium | ✔ |
| 5 | 005-reviewer-evidence | reviewer | evidence | 1 | codex | gpt-5.6-sol | medium | ✔ |
| 6 | 006-reviewer-story | reviewer | story | 1 | claude_code | claude-sonnet-5 | medium | ✔ |
| 7 | 007-writer | writer | – | 2 | claude_code | claude-opus-5 | medium | ✔ |
| 8 | 008-reviewer-evidence | reviewer | evidence | 2 | codex | gpt-5.6-sol | medium | ✔ |
| 9 | 009-reviewer-story | reviewer | story | 2 | claude_code | claude-sonnet-5 | medium | ✔ |
| 10 | 010-reviewer-clarity | reviewer | clarity | 2 | claude_code | claude-sonnet-5 | medium | ✔ |
| 11 | 011-writer | writer | – | 3 | claude_code | claude-opus-5 | medium | ✔ |
| 12 | 012-reviewer-evidence | reviewer | evidence | 3 | codex | gpt-5.6-sol | medium | ✔ |
| 13 | 013-reviewer-story | reviewer | story | 3 | claude_code | claude-sonnet-5 | medium | ✔ |
| 14 | 014-reviewer-clarity | reviewer | clarity | 3 | claude_code | claude-sonnet-5 | medium | ✔ |
| 15 | 015-reviewer-copy | reviewer | copy | 3 | codex | gpt-5.6-luna | low | ✔ |

All **eight** distinct identities appear (pm, researcher, story_editor, writer,
reviewer_evidence, reviewer_story, reviewer_clarity, reviewer_copy). Zero deviations from the
contract table (contract §2). Record shape matches `docs/artifacts.md:52-64` and
`internal/app/models.go:81-90`, including `role: "reviewer"` + separate `lens`. All 15 are
`-r--r--r--`, consistent with `writeAtomicNoReplace(..., 0o444)` at `internal/app/app.go:1613-1638`.

**Policy digest binding.**
```
$ shasum -a 256 prompts/models.json smoke-runs/mixed-20260825-2/model-policy.json
f5a91c462dea2aebe51b2870213c4a193ce0c1576f95ec983b018ae30b16752b  prompts/models.json
f5a91c462dea2aebe51b2870213c4a193ce0c1576f95ec983b018ae30b16752b  smoke-runs/mixed-20260825-2/model-policy.json
$ cmp prompts/models.json smoke-runs/mixed-20260825-2/model-policy.json   # identical
```
`workflow.json:24` `model_policy_digest` and **all 15** audit records carry exactly
`sha256:f5a91c46…16752b`. `model-policy.json` is byte-identical to `prompts/models.json` and
mode `-r--r--r--`. **Match.**

**Final reviewers bind the published revision (explicit contract clause).** Every
candidate-3 review binds the published digest:
`reviews/article-003/{evidence,story,clarity,copy}/result.json:4` and the corresponding
`report.md` all carry `sha256:1f134fc8…0c9844`, equal to `workflow.json` `current_revision`.
Candidate-1 reviews bind `a4adaf17…`, candidate-2 reviews bind `de5cf222…` — i.e. each chain
bound the draft that existed at the time, and only the final chain binds the published article.

**Ordered review chain rerun per revision, same profile, no drift.** The declared order is
`evidence, story, clarity, copy` (`internal/app/app.go:565`, `:1444`). The run shows:
- candidate 1: 005 evidence `clean` → 006 story `fix_required` → chain stops, revise
- candidate 2: 008 evidence `clean` → 009 story `fix_required` → 010 clarity `fix_required` → revise
- candidate 3: 012 evidence → 013 story → 014 clarity → 015 copy, all `clean` → publish

Order is correct in every chain, and each lens/role kept an identical provider/model/effort
across candidates 1→2→3 (rows 5/8/12, 6/9/13, 10/14, 4/7/11 above). No drift.

**Not doctored — independent digest recomputation.** `review_digest` is
`sha256(result.json || 0x00 || report.md)` (`internal/app/files.go:510-516`). Recomputing from
the on-disk files reproduces **all nine** PM-decision digests exactly:

```
001/evidence e573721c…72695   001/story 5360fc31…f05a71
002/evidence 225da109…82ac95  002/story 0fed74b6…f133bc3  002/clarity 861966a0…4c5f2
003/evidence 2bb935e4…fb7409  003/story 975c2986…6dce5   003/clarity c13d3f5f…e720e8  003/copy 92a5c9e8…d240f2a
```
matching `pm-decisions/article-001.md:7,12`, `article-002.md:7,12,23`, `article-003.md:7,12,17,22`.
Any hand edit to a report, result, article, or policy would break one of these chains.

**Timestamps and exit codes are internally consistent with the code, not hand-set.**
`.control/invocations/*.json` mtimes are strictly increasing and staggered across the run window
(00:25:23 → 00:49:09 JST), matching `started_at` 2026-08-24T15:25:23Z / `completed_at`
15:50:15Z (UTC+9). `.control/{prompts,logs,exits}` all carry 00:50:14-15 because `archive` copies
them out of the controller-private root only at terminal cleanup
(`internal/app/tmux.go:1017-1041`, called from `succeed` at `internal/app/app.go:1362` **after**
`runtime.cleanup(true, control.pm)` at `:1359`) — exactly as `docs/roles.md:55-57` states.
`001-pm.exit` = `143` (SIGTERM) is the PM being terminated by that same terminal cleanup, and is
identical to the three pre-existing runs already on `main`
(`smoke-runs/real-20260823-{29,31,34}/.control/exits/001-pm.exit` = 143). All 14 worker exits are `0`.

**No credentials, no raw auth identity, no fallback/Fable/billing/undeclared model.**
```
$ grep -rniE 'ANTHROPIC|authMethod|subscriptionType|orgId|sk-ant|Bearer |api[_-]?key|access[_-]?token|refresh[_-]?token|\bsecret\b' smoke-runs/mixed-20260825-2   # no output
$ grep -rniE 'token|@…\.(com|ai|jp|io)|keychain' smoke-runs/mixed-20260825-2
  → only 4 hits, all the Codex CLI's own "tokens used" usage counter (e.g. .control/logs/015-reviewer-copy.log:339)
$ grep -rniE 'fable|fast mode|api billing|usage credit|extra usage' smoke-runs/mixed-20260825-2   # no output
$ grep -rniE 'fallback|fall back' smoke-runs/mixed-20260825-2
  → editorial prose only, all asserting the ABSENCE of fallback (article.md:8, outline.md:51,141, claim-ledger.md:38,49)
$ grep -rnoiE 'gpt-[a-z0-9.-]+|claude-[a-z0-9.-]+|sonnet|opus|haiku|fable' smoke-runs/mixed-20260825-2 | sed 's/.*://' | sort | uniq -c
  183 claude-sonnet-5   126 gpt-5.6-sol   122 claude-opus-5   61 gpt-5.6-luna
```
Exactly the four declared models appear anywhere in the run; no undeclared model name, no
`fable`, no bare `opus`/`sonnet`/`haiku` alias, no billing or Fast-mode term.

**Cleanup.** No `.write-uuter-private-*` root anywhere in the worktree
(`find . -name '.write-uuter-private-*'` → empty); `smoke-runs/` contains only the four run
directories. No live tmux session belongs to this run: the three `write-uuter-*` sessions present
were created 2026-08-24 21:07/21:11 (before the run's 00:25 start) and 2026-08-25 00:59 (after its
00:50 completion) — see Finding 1 for the two stale ones. The private-root name that appears
inside the archived logs (`write-uuter-private-4016323721`) no longer exists on disk.

**Blocked attempts not passed off as successes.** `git ls-tree -d --name-only main smoke-runs/`
and `HEAD` both list only `real-20260823-{29,31,34}`; `git status --porcelain -- smoke-runs/`
adds only `mixed-20260825-2/`. The blocked `mixed-20260824-*` runs are gone from the product
tree; nothing labelled `blocked` remains, and the one run present genuinely reports `succeeded`.
The run-specific brief is preserved at `smoke-runs/mixed-20260825-2/brief.md`.

### MF3 — README/docs contradicted the code about HOME — **FIXED**

Round-1 Finding 1 (`README.md:147`) and Finding 2 (`docs/roles.md:35-38`) are both rewritten and
now accurate sentence by sentence:

| Doc claim | Code | Verdict |
| --- | --- | --- |
| `README.md:148-150`, `docs/roles.md:37-40`: "Claude invocations keep the real `HOME` … only `CLAUDE_CODE_TMPDIR` is redirected, to a run-owned scratch directory removed with the run" | `providerBaseEnvironment` copies `HOME` verbatim (`internal/app/agent_runner.go:181-198`); `agentEnvironment` appends only `CLAUDE_CODE_TMPDIR=`+providerHome for claude_code (`:202-213`); provider home lives under the private root removed at cleanup (`internal/app/tmux.go:114`, `:1135`) | accurate |
| `README.md:150-152`, `roles.md:40-44`: "The OS sandbox, not the environment, is the boundary; the exact staged Claude client … is the only process path allowed to read `~/.claude.json`" | `claudeAuthenticationPaths` returns exactly `~/.claude.json` (`internal/app/isolation_darwin.go:185-187`), granted only under `(with-filter (process-path <client>) (allow file-read* …))` (`:131-134`) | accurate |
| "…and to start the system keychain client on the narrowly granted keychain path" | `internal/app/isolation_darwin.go:149-163`: only `/usr/bin/security` may read the keychain stores, and only the staged client may `process-exec` it | accurate |
| "`--safe-mode` stops the client loading non-managed customizations" | `internal/app/agent_runner.go:151-156` + `internal/app/isolation_darwin.go:42-44`, `:189-196` — scope stated correctly (it does **not** stop admin-managed policy) | accurate |
| `README.md:155-157`, `roles.md:44-47`: home directory, `~/.claude`, settings, history, plugins, skills, hooks, MCP config, sessions/projects, **and the admin-managed settings tree** "stay denied to the client" | default `(deny default)` (`isolation_darwin.go:65`); the `/Library/Application Support/ClaudeCode` grant is gone and both managed trees are denied **last** so no earlier rule can shadow them (`:164-177`, `claudeManagedPolicyPaths` `:197-200`) — this is also the MF4 fix, and it makes the doc sentence true | accurate |
| `README.md:157-159`, `roles.md:47-50`: a model-invoked tool "can read neither the account record nor the keychain, cannot start the keychain client, and cannot reach the user's home" | different process path → the `with-filter` grants do not apply; `(with-filter (require-not (process-path <client>)) (deny mach-lookup …))` (`:140-142`) and `(with-filter (require-not (process-path <client>)) (deny process-exec "/usr/bin/security"))` (`:161-162`) | accurate |

No new inaccuracy introduced. `docs/roles.md:11-25` model table matches `prompts/models.json`
value-for-value; `roles.md:55-57` ("copied to `.control/prompts/` after the terminal cleanup
attempt") is confirmed by the archive ordering *and* by the smoke run's mtimes.

`docs/workflow.md` and `docs/artifacts.md` stayed accurate: `workflow.md:5-15` ordering matches
`internal/app/app.go:95-150`; its flowchart (`:17-40`, evidence→story→clarity→copy with a
must-fix branch and a "candidate below 003?" cap) is exactly what the smoke run executed.
`docs/artifacts.md:5-32` run layout matches the real directory tree file-for-file, including
`.control/prompts` "post-cleanup audit copies"; `:36-70` matches `internal/app/app.go:193-205`
and the real record shape.

### MF2 — vacuous `codex-homes` assertions — **NOT IN MY LENS**
(Confirmed only that no test function was deleted; see the sweep below.)

### MF4 — admin-managed settings tree grant — **NOT IN MY LENS**
(Verified incidentally as part of the MF3 doc/code check: the grant is removed and the trees are
explicitly denied last at `internal/app/isolation_darwin.go:164-177`.)

### MF5 — Max preflight lifecycle bound — **NOT IN MY LENS**

## Full contract sweep (Priority 3)

**Policy content — PASS.** `prompts/models.json` declares `"schema_version": 1` and exactly the
eight contracted roles with the contracted values, and nothing else (`grep -c '"provider"'` = 8).
No `visual_editor`, no `human_editor`, no extra key.

**No scope creep — PASS.**
`grep -rniE 'visual_editor|image gener|fable|fast mode|api billing|usage credit'` over the
product tree (excluding `smoke-runs/`, `tmp/`, `docs/review/`) matches only prose about
*removing* API billing (`README.md:146`, `internal/app/agent_runner.go:181`,
`internal/app/claude_auth.go:28`) and three negative test fixtures
(`internal/app/model_policy_blackbox_test.go:271,442-443,445`). The changed file set still
matches the contract's in-scope list; `smoke-runs/` and `tmp/` are evidence/scratch per contract.

**Rejection cases still pre-run — PASS.** `TestBlackBoxInvalidPolicyFailsBeforeRunCreation` and
`TestBlackBoxClaudeMaxPreflightRunsBeforeRunCreation` pass (command above), each subtest still
asserting `os.Stat(runDir)` is `ErrNotExist` and zero launched agents. Effort vocabularies in
`README.md:109-112` still match `internal/app/policy.go:35-38`; the rejection list in
`README.md:116-120` still matches `parseModelPolicy`.

**Eleven required black-box items — all still covered, all passing.**

| # | Required item | Covering test | Status |
| --- | --- | --- | --- |
| 1 | Exact profile per role/lens | `TestBlackBoxEveryRoleLaunchesItsDeclaredProfile` | pass |
| 2 | Revisions preserve profiles | `TestBlackBoxRevisionsPreserveDeclaredProfiles` | pass |
| 3 | Separate executables / empty value / unused provider | `TestBlackBoxProviderExecutableSelectionIsIndependent` + `TestBlackBoxExplicitEmptyExecutableOverridesFailBeforeRunInitialization` | pass |
| 4 | Max preflight failure modes pre-run | `TestBlackBoxClaudeMaxPreflightRunsBeforeRunCreation` | pass |
| 5 | Claude flags / no `--bare` / no external credential | `assertLaunchedProfile` + `TestBlackBoxProviderProcessesNeverReceiveExternalCredentials` | pass |
| 6 | Sandbox child-tool denial | `TestBlackBoxClaudeKeychainAccessIsProcessScoped` (+ `internal/app/isolation_darwin_test.go` for the managed tree) | pass |
| 7 | Strict parsing rejections | `TestBlackBoxInvalidPolicyFailsBeforeRunCreation` | pass |
| 8 | Post-validation mutation identity | `TestBlackBoxPolicyMutationAfterValidationCannotChangeLaunchedProfiles` | pass |
| 9 | Audit/argv/digest binding | `assertPolicyBinding` (used by 1, 2, 5, 8, 10) | pass |
| 10 | Unavailable model blocks, no fallback | `TestBlackBoxUnavailableModelBlocksWithoutFallback` | pass |
| 11 | Lifecycle cleanup | `TestBlackBoxTimeoutBlocksAndCleansProcesses`, `TestBlackBoxDetachedDescendantsAreKilledOnTerminalPaths`, `TestBlackBoxPersistentCleanupFailurePreservesOwnershipButDeletesCredentials` | pass |

No item is now uncovered.

**No test or validation deleted or weakened — PASS.** Test-function sets are unchanged versus
`main` for every pre-existing test file:

```
$ diff <(git show main:internal/app/blackbox_test.go | grep -o '^func Test[A-Za-z0-9_]*' | sort) \
       <(grep -o '^func Test[A-Za-z0-9_]*' internal/app/blackbox_test.go | sort)     # no output
blackbox_test.go 56 → 56;  security_regression_test.go 5 → 5;  credential_boundary_unix_test.go 1 → 1
```
The only assertions removed from those files are exactly the four vacuous `codex-homes` checks
that MF2 replaced with the stronger `provider-homes` helper — a strengthening, not a weakening.
No editorial artifact validation was relaxed: the run's candidate 1 and 2 were legitimately sent
back by the Story and Clarity reviewers under the existing contract, which is what produced the
three candidates.

## Findings

### 1. Two orphaned `agent-runner` processes and their tmux sessions survive from an earlier black-box test run
- **Severity**: Low (**uncertain** — origin not conclusively established; not attributable to the smoke run)
- **Location**: `internal/app/tmux.go:1083-1140` (`archiveAll`/`cleanup` path) — observed at runtime, not in a diff hunk
- **Description**: While verifying MF1's "no run-owned tmux session, no orphan process" clause I
  found two live `write-uuter-*` tmux sessions, each holding a running `agent-runner` process,
  from black-box test runs started 2026-08-24 21:07:37 and 21:11:56 — roughly four hours before
  the check, with no `app.test` binary still alive. Their private roots under `$TMPDIR` also
  survive, which means the owning test binary never reached its cleanup. This is most likely an
  interrupted/killed `go test` (if the controller is SIGKILLed it cannot run its own cleanup,
  which is an acknowledged limitation, not a defect), so I am **not** claiming a product bug —
  but it is a live counter-example to black-box item 11's guarantee that no run-owned process
  survives, and it should be dispositioned rather than left unexplained.
  **It is not from `smoke-runs/mixed-20260825-2`**: both sessions predate that run's 00:25:23
  start, and the run's own private root and session are gone.
- **Evidence**:
  ```
  $ tmux list-sessions
  write-uuter-a4a91b09fd19: 1 windows (created Mon Aug 24 21:11:56 2026)
  write-uuter-bbefd43d9e48: 1 windows (created Mon Aug 24 21:07:37 2026)
  $ tmux list-panes -s -t '=write-uuter-a4a91b09fd19' -F '#{pane_pid} #{pane_current_command}'
  8044 agent-runner
  $ ps -o pid,lstart,command -p 8044 -p 44070
   8044 Mon Aug 24 21:11:56 2026  …/TestBlackBoxMalformedReviewReportsAndWhitespaceAreRejected…/.write-uuter-private-4099281063/control/agent-runner __agent --provider codex --model gpt-5.6-sol --effort high …
  44070 Mon Aug 24 21:07:37 2026  …/TestBlackBoxPMHistoryCannotBePrepopulatedOrDropped…/.write-uuter-private-296910389/control/agent-runner …
  $ ps -eo command | grep 'app.test'   # only the concurrently running suite started 2026-08-25 00:59:36
  ```
  For contrast, the smoke run is clean: `find . -name '.write-uuter-private-*'` → empty, and no
  session exists in the run's 00:25–00:50 window.
- **Suggestion**: Confirm whether these two came from an interrupted `go test` (in which case
  simply `tmux kill-session -t write-uuter-a4a91b09fd19 write-uuter-bbefd43d9e48` and record the
  cause). If instead the suite can leave them behind on a *normal* exit, that is a real
  regression against required item 11 and belongs to the resilience/lifecycle lens — reproduce by
  running `TestBlackBoxMalformedReviewReportsAndWhitespaceAreRejected` and
  `TestBlackBoxPMHistoryCannotBePrepopulatedOrDropped` alone to completion and re-checking
  `tmux list-sessions`.
