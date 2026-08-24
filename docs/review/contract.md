# Review Contract — issue uuta/write-uuter#4

- Issue: https://github.com/uuta/write-uuter/issues/4 "Configure and record model and reasoning effort for every role"
- Review range: worktree `feat/4` (tracked + untracked implementation files) vs `main`.
  Diff captured at `docs/review/diff.txt` (excludes `smoke-runs/`, `tmp/`).
  Smoke artifacts under `smoke-runs/` and PM control files under `tmp/` are evidence, not product code.

## Goal

Make the agent backend (provider), model, and reasoning effort for every write-uuter role
explicit, version-controlled, validated pre-run, passed to each invocation, and recorded in
run artifacts. Add Claude Code as a second provider using the user's Claude Max subscription.

## Canonical behavior after the change

1. **Policy file.** A required, version-controlled `models.json` lives in the resolved prompt
   bundle and is resolved/bound through the SAME stable no-follow boundary as role prompts.
   The checked-in bundle and an explicit `--prompts-dir` bundle are each complete policies.
2. **Schema v1.** Providers: only `claude_code`, `codex`. Roles exactly:
   pm=codex/gpt-5.6-sol/high; researcher=claude_code/claude-sonnet-5/medium;
   story_editor=claude_code/claude-opus-5/high; writer=claude_code/claude-opus-5/medium;
   reviewer_evidence=codex/gpt-5.6-sol/medium; reviewer_story=claude_code/claude-sonnet-5/medium;
   reviewer_clarity=claude_code/claude-sonnet-5/medium; reviewer_copy=codex/gpt-5.6-luna/low.
   Human Editor has no profile. Issue #3 `visual_editor`/assembly policy is NOT implemented here.
3. **Strict validation before run initialization / tmux.** Reject: missing policy, unsupported
   schema, duplicate JSON key, missing/unknown role, unknown field, unsupported provider, empty
   model, invalid provider or effort value, and `gpt*` on claude_code / `claude*` on codex.
   No global model allowlist, no defaults, no profile sharing, no per-run override, no dynamic
   routing, no fallback, no Fable 5, no API billing, no Fast mode, no automatic escalation.
4. **CLI.** Public `--claude <path>` alongside `--codex`; default `claude`; explicit empty value
   rejected. Canonicalize/stage ONLY provider executables referenced by the validated policy.
   Deterministic separate fake CLIs must be supportable.
5. **Claude Max preflight.** If the policy uses claude_code, before run initialization run
   sanitized `env -u ANTHROPIC_API_KEY claude auth status`; strictly require
   loggedIn=true, authMethod=claude.ai, subscriptionType=max, with the same provider credential
   filtering used for Claude invocations. Never persist raw identity/status/auth/env values.
   Skip only when no role uses claude_code.
6. **Provider-neutral runner.** One runner interface; implementations receive identical immutable
   role/task prompt, workspace boundary, timeout/cancellation, audit identity.
   Codex argv explicitly carries model and `model_reasoning_effort` plus `exec --ephemeral
   --ignore-user-config`. Claude argv uses `--print --safe-mode --dangerously-skip-permissions
   --no-session-persistence` + explicit `--model`/`--effort`, prompt via stdin; NEVER `--bare`.
7. **Preserved behavior.** Controller-owned logs, OS sandbox, lifecycle/process-group cleanup,
   timeouts, credential cleanup, candidate budget, revision behavior, reviewer order.
   Real home and `~/.claude`, `~/.claude.json`, settings/history/plugins/skills/hooks/MCP/session
   state stay outside the agent-readable sandbox. Only the exact staged Claude client gets
   OAuth/Keychain access; its spawned tools keep existing denials. `ANTHROPIC_API_KEY` and
   alternative API/Bedrock/Vertex/Foundry/provider-selection credentials removed from children.
   Quota/provider/model unavailability => `blocked` with effective profile, never fallback.
8. **Artifacts.** Copy validated policy to run `model-policy.json`; record its sha256 digest in
   `workflow.json`; atomically publish an immutable `.control/invocations/` audit entry BEFORE
   readiness for every controller invocation, containing invocation, role, lens/candidate as
   applicable, provider/model/reasoning_effort/model_policy_digest generated from the SAME
   profile used to build argv. Retain on success/blocked/timeout/non-zero exit.
   No secrets, prompts, env, or auth values.
9. **Docs.** README, docs/workflow.md, docs/roles.md, docs/artifacts.md updated.
10. **Tests.** Deterministic fake-provider black-box coverage for: exact profiles + revisions;
    separate executable selection + unused-provider behavior; all Max preflight failure modes
    failing before run creation; Claude flags/env/provider isolation; sandbox child-tool denial;
    strict parsing + post-validation mutation identity; audit/argv/digest binding; no-fallback
    blocked behavior; lifecycle cleanup.

## Old behavior handling

`codex exec --ignore-user-config` without explicit model/effort must no longer be reachable.
No production invocation may reach either CLI without a validated explicit provider/model/effort.

## In scope

`prompts/models.json`, `internal/app/{policy,claude_auth,models,prompts,app,agent_runner,tmux,isolation*}.go`,
`cmd/write-uuter/main.go`, tests under `internal/app`, `internal/app/testdata/fakeagent`,
README + docs/{workflow,roles,artifacts}.md.

## Out of scope

Issue #3 Visual Editor / AI image generation; runtime model selection or automatic fallback;
Fable 5; per-run CLI/env overrides; pricing/usage accounting; editorial prompt or role
responsibility changes; enabling usage credits / API billing / Fast mode.
Files under `tmp/` are PM control scratch, not product scope.

## Accepted deferrals / known state

- Product changes and smoke artifacts are currently uncommitted. That is expected at this stage.
- A temporary debug test was already removed.

## Known verification gap (do NOT re-litigate, but DO confirm scope)

The engineer reported `go test ./... -count=1`, gofmt/vet/build, and targeted auth/provider
isolation tests clean. However **no authenticated mixed-provider smoke run completed
successfully**: `mixed-20260824-1` and `-2` blocked at Writer on an unresolved TODO artifact;
`-3` blocked at Researcher because the claim ledger did not distinguish inference.
This is a concrete REQUIRED-VERIFICATION GAP. Reviewers must not treat launcher exit 0 as a
successful workflow, and must not propose weakening existing editorial artifact validation
to make the smoke pass.

## Not done if

Any profile remains implicit/global/shared; lenses cannot differ; ambient config/env changes
policy; Claude gets external credentials, `--bare`, user customizations, or session state, or
child tools get credential access; auth/config failure occurs after run initialization;
artifacts diverge from argv; partial invalid runs are created; failures fall back.

---

# ROUND 2 ADDENDUM (2026-08-25)

Contract above is UNCHANGED. Round 2 is a fresh manager-led re-review of the complete current
diff, with focused regression validation of the five round-1 must-fix items, all of which the PM
validated as `valid_must_fix` (gate round 1 outcome `fix`, review digest
sha256:0316067f1da99cd1ab17109769a50071e52cfc5e93bde80dfc072e4e445767cd, archive
tmp/pm-review-rounds/round-001).

A prior finding may be re-raised ONLY if it remains unfixed, or if new evidence / changed code
invalidates the prior resolution. Do not re-litigate a fixed item.

## Round-1 must-fix items and the fix evidence to verify INDEPENDENTLY

- **MF1 — no successful authenticated mixed-provider smoke run.**
  Claimed fixed by `smoke-runs/mixed-20260825-2/`: `status: succeeded`, `phase: complete`,
  `current_candidate: 3`, `review_attempt_count: 9`, 15 audit entries (revisions reran the
  ordered review chain). Verify: all eight distinct role/lens identities appear; exact
  provider/model/effort per the table; policy digest and argv binding agree; the FINAL reviewers
  bind the published article revision; no secrets, no fallback, no Fable, no API billing, no
  undeclared model; exact process / tmux / private-root cleanup. The run-specific brief is
  preserved in the run. Superseded blocked attempts were moved out of the product tree — confirm
  they are not misrepresented as successes and that nothing was doctored.

- **MF2 — four vacuous `codex-homes` cleanup assertions.**
  Claimed fixed in `internal/app/blackbox_test.go` with a non-vacuous helper rooted at
  `provider-homes` that also checks concrete `auth.json` / `installation_id` absence. Verify all
  four original locations (round-1 lines 1150, 1331, 1429, 1463) and that the new helper is
  mutation-resistant — i.e. it would actually FAIL if the credential directory survived.

- **MF3 — README/docs/roles.md contradicted the code about HOME.**
  Claimed fixed: docs now describe the real HOME, run-owned `CLAUDE_CODE_TMPDIR`, client-only
  `~/.claude.json` / Keychain access, spawned-tool denial, and the true scope of `--safe-mode`.
  Verify each sentence against `internal/app/agent_runner.go` and `internal/app/isolation_darwin.go`.

- **MF4 — sandbox granted the admin-managed-settings tree.**
  Claimed fixed: the `/Library/Application Support/ClaudeCode` read grant was removed, with
  regression coverage in `internal/app/isolation_darwin_test.go` and
  `internal/app/model_policy_blackbox_test.go`. Verify no BROADER ancestor, subpath, or system
  read rule still exposes that tree (check `/Library`, `/Library/Apple`, metadata-ancestor rules,
  and any `(allow file-read* (subpath ...))` that could cover it), and that the new test would
  actually fail if the grant were reintroduced.

- **MF5 — Max preflight had no hard lifecycle/output bound.**
  Claimed fixed in `internal/app/claude_auth.go` plus `claude_auth_test.go` and the
  process-group helpers (`internal/app/process_group_unix.go`, `process_group_other.go`): the
  full probe tree, inherited pipes, and stdout size are now bounded. Verify the timeout /
  descendant / orphan / oversize tests are deterministic (no sleeps that race), run PRE-run (no
  run directory created), and leak no processes.

## Round-2 scope notes

- No web UI files in the diff (`docs/review/touched.txt` is Go, Markdown, JSON only), so
  `ui-visual` is NOT required.
- Do NOT propose weakening editorial artifact validation or any existing test to make something pass.
- Round-1 optional items OPT1-OPT8 remain optional; do not promote them without new evidence.
