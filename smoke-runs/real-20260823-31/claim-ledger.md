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
