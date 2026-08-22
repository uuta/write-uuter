# Claim Ledger

Classifications used explicitly below: **Fact**, **Firsthand observation**,
**Inference**, **Opinion**, and **Unresolved**. Source IDs refer to
`evidence/sources.md`.

## Fact

| ID | Claim | Support |
| --- | --- | --- |
| F1 | write-uuter is a Go CLI that turns a Markdown brief into a reviewed article and preserves evidence, outline, candidates, reviews, and PM decisions. | S1, opening description |
| F2 | The controller validates all required brief sections and requires a new target before atomically initializing the run directory. | S1, “Run”; S2, “Controller sequence” |
| F3 | Go owns state transitions, artifact validation, revision hashes, timeouts, routing, and cleanup; agents write only role-owned outputs in isolated workspaces. | S1, “Runtime model”; S2, “Artifact gates” and “Lifecycle”; S3, introduction |
| F4 | The editorial production roles run in order: Researcher, Story Editor, then Writer; the Writer creates candidate 001 from the brief, evidence, ledger, and outline. | S2, controller diagram; S3, Researcher/Story Editor/Writer |
| F5 | Evidence, Story, Clarity, and Copy reviews run in that order, sequentially, using a fresh Codex process for each lens. | S1, “Runtime model”; S2, controller diagram and review-lens rule; S3, “Reviewers” |
| F6 | Reviewers receive the brief, exact candidate and revision, their durable lens instructions, and only lens-specific additional context; they do not edit candidates. | S3, “Reviewers” |
| F7 | The persistent PM classifies every reviewer finding as `valid_must_fix`, `valid_optional`, `invalid` (with a reason), or `needs_human_judgment`; Go independently validates and applies the result. | S3, “PM”; S2, “Artifact gates” |
| F8 | A PM-validated must-fix stops the remaining lenses for that candidate, causes a new candidate when budget remains, and restarts review at Evidence. Optional and invalid findings do not consume a candidate; human judgment blocks. | S2, controller diagram and paragraph after it |
| F9 | Candidate 003 is the hard limit. A validated must-fix at that point blocks the run rather than creating a fourth candidate. | S1, “Runtime model”; S2, controller diagram |
| F10 | Agent completion is not inferred from a final message or process exit alone: owned files must exist and pass role-specific validation gates. | S2, “Artifact gates” |
| F11 | Review artifacts are bound to an allowed lens and the exact SHA-256 candidate revision, and PM decisions are bound to the revision, request ID, and review digest while preserving prior reached-lens decisions. | S2, “Artifact gates”; S3, “PM”; S4, “Review result” and “PM decision” |
| F12 | On success, `article.md` is non-empty and byte-for-byte identical to the candidate accepted through all four final review lenses. | S1, “Run”; S4, “Run layout” |
| F13 | Earlier candidates, partial lens sequences, reviews, and PM decisions are retained when revision occurs or the run blocks. | S4, “Run layout” |
| F14 | `workflow.json` is atomically rewritten as the controller's source of truth and records status, phase, current candidate/revision, active role, artifact paths, review-attempt count, timestamps, and a terminal block reason when blocked. | S4, “workflow.json” |
| F15 | `.control/` preserves post-cleanup audit copies of generated assignments, logs, and exit markers; editorial completion does not depend on chat transcripts or tmux scrollback. | S3, introduction; S4, “workflow.json” |
| F16 | The current implementation is single-run and non-resumable; parallel runs, resume after controller restart, and editing completed runs are not implemented. | S2, opening and final sentence; S3, “Human Editor” |
| F17 | Real runs are documented as macOS-only because the controller uses native Seatbelt isolation; Linux may be cross-built but execution fails closed without an equivalent read-isolation backend. | S1, “Requirements” |
| F18 | The documented containment guarantee excludes an intentionally ancestry-escaping hostile process; stronger containment is deferred to a future container/VM design. | S1, “Runtime model”; S2, “Lifecycle”; S3, “Reviewers”; S4, “workflow.json” |

## Firsthand observation

None. The research consisted only of reading staged documentation. The CLI was
not executed, its artifacts were not generated, and implementation behavior
was not independently observed. Accordingly, no `evidence/firsthand.md` or
firsthand assets were created.

## Inference

| ID | Claim | Basis and limit |
| --- | --- | --- |
| I1 | Durable artifact gates make the workflow inspectable because progress and routing can be reconstructed from validated files rather than ephemeral agent conversation. | Inferred from F10–F15, especially S4's statement that completion does not depend on transcripts or scrollback. “Reconstructed” is an interpretation, not a separately tested property. |
| I2 | Separation of artifact ownership reduces the chance that a reviewer silently repairs the text it is judging or that the PM directly authors a preferred outcome. | Inferred from S3's explicit prohibitions on reviewers and the PM. The documentation establishes separation, not an empirical reduction in error. |
| I3 | Hash, request-ID, and digest bindings make stale or mismatched decisions detectable by the controller. | Inferred directly from the rejection rules in S2 and S4; no adversarial test was performed here. |
| I4 | The three-candidate budget converts an otherwise open-ended revision loop into a deterministic terminal choice: success or blocked. | Inferred from F8–F9 and terminal-state rules in S2. |

## Opinion

| ID | Claim | Note |
| --- | --- | --- |
| O1 | The workflow is a useful small example of artifact-driven editorial orchestration for engineers. | Evaluative framing consistent with the intended audience, not a repository fact. |
| O2 | Preserving blocked runs is preferable to hiding failed attempts because it supports diagnosis and informed retry. | Editorial judgment; the sources establish preservation and later inspection, not that this policy is universally preferable. |

## Unresolved

| ID | Question or unsupported claim | Why unresolved |
| --- | --- | --- |
| U1 | Does the shipped implementation behave exactly as the documentation specifies under real Codex runs? | No CLI execution or code-level verification was performed; the brief permits only README/docs support. |
| U2 | How effective is the workflow at improving article quality compared with another editorial process? | No comparative data, quality metric, or external workflow evidence is in the allowed sources. |
| U3 | How often do runs reach candidate 003, block for human judgment, or fail cleanup? | The allowed sources provide contracts but no operational dataset. |
| U4 | What would publishing integrations or a web interface look like? | Explicitly out of scope and not supported by S1–S4. |
| U5 | When will resume, parallel execution, or stronger container/VM containment be implemented? | The docs identify these as absent or future work but give no schedule. |
