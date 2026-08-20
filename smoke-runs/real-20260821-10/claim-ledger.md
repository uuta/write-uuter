# Claim Ledger

Classifications used explicitly in this ledger: **Fact**, **Firsthand observation**, **Inference**, **Opinion**, and **Unresolved**.

## Fact

| ID | Claim | Support |
| --- | --- | --- |
| F01 | write-uuter is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | `context/source-hints/001-README.md`, opening description |
| F02 | The issue-1 controller is single-run and non-resumable; it validates required brief sections and a new target, initializes the run atomically, and does not implement parallel runs, restart resume, or editing completed runs. | `002-workflow.md`, “Controller sequence” and “Lifecycle and terminal states” |
| F03 | Go, rather than an agent, owns workflow transitions, validation, revision hashes, timeouts, routing, process cleanup, and final publication. | `001-README.md`, “Runtime model”; `002-workflow.md`, “Artifact gates” and “Lifecycle and terminal states”; `003-roles.md`, introduction and PM section |
| F04 | The editorial roles run in this order: Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewers; the review lenses are sequential, not parallel. | `001-README.md`, “Runtime model”; `002-workflow.md`, flowchart and text below it; `003-roles.md`, “Reviewers” |
| F05 | The Researcher owns `evidence/sources.md`, optional firsthand evidence/assets, and `claim-ledger.md`; the Story Editor owns `outline.md`; the Writer owns one assigned candidate; reviewers own lens-specific `result.json` and `report.md`; the PM owns recorded classifications but never writes a candidate or review. | `003-roles.md`, role sections |
| F06 | Roles work in controller-created private workspaces. Go stages only allowed context, validates successful outputs, and copies validated regular files into the durable run; the macOS sandbox blocks access to the durable run, other role workspaces, controller-private state, and unrelated host files. | `001-README.md`, “Runtime model”; `002-workflow.md`, “Lifecycle and terminal states”; `003-roles.md`, introduction and “Reviewers” |
| F07 | A worker's message or exit alone does not advance the workflow: its owned files must exist and satisfy role-specific validation gates after successful exit. | `002-workflow.md`, “Artifact gates” |
| F08 | Candidate review begins at Evidence and proceeds through Story, Clarity, and Copy. After each reached lens, the persistent PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. | `002-workflow.md`, flowchart; `003-roles.md`, “PM” and “Reviewers” |
| F09 | A PM-validated must-fix stops the remaining lenses for that candidate. If the candidate number is below 003, the Writer creates the next candidate and review restarts at Evidence. Optional and invalid findings do not consume a candidate. | `001-README.md`, “Runtime model”; `002-workflow.md`, text below flowchart |
| F10 | Candidate 003 is the hard limit. An exhausted third candidate, human-judgment decision, timeout, malformed/stale artifact, premature exit, or cleanup failure blocks the run with an actionable `workflow.json.block_reason`. | `001-README.md`, “Run” and “Runtime model”; `002-workflow.md`, flowchart and “Lifecycle and terminal states” |
| F11 | A reviewer cannot edit its candidate. Review metadata must bind to the exact lens and SHA-256 candidate revision, and stale or mismatched artifacts are rejected. | `001-README.md`, “Runtime model”; `002-workflow.md`, “Artifact gates”; `004-artifacts.md`, “Review result” |
| F12 | PM decisions bind to the active request, review digest, and candidate revision; accumulated decisions must retain every earlier reached lens without changing its accepted classifications or routing outcome. | `002-workflow.md`, “Artifact gates”; `003-roles.md`, “PM”; `004-artifacts.md`, “PM decision” |
| F13 | On success, the controller revalidates the candidate hash, final reviews, PM request bindings, and accepted classification lists before writing `article.md`; the file is byte-for-byte identical to the candidate accepted through all four lenses. | `001-README.md`, “Run”; `002-workflow.md`, “Lifecycle and terminal states”; `004-artifacts.md`, “Run layout” |
| F14 | Earlier candidates, partial lens sequences, reviews, and PM decisions remain available after revision or blocking; `workflow.json` records status, phase, current candidate/revision, active role, artifact paths, review count, timestamps, and any terminal block reason. | `004-artifacts.md`, “Run layout” and “workflow.json” |
| F15 | After verified cleanup, `.control/` preserves audit copies of generated assignments, per-invocation logs, and natural exit markers, but not live runner executables, sandbox profiles, process-group records, PM requests, or agent workspaces. | `003-roles.md`, introduction; `004-artifacts.md`, “workflow.json” |
| F16 | Editorial completion does not depend on tmux scrollback or chat transcripts. | `004-artifacts.md`, final sentence |

## Firsthand observation

| ID | Claim | Support |
| --- | --- | --- |
| H01 | None. No firsthand run, code inspection, experiment, interview, or direct product observation was performed for this assignment. | Research method recorded in `evidence/sources.md` |

## Inference

| ID | Claim | Basis and caution |
| --- | --- | --- |
| I01 | The workflow is inspectable because its important handoffs and decisions are represented by retained, validated files rather than only transient agent conversation. | Inferred from F05, F07, F12, F14–F16. “Inspectable” is a synthesis of the documented artifact design, not a measured usability result. |
| I02 | Artifact gates reduce the chance that a stale review, incomplete decision, or silently changed candidate is treated as approval. | Inferred from F07, F11–F13. The docs specify rejection checks; no empirical error-rate claim is supported. |
| I03 | Restarting every revised candidate at Evidence ensures later review lenses never stand in for evidence review of changed text. | Inferred from F04 and F09. This describes the structural consequence of the sequence, not proof that all factual errors will be found. |
| I04 | The three-candidate cap converts an otherwise potentially open-ended revision loop into a bounded controller outcome. | Inferred from F09–F10. “Bounded” refers to candidate count, not total duration, which also depends on configured timeouts and cleanup. |
| I05 | Separating Go's control duties from Codex's editorial duties makes the workflow more deterministic at its transitions than if agents controlled their own advancement. | Inferred from F03, F07, F11–F13. This is architectural reasoning; no comparative implementation was studied. |

## Opinion

| ID | Claim | Basis and caution |
| --- | --- | --- |
| O01 | The durable artifact gates are a sensible design for engineers who value auditability over conversational continuity. | Evaluative judgment based on F07, F14–F16; not stated as fact by the sources. |
| O02 | The candidate cap is a pragmatic safety valve, though teams may differ on whether three candidates are enough. | Evaluation prompted by F10; adequacy is not established by the documentation. |

## Unresolved

| ID | Question or unsupported claim | Why unresolved |
| --- | --- | --- |
| U01 | How often do the validation gates catch real defects, and how often do runs block? | The allowed sources provide contracts and behavior, not production metrics or run data. |
| U02 | Does the four-lens sequence improve article quality compared with another workflow? | No comparative evaluation is present; such a claim would exceed repository-specific documentation. |
| U03 | How easy is it for an engineer to inspect a large or blocked run in practice? | The artifact layout is documented, but no usability study or firsthand inspection was performed. |
| U04 | Will Linux execution gain an equivalent native read-isolation backend, resume support, parallel runs, publishing integrations, or a web interface? | The sources identify current omissions or out-of-scope features but give no supported delivery commitments. |
| U05 | What should a human do after `needs_human_judgment` blocks a run beyond inspecting artifacts and retrying in a new run directory? | `003-roles.md` says issue 1 does not resume and supports a later inspected retry, but does not specify a broader operational process. |

## Guardrails for downstream use

- Treat F01–F16 as documentation-supported facts, with the cited source locations retained nearby.
- Attribute I01–I05 as explanations or architectural inferences, not measured outcomes.
- Do not present O01–O02 as repository claims.
- Do not resolve U01–U05 without new allowed evidence.
- Avoid claims about publishing, web interfaces, workflows outside this repository, editorial quality gains, or real-world reliability.
