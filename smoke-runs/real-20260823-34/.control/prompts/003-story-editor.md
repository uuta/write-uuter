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

Accessed 2026-08-23. These are controller-staged copies of the repository sources named by the brief; no outside sources were used.

## 1. Repository README

- Location: `context/source-hints/001-README.md` (staged from `README.md`)
- Type: Repository documentation
- Useful summary: Defines `write-uuter` as a Go CLI that turns a Markdown brief into a reviewed article while preserving intermediate evidence, outline, candidates, reviews, and PM decisions. Documents prerequisites, brief validation, prompt resolution, isolated per-role workspaces, controller ownership of transitions and cleanup, sequential role order, the four review lenses, the three-candidate ceiling, and success/blocked outcomes. It states that successful `article.md` is an exact copy of the candidate accepted through all four reviews.
- Most useful sections: opening description; `Run`; `Runtime model`.

## 2. Workflow documentation

- Location: `context/source-hints/002-workflow.md` (staged from `docs/workflow.md`)
- Type: Repository documentation
- Useful summary: Gives the controller sequence from brief validation and atomic run initialization through Researcher, Story Editor, Writer, Evidence/Story/Clarity/Copy review, PM classification, revision, and terminal publication or blocking. Explains that review lenses are sequential, a must-fix stops the remaining lenses and restarts the next candidate at Evidence, candidate 003 is the limit, and human-judgment routing blocks. Defines artifact gates and terminal checks, including worker exit, file validation, hash/revision binding, process cleanup, and atomic publication.
- Most useful sections: `Controller sequence`; `Artifact gates`; `Lifecycle and terminal states`.

## 3. Role contracts

- Location: `context/source-hints/003-roles.md` (staged from `docs/roles.md`)
- Type: Repository documentation
- Useful summary: Assigns ownership and limits for the Human Editor, persistent PM, Researcher, Story Editor, Writer, and four fresh reviewers. The PM classifies findings but does not write candidates or reviews; Go validates decisions and applies routing. The Writer owns one candidate at a time. Reviewers receive lens-specific context, write `result.json` and `report.md`, and cannot edit candidates. The Researcher owns the sources and claim ledger, while the Story Editor owns an outline with purpose, evidence, and reader takeaway.
- Most useful sections: `PM`; `Researcher`; `Story Editor`; `Writer`; `Reviewers`.

## 4. Artifact documentation

- Location: `context/source-hints/004-artifacts.md` (staged from `docs/artifacts.md`)
- Type: Repository documentation
- Useful summary: Specifies the durable run layout and validation contracts for reviewer results, PM decisions, `workflow.json`, and `.control/` audit copies. States that earlier candidates, partial review sequences, reviews, and PM decisions remain available after revision or blocking. Defines strict JSON validation and says `article.md` appears only on success and is byte-for-byte identical to the accepted candidate.
- Most useful sections: `Run layout`; `Review result`; `PM decision`; `workflow.json`.

## Source boundaries

- The brief requires facts to come only from `README.md` and `docs/`; all factual entries in the claim ledger below trace to the four staged copies above.
- No firsthand testing, execution of `write-uuter`, interviews, or independent code inspection was performed. Therefore `evidence/firsthand.md` and `evidence/assets/` were not created.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

This ledger uses all five required classifications: **Fact**, **Firsthand observation**, **Inference**, **Opinion**, and **Unresolved**. Proposed article claims should rely on the Fact entries; the other classifications mark interpretation or research limits rather than established repository behavior.

## Fact

| ID | Claim | Support |
| --- | --- | --- |
| F1 | `write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | `context/source-hints/001-README.md`, opening description. |
| F2 | Go owns workflow state transitions, artifact validation, revision hashes, timeouts, routing enforcement, and process cleanup. | `001-README.md`, `Runtime model`; `002-workflow.md`, `Artifact gates` and `Lifecycle and terminal states`; `003-roles.md`, opening and `PM`. |
| F3 | The controller validates all required brief sections and requires a new target before atomically initializing a run directory. | `001-README.md`, `Run` and required headings; `002-workflow.md`, `Controller sequence`. |
| F4 | The editorial roles run in this order: Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewer processes. | `001-README.md`, `Runtime model`; `002-workflow.md`, controller diagram; `003-roles.md`, `Reviewers`. |
| F5 | The Researcher produces sources and a claim ledger; the Story Editor produces an outline whose sections record purpose, supporting evidence, and reader takeaway; the Writer produces exactly one assigned candidate. | `003-roles.md`, `Researcher`, `Story Editor`, and `Writer`; `002-workflow.md`, `Artifact gates`. |
| F6 | Each role works in a fresh private workspace outside the durable run directory, receives only contracted context, and has validated regular-file outputs copied back by Go. | `001-README.md`, `Runtime model`; `002-workflow.md`, `Lifecycle and terminal states`; `003-roles.md`, opening and `Reviewers`. |
| F7 | The four review lenses never run in parallel. Each reviewer is a fresh process, cannot edit the candidate, and writes a validated JSON result plus a matching Markdown report. | `002-workflow.md`, `Controller sequence` and `Artifact gates`; `003-roles.md`, `Reviewers`; `004-artifacts.md`, `Review result`. |
| F8 | After each reached lens, the persistent PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go validates the record and performs routing. | `003-roles.md`, `PM`; `004-artifacts.md`, `PM decision`. |
| F9 | A PM-validated must-fix ends the remaining review lenses for that candidate, sends the Writer a revision assignment, and restarts review at Evidence. Optional and invalid findings do not consume a candidate; a human-judgment decision blocks the run. | `002-workflow.md`, `Controller sequence`; `003-roles.md`, `PM` and `Writer`. |
| F10 | Candidate 003 is the hard ceiling. A validated must-fix after that budget is exhausted blocks the run and preserves artifacts. | `001-README.md`, `Run` and `Runtime model`; `002-workflow.md`, controller diagram. |
| F11 | Artifact gates require successful worker exit and structurally valid owned files; chat messages, process exit alone, tmux scrollback, and chat transcripts are not editorial completion evidence. | `002-workflow.md`, `Artifact gates`; `004-artifacts.md`, final paragraph. |
| F12 | Review and PM artifacts are bound to the current candidate revision, and stale or mismatched lens, revision, request, digest, finding, or prior-decision data is rejected. | `002-workflow.md`, `Artifact gates`; `004-artifacts.md`, `Review result` and `PM decision`. |
| F13 | `workflow.json` is the controller's atomically rewritten source of truth and records status, phase, candidate/revision, active role, artifact paths, review-attempt count, timestamps, and a block reason when blocked. | `004-artifacts.md`, `workflow.json`. |
| F14 | Earlier candidates, partial lens sequences, reviews, and PM decisions are retained when revision occurs or the run blocks; `.control/` preserves post-cleanup audit copies such as assignments, logs, and exit markers. | `004-artifacts.md`, `Run layout` and `workflow.json`; `003-roles.md`, opening. |
| F15 | Success requires all four final-candidate lenses to pass PM routing, final revision and decision revalidation, and verified cleanup; `article.md` is then atomically published as a byte-for-byte copy of that candidate. | `001-README.md`, `Run`; `002-workflow.md`, `Lifecycle and terminal states`; `004-artifacts.md`, `Run layout`. |
| F16 | A timeout, malformed artifact, premature exit, stale review, cleanup failure, need for human judgment, or exhausted candidate budget produces a blocked workflow with an actionable block reason. | `001-README.md`, `Run`; `002-workflow.md`, `Lifecycle and terminal states`. |
| F17 | The implemented controller is single-run and non-resumable; parallel runs, resuming after controller restart, and editing completed runs are not implemented. | `002-workflow.md`, opening and final sentence; `003-roles.md`, `Human Editor`. |

## Firsthand observation

| ID | Claim | Support |
| --- | --- | --- |
| O1 | No firsthand observation is claimed in this research pass because the CLI was not run and the implementation was not independently inspected. | Research method for this assignment; the work used only the supplied staged documentation. |

## Inference

| ID | Claim | Basis and caution |
| --- | --- | --- |
| I1 | The durable artifacts make the editorial path inspectable because a reader can relate sources, planned structure, candidate revisions, lens findings, PM classifications, and terminal state after the processes finish. | Inferred from F1, F12–F15 and the run layout in `004-artifacts.md`. “Inspectable” is a synthesis, not a measured usability result. |
| I2 | Revision/hash binding and preservation of prior classifications are designed to prevent later decisions from silently being applied to a different candidate or rewriting earlier routing history. | Inferred from F12 and the validation rules in `002-workflow.md` and `004-artifacts.md`. This states the apparent purpose of the mechanism, not a security proof. |
| I3 | The three-candidate ceiling makes failure bounded and visible instead of allowing an indefinite automated rewrite loop. | Inferred from F9–F10. The documentation specifies the bound; the characterization of its purpose is interpretive. |
| I4 | Separating role workspaces and limiting staged inputs reduces unintended cross-role context and makes artifact files, rather than shared conversation, the handoff mechanism. | Inferred from F6–F7 and `003-roles.md`. Do not broaden this into a claim of complete containment; the docs explicitly exclude intentional ancestry escapes from the guarantee. |

## Opinion

| ID | Claim | Note |
| --- | --- | --- |
| P1 | The workflow is a useful example for engineers evaluating a small artifact-driven editorial system. | Editorial assessment aligned with the stated audience, not a repository-proven fact. |
| P2 | The artifact gates are worth their operational complexity when auditability matters. | Value judgment; evidence can explain the gates and their effects but cannot establish this preference universally. |

## Unresolved

| ID | Question or unverified claim | Why unresolved |
| --- | --- | --- |
| U1 | Does the workflow behave exactly as documented in a real authenticated Codex run? | No firsthand execution or independent code inspection was performed, and the brief restricts factual support to README/docs. |
| U2 | How usable are the preserved artifacts for human debugging or audit in practice? | The sources describe what is retained but provide no usability study, timing data, or user evidence. |
| U3 | What happens in future parallel, resumable, publishing, or web-interface variants? | These capabilities are either explicitly unimplemented or out of scope, so no broader workflow claim is supported. |
| U4 | Does the sandbox provide complete containment against intentionally ancestry-escaping hostile processes? | The documentation explicitly says this case is outside the current guarantee and defers complete containment to a future container/VM design. |

</write-uuter-context>