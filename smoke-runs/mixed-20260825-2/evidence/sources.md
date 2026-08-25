# Sources

All sources below are staged, read-only local copies provided in
`context/source-hints/`, corresponding to the repository files listed in the
brief's Source hints. Accessed 2026-08-24.

## 001-README.md (staged copy of `../README.md`)

Location: `context/source-hints/001-README.md`

Summary: Top-level repository README for `write-uuter`, a Go CLI that turns a
Markdown brief into a reviewed article. Contains the canonical "Model policy"
section (lines ~87-122) with the full eight-role table of provider/model/
reasoning-effort assignments, the rule that `prompts/models.json` is bound
through the same stable no-follow boundary as role prompts, the statement that
there is no implicit default, shared reviewer profile, runtime routing, or
fallback, the accepted `reasoning_effort` vocabularies per provider (`minimal`,
`low`, `medium`, `high` for codex; `low`, `medium`, `high`, `xhigh`, `max` for
claude_code), and the list of validation rejections enforced before a run
directory is created. Also documents the "Claude Max preflight" behavior,
credential stripping from provider child environments, and the "Runtime
model" section describing sequential role execution order and sandbox
boundaries.

## 002-roles.md (staged copy of `../docs/roles.md`)

Location: `context/source-hints/002-roles.md`

Summary: Describes the same eight-role model-policy table (Role/Lifecycle/
Provider/Model/Effort) with an added Lifecycle column showing when each role
process runs (e.g., PM is one persistent process per run; Researcher and
Story Editor run once before the first candidate; Writer runs once per
candidate or revision; each reviewer is a fresh process per candidate).
States that reviewer profiles are selected by the combined `reviewer_<lens>`
key so the four lenses are independent, that retries and revised candidates
reuse the same declared profile, and that a different policy means a
different version-controlled prompt bundle. Also documents per-role
responsibilities: Human Editor, PM (decision classifications
`valid_must_fix`, `valid_optional`, `invalid`, `needs_human_judgment`),
Researcher (owns `evidence/sources.md`, optional `evidence/firsthand.md`/
assets, `claim-ledger.md`, distinguishing fact/firsthand observation/
inference/opinion/unresolved), Story Editor (owns `outline.md`), Writer (owns
one assigned `drafts/article-00N.md`), and the four Reviewers (Evidence,
Story, Clarity, Copy) with their per-lens additional context.

## 003-artifacts.md (staged copy of `../docs/artifacts.md`)

Location: `context/source-hints/003-artifacts.md`

Summary: Documents the run directory layout, including `model-policy.json`
described as "exact validated policy for this run". States that
`model-policy.json` is a byte-exact copy of the validated `models.json`, and
that `workflow.json.model_policy_digest` is its SHA-256 digest, both written
during initialization so even a blocked run preserves them. Documents the
per-invocation audit records published atomically under
`.control/invocations/` before any launched process is considered ready, each
containing `role`, `lens` (if applicable), `candidate`, `provider`, `model`,
`reasoning_effort`, and `model_policy_digest`, with the note that "the
recorded values come from the same validated profile that built the process
arguments, so the artifacts and the launched command cannot disagree."
States records are retained for successful, blocked, timed-out, and non-zero
invocations, and that a run blocking on provider/model/quota unavailability
records the effective provider, model, and reasoning effort in
`block_reason` and never retries with a different profile. Also documents
`workflow.json` fields, including `model_policy_digest` and the
`artifact_paths` entries for `model_policy` and `invocations`.
