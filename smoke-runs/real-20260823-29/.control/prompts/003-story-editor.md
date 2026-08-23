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

Accessed 2026-08-22. All research used the controller-staged copies in
`context/source-hints/`; no source-repository files were read directly and no
network sources were used.

## S1 — README

- Staged location: `context/source-hints/001-README.md`
- Source hint represented: `../README.md`
- Useful summary: Defines write-uuter as a Go CLI that turns a Markdown brief
  into a reviewed article while preserving the evidence, outline, candidates,
  reviews, and PM decisions. Describes brief validation, atomic run creation,
  the runtime division of responsibility, isolated role workspaces, the
  sequential role/reviewer order, the three-candidate ceiling, success as an
  exact copy of the accepted candidate, and blocked terminal behavior.
- Especially useful for: the top-level workflow, controller/agent boundary,
  platform constraints, success and failure semantics, and candidate budget.

## S2 — Workflow

- Staged location: `context/source-hints/002-workflow.md`
- Source hint represented: `../docs/workflow.md`
- Useful summary: Gives the controller sequence from brief validation through
  research, outlining, drafting, four fresh review lenses, PM classification,
  revision, and publication or blocking. Specifies that lenses are sequential,
  a must-fix stops the remaining lenses and restarts review at Evidence on the
  next candidate, optional/invalid findings do not consume a candidate, and
  human judgment blocks. Enumerates validation gates and lifecycle/cleanup
  requirements, including hash binding and failure preservation.
- Especially useful for: sequence, artifact gates, revision routing, terminal
  validation, timeouts, non-resumability, and failure modes.

## S3 — Roles

- Staged location: `context/source-hints/003-roles.md`
- Source hint represented: `../docs/roles.md`
- Useful summary: Defines ownership and prohibitions for the Human Editor, PM,
  Researcher, Story Editor, Writer, and Evidence/Story/Clarity/Copy reviewers.
  Explains the persistent PM protocol, its four finding classifications, the
  context supplied to each reviewer, and the rule that reviewers neither edit
  candidates nor inherit prior reviewers' conversations.
- Especially useful for: explaining each role, separation of duties, reviewer
  context, and why revisions remain attributable.

## S4 — Artifacts

- Staged location: `context/source-hints/004-artifacts.md`
- Source hint represented: `../docs/artifacts.md`
- Useful summary: Defines the durable run layout and validated schemas for
  reviews, PM decisions, and `workflow.json`. States that earlier candidates,
  partial reviews, reviews, and decisions remain available after revision or
  blocking; `article.md` exists only on success and is byte-for-byte identical
  to the accepted candidate. Describes `.control/` audit copies and emphasizes
  that completion does not depend on chat transcripts or tmux scrollback.
- Especially useful for: inspectability, schema-level gates, terminal artifact
  meanings, audit records, and exact-copy publication.

## Source-use boundaries

- The brief limits support to README and `docs/`; S1–S4 are the complete source
  set used here.
- Claims about publishing integrations, web interfaces, or workflows outside
  this repository are unsupported and excluded.
- The documentation describes the shipped contracts; this research did not run
  the CLI or independently test implementation behavior.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
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

</write-uuter-context>