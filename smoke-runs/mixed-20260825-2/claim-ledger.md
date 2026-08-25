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
