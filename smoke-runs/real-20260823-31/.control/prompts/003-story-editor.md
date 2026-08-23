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

Accessed 2026-08-23. These are controller-staged copies of the brief's local
source hints; no source-repository paths were read directly and no network
sources were used.

## S1 — README.md

- Location: `context/source-hints/001-README.md`
- Original hint: `../README.md`
- Useful summary: Defines write-uuter as a Go CLI that turns a Markdown brief
  into a reviewed article while preserving evidence, outline, candidates,
  reviews, and PM decisions. States that Go owns state transitions,
  validation, revision hashes, timeouts, and cleanup. Gives the role order,
  the three-candidate ceiling, success behavior (an exact copy of the accepted
  candidate), and blocked-run behavior.
- Most useful sections: opening description; **Run**; **Runtime model**.

## S2 — docs/workflow.md

- Location: `context/source-hints/002-workflow.md`
- Original hint: `../docs/workflow.md`
- Useful summary: Gives the controller sequence and review loop. Review lenses
  are sequential; a validated must-fix stops later lenses for that candidate,
  sends work back to the Writer if the candidate is below 003, and restarts
  review at Evidence. Optional or invalid findings do not use a candidate;
  human judgment blocks. Documents artifact gates, isolation/lifecycle checks,
  final revalidation, and unsupported resume/parallel-run behavior.
- Most useful sections: **Controller sequence**; **Artifact gates**;
  **Lifecycle and terminal states**.

## S3 — docs/roles.md

- Location: `context/source-hints/003-roles.md`
- Original hint: `../docs/roles.md`
- Useful summary: Defines ownership and authority for the Human Editor, PM,
  Researcher, Story Editor, Writer, and four reviewer lenses. The PM classifies
  every finding but does not write candidates or reviews; Go independently
  validates PM decisions and applies routing. Reviewers are fresh, sequential,
  read only lens-specific context, and cannot edit candidates.
- Most useful sections: **PM**; **Researcher**; **Story Editor**; **Writer**;
  **Reviewers**.

## S4 — docs/artifacts.md

- Location: `context/source-hints/004-artifacts.md`
- Original hint: `../docs/artifacts.md`
- Useful summary: Specifies the durable run layout and schemas for review
  results, PM decisions, and workflow state. Earlier candidates and partial
  review histories remain available after revision or blockage. `article.md`
  exists only on success and is byte-for-byte identical to the accepted final
  candidate. `.control/` holds post-cleanup audit copies; editorial completion
  does not depend on chat transcripts or tmux scrollback.
- Most useful sections: **Run layout**; **Review result**; **PM decision**;
  **workflow.json**.

## Source boundaries

The supplied brief permits only README.md and docs/ facts. Accordingly, this
research does not make claims about publishing integrations, web interfaces,
or workflows outside this repository. The documents describe the shipped
contracts and also name limitations: runs are single-run and non-resumable,
review lenses are not parallel, completed runs are not edited, Linux execution
fails closed pending a native read-isolation backend, and intentional ancestry
escape is outside the current containment guarantee.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

Classification vocabulary used explicitly in this ledger: **Fact**,
**Firsthand observation**, **Inference**, **Opinion**, and **Unresolved**.
Source IDs refer to `evidence/sources.md`.

## Facts

| ID | Classification | Claim | Support |
| --- | --- | --- | --- |
| F1 | Fact | `write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | S1, opening description |
| F2 | Fact | Before role work begins, the controller validates all required brief sections, requires a new target, initializes the run in a temporary sibling directory, and commits it with a no-replace rename. | S2, Controller sequence |
| F3 | Fact | Go—not an agent final message—owns state transitions and validates files, revisions, timeouts, routing, and cleanup. A worker must exit successfully and its owned files must pass artifact validation before the workflow advances. | S1, Runtime model; S2, Artifact gates |
| F4 | Fact | The main production order is persistent PM, Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewers. The reviewer lenses execute sequentially. | S1, Runtime model; S2, Controller sequence; S3, Reviewers |
| F5 | Fact | The Researcher owns `evidence/sources.md`, optional firsthand evidence/assets, and `claim-ledger.md`; the Story Editor owns `outline.md`; the Writer owns one assigned candidate; each reviewer owns a lens-specific `result.json` and `report.md`. | S3, role sections |
| F6 | Fact | Roles run in separate controller-created workspaces outside the durable run directory, receive only contracted context, and have validated regular-file outputs copied back into the run. | S1, Runtime model; S2, Lifecycle and terminal states |
| F7 | Fact | Evidence, Story, Clarity, and Copy reviewers receive the brief, exact candidate and revision, their durable lens prompt, and only the documented lens-specific additional context. They cannot edit candidates and do not inherit prior-lens conversations. | S3, Reviewers |
| F8 | Fact | After each reached lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` (with a reason), or `needs_human_judgment`; the PM cannot write candidates or reviews, and Go validates its decisions and applies routing. | S3, PM |
| F9 | Fact | A PM-validated must-fix stops the remaining lenses for the current candidate. If fewer than three candidates have been used, the Writer creates the next candidate and review restarts at Evidence. Optional and invalid findings do not consume a candidate. | S2, Controller sequence |
| F10 | Fact | Candidate 003 is the hard limit. An exhausted third candidate, a human-judgment decision, timeout, malformed or stale artifacts, premature exit, or cleanup failure produces a blocked workflow with an actionable reason. | S1, Run/Runtime model; S2, Lifecycle and terminal states |
| F11 | Fact | Review artifacts are revision-bound and schema-validated: lens and SHA-256 revision must match, findings must be complete and unique, and the Markdown report must reproduce the JSON findings in order. Stale metadata is rejected. | S2, Artifact gates; S4, Review result |
| F12 | Fact | PM decisions are request- and review-digest-bound, must cover every finding exactly once, retain previously reached lenses and classifications, and cannot prepopulate future lenses. | S3, PM; S4, PM decision |
| F13 | Fact | Earlier candidates, partial lens sequences, reviews, and PM decisions remain in the run when revision occurs or the workflow blocks. `workflow.json` records running/succeeded/blocked status, phase, current candidate/revision, counts, paths, timestamps, and a terminal block reason when applicable. | S4, Run layout/workflow.json |
| F14 | Fact | Success requires all four final-candidate lenses to pass PM routing plus terminal revalidation. Only then is `article.md` written, byte-for-byte identical to the accepted candidate. | S1, Run; S2, Lifecycle and terminal states; S4, Run layout |
| F15 | Fact | Generated prompts, invocation logs, and exit markers are retained as post-cleanup audit copies under `.control/`; completion does not depend on chat transcripts or tmux scrollback. | S3, opening; S4, workflow.json |
| F16 | Fact | The shipped workflow is single-run and non-resumable; parallel review lenses, resume after controller restart, and editing completed runs are not implemented. | S2, Controller sequence/Lifecycle and terminal states |

## Firsthand observations

No firsthand work was performed. No command execution, live workflow run,
artifact inspection from an actual run, interview, or original measurement is
claimed as a **Firsthand observation**. Therefore `evidence/firsthand.md` and
`evidence/assets/` were not created.

## Inferences

| ID | Classification | Claim | Basis and boundary |
| --- | --- | --- | --- |
| I1 | Inference | The workflow is inspectable because consequential inputs, intermediate candidates, reviews, classifications, state, and terminal output are represented as durable files rather than existing only in agent conversation. | Derived from F1, F5, F11–F15. “Inspectable” is an interpretation of the documented artifact retention and validation model. |
| I2 | Inference | The artifact gates reduce ambiguity at handoffs by requiring each role to produce a narrowly specified, machine-checkable output before Go advances. | Derived from F3, F5, F11, and F12. The sources specify the checks; “reduce ambiguity” is the inferred benefit. |
| I3 | Inference | Restarting every revised candidate at the Evidence lens prioritizes rechecking factual support before spending work on later story, clarity, or copy checks. | Derived from F4 and F9. The priority rationale is inferred; the documents specify the order but do not state this motivation verbatim. |
| I4 | Inference | The three-candidate budget converts an otherwise potentially open-ended agent revision loop into a bounded workflow with a visible blocked outcome. | Derived from F9, F10, and F13. “Bounded” follows directly from the hard ceiling; the design intention is interpretive. |
| I5 | Inference | Keeping review and PM classification separate from candidate writing creates inspectable separation between detecting a problem, deciding whether it must be fixed, and changing prose. | Derived from F5, F7, F8, F12, and F13. “Separation” describes documented authority boundaries; its inspectability benefit is inferred. |

## Opinions

| ID | Classification | Claim | Treatment |
| --- | --- | --- | --- |
| O1 | Opinion | For engineers evaluating editorial automation, explicit artifact contracts are generally preferable to relying on chat history alone. | Editorial judgment, not a repository fact. If used, present as analysis and ground it in F11–F15. |
| O2 | Opinion | A three-candidate ceiling is a pragmatic budget for a small workflow. | Value judgment; the sources establish the ceiling but not that it is optimal or pragmatic. Avoid presenting it as fact. |

## Unresolved

| ID | Classification | Question or claim not established | Why unresolved |
| --- | --- | --- | --- |
| U1 | Unresolved | How often the workflow succeeds, blocks, or reaches candidate 003 in real use. | No operational dataset or run history is included in the permitted sources. |
| U2 | Unresolved | Whether this workflow improves article quality, reviewer agreement, cost, or elapsed time compared with another process. | The permitted sources define implementation and contracts, not comparative outcomes. |
| U3 | Unresolved | Whether the three-candidate budget is the best limit. | The documents state the limit but provide no optimization evidence. |
| U4 | Unresolved | How a future resume mechanism, parallel-run support, Linux isolation backend, container/VM containment, publishing integration, or web interface would work. | These capabilities are absent, deferred, or outside the brief's scope. |

## Safe synthesis boundary

A repository article can state F1–F16 as documented facts and may use I1–I5
when clearly framed as interpretation. O1–O2 must remain opinion, and U1–U4
must not be resolved without additional allowed evidence. No firsthand claims
are available.

</write-uuter-context>