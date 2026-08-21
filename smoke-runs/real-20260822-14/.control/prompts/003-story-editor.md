# Story Editor role contract

Create `outline.md` from the supplied brief and evidence. Build a coherent
section sequence. Under every planned article section, explicitly record:

- Purpose
- Supporting evidence
- Reader takeaway

Do not write a draft, change research, or make reviewer decisions. Finish only
after `outline.md` is complete on disk.


## Provided context: brief.md

<write-uuter-context name="brief.md">
# Brief

## Question

How does write-uuter turn a brief into an inspectable reviewed article?

## Audience

Engineers evaluating a small, artifact-driven editorial workflow.

## Provisional takeaway

A deterministic Go controller can coordinate isolated Codex roles while
leaving evidence, drafts, reviews, and decisions available for inspection.

## Scope

Describe the workflow implemented by this repository and the reason its
durable artifact gates matter.

## Out of scope

Publishing integrations, web interfaces, and claims about workflows outside
this repository.

## Publication target

A concise repository article for technical readers.

## Constraints

Use only facts supported by README.md and docs/. Keep the article under 900
words and explain terms in plain language.

## Done when

The article accurately explains the roles, sequential review loop, candidate
budget, and inspectable terminal artifacts.

## Source hints

- ../README.md
- ../docs/workflow.md
- ../docs/roles.md
- ../docs/artifacts.md

</write-uuter-context>

## Provided context: evidence/sources.md

<write-uuter-context name="evidence/sources.md">
# Sources

Research method: close reading of the controller-staged copies named by the brief. No network research or firsthand execution was performed. Accessed 2026-08-22.

## S1 — Repository README

- Location: `context/source-hints/001-README.md` (staged copy of `README.md`)
- Useful summary: Defines write-uuter as a Go CLI that converts a Markdown brief into a reviewed article while retaining evidence, outline, candidates, reviews, and PM decisions. Summarizes the runtime division of responsibility: Go owns state, validation, hashes, timeouts, cleanup, isolation, and artifact copying; isolated Codex roles perform editorial work. States the role order, sequential fresh reviewers, candidate-003 limit, success behavior, and blocked-run behavior.
- Particularly useful for: the high-level answer, system requirements, controller/agent boundary, success and failure outcomes.

## S2 — Workflow documentation

- Location: `context/source-hints/002-workflow.md` (staged copy of `docs/workflow.md`)
- Useful summary: Gives the exact controller sequence from brief validation and atomic run initialization through research, outline, candidate creation, four review lenses, PM classification, revision, and terminal publication or blocking. Defines artifact validation gates, explains that review lenses are sequential, and states that a must-fix stops later lenses and makes the replacement restart at Evidence. Documents lifecycle, isolation, timeout, cleanup, and terminal revalidation behavior.
- Particularly useful for: sequential review loop, routing rules, artifact gates, failure modes, and candidate budget.

## S3 — Role contracts

- Location: `context/source-hints/003-roles.md` (staged copy of `docs/roles.md`)
- Useful summary: Defines ownership and prohibited actions for the Human Editor, persistent PM, Researcher, Story Editor, Writer, and four reviewers. Specifies the PM's four finding classifications, the context supplied to each reviewer lens, and the fact that reviewers are fresh sequential processes that cannot edit candidates. Explains which durable files each role owns.
- Particularly useful for: role boundaries, PM-versus-controller responsibilities, reviewer inputs, and why artifacts remain attributable and inspectable.

## S4 — Artifact contracts

- Location: `context/source-hints/004-artifacts.md` (staged copy of `docs/artifacts.md`)
- Useful summary: Defines the run-directory layout and validated schemas for reviewer results, PM decisions, and `workflow.json`. States that earlier candidates and partial review sequences are retained, while `article.md` exists only on success and exactly matches the accepted candidate. Documents recursive JSON validation, atomic state/marker writes, audit copies, symlink and regular-file protections, and the rule that editorial completion does not depend on chat transcripts or tmux scrollback.
- Particularly useful for: inspectable terminal artifacts, schema-level validation, retained revision history, and exact publication semantics.

## Source-use boundaries

- These four staged files are the complete source set permitted by the brief.
- Claims about publishing integrations, web interfaces, other repositories, or editorial systems in general are unsupported and outside scope.
- The documentation describes the shipped issue-1 workflow; it explicitly says parallel runs, resume after controller restart, and editing completed runs are not implemented.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

This ledger uses the five required classifications explicitly. Source keys refer to `evidence/sources.md`.

## Fact

| ID | Claim | Support |
| --- | --- | --- |
| F1 | write-uuter is a Go CLI that turns a Markdown brief into a reviewed article and preserves the artifacts that produced it. | S1 |
| F2 | The controller validates the required brief sections and requires a new target before atomically initializing the run directory. | S1, S2 |
| F3 | Go, rather than an editorial agent, owns workflow transitions, artifact validation, revision hashes, timeouts, routing, process cleanup, and terminal publication. | S1, S2, S3 |
| F4 | The editorial roles run in this order: Researcher, Story Editor, Writer, then Evidence, Story, Clarity, and Copy reviewers. | S1, S2, S3 |
| F5 | The PM is one persistent Codex process; workers and reviewers run separately, with each reviewer lens using a fresh process. | S1, S2, S3 |
| F6 | Role workspaces are isolated from the durable run and from other role workspaces; Go stages allowed context and copies validated regular-file outputs back into the run. | S1, S2, S3 |
| F7 | Completion is gated on owned files existing and passing validation after the worker exits successfully; an agent's final message alone does not advance the workflow. | S2 |
| F8 | Research must include non-empty sources and a claim ledger that names Fact, Firsthand observation, Inference, Opinion, and Unresolved. | S2, S3 |
| F9 | Each outline section must record purpose, supporting evidence, and reader takeaway, and each candidate must be non-empty with no TODO placeholder. | S2, S3 |
| F10 | Reviewers run sequentially in the fixed order Evidence, Story, Clarity, Copy and do not edit candidates. | S1, S2, S3 |
| F11 | Reviewer output is revision-bound: its lens and SHA-256 revision must match the assignment, and its JSON findings must agree with the report. | S2, S4 |
| F12 | After each reached lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go validates the decision and applies routing. | S2, S3, S4 |
| F13 | A validated must-fix stops the remaining lenses for that candidate, causes the Writer to create the next candidate, and restarts review at Evidence. Optional and invalid findings do not consume a candidate. | S2 |
| F14 | Candidate 003 is the hard limit. A must-fix at that limit, a need for human judgment, or a runtime/artifact failure blocks the run and preserves artifacts. | S1, S2, S3 |
| F15 | Success requires all four lenses on the final candidate to pass PM routing; `article.md` is then a byte-for-byte copy of that candidate. | S1, S2, S4 |
| F16 | Earlier candidates, partial lens sequences, reviews, PM decisions, workflow state, prompts/logs/exit-marker audit copies, and terminal block reasons remain available in the run as applicable. | S2, S4 |
| F17 | `workflow.json` is the controller's atomically rewritten source of truth and records status, phase, candidate/revision, active role, artifact paths, review attempts, timestamps, and a terminal block reason when blocked. | S4 |
| F18 | Parallel runs, resuming a run after controller restart, and editing completed runs are not implemented in the documented issue-1 workflow. | S2, S3 |

## Firsthand observation

None. This assignment involved document research only; the CLI was not built, run, or directly observed.

## Inference

| ID | Claim | Basis and caution |
| --- | --- | --- |
| I1 | The workflow is inspectable because editorial state changes are represented by retained, validated files rather than depending on transient agent conversation. | Inferred from F7, F11, F16, F17 and S4's explicit statement that completion does not depend on tmux scrollback or chat transcripts. “Inspectable” is a synthesis, not a measured usability result. |
| I2 | Separating PM classification from Go routing makes editorial judgment auditable while keeping control-flow enforcement deterministic. | Inferred from F3 and F12. The sources establish the responsibility split; “auditable” is an interpretation of the retained decision records. |
| I3 | Restarting each revised candidate at Evidence prevents later review lenses from being treated as valid after evidence-related content may have changed. | Inferred from F11 and F13. The repository documents the restart rule but does not explicitly state this rationale. |
| I4 | The three-candidate budget bounds automated revision and converts unresolved repeated must-fix work into an inspectable blocked state. | Inferred from F14 and retained-artifact behavior in F16. “Bounds” follows logically; the design motivation is not directly stated. |

## Opinion

| ID | Claim | Treatment |
| --- | --- | --- |
| O1 | Durable artifact gates are a useful design for engineers who value reproducibility and post-run diagnosis. | Editorial evaluation, not a repository fact; if used, label it as judgment and ground it in F7, F11, and F16. |
| O2 | A limit of three candidates is a reasonable tradeoff between autonomous revision and escalation. | Value judgment; the sources establish the limit but do not claim it is optimal or reasonable. |

## Unresolved

| ID | Question or claim not established | What would resolve it |
| --- | --- | --- |
| U1 | How often real runs succeed, block, time out, or reach candidate 003. | Empirical run data; none is provided by the allowed sources. |
| U2 | Whether the workflow improves article accuracy or quality compared with another editorial process. | A defined comparative evaluation; outside the supplied documentation. |
| U3 | How the workflow behaves at scale across many simultaneous jobs. | Implementation and tests for parallel runs; the documented implementation does not support them. |
| U4 | Whether or how blocked artifacts can be resumed in place. | A future resume contract or implementation; issue 1 explicitly does not resume blocked runs. |
| U5 | Support for Linux execution. | An equivalent native read-isolation backend and updated documentation; current Linux execution fails closed. |

## Guardrails for the article team

- Keep the workflow description within F1–F18 and identify I1–I4 as explanations or interpretations rather than direct facts.
- Do not present O1–O2 as proven outcomes.
- Do not imply answers to U1–U5.
- Do not claim support for publishing integrations, web interfaces, resume, or workflow behavior outside this repository.

</write-uuter-context>