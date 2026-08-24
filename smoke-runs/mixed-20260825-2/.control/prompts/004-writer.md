# Writer role contract

Write only the assigned versioned candidate under `drafts/`. Expand the
supplied outline into publishable prose supported by the supplied evidence and
brief. Do not leave TODOs or unresolved placeholders.

For a revision, apply every PM-validated must-fix decision using the prior
candidate and the reached review result/report as input. Use the matching
finding's problem, location, and suggested direction to make the correction,
then verify that the revised wording actually resolves it. Do not accept or
reject findings yourself. Never edit a review result, PM decision, earlier
draft, or final `article.md`. Finish only after the assigned candidate is
complete on disk.


## Assignment

Write candidate 001 to `drafts/article-001.md` in this isolated workspace.

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

## Provided context: evidence/sources.md

<write-uuter-context name="evidence/sources.md">
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

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

Classifications: **Fact** (directly stated in a staged source), **Firsthand
observation** (something this researcher directly observed by using the
system, not applicable here — no firsthand work was performed), **Inference**
(a conclusion drawn by combining or interpreting stated facts, not itself
stated verbatim), **Opinion** (a value judgment, not verifiable from source
text), **Unresolved** (a question the staged sources do not answer).

No firsthand observation entries are recorded because this research relied
entirely on the three staged source-hint files; no direct use of the CLI or
a live run was performed.

## Fact

1. `prompts/models.json` is part of the durable prompt bundle and declares the
   agent backend (provider), model, and reasoning effort for every role.
   [001-README.md L87-94; 002-roles.md L4-9]
2. The model policy schema defines exactly eight role keys: `pm`,
   `researcher`, `story_editor`, `writer`, `reviewer_evidence`,
   `reviewer_story`, `reviewer_clarity`, `reviewer_copy`. [001-README.md
   L96-108]
3. The declared profile for each role is: PM = codex / `gpt-5.6-sol` / high;
   Researcher = claude_code / `claude-sonnet-5` / medium; Story Editor =
   claude_code / `claude-opus-5` / high; Writer = claude_code /
   `claude-opus-5` / medium; Evidence Reviewer = codex / `gpt-5.6-sol` /
   medium; Story Reviewer = claude_code / `claude-sonnet-5` / medium; Clarity
   Reviewer = claude_code / `claude-sonnet-5` / medium; Copy Reviewer = codex
   / `gpt-5.6-luna` / low. [001-README.md L96-106; 002-roles.md L11-20]
4. Only two providers are supported: `claude_code` and `codex`.
   [001-README.md L108]
5. Accepted `reasoning_effort` values are `minimal`, `low`, `medium`, `high`
   for codex, and `low`, `medium`, `high`, `xhigh`, `max` for claude_code —
   provider, model, and prompt bundle are fixed per role, and reasoning
   effort is the only value in the schema that varies across an accepted
   range of named settings. [001-README.md L109-110]
6. There is no implicit default, no shared reviewer profile, no runtime
   routing, and no fallback for the model policy. [001-README.md L92-94]
7. The controller validates the policy completely before creating a run and
   resolves each invocation by its role key, so no role inherits another
   role's profile and no invocation reaches a CLI without an explicit
   profile. [002-roles.md L6-9]
8. Reviewer profiles are selected by the combined `reviewer_<lens>` key
   (e.g. `reviewer_evidence`), so the four reviewer lenses are configured
   independently of one another. [002-roles.md L22-23]
9. Retries and revised candidates reuse the same declared profile; evaluating
   a different policy means running a different version-controlled prompt
   bundle, and per-run model overrides, automatic routing, and dynamic
   fallback are not implemented. [001-README.md L120-122; 002-roles.md
   L23-25]
10. The following are rejected before the run directory is created and before
    tmux starts: a missing or empty policy, an unsupported `schema_version`,
    a duplicate JSON key, a missing or unknown role, an unknown field, an
    unsupported provider, an empty model, an effort the provider does not
    accept, a `claude-*` model on `codex`, and a `gpt-*` model on
    `claude_code`. [001-README.md L114-118]
11. There is no global model allowlist: exact model availability is decided
    by the selected CLI, which blocks the run instead of substituting another
    model. [001-README.md L111-112]
12. `model-policy.json` in the run directory is a byte-exact copy of the
    `models.json` the controller validated for that run, and
    `workflow.json.model_policy_digest` is its SHA-256 digest; both are
    written during initialization, so even a blocked run preserves them.
    [003-artifacts.md L40-45]
13. Before any launched process is considered ready, the controller
    atomically publishes one immutable record per invocation under
    `.control/invocations/`, containing (among other fields) `role`,
    `lens` (when applicable), `candidate`, `provider`, `model`,
    `reasoning_effort`, and `model_policy_digest`. [003-artifacts.md L47-61]
14. The recorded per-invocation values come from the same validated profile
    that built the process arguments, so the artifacts and the launched
    command cannot disagree. [003-artifacts.md L63-64]
15. Invocation records are retained for successful, blocked, timed-out, and
    non-zero invocations, and never contain authentication values,
    environment values, prompts, or secrets. [003-artifacts.md L65-66]
16. A run that blocks because a provider, model, or quota is unavailable
    records the effective provider, model, and reasoning effort in
    `block_reason` and never retries with a different profile.
    [003-artifacts.md L67-69]
17. `workflow.json`'s `artifact_paths` include `model_policy` and
    `invocations` among its stable relative paths. [003-artifacts.md
    L138-147]
18. A provider-neutral runner gives Codex and Claude Code invocations the
    same immutable role/task prompt, workspace boundary, timeout and
    cancellation signal, and audit identity; Codex invocations add an
    explicit `--model` and `--config model_reasoning_effort=...`, while
    Claude Code invocations use `--print`, `--safe-mode`,
    `--dangerously-skip-permissions`, `--no-session-persistence`, and
    explicit `--model`/`--effort`. [001-README.md L136-142; 002-roles.md
    L27-35]
19. Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity,
    and Copy reviewer processes run sequentially; reviewers never receive the
    run directory or edit candidates. [001-README.md L198-201]
20. Human Editor is a human role and has no model profile. [001-README.md
    L107; 002-roles.md L25]
21. Lifecycle differs by role: PM is one persistent process per run;
    Researcher and Story Editor each run once before the first candidate;
    Writer runs once per candidate or revision; each of the four reviewers
    is a fresh process per candidate. [002-roles.md L11-20]
22. Evidence and Copy reviewers run on Codex, while Story and Clarity
    reviewers run on Claude Code, so a candidate is not reviewed only by the
    Writer's own provider. [002-roles.md L114-116]

## Inference

1. Because reasoning effort is the only field in the schema with a named,
   provider-specific accepted range of alternative values (versus provider
   and model, which are fixed single values per role in the checked-in
   bundle, and roles/schema version, which are fixed keys), it functions as
   the sole tuning parameter exposed by the model-policy schema for adjusting
   a role's behavior without changing provider or model. This is a
   synthesis of facts 3, 5, and 9 above rather than a sentence stated
   verbatim in any source.
2. Separating reviewer roles into four independently keyed profiles
   (`reviewer_<lens>`), combined with mixing providers across lenses (fact
   22) and running reviewers as fresh, sequential, candidate-scoped
   processes with no shared conversation (002-roles.md L139-141), together
   support the brief's premise that the roles are separated to keep each
   lens's judgment independent of the others and of the Writer's own
   provider.
3. Because `model-policy.json` is a byte-exact copy of the validated policy,
   the invocation records are built from that same validated profile, and
   both are preserved even for blocked runs, a completed or blocked run's
   directory contains everything needed to determine after the fact exactly
   which provider/model/effort each role invocation used, without relying on
   external logs or memory of how the run was launched.

## Opinion

None recorded. No source text expresses a value judgment distinct from
stated facts about the role/model-policy design.

## Unresolved

1. The staged sources do not state why these particular eight roles (rather
   than some other decomposition) were chosen, beyond describing what each
   role owns and when it runs.
2. The staged sources do not state why the specific model or effort values
   in the checked-in policy table (e.g., `high` for PM and Story Editor
   versus `medium` or `low` elsewhere) were selected.

</write-uuter-context>

## Provided context: outline.md

<write-uuter-context name="outline.md">
# Outline

**Working title:** One declared profile per role

**Question:** How does write-uuter bind every editorial role to one declared
model profile?

**Controlling idea:** Pinning each role to a version-controlled provider,
model, and reasoning effort keeps a multi-agent run reproducible and
inspectable after the fact.

**Target length:** under 700 words (hard constraint from the brief). Section
budgets below sum to roughly 660 words, leaving headroom.

**Sequence logic:** declaration → enforcement → the one tunable → why the
roles are split → what the finished run preserves. The article moves from
what is written down, to what the controller refuses to run without, to what
survives the run — so the "auditable" claim in the takeaway lands only after
the reader has seen both halves of it (declared up front, recorded on disk).

**Term-introduction plan** (brief requires each explained in plain language at
first use, within what the sources support):

| Term | Introduced in | Plain-language gloss to use |
| --- | --- | --- |
| prompt bundle | Section 2 | the version-controlled set of role prompts and `prompts/models.json` that a run is launched from |
| profile | Section 2 | one role's declared provider, model, and reasoning effort |
| candidate | Section 4 | one numbered draft of the article, written by the Writer as `drafts/article-00N.md` |
| revision | Section 4 | a rewrite of a candidate after review, rather than a new candidate |
| lens | Section 5 | one reviewer's assigned angle of judgment — evidence, story, clarity, or copy |

Section 2 must therefore name the four reviewer role keys without using the
word "lens"; Section 5 introduces it.

---

## Section 1 — Lead: the question, stated concretely

**Approx. length:** 70 words.

**Purpose:** Frame the article's single question — where a role's model choice
lives and who decides it — and state up front that in this repository the
answer is a file in version control rather than anything decided at runtime.
Establishes stakes for the engineer audience without a comparative claim.

**Supporting evidence:**
- `prompts/models.json` is part of the durable prompt bundle and declares the
  provider, model, and reasoning effort for every role (Fact 1;
  001-README.md L87-94, 002-roles.md L4-9).
- There is no implicit default, no shared reviewer profile, no runtime
  routing, and no fallback (Fact 6; 001-README.md L92-94).

**Reader takeaway:** Model selection here is a declaration, not a runtime
decision — so it can be read before a run and checked after one.

---

## Section 2 — The declaration: eight role keys, eight profiles

**Approx. length:** 150 words including the table.

**Purpose:** Deliver the brief's first "done when" item — the eight declared
role profiles — as the article's factual spine. Define *prompt bundle* and
*profile* at first use. Present the full table so the reader sees that the
assignments are heterogeneous by role and that both supported providers are
in play.

**Supporting evidence:**
- Exactly eight role keys: `pm`, `researcher`, `story_editor`, `writer`,
  `reviewer_evidence`, `reviewer_story`, `reviewer_clarity`, `reviewer_copy`
  (Fact 2; 001-README.md L96-108).
- The per-role assignments — PM = codex / `gpt-5.6-sol` / high; Researcher =
  claude_code / `claude-sonnet-5` / medium; Story Editor = claude_code /
  `claude-opus-5` / high; Writer = claude_code / `claude-opus-5` / medium;
  Evidence Reviewer = codex / `gpt-5.6-sol` / medium; Story Reviewer =
  claude_code / `claude-sonnet-5` / medium; Clarity Reviewer = claude_code /
  `claude-sonnet-5` / medium; Copy Reviewer = codex / `gpt-5.6-luna` / low
  (Fact 3; 001-README.md L96-106, 002-roles.md L11-20).
- Only two providers are supported: `claude_code` and `codex` (Fact 4;
  001-README.md L108).
- Human Editor is a human role and carries no model profile (Fact 20;
  001-README.md L107, 002-roles.md L25).

**Reader takeaway:** Eight machine roles, eight separately written profiles,
two providers — and one role in the pipeline that is a person and therefore
has no profile at all.

---

## Section 3 — Enforcement: validated before a run directory exists

**Purpose:** Show that the declaration is binding rather than advisory. This
is the section that earns the word "auditable": the controller resolves each
invocation by its role key and refuses malformed policies up front, so a
run either starts under a complete explicit policy or does not start.

**Approx. length:** 110 words.

**Supporting evidence:**
- The controller validates the policy completely before creating a run and
  resolves each invocation by its role key, so no role inherits another
  role's profile and no invocation reaches a CLI without an explicit profile
  (Fact 7; 002-roles.md L6-9).
- Rejected before the run directory is created and before tmux starts: a
  missing or empty policy, an unsupported `schema_version`, a duplicate JSON
  key, a missing or unknown role, an unknown field, an unsupported provider,
  an empty model, an effort the provider does not accept, a `claude-*` model
  on `codex`, and a `gpt-*` model on `claude_code` (Fact 10; 001-README.md
  L114-118).
- There is no global model allowlist: exact model availability is decided by
  the selected CLI, which blocks the run instead of substituting another
  model (Fact 11; 001-README.md L111-112).

**Reader takeaway:** An incomplete or contradictory policy fails before any
artifacts exist, and an unavailable model blocks the run rather than being
quietly swapped — so a run that started did so under a complete, explicit
policy.

---

## Section 4 — The one tunable: reasoning effort

**Approx. length:** 90 words.

**Purpose:** Deliver the brief's second "done when" item. Show *why* effort is
the only tuning vocabulary: provider and model are fixed single values per
role in the checked-in bundle, while effort is the one field with a named,
provider-specific accepted range. Introduce *candidate* and *revision* here,
because the natural place to say "retries and revised candidates reuse the
same profile" is the moment the reader wonders whether anything can shift
mid-run.

**Supporting evidence:**
- Accepted `reasoning_effort` values are `minimal`, `low`, `medium`, `high`
  for codex, and `low`, `medium`, `high`, `xhigh`, `max` for claude_code;
  provider, model, and prompt bundle are fixed per role, and effort is the
  only schema value that varies across an accepted range of named settings
  (Fact 5; 001-README.md L109-110).
- Retries and revised candidates reuse the same declared profile; evaluating a
  different policy means running a different version-controlled prompt bundle,
  and per-run overrides, automatic routing, and dynamic fallback are not
  implemented (Fact 9; 001-README.md L120-122, 002-roles.md L23-25).
- Inference 1 (effort as the sole tuning parameter) is a synthesis of Facts 3,
  5, and 9 — the article should present it as a reading of the schema, not as
  a sentence quoted from the sources.

**Reader takeaway:** The only dial the schema offers is how hard a role
thinks; changing anything else means changing the bundle you ran.

---

## Section 5 — Why the roles are split this way

**Approx. length:** 110 words.

**Purpose:** Answer the brief's third scope item — why the roles are separated
as they are — using only what the sources support. Introduce *lens*. Keep the
claim modest: independent keys plus mixed providers plus fresh per-candidate
processes are what the sources state; the "independence" reading is an
inference and should be phrased as such.

**Supporting evidence:**
- Reviewer profiles are selected by the combined `reviewer_<lens>` key, so the
  four reviewer lenses are configured independently of one another (Fact 8;
  002-roles.md L22-23).
- Evidence and Copy reviewers run on Codex while Story and Clarity reviewers
  run on Claude Code, so a candidate is not reviewed only by the Writer's own
  provider (Fact 22; 002-roles.md L114-116).
- Lifecycle differs by role: PM is one persistent process per run; Researcher
  and Story Editor each run once before the first candidate; Writer runs once
  per candidate or revision; each reviewer is a fresh process per candidate
  (Fact 21; 002-roles.md L11-20).
- Reviewers never receive the run directory or edit candidates, and roles run
  sequentially (Fact 19; 001-README.md L198-201).
- Inference 2 supports the "independent judgment" framing; mark it as a
  reading of Facts 8, 21, and 22 rather than a stated conclusion.

**Reader takeaway:** Separate keys per lens, providers mixed across lenses,
and a fresh process per candidate mean each review is configured — and runs —
on its own terms.

**Note for the Writer:** The sources do not say *why* these eight roles were
chosen or why these particular effort values were picked (Unresolved 1 and 2).
Do not supply a rationale for either; describe the arrangement and stop.

---

## Section 6 — What the finished run keeps

**Approx. length:** 120 words.

**Purpose:** Deliver the brief's third "done when" item and close the loop
opened in Section 1: the declaration is not only readable beforehand, it is
preserved verbatim afterward, alongside a per-invocation record that cannot
contradict the command that ran. This is the section that makes
"reproducible and inspectable after the fact" a factual statement rather than
an aspiration.

**Supporting evidence:**
- `model-policy.json` is a byte-exact copy of the validated `models.json`, and
  `workflow.json.model_policy_digest` is its SHA-256 digest; both are written
  during initialization, so even a blocked run preserves them (Fact 12;
  003-artifacts.md L40-45).
- Before any launched process is considered ready, one immutable record per
  invocation is published atomically under `.control/invocations/`, carrying
  `role`, `lens` where applicable, `candidate`, `provider`, `model`,
  `reasoning_effort`, and `model_policy_digest` (Fact 13; 003-artifacts.md
  L47-61).
- The recorded values come from the same validated profile that built the
  process arguments, so the artifacts and the launched command cannot
  disagree (Fact 14; 003-artifacts.md L63-64).
- Records are retained for successful, blocked, timed-out, and non-zero
  invocations, and never contain authentication values, environment values,
  prompts, or secrets (Fact 15; 003-artifacts.md L65-66).
- A run blocked on provider, model, or quota unavailability records the
  effective provider, model, and reasoning effort in `block_reason` and never
  retries with a different profile (Fact 16; 003-artifacts.md L67-69).
- `workflow.json`'s `artifact_paths` include `model_policy` and `invocations`
  (Fact 17; 003-artifacts.md L138-147).

**Reader takeaway:** A finished run — or a blocked one — carries its own
policy and a per-invocation record of what actually launched, so the model
choices can be reconstructed from the run directory alone.

---

## Section 7 — Close

**Approx. length:** 40 words.

**Purpose:** Restate the provisional takeaway in the article's own terms,
grounded in what Sections 2-6 established. No new facts, no forward-looking
claims, no comparison to anything outside this repository.

**Supporting evidence:** Inference 3 — because the policy copy is byte-exact,
the invocation records are built from that same validated profile, and both
survive a blocked run, the run directory contains what is needed to determine
after the fact which provider, model, and effort each invocation used, without
external logs.

**Reader takeaway:** Declared in version control, enforced before launch,
recorded per invocation — the model policy is a fact about the run, not a
memory of it.

---

## Constraints the Writer must hold

- Under 700 words total. If cutting is needed, compress Sections 3 and 5
  before Sections 2, 4, or 6 — those three carry the brief's "done when"
  items.
- The eight-row table in Section 2 is load-bearing; keep it complete.
- Every claim traceable to a staged source hint. Sections 4, 5, and 7 lean on
  Inferences 1, 2, and 3 — present those as readings of the cited facts, not
  as source statements.
- No comparative or industry-wide claims about other tools, pipelines, or
  practices; no meaning asserted for a term beyond what the sources state.
- Explain prompt bundle, profile, candidate, revision, and lens at first use,
  per the table above.
- Finished prose only: no placeholders, no bracketed stand-ins, no unfinished
  sections.

</write-uuter-context>