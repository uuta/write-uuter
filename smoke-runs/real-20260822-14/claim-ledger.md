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
