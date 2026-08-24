# Clarity Reviewer lens contract

Review only whether the supplied audience can understand and act on the exact
candidate within the supplied constraints. Do not perform evidence, story, or
copy review.


# Shared reviewer output contract

Use only the files under the supplied `context/` directory. Do not inspect its
parent, another workspace, the source repository, `.control/`, `reviews/`, PM
decisions, logs, or another lens's output. Never edit `context/article.md`.
The `context/` directory contains every permitted input and no other run
artifact. Write only `result.json` and `report.md` in the workspace root. Use
status `clean`, `fix_required`, or `blocked`; the exact supplied lens and
revision; and an array of findings. Every finding requires a stable ID,
severity, location, problem, and `suggested_direction`. The report must repeat
every machine finding field verbatim.

For each finding, use these five labels in this order (bullets and blank lines
between fields are optional): `id`, `severity`, `location`, `problem`, and
`suggested_direction`. Do not split a field value across lines.

The JSON field name for the revision is exactly `reviewed_revision` (never
`revision`). Use this exact shape, retaining the finding objects only when
there are findings:

```json
{
  "status": "clean",
  "lens": "evidence",
  "reviewed_revision": "sha256:the-exact-assigned-revision",
  "findings": []
}
```

Before exiting, re-read `result.json` and verify that it contains all four
top-level keys: `status`, `lens`, `reviewed_revision`, and `findings`.


## Assignment

Lens: `clarity`
Candidate: `article-002`
Revision: `sha256:de5cf22227acee3d99de7f8fe6ee985a0fe473d3843a49a66b71d5e076d5b67d`

## Provided context: brief.md

<write-uuter-context name="brief.md">
# Brief

## Question

How does write-uuter bind every editorial role to one declared model profile?

## Audience

Engineers evaluating how a small editorial pipeline makes its model choices
explicit and auditable.

## Provisional takeaway

Pinning each role to a version-controlled provider, model, and reasoning
effort keeps a multi-agent run reproducible and inspectable after the fact.

## Scope

Explain the per-role model policy this repository declares, how a run records
the policy it used, and why the roles are separated the way they are.

## Out of scope

Publishing integrations, web interfaces, provider pricing, and any claim about
tools outside this repository.

## Publication target

A concise repository article for technical readers.

## Constraints

Use only facts supported by the staged source hints. Every claim must be
traceable to those sources: make no comparative or industry-wide claims about
other tools, pipelines, or practices, and do not assert a meaning for a term
beyond what the sources state. Explain each repository-specific term the
article uses - prompt bundle, profile, candidate, revision, lens - in plain
language at its first use, staying within what the sources support. Keep the
article under 700 words. Deliver finished prose: no placeholder markers, no
bracketed stand-ins, and no unfinished sections.

Process note, not article content: the staged read-only copies of the source
hints live in `context/source-hints/` and are named by their listing position
and file name - `001-README.md`, `002-roles.md`, `003-artifacts.md`. Read those
paths directly. Shell, directory-listing, and network tools are unavailable in
this environment, so do not spend effort on them and do not report their
absence as research.

## Done when

The article accurately explains the eight declared role profiles, the fact
that reasoning effort is the only tuning vocabulary, and how a completed run
preserves the exact policy it ran under.

## Source hints

- ../README.md
- ../docs/roles.md
- ../docs/artifacts.md

</write-uuter-context>

## Provided context: clarity-fields.md

<write-uuter-context name="clarity-fields.md">
Audience:
Engineers evaluating how a small editorial pipeline makes its model choices
explicit and auditable.

Constraints:
Use only facts supported by the staged source hints. Every claim must be
traceable to those sources: make no comparative or industry-wide claims about
other tools, pipelines, or practices, and do not assert a meaning for a term
beyond what the sources state. Explain each repository-specific term the
article uses - prompt bundle, profile, candidate, revision, lens - in plain
language at its first use, staying within what the sources support. Keep the
article under 700 words. Deliver finished prose: no placeholder markers, no
bracketed stand-ins, and no unfinished sections.

Process note, not article content: the staged read-only copies of the source
hints live in `context/source-hints/` and are named by their listing position
and file name - `001-README.md`, `002-roles.md`, `003-artifacts.md`. Read those
paths directly. Shell, directory-listing, and network tools are unavailable in
this environment, so do not spend effort on them and do not report their
absence as research.

</write-uuter-context>

## Provided context: drafts/article-002.md

<write-uuter-context name="drafts/article-002.md">
# One declared profile per role

In write-uuter, a role's model choice lives in a file rather than in anything
decided at runtime. `prompts/models.json` declares the provider, model, and
reasoning effort for every role, and is version-controlled alongside the role
prompts. That trio — one provider, one model, one reasoning effort — is a
role's *profile*. There is no implicit default, no shared reviewer profile, no
runtime routing, and no fallback. The choice can be read before a run and
checked after one.

## Eight role keys, eight profiles

That file travels as part of the *prompt bundle* — the version-controlled set
of role prompts plus `models.json` that a run is launched from. Every role in
the bundle carries its own separately written profile. The schema defines
exactly eight role keys, and only two providers are supported, `claude_code`
and `codex`.

| Role | Provider | Model | Effort |
| --- | --- | --- | --- |
| `pm` | codex | `gpt-5.6-sol` | high |
| `researcher` | claude_code | `claude-sonnet-5` | medium |
| `story_editor` | claude_code | `claude-opus-5` | high |
| `writer` | claude_code | `claude-opus-5` | medium |
| `reviewer_evidence` | codex | `gpt-5.6-sol` | medium |
| `reviewer_story` | claude_code | `claude-sonnet-5` | medium |
| `reviewer_clarity` | claude_code | `claude-sonnet-5` | medium |
| `reviewer_copy` | codex | `gpt-5.6-luna` | low |

The Human Editor is a ninth role in the pipeline and a person, so it carries no
model profile at all.

## Validated before a run directory exists

The controller validates the policy completely before creating a run directory
and before tmux starts, rejecting a missing or empty policy, an unsupported
`schema_version`, a duplicate JSON key, a missing or unknown role, an unknown
field, an unsupported provider, an empty model, an effort the provider does not
accept, a `claude-*` model on `codex`, or a `gpt-*` model on `claude_code`.
Each invocation is then resolved by its role key, so no role inherits another's
profile and none reaches a CLI without an explicit one. There is no global
model allowlist: availability is decided by the selected CLI, which blocks the
run rather than substituting something else.

## The one tunable

Reasoning effort accepts `minimal`, `low`, `medium`, and `high` on codex, and
`low`, `medium`, `high`, `xhigh`, and `max` on claude_code. Provider, model,
and prompt bundle are fixed per role. Read together, that makes effort the only
value in the schema that varies across a range of named settings — the sole
dial for adjusting a role's behavior without changing anything else. Nothing
shifts mid-run either. A *candidate* is one numbered draft,
`drafts/article-00N.md`; a *revision* is a rewrite of that candidate after
review rather than a new one. Retries and revisions reuse the same declared
profile, and evaluating a different policy means running a different bundle.

## Why the roles split this way

Reviewer profiles are selected by the combined `reviewer_<lens>` key, where a
*lens* is one reviewer's assigned angle of judgment: evidence, story, clarity,
or copy. The four are therefore configured independently. Evidence and Copy run
on Codex while Story and Clarity run on Claude Code, so a candidate is not
reviewed only by the Writer's own provider. Lifecycles differ too: PM is one
persistent process per run; Researcher and Story Editor each run once before the
first candidate; Writer runs once per candidate or revision; each reviewer is a
fresh process per candidate that never receives the run directory or edits
candidates. Read together, those facts describe an arrangement that keeps each
lens's judgment separate from the others'.

## What the finished run keeps

`model-policy.json` in the run directory is a byte-exact copy of the validated
`models.json`, and `workflow.json.model_policy_digest` is its SHA-256 digest.
Both are written during initialization, so even a blocked run preserves them.
Before any launched process is considered ready, one immutable record per
invocation is published atomically under `.control/invocations/`, carrying
`role`, `lens` where applicable, `candidate`, `provider`, `model`,
`reasoning_effort`, and `model_policy_digest`. Those values come from the same
validated profile that built the process arguments, so the artifacts and the
launched command cannot disagree. Records are kept for successful, blocked,
timed-out, and non-zero invocations, and never carry credentials, environment
values, prompts, or secrets. A run blocked on provider, model, or quota
unavailability records the effective provider, model, and effort in
`block_reason`, and never retries under a different profile. Both the policy
copy and the invocation directory are listed among `workflow.json`'s stable
artifact paths.

Declared in version control, enforced before launch, recorded per invocation:
the policy a run used is recoverable from its own directory, without external
logs.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:de5cf22227acee3d99de7f8fe6ee985a0fe473d3843a49a66b71d5e076d5b67d

</write-uuter-context>