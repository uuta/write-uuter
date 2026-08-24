# Manager Pass — issue uuta/write-uuter#4 (branch feat/4)

## Perspectives run

| Lens | Engine / model / effort | Output |
| --- | --- | --- |
| requirements (floor) | claude opus, medium | docs/review/requirements.md |
| correctness (floor) | codex gpt-5.6-sol, medium | docs/review/correctness.md |
| security (floor) | codex gpt-5.6-sol, medium | docs/review/security.md |
| resilience (optional) | codex gpt-5.6-sol, low | docs/review/resilience.md |
| reuse (optional) | codex gpt-5.6-sol, low | docs/review/reuse.md |
| api-contract (optional) | codex gpt-5.6-sol, medium | docs/review/api-contract.md |

Run-scoped tmux targets recorded in `docs/review/tmux-targets.env` (tag `review-issue4-wt4-42380`,
windows `@47463`-`@47468`, panes `%47490`-`%47495`); all six killed after the pass.

Optional lenses matched but dropped (cap ~3, highest risk kept):
- `concurrency` — process-group/ownership primitives are touched, but the diff only generalizes
  the pre-existing owner; `resilience` covers the lifecycle risk. Dropped for cost.
- `data-migration` — `schema_version` is a config schema, not a stored-data migration;
  `api-contract` covers the schema surface. Dropped for cost.
- `i18n-a11y` — no user-facing strings or interactive UI in the diff.
- `ui-visual` — NOT required: the diff contains no web or Flutter UI files
  (`docs/review/touched.txt` is Go, Markdown, and JSON only). Confirmed by classification,
  not assumed.

## Overall assessment

The contract is met on every clause that can be checked statically or with the deterministic
fake-CLI suite. `prompts/models.json` carries exactly the eight contracted profiles; it is bound
through the same no-follow `os.Root` bundle boundary as the role prompts; every rejection case
fails before the run directory exists and before tmux starts; `--claude` behaves as specified;
the Max preflight is strict, sanitized, and correctly ordered; argv and the `.control/invocations`
audit entry are both generated from the same `roleProfile` value; the run policy copy is
byte-identical to `prompts/models.json` and its `sha256:` digest agrees across `workflow.json`
and every audit record. No scope creep into Issue #3 and no Fable/fallback/API-billing path exists.
The reuse lens found no parallel implementation.

What blocks a clean verdict is verification, plus four concrete defects.

## Confirmed blockers

1. **No successful authenticated mixed-provider smoke run** (required verification item).
   The only retained new run, `smoke-runs/mixed-20260824-1/`, is `status: blocked` at Writer.
   Runs -2 and -3 are not retained. Consequently the four reviewer profiles
   (`reviewer_evidence`, `reviewer_story`, `reviewer_clarity`, `reviewer_copy`) have **zero
   runtime evidence** — only four invocations ever launched — and "final reviewers bind the
   published revision" is unverified at runtime. Aggravating context: the three pre-existing
   all-Codex runs on `main` (`smoke-runs/real-20260823-{29,31,34}`) all reached
   `status: succeeded`, so the mixed-provider blockage is specific to this change, not a
   pre-existing flakiness. Do not weaken editorial artifact validation to clear it.

2. **Four credential-cleanup assertions were silently disarmed by the `codex-homes` →
   `provider-homes` rename.** `internal/app/blackbox_test.go:1150,1331,1429,1463` still assert the
   absence of `codex-homes`; production only ever creates `provider-homes`
   (`internal/app/tmux.go:114`, removed at `:783`). Those `os.Lstat`/`os.Stat` calls now always
   return `ErrNotExist`, so the assertions pass unconditionally and no longer prove that staged
   credentials are removed on the probe-failure and persistent-cleanup-failure paths.

3. **README and docs/roles.md state the opposite of what the code does about `HOME`.**
   `README.md:147` — "Claude processes receive a run-owned home instead of the real user home";
   `docs/roles.md:35-38` — "Claude processes run with a run-owned home and scratch directory
   rather than the real user home, so user Claude configuration ... stay outside the sandbox".
   The code deliberately passes the real `HOME` through (`internal/app/agent_runner.go:186`,
   `:205-213`) and redirects only `CLAUDE_CODE_TMPDIR`; the project's own test asserts the real
   home reaches Claude (`internal/app/model_policy_blackbox_test.go:283-300`). The roles.md
   sentence additionally overstates the boundary: the sandbox grants the exact staged client
   read access to `~/.claude.json` (`internal/app/isolation_darwin.go:52-55`, granted at
   `:134-137`); only model-invoked tools are denied it.

4. **The sandbox grants the Claude client the admin-managed-settings tree.**
   `internal/app/isolation_darwin.go:52-55` adds `/Library/Application Support/ClaudeCode` to
   `claudeClientReads`. That is the documented macOS location of `managed-settings.json`, which
   can carry an `env` object (`ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`,
   Bedrock/Vertex selectors, `ANTHROPIC_MODEL`) and `apiKeyHelper`. `--safe-mode` does not
   suppress it — verified locally: `claude --help` states "Admin-managed (policy) settings still
   apply." Those values are applied by Claude *after* `providerBaseEnvironment()` has sanitized
   the inherited environment, so the allowlist does not close this channel, and the preflight
   checks only three auth-status fields. Latent, not live, on this host: the directory does not
   exist here, so `existingIsolationPaths` emits no grant today. But the grant is a deliberate
   ambient-config channel into exactly the two "Not done if" clauses about external provider
   credentials and ambient config changing the recorded policy.

5. **The Max preflight has no hard lifecycle or output bound.**
   `internal/app/claude_auth.go:32-43` uses `exec.CommandContext` with a `bytes.Buffer` stdout and
   no `configureProcessGroup`, no `WaitDelay`, and no size limit. `CommandContext` kills only the
   direct child; a surviving descendant that inherited the stdout pipe keeps the copy goroutine
   alive, so `command.Run()` can outlive the 30s context indefinitely. This runs before
   `tmuxRuntime` exists, so none of the ownership-tracker cleanup applies. The launched-agent path
   does this correctly (`internal/app/agent_runner.go:111,115-140`).

## Findings judged NOT blocking (kept as optional)

- Preflighted-vs-staged executable identity (`internal/app/app.go:126` → `internal/app/tmux.go:83-93,137-142`):
  only a pathname, not a descriptor or `os.SameFile` identity, connects the preflight to the
  staged bytes. Real TOCTOU, but exploiting it requires write access to the resolved `claude`
  binary, at which point the host is already compromised. Worth hardening, not a blocker.
- Unbounded preflight stdout accumulation — folded into blocker 5's fix, listed separately as
  hardening.
- `assertLaunchedProfile` never compares `audit.Role` (`internal/app/model_policy_blackbox_test.go:134-142`).
- Unknown role-profile fields are rejected with `json: unknown field "x"` and no role name
  (`internal/app/policy.go:71` → `internal/app/strict_json.go:24-27`); the test at
  `:443-445` only asserts the substring `unknown field`.
- `docs/artifacts.md:42` does not state the `sha256:[0-9a-f]{64}` grammar or that `lens`/`candidate`
  are conditionally omitted (`internal/app/models.go:76-77`).
- `--codex` is resolved inside `newTmuxRuntime`, i.e. after `initialize`, while `--claude` is
  resolved pre-run. Pre-existing `main` behavior; the issue only mandates a pre-run gate for the
  Claude preflight. Inconsistency, not a contract breach.
- `WRITE_UUTER_PROMPTS_DIR` selects the policy along with the prompts. Documented bundle
  precedence, and the selected policy is always copied and digested, so it is not "silent".
  README's Model policy section mentions only `--prompts-dir`.

## Residual risks / verification gaps

- Reviewer-lens profiles are covered only by fake-CLI tests. No real reviewer invocation exists.
- `TestBlackBoxClaudeKeychainAccessIsProcessScoped` is darwin-only; the non-darwin isolation path
  is a hard error (`internal/app/isolation_other.go`), so Linux has no equivalent coverage.
- The full suite is slow (~10 min unloaded) and exceeds Go's default 10-minute package timeout
  under CPU load. It is not flaky in content, but `-timeout` must be raised in CI.
