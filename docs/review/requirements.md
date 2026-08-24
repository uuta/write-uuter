# Requirements Compliance and Scope Control Review

## Summary
Every clause of the issue-4 contract that can be checked statically or with the deterministic
fake-CLI suite is implemented and covered: `prompts/models.json` holds exactly the eight
contracted profiles, is bound through the same no-follow bundle boundary as the role prompts,
all twelve rejection cases fail before the run directory exists and before tmux starts, the
`--claude` flag and Claude Max preflight behave as specified, argv and audit artifacts are
generated from the same validated profile, and there is no scope creep into Issue #3 or the
out-of-scope list. Two documentation statements (README and `docs/roles.md`) assert a
"run-owned home instead of the real user home" for Claude processes, which the code and its
own test explicitly contradict.

## Verification commands run

```
$ go build ./...                                   # exit 0, no output
$ go vet ./...                                     # exit 0, no output
$ gofmt -l .                                       # no output
$ go test ./internal/app -count=1 -run 'TestBlackBox(EveryRole|InvalidPolicy|ClaudeMaxPreflight|ProviderExecutableSelection|ReviewerLenses|Revisions|PolicyMutation|UnavailableModel|ProviderProcessesNever)'
ok  	github.com/uuta/write-uuter/internal/app	101.619s
$ shasum -a 256 prompts/models.json
f5a91c462dea2aebe51b2870213c4a193ce0c1576f95ec983b018ae30b16752b  prompts/models.json
```

## Clause-by-clause result

### 1. Policy file exists, is version-controlled, schema v1, exactly eight roles — PASS
`prompts/models.json:1-42` is a new tracked file (`git status`: `A prompts/models.json`) with
`"schema_version": 1` and exactly the eight contracted keys and values:
pm=codex/gpt-5.6-sol/high, researcher=claude_code/claude-sonnet-5/medium,
story_editor=claude_code/claude-opus-5/high, writer=claude_code/claude-opus-5/medium,
reviewer_evidence=codex/gpt-5.6-sol/medium, reviewer_story=claude_code/claude-sonnet-5/medium,
reviewer_clarity=claude_code/claude-sonnet-5/medium, reviewer_copy=codex/gpt-5.6-luna/low.
No `visual_editor`, no `human_editor`, no extra or missing role
(`grep -c '"provider"' prompts/models.json` = 8). `internal/app/policy.go:23-32` declares the
same eight keys as the exact required set, with the Human-Editor exclusion documented at
`internal/app/policy.go:19-22`.

### 2. Same stable no-follow bundle boundary as role prompts — PASS
`internal/app/prompts.go:14-32` puts `models.json` first in `requiredPromptFiles`, so it is
opened by `openPromptBundle` → `openValidated` (`internal/app/prompts.go:87-148`) with the same
`os.Root`, `Lstat` symlink/regular-file check, and `os.SameFile` identity re-check as every
role prompt, and is read from the held descriptor in `load` (`internal/app/prompts.go:152-175`).
`internal/app/app.go:100-110` loads and validates it through that bound bundle for both the
ambient checked-in bundle and an explicit `--prompts-dir` bundle (the same
`openPromptsBundle(config.PromptsDir, explicit)` call at `internal/app/app.go:95`).
Verified behaviourally by `TestBlackBoxPolicyMutationAfterValidationCannotChangeLaunchedProfiles`
(`internal/app/model_policy_blackbox_test.go:490`), which replaces the policy file and then the
whole bundle root mid-run and asserts the run copy and every launched profile are unchanged.

### 3. All rejection cases implemented, before run dir creation and before tmux — PASS
Ordering: `internal/app/app.go:104-110` (load + `parseModelPolicy`) and
`internal/app/app.go:121-136` (executable resolution + Max preflight) all run before
`control.initialize` at `internal/app/app.go:150`, which is the first call that creates any
directory (`internal/app/app.go:156-165`), and long before `newTmuxRuntime` at
`internal/app/app.go:264`.

| Rejection case | Implementation | Test case |
| --- | --- | --- |
| missing policy | `internal/app/prompts.go:110-116` (required bundle file) | `"missing file"` |
| unsupported schema_version | `internal/app/policy.go:74-75` | `"unsupported schema"` |
| duplicate JSON key | `internal/app/strict_json.go:12-16`, `:37-57` | `"duplicate key"` |
| missing role | `internal/app/policy.go:96-102` | `"missing role"` |
| unknown role | `internal/app/policy.go:84-94` | `"unknown role"` |
| unknown field (top level and per role) | `internal/app/strict_json.go:23` (`DisallowUnknownFields`) | `"unknown top-level field"`, `"unknown role field"` |
| unsupported/invalid provider value | `internal/app/policy.go:116-120` | `"unsupported provider"` |
| empty model | `internal/app/policy.go:121-123` | `"empty model"` |
| invalid effort value | `internal/app/policy.go:132-140` | `"invalid effort for provider"`, `"unknown effort"` |
| `gpt*` on claude_code | `internal/app/policy.go:42-45`, `:127-129` | `"gpt model on claude"` |
| `claude*` on codex | same | `"claude model on codex"` |

All test cases are subtests of `TestBlackBoxInvalidPolicyFailsBeforeRunCreation`
(`internal/app/model_policy_blackbox_test.go:407-478`), each asserting both
`os.Stat(runDir)` is `ErrNotExist` and zero launched agents
(`internal/app/model_policy_blackbox_test.go:471-477`). Extra strictness beyond the contract
(padded model at `internal/app/policy.go:124-126`, missing `roles` object at
`internal/app/policy.go:77-79`, empty policy file at `internal/app/prompts.go:171-173`) is
also covered.

### 4. No allowlist / defaults / sharing / override / routing / fallback / Fable 5 / billing / Fast mode — PASS
- No model allowlist: `internal/app/policy.go:33-39` documents and implements effort-only
  vocabulary; there is no model list anywhere.
- No default and no sharing: `profileFor` (`internal/app/policy.go:148-158`) returns an error
  for an unmapped key instead of a default; every call site passes a literal policy role key
  (`internal/app/app.go:276` `"pm"`, `:344` `"researcher"`, `:459` `"story_editor"`, `:524`
  `"writer"`, `:638` `"reviewer_"+lens`).
- No per-run override: `cmd/write-uuter/main.go:26-33` exposes no model/effort flag, and
  `grep -rn 'MODEL\|EFFORT' internal/app/*.go cmd/write-uuter/main.go` (non-test) returns
  nothing — no environment override path exists.
- No fallback/routing: `internal/app/agent_runner.go:150-176` always emits explicit
  `--model`/effort for both providers with no retry or alternate-model branch;
  `TestBlackBoxUnavailableModelBlocksWithoutFallback`
  (`internal/app/model_policy_blackbox_test.go:540-575`) asserts exactly one writer launch and
  a `blocked` status carrying `provider=claude_code model=claude-opus-5 reasoning_effort=medium`.
- No Fable 5 / API billing / Fast mode / escalation:
  `grep -rniE 'visual_editor|image gener|fable|fast mode|billing|credit' --include='*.go' --include='*.md' --include='*.json'`
  outside `smoke-runs/`, `tmp/`, and `docs/review/` matches only prose about *removing* API
  billing (`README.md:146`, `internal/app/agent_runner.go:181`, `internal/app/claude_auth.go:20`)
  and two negative test fixtures (`internal/app/model_policy_blackbox_test.go:271`, `:444`).

### 5. `--claude` flag semantics and provider-scoped canonicalization/staging — PASS
`cmd/write-uuter/main.go:30` declares `--claude` with default `"claude"`;
`cmd/write-uuter/main.go:42-43` records explicitness via `flags.Visit`;
`internal/app/app.go:72-74` rejects an explicit empty value and `internal/app/app.go:81-83`
applies the default only when unset. Only referenced providers are resolved
(`internal/app/app.go:121-136`) and staged (`internal/app/tmux.go:75-93`, `:133-160`).
Covered by `TestBlackBoxExplicitEmptyExecutableOverridesFailBeforeRunInitialization`
(`internal/app/blackbox_test.go:814-842`, includes a `--claude=` subtest) and
`TestBlackBoxProviderExecutableSelectionIsIndependent`
(`internal/app/model_policy_blackbox_test.go:315-353`, both separate-executable and
unused-provider-not-required subtests). Separate deterministic fakes are supported through
`executableTag()` in `internal/app/testdata/fakeagent/main.go:73-79`.

### 6. Claude Max preflight semantics and ordering — PASS
`internal/app/claude_auth.go:29-49` runs `claude auth status` with
`command.Env = providerBaseEnvironment()` — the identical allowlist used to build invocation
environments (`internal/app/agent_runner.go:183-206`), so `ANTHROPIC_API_KEY` and every
alternative provider credential are absent by construction (stronger than the issue's
`env -u ANTHROPIC_API_KEY`). `internal/app/claude_auth.go:53-90` strictly requires
`loggedIn=true`, `authMethod=claude.ai`, `subscriptionType=max`, rejects duplicate keys and
multiple documents, and reads only those three fields — no identity value is returned or
persisted. It is invoked at `internal/app/app.go:132` on the canonicalized executable
(`internal/app/app.go:126`, `resolveClaudeExecutable` at `internal/app/app.go:1597-1607`),
i.e. before `initialize` (`internal/app/app.go:150`) and before `newTmuxRuntime`
(`internal/app/app.go:264`), and is skipped entirely when `policy.usesProvider(providerClaudeCode)`
is false (`internal/app/app.go:125`, `internal/app/policy.go:161-171`). Nine subtests in
`TestBlackBoxClaudeMaxPreflightRunsBeforeRunCreation`
(`internal/app/model_policy_blackbox_test.go:355-405`) cover logged out, apiKey, non-max,
malformed, missing field, duplicate key, two documents, non-zero exit, and missing executable,
each asserting no run directory and zero launched agents, plus asserting that the fixture's
`email`/`orgId` (`internal/app/testdata/fakeagent/main.go:152`) never appear in output.

### 7. Codex and Claude argv exactly as contracted; `--bare` never appears — PASS
`internal/app/agent_runner.go:153-176`. Claude:
`--print --safe-mode --dangerously-skip-permissions --no-session-persistence --model <m> --effort <e>`
with the prompt on stdin (`internal/app/agent_runner.go:103`); no `--bare` token exists anywhere
in the repository (`grep -rn '\-\-bare' internal/app cmd` → only the forbidding comment at
`internal/app/agent_runner.go:157` and the negative assertion at
`internal/app/model_policy_blackbox_test.go:110`). Codex:
`--model <m> --config model_reasoning_effort="<e>"` prepended to the pre-existing
`--dangerously-bypass-approvals-and-sandbox -C <ws> exec --ephemeral --ignore-user-config
--ignore-rules --skip-git-repo-check -` (identical to `main`'s vector per
`git diff main -- internal/app/agent_runner.go`, so preserved behavior is intact).
`internal/app/agent_runner.go:47-59` makes `model` and `effort` mandatory non-empty runner
arguments, so the old `codex exec --ignore-user-config` without explicit model/effort is
unreachable.

### 8. Artifact contract — PASS
- Run policy copy: `internal/app/app.go:193-195` writes `model-policy.json` (mode `0o444`) from
  the exact validated bytes captured at `internal/app/policy.go:112`.
- Digest in `workflow.json`: `internal/app/models.go:16` field, set at `internal/app/app.go:199`.
- Audit entries: `InvocationAudit` (`internal/app/models.go:81-90`) carries invocation, role,
  lens, candidate, provider, model, reasoning_effort, model_policy_digest;
  `publishInvocationAudit` (`internal/app/app.go:1613-1638`) builds them from `inv.Profile` —
  the same value passed to `prepareInvocation` and thence into argv — and writes them with
  `writeAtomicNoReplace(..., 0o444)`.
- Before readiness: published at `internal/app/app.go:284` (PM) and `internal/app/app.go:1215`
  (workers), both strictly before `startWorker`/`startPM` and before the ready marker is
  published in `internal/app/agent_runner.go:120`.
- Retained on all exit paths: nothing removes `.control/invocations`; verified for the blocked
  path by `TestBlackBoxUnavailableModelBlocksWithoutFallback`
  (`internal/app/model_policy_blackbox_test.go:559-563`) and by the retained blocked smoke run.
- No secrets/prompts/env/auth: asserted per record in `readAuditRecords`
  (`internal/app/model_policy_blackbox_test.go:55-58`).
- Field shape matches the issue example exactly, including `role: "reviewer"` + `lens: "copy"`
  (the lens suffix is trimmed at `internal/app/app.go:1615-1618`).

### 9. Documentation — ONE CONTRADICTION (see Finding 1); everything else accurate
Spot-checked against code: the README and `docs/roles.md` model tables match
`prompts/models.json` value-for-value; the effort vocabularies in `README.md:109-110` match
`internal/app/policy.go:35-39`; the rejection list in `README.md:114-118` matches
`parseModelPolicy`; the flag names/defaults in `README.md:58-59` match
`cmd/write-uuter/main.go:26-33`; the artifact names, paths, and the JSON example in
`docs/artifacts.md:40-70` match `internal/app/app.go:193-205` and the real record shape; the
`docs/workflow.md:2-16` ordering claim matches `internal/app/app.go:95-150`.

### 10. Test coverage of the eleven required black-box items

| # | Required item | Covering test | Status |
| --- | --- | --- | --- |
| 1 | Happy path: exact executable/model/effort per role and lens | `TestBlackBoxEveryRoleLaunchesItsDeclaredProfile` (`internal/app/model_policy_blackbox_test.go:193`) | covered |
| 2 | Revision path preserves profiles | `TestBlackBoxRevisionsPreserveDeclaredProfiles` (`:212`) | covered |
| 3 | `--claude`/`--codex` select separate fakes; empty paths fail; unused provider not required | `TestBlackBoxProviderExecutableSelectionIsIndependent` (`:315`) + `TestBlackBoxExplicitEmptyExecutableOverridesFailBeforeRunInitialization` (`internal/app/blackbox_test.go:814`) | covered |
| 4 | Max preflight: pass plus missing/malformed/logged-out/API/non-Max/non-zero fail pre-run | `TestBlackBoxClaudeMaxPreflightRunsBeforeRunCreation` (`:355`); the pass case is exercised by every happy-path scenario via the `auth_ok` fixture (`internal/app/testdata/fakeagent/main.go:152`) | covered |
| 5 | Claude argv flags, never `--bare`; hostile config/env cannot change argv or artifacts; no external credential reaches the process | `assertLaunchedProfile` (`:105-116`) + `TestBlackBoxProviderProcessesNeverReceiveExternalCredentials` (`:264`) | covered |
| 6 | Sandbox regression: staged client authenticates, model-invoked tool cannot read Keychain/`~/.claude`/`~/.claude.json` | `TestBlackBoxClaudeKeychainAccessIsProcessScoped` (`:577`) | covered (darwin-only; `t.Skip` elsewhere) |
| 7 | Strict config: malformed/missing/unknown/mismatched/invalid fail pre-run | `TestBlackBoxInvalidPolicyFailsBeforeRunCreation` (`:407`) | covered |
| 8 | Policy identity under post-validation mutation | `TestBlackBoxPolicyMutationAfterValidationCannotChangeLaunchedProfiles` (`:490`) | covered |
| 9 | Audit binding to captured argv and copied digest | `assertPolicyBinding` (`:139-190`), used by items 1, 2, 5, 8, 10 | covered |
| 10 | Unavailable model/provider/quota blocks with no fallback process | `TestBlackBoxUnavailableModelBlocksWithoutFallback` (`:540`) | covered |
| 11 | Lifecycle: no run-owned process on success or failure paths | `assertProcessesGone` in `TestBlackBoxUnavailableModelBlocksWithoutFallback` (`:574`) plus the preserved suite (`TestBlackBoxTimeoutBlocksAndCleansProcesses`, `TestBlackBoxDetachedDescendantsAreKilledOnTerminalPaths`, `TestBlackBoxPersistentCleanupFailurePreservesOwnershipButDeletesCredentials`) | covered |

No required black-box item is uncovered. One narrow gap inside item 4 is noted as Finding 3.

### 11. Scope creep — NONE FOUND
No `visual_editor` role, image generation, pricing/usage accounting, Fable 5 path, per-run
override, or Fast-mode/credit enabling exists in product code (grep above). The changed file
set matches the contract's in-scope list exactly; `tmp/engineer-brief.txt` is PM scratch and
`smoke-runs/` is evidence, both excluded from product scope by the contract.

### 12. Retained smoke evidence
`smoke-runs/mixed-20260824-1/` is the only new smoke run (the three `real-20260823-*` runs are
already tracked on `main`). Its claims are internally consistent: `workflow.json` reports
`"status": "blocked"` with `block_reason: "writer artifact contract failed after process
completion: candidate contains unresolved TODO placeholder"`, `drafts/` is empty, `article.md`
is absent, and exactly four invocations (`001-pm`, `002-researcher`, `003-story-editor`,
`004-writer`) have matching `.control/{invocations,prompts,logs,exits}` entries. The run proves
mixed-provider execution (`001-pm` = codex/gpt-5.6-sol/high, `002/003/004` = claude_code) and
that the Max preflight passed against the real CLI. `model-policy.json` is byte-identical to
`prompts/models.json` (`diff` → identical), and the digest recorded in `workflow.json` and in
all four audit records, `sha256:f5a91c46…16752b`, equals `shasum -a 256 prompts/models.json`.
`grep -rniE "ANTHROPIC|authMethod|subscriptionType|token|secret|api[_-]?key|sk-ant|Bearer|@…\.(com|ai|jp)"`
over the run returns only editorial prose about the artifact contract itself (e.g.
`smoke-runs/mixed-20260824-1/claim-ledger.md:100`, `outline.md:239`) — no credential, no raw
auth identity field. This run does not demonstrate a *successful* mixed-provider workflow,
which is the already-known gap recorded in the contract and is not re-litigated here.

## Findings

### 1. README states Claude processes get a run-owned home; the code deliberately keeps the real user home
- **Severity**: Medium
- **Location**: `README.md:147`
- **Description**: The README says "Claude processes receive a run-owned home instead of the
  real user home". The implementation does the opposite by design: only `CLAUDE_CODE_TMPDIR`
  is redirected to the run-owned provider home, while `HOME` is passed through unchanged
  because the Max session is resolved from the user's top-level Claude configuration file and
  the login keychain, with the sandbox — not the environment — enforcing isolation. A reader
  auditing the credential boundary from the README would conclude the wrong mechanism is in
  force and would not know the sandbox is the only thing keeping user Claude state out of reach.
- **Evidence**:
  `internal/app/agent_runner.go:205-213`:
  ```go
  // ... HOME is deliberately left alone:
  // the Max session is resolved from the user's top-level Claude
  // configuration file and the login keychain, and the sandbox - not the
  // environment - is what keeps every other part of the user's Claude
  // state out of reach.
  environment = append(environment, "CLAUDE_CODE_TMPDIR="+providerHome)
  ```
  `providerBaseEnvironment` (`internal/app/agent_runner.go:186`) copies `HOME` verbatim, and the
  project's own test asserts the real home reaches Claude invocations —
  `internal/app/model_policy_blackbox_test.go:283-300`:
  ```go
  realHome, err := os.UserHomeDir()
  ...
  if record.Environment["HOME"] != realHome {
      t.Errorf("%s Claude invocation did not receive the authenticated user home: %q", ...)
  }
  ```
- **Suggestion**: Replace the sentence with what the code does, e.g. "Claude processes keep the
  real user home so the Max session resolves, but their scratch root
  (`CLAUDE_CODE_TMPDIR`) is run-owned, and the OS sandbox — not the environment — is what keeps
  `~/.claude`, settings, history, plugins, hooks, MCP configuration, and session state out of
  reach."

### 2. docs/roles.md repeats the run-owned-home claim and overstates what is outside the sandbox
- **Severity**: Medium
- **Location**: `docs/roles.md:35-38`
- **Description**: Two statements are contradicted by the code. (a) "Claude processes run with a
  run-owned home … rather than the real user home" — same contradiction as Finding 1. (b) "so
  user Claude configuration … stay outside the sandbox" attributes the isolation to the wrong
  mechanism *and* overstates it: the sandbox profile explicitly grants the exact staged Claude
  client read access to `~/.claude.json` (the account record that identifies the Max session).
  Only model-invoked tools are denied it. The doc as written would lead a reader to believe the
  client itself cannot read any user Claude configuration.
- **Evidence**:
  `internal/app/isolation_darwin.go:52-55`:
  ```go
  claudeClientReads = existingIsolationPaths(
      filepath.Join(home, ".claude.json"),
      "/Library/Application Support/ClaudeCode",
  )
  ```
  granted at `internal/app/isolation_darwin.go:134-137` with `(with-filter (process-path <client>)
  (allow file-read* ...))`. The test encodes exactly this split —
  `internal/app/model_policy_blackbox_test.go:594-606`: `client_user_claude_config` must contain
  `READ_SUCCEEDED`, while `client_user_claude_dir`, `client_user_home`, `client_keychain_file`,
  `tool_user_claude_config`, `tool_keychain_file`, and `tool_keychain_client` must not succeed.
- **Suggestion**: Rewrite as: the real home is kept, scratch is run-owned; the OS sandbox denies
  the client the user home and `~/.claude`, permits only the account record `~/.claude.json` to
  the exact staged client, and denies model-invoked tools the account record, keychain, and user
  home entirely; `--safe-mode` additionally stops the client loading customizations.

### 3. The Max preflight's credential filtering is asserted only by construction, not by a test
- **Severity**: Low
- **Location**: `internal/app/claude_auth.go:41`
- **Description**: Contract clause 5 requires the preflight to run "with the same provider
  credential filtering used for Claude invocations". The code satisfies this by sharing
  `providerBaseEnvironment()`, but no test observes the preflight's own environment — the fake
  fixture answers `auth status` without inspecting env
  (`internal/app/testdata/fakeagent/main.go:170`), and
  `TestBlackBoxProviderProcessesNeverReceiveExternalCredentials` only captures *invocation*
  environments. A future change that gave the preflight its own env construction would not fail
  any test. Marked Low because the shared helper makes present behavior correct, and this is not
  one of the eleven required verification items.
- **Evidence**: `internal/app/claude_auth.go:41` `command.Env = providerBaseEnvironment()`;
  `internal/app/testdata/fakeagent/main.go:170` `if len(os.Args) > 2 && os.Args[1] == "auth" && os.Args[2] == "status"`
  — the branch records nothing about its environment.
- **Suggestion**: Have the `auth status` branch of the fixture dump its observed environment to
  the fixture log directory and assert in the preflight test that no external provider
  credential is present, mirroring the invocation-side assertions.

### 4. `--codex` resolution failure still occurs after the run directory is created (preserved asymmetry)
- **Severity**: Low
- **Location**: `internal/app/app.go:122-123`, `internal/app/tmux.go:83-93`
- **Description**: Contract clause 4 asks for canonicalizing and staging only the referenced
  provider executables. `--claude` is looked up and canonicalized pre-run
  (`internal/app/app.go:126`), but `--codex` is only stored verbatim and is first resolved inside
  `newTmuxRuntime`, which runs after `initialize`. A policy that references `codex` with an
  unusable `--codex` path therefore creates a run directory and blocks, whereas the equivalent
  Claude failure creates nothing. This is `main`'s pre-existing behavior
  (`git show main:internal/app/tmux.go` resolves Codex in `newTmuxRuntime` too) and the issue
  only mandates a pre-run gate for the Claude preflight, so it is reported as an inconsistency
  rather than a contract breach. It is uncertain whether the author intended symmetry here.
- **Evidence**: `internal/app/app.go:122-123`:
  ```go
  if policy.usesProvider(providerCodex) {
      providerExecutables[providerCodex] = config.CodexExecutable
  }
  ```
  versus `internal/app/app.go:126-135`, which calls `resolveClaudeExecutable` and
  `verifyClaudeMaxSubscription` before `control.initialize` at `internal/app/app.go:150`.
- **Suggestion**: If pre-run symmetry is wanted, call `exec.LookPath` + `filepath.EvalSymlinks`
  for the Codex path next to the Claude resolution and pass the canonical path into
  `newTmuxRuntime`. Otherwise, leave as is — no doc currently claims otherwise.

### 5. Ambient `WRITE_UUTER_PROMPTS_DIR` can select which policy a run uses
- **Severity**: Low
- **Location**: `internal/app/prompts.go:65-68`
- **Description**: The issue's "Not done if" list includes "Ambient Claude/Codex config or
  environment silently changes the recorded policy." Because `models.json` lives in the prompt
  bundle, the pre-existing ambient bundle precedence means `WRITE_UUTER_PROMPTS_DIR` selects the
  policy along with the prompts. This is not silent — the chosen policy is copied to
  `model-policy.json` and digested in `workflow.json`, and the clause is aimed at Claude/Codex
  ambient config rather than write-uuter's own documented bundle precedence — so I read this as
  compliant, and record it only so the reviewer chain sees it was considered.
- **Evidence**: `internal/app/prompts.go:65-68`:
  ```go
  if fromEnvironment := os.Getenv("WRITE_UUTER_PROMPTS_DIR"); fromEnvironment != "" {
      candidates = append(candidates, fromEnvironment)
  }
  ```
  Documented precedence at `README.md:71-79`; the resulting policy is always recorded verbatim
  (`internal/app/app.go:193-199`).
- **Suggestion**: No change required. Optionally note in the README's "Model policy" section that
  `WRITE_UUTER_PROMPTS_DIR` also selects the policy, since that section only mentions
  `--prompts-dir`.
