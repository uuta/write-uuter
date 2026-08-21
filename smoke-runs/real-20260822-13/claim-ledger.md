# Claim Ledger

This ledger uses all five required classifications explicitly. Source locations point to the staged copies recorded in `evidence/sources.md`.

## Fact

| ID | Supported claim | Evidence |
| --- | --- | --- |
| F1 | write-uuter is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | `001-README.md`, lines 1-5 |
| F2 | Go owns workflow state transitions, artifact validation, revision hashes, timeouts, process cleanup, and the decision to advance or terminate. | `001-README.md`, lines 68-80; `003-roles.md`, lines 1-9 |
| F3 | Each role receives a controller-created isolated workspace outside the durable run directory, with only contracted context staged in and validated regular-file outputs copied back. | `001-README.md`, lines 70-77; `002-workflow.md`, lines 64-88 |
| F4 | The main production order is Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewer processes. | `001-README.md`, lines 81-84; `002-workflow.md`, lines 12-39 |
| F5 | The Researcher owns `evidence/sources.md`, optional firsthand evidence/assets, and `claim-ledger.md`; the Story Editor owns `outline.md`; the Writer owns one assigned numbered draft. | `003-roles.md`, lines 37-55 |
| F6 | Each outline section must record its purpose, supporting evidence, and reader takeaway. | `002-workflow.md`, lines 51-54; `003-roles.md`, lines 43-46 |
| F7 | Review lenses run sequentially, never in parallel, and each lens uses a fresh Codex invocation. | `002-workflow.md`, lines 41-43 and 87-88; `003-roles.md`, lines 57-60 |
| F8 | After each review lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go independently validates the decision and applies routing. | `003-roles.md`, lines 17-35 |
| F9 | A validated must-fix stops the remaining lenses for that candidate; if budget remains, the Writer creates the next candidate and review restarts at Evidence. Optional and invalid findings do not consume a candidate, while human judgment blocks the run. | `002-workflow.md`, lines 19-43 |
| F10 | Candidate 003 is the hard limit. Exhausting it blocks the run rather than creating a fourth candidate. | `001-README.md`, lines 81-84; `002-workflow.md`, lines 28-38 and 90-96 |
| F11 | The controller advances only after owned artifacts exist and satisfy role-specific validation; an agent's final message or process exit alone is not completion. | `002-workflow.md`, lines 45-62 |
| F12 | Review artifacts are bound to the exact lens and SHA-256 candidate revision, and stale metadata is rejected. Reviewers may not edit candidates. | `002-workflow.md`, lines 54-62; `003-roles.md`, lines 57-73 |
| F13 | PM decisions must cover every finding, bind to the current review request and digest, preserve earlier accepted lens classifications and routing, and contain no future lens. | `002-workflow.md`, lines 54-59; `003-roles.md`, lines 28-35; `004-artifacts.md`, lines 65-100 |
| F14 | On success, `article.md` is created only after all four lenses pass PM routing and is byte-for-byte identical to the accepted candidate. | `001-README.md`, lines 41-46; `004-artifacts.md`, lines 30-33 |
| F15 | Earlier candidates, partial review sequences, reviews, and PM decisions remain available when revision occurs or a run blocks. | `004-artifacts.md`, lines 30-33 |
| F16 | `workflow.json` is atomically rewritten as the controller's source of truth and records status, phase, current candidate/revision, active role, artifact paths, review attempt count, timestamps, and a terminal block reason when blocked. | `004-artifacts.md`, lines 102-114 |
| F17 | `.control/` preserves post-cleanup audit copies of assignments, invocation logs, and exit markers, while live requests, workspaces, sandbox profiles, and ownership state remain private and are removed after verified cleanup. | `004-artifacts.md`, lines 116-125 |
| F18 | Editorial completion does not depend on agent chat transcripts or tmux scrollback. | `004-artifacts.md`, lines 126-129 |
| F19 | The shipped workflow is single-run and non-resumable; parallel runs, resuming after controller restart, and editing completed runs are not implemented. | `002-workflow.md`, lines 3-10 and 106-107 |

## Firsthand observation

| ID | Observation | Evidence |
| --- | --- | --- |
| HO1 | No firsthand observation was performed for this assignment. | Research activity record; no `evidence/firsthand.md` or assets were created. |

## Inference

| ID | Interpretive claim | Basis and limits |
| --- | --- | --- |
| I1 | The workflow is inspectable because consequential stages produce durable, structured artifacts tied to candidates and revisions, rather than relying on conversational history. | Derived from F11-F18. This is an explanation of the documented design, not a measured usability result. |
| I2 | Artifact gates make role completion machine-checkable and reduce the chance that malformed, stale, or incomplete outputs silently advance the workflow. | Derived from `002-workflow.md`, lines 45-62, and the schema rejection rules in `004-artifacts.md`, lines 35-100. “Reduce the chance” is architectural reasoning; no comparative error-rate study is supplied. |
| I3 | Separating PM classification from Go-controlled validation and routing leaves editorial judgment with the PM while keeping transition enforcement deterministic. | Derived from `003-roles.md`, lines 17-35. “Editorial judgment” refers only to the documented classification task, and “deterministic” to controller enforcement described by the sources. |
| I4 | Restarting revised candidates at Evidence prevents later lenses from treating an evidence-invalidated draft as already cleared. | Derived from the routing order in `002-workflow.md`, lines 19-43. The sources specify the behavior but do not explicitly state this rationale. |
| I5 | The three-candidate ceiling bounds automated revision attempts and turns unresolved work into an inspectable blocked state. | Derived from F10, F15, and F16. It does not establish that three is the optimal budget. |

## Opinion

| ID | Value judgment | Treatment |
| --- | --- | --- |
| O1 | Durable artifact gates are a useful design choice for engineers who need to audit how an article was produced. | Clearly present as an evaluative interpretation; may be argued from I1-I3 but must not be presented as a measured fact. |
| O2 | A three-candidate cap is a sensible tradeoff between automated iteration and bounded execution. | Opinion only. The sources document the cap but provide no evidence that three is the best threshold. |

## Unresolved

| ID | Open claim or question | Why unresolved |
| --- | --- | --- |
| U1 | How often reviewer findings cause revisions or human-judgment blocks in real use. | No run corpus or operational metrics appear in the allowed sources. |
| U2 | Whether the workflow improves article quality, review consistency, cost, or latency compared with another editorial process. | The sources describe implementation and contracts, not comparative outcomes. |
| U3 | Why the candidate budget is exactly three rather than another number. | The sources state the hard limit but do not give a selection rationale. |
| U4 | How a blocked run is retried in practice beyond inspection followed by a new run directory. | `003-roles.md`, lines 11-15 says issue 1 does not resume and supports a later inspected retry, but does not prescribe a full human procedure. |
| U5 | Whether Linux will receive a native read-isolation backend, and on what timeline. | `001-README.md`, lines 17-18 describes the current fail-closed limitation but gives no roadmap. |

## Guardrails for downstream use

- Keep factual descriptions within the repository implementation documented above.
- Attribute design rationale to inference unless a source states it explicitly.
- Do not convert O1 or O2 into factual performance claims.
- Do not claim support for publishing integrations, web interfaces, workflow resumption, or generalized results outside this repository.
