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
