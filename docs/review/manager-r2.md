# Manager Pass — Round 2 — issue uuta/write-uuter#4 (branch feat/4)

## Perspectives run

| Lens | Engine / model / effort | Owns | Output |
| --- | --- | --- | --- |
| requirements (floor) | claude opus, medium | MF1, MF3 | docs/review/requirements-r2.md |
| correctness (floor) | codex gpt-5.6-sol, medium | MF2 | docs/review/correctness-r2.md |
| security (floor) | codex gpt-5.6-sol, medium | MF4 | docs/review/security-r2.md |
| resilience (optional) | codex gpt-5.6-sol, medium | MF5 | docs/review/resilience-r2.md |
| concurrency (optional) | codex gpt-5.6-sol, medium | MF5 cross-check | docs/review/concurrency-r2.md |

Run-scoped tag `review-issue4-wt4-r2-57531`, windows `@50762`-`@50766`, panes `%50796`-`%50800`,
recorded in `docs/review/tmux-targets.env`; all five killed by exact id. Unrelated
`review-153-r2-*` windows from a concurrent issue review were left untouched — the run-scoped
prefix is what made that separation safe.

Optional lenses dropped, with reason:
- `reuse` — round 1 found no issues and the round-2 delta adds no new abstraction (one helper,
  one path list, two test files). Dropped for cost.
- `api-contract` — the CLI surface, `models.json` schema, and artifact schema are unchanged since
  round 1; the round-1 pass produced only two Low/optional items, both still optional.
- `data-migration`, `i18n-a11y` — not triggered.
- `ui-visual` — NOT required and correctly excluded: `docs/review/touched.txt` is Go, Markdown
  and JSON only, with zero web or Flutter UI files. Verified by classification, not assumed.

## Verdict: all five round-1 must-fix items resolved

| Item | Verdict | Strongest independent evidence |
| --- | --- | --- |
| MF1 | FIXED | `smoke-runs/mixed-20260825-2` is genuinely `succeeded`; all 8 identities, exact profiles, digest and revision binding all verified; forensic digest-chain recomputation proves it was not hand-edited |
| MF2 | FIXED (strengthened) | All four sites use `assertNoStagedProviderCredentials`; mutation experiment in an out-of-repo scratch copy proved the helper rejects surviving credentials |
| MF3 | FIXED | Sentence-by-sentence doc-vs-code table; every claim now matches `agent_runner.go` / `isolation_darwin.go` |
| MF4 | FIXED (strengthened) | Grant removed AND both managed trees explicitly denied last; overlay mutation re-adding the grant made the new test FAIL |
| MF5 | FIXED as scoped; one narrower residual | The 30s bound is now genuinely hard; two lenses found a detached-descendant escape that does not breach a contract clause — carried as optional |

### MF1 — the decisive item

Verified independently before reading any lens output, then corroborated by the requirements lens:
- All 15 audit records enumerated; all eight distinct role/lens identities present with exactly
  the contracted provider/model/effort, and no drift across candidates 1→2→3.
- `model-policy.json` byte-identical to `prompts/models.json`; its sha256 equals the single
  digest carried by `workflow.json` and all 15 audit records.
- `article.md` == `drafts/article-003.md`, sha256 `1f134fc8…0c9844` == `workflow.json`
  `current_revision`; all four final-lens reviews bind that exact digest, while candidate-1 and
  candidate-2 chains bind their own drafts. The published-revision binding clause holds.
- Reviewer order evidence→story→clarity→copy holds in every chain; `review_attempt_count: 9`
  reconciles with 2+3+4 review directories and 9 pm-decision entries.
- The requirements lens recomputed all nine `sha256(result.json || 0x00 || report.md)` review
  digests from disk and reproduced every value recorded in `pm-decisions/` — any hand edit to a
  report, result, article, or the policy would have broken one of those chains. This is the
  strongest available proof the run was not doctored.
- Zero credential / raw-auth / identity material; only the four declared model names appear
  anywhere in the run; no Fable, fallback, API-billing, or Fast-mode term.
- Superseded blocked attempts are absent from the product tree and nothing labelled `blocked` is
  presented as a success. The run-specific brief is preserved at `brief.md`.

### The one live counter-example, disposed

Both the requirements lens and I found two live `write-uuter-*` tmux sessions holding running
`agent-runner` processes. The requirements lens flagged this Low/uncertain and asked for
disposition. I can close it definitively, and it is NOT a finding:

- Both were created 2026-08-24 21:07:37 and 21:11:56, inside my own round-1
  `go test ./... -count=1` run, which I launched WITHOUT `-timeout` and which Go killed with a
  panic at the default 10-minute package limit. A panic-killed test binary cannot run `t.Cleanup`,
  so its children are orphaned by construction.
- Their paths are `$TMPDIR/TestBlackBox*/.write-uuter-private-*/control/agent-runner` — test
  fixtures, not product runs.
- Both of my clean full-suite runs left nothing behind: the round-2 run executed 00:59→01:05 and
  exited 0, and `tmux list-sessions` immediately afterwards showed only the two 21:07/21:11
  sessions. The transient session the requirements lens observed at 00:59 is gone.
- The smoke run itself is clean: no `.write-uuter-private-*` anywhere, no session in its
  00:25–00:50 window.

Conclusion: cleanup works on normal and failure paths; these orphans are an artifact of my own
aborted verification run, not a product regression against black-box item 11.

## The one residual, judged optional rather than must-fix

The resilience and concurrency lenses both rated the detached-descendant escape High and both
called MF5 "partially fixed". I am not adopting that severity, and reviewer agreement is not the
arbiter — the contract is. My reasoning:

- MF5 as the PM validated it was that the probe "lacked process-group termination, WaitDelay/pipe
  bound, and output size bound, so the 30s Max preflight limit was not hard." All three named
  mechanisms are now present and the bound IS now hard: both lenses confirm the function returns
  within the deadline even when a descendant holds stdout, and the new tests assert
  `elapsed < timeout` directly. The scoped defect is fixed.
- The residual is narrower and new: a descendant that calls `setsid(2)`/`setpgid(2)` leaves the
  group and survives `kill(-pgid)`. No contract clause covers it. The preflight is pre-run, so a
  probe descendant is not a "run-owned process" under required verification item 11; and nothing
  regressed, because `main` had no preflight at all.
- Real-world impact is bounded: one orphaned child of the user's own `claude` CLI, running under
  the sanitized allowlist environment. No credential exposure, no artifact divergence, no policy
  bypass.
- The code comments at `claude_auth.go:13-16` and `:74-77` are mildly overstated for the detached
  case, but read in context they describe signalling the group and bounding the wait, both of
  which are accurate.

It is carried as a prominent optional item with both lenses' evidence preserved, so the PM can
promote it if they read the contract differently.

## Residual risks / verification gaps

- The new preflight tests use 2s/3s budgets; on a heavily loaded machine they could flake. They
  passed repeatedly here, including under `-race`.
- `GOOS=windows` build fails, but the correctness lens confirmed the same failure on `main` —
  a pre-existing baseline gap, not a regression.
- Reviewer-lens coverage on non-darwin remains test-only; `isolation_other.go` hard-errors.
- The full suite takes ~6 minutes unloaded and exceeds Go's default 10-minute package timeout
  under load. CI must pass `-timeout`.
