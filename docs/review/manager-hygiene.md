# Manager Review — PR #5 repository hygiene (issue #4, round 3)

Deletion-and-ignore-only delta on top of `5d3d106`. Verdict: **clean**.

## Delta shape

`git diff HEAD --name-status` = 1 modified (`.gitignore`) + 61 deletions, nothing else.
`git diff HEAD --check` clean. No `.go`, `README`, `docs/{roles,workflow,artifacts}.md`,
`prompts/**`, or `cmd/**` file is touched, so no runtime behavior can change.

Deletions by class — all four inside the allowed set, none outside:

| Class | Count |
| --- | --- |
| `smoke-runs/mixed-20260825-2/.control/logs/**` | 15 |
| `smoke-runs/mixed-20260825-2/.control/prompts/**` | 15 |
| `smoke-runs/mixed-20260825-2/.control/exits/**` | 15 |
| `docs/review/*` ephemeral inputs + specialist raw reports | 16 |

No `smoke-runs/real-20260823-*` path is deleted or edited. `docs/review/manager-r2.md` is
retained as the minimal manager summary.

PR size: `main..HEAD` 133 files → `main..worktree` 73 files (−60, −45%).

## Retained evidence re-verified (not assumed)

- `workflow.json`: `succeeded` / `complete`, candidate 3, `review_attempt_count` 9.
- `.control/invocations/`: 15 records, **8 distinct role/lens identities**, each carrying exactly
  the contracted provider/model/effort, and one single digest across all 15.
- `model-policy.json` is byte-identical to `prompts/models.json`; recomputed sha256 of both =
  `sha256:f5a91c46…16752b` = `workflow.json.model_policy_digest` = every audit record.
- `article.md` == `drafts/article-003.md`, recomputed `sha256:1f134fc8…0c9844` ==
  `workflow.json.current_revision`; all four `reviews/article-003/{evidence,story,clarity,copy}`
  bind that exact revision.
- **Digest chain survives deletion.** All nine `sha256(result.json ‖ 0x00 ‖ report.md)` review
  digests recompute from the *retained* files alone and match all three `pm-decisions/` files —
  zero mismatches. Removing the raw transcripts did not cost any verifiability.

## Ignore rules

Probed with `git check-ignore -q --no-index` (plain `check-ignore` silently skips tracked files
and misreports negations — the exit code is the reliable signal):

- IGNORED: smoke `.control/{logs,prompts,exits}/*`, `docs/review/{diff.txt,touched.txt,
  tmux-targets.env}`, and every specialist `*-r2.md` / `*-hygiene.md` raw report.
- NOT ignored: `docs/review/manager*.md` (negation is effective — `/docs/review/*` is a file
  glob, not a directory exclusion, so the re-include works), `.control/invocations/**`,
  `workflow.json`, `model-policy.json`, `article.md`, `reviews/**`, `pm-decisions/**`.
- Pre-existing `smoke-runs/real-20260823-*` transcripts stay tracked; gitignore never untracks,
  so that history is unchanged.

## No regression of the five PM-validated fixes

No source or test file changed, and the focused suite passes (33.9s, exit 0):
`TestBlackBox(EveryRoleLaunchesItsDeclaredProfile|RevisionsPreserveDeclaredProfiles|
PolicyMutationAfterValidationCannotChangeLaunchedProfiles|
PersistentCleanupFailurePreservesOwnershipButDeletesCredentials|
AdminManagedClaudeSettingsCannotCrossSandboxBoundary|ClaudeMaxPreflightRunsBeforeRunCreation)`
plus `TestClaude(IsolationNeverGrantsAdminManagedPolicy|MaxPreflight)` — covering MF2, MF4 and
MF5 directly; MF1 evidence re-verified above; MF3 docs untouched by this delta.

## Run-scoped review isolation

Tag `review-issue4-wt4-hygiene-r3-14549`; windows `@51650`-`@51652`, panes `%51685`-`%51687`,
recorded in `docs/review/tmux-targets.env` (itself intentionally ignored and unstaged). Prompts
delivered per-pane via `load-buffer` → `paste-buffer` → `send-keys C-m`; submission confirmed by
re-capture. No static window names.
