# Writer role contract

Write only the assigned versioned candidate under `drafts/`. Expand the
supplied outline into publishable prose supported by the supplied evidence and
brief. Do not leave TODOs or unresolved placeholders.

For a revision, apply every PM-validated must-fix decision using the prior
candidate and the reached review result/report as input. Use the matching
finding's problem, location, and suggested direction to make the correction,
then verify that the revised wording actually resolves it. Do not accept or
reject findings yourself. Never edit a review result, PM decision, earlier
draft, or final `article.md`. Finish only after the assigned candidate is
complete on disk.


## Assignment

Write candidate 001 to `drafts/article-001.md` in this isolated workspace.

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

Accessed 2026-08-23. All sources are controller-staged copies of the repository files named in the brief; no external sources were used.

## README.md

- Location: `context/source-hints/001-README.md`
- Useful sections: introduction; “Run”; “Runtime model.”
- Summary: Defines write-uuter as a Go CLI that turns a Markdown brief into a reviewed article while preserving intermediate evidence, outlines, candidates, reviews, and PM decisions. Describes success and blocked outcomes, the three-candidate hard limit, the controller’s ownership of state transitions and validation, isolated role workspaces, sequential roles and review lenses, and the requirement that a successful `article.md` exactly copy the accepted candidate.
- Useful for: high-level purpose, controller responsibilities, role order, candidate budget, and success/blocked behavior.

## docs/workflow.md

- Location: `context/source-hints/002-workflow.md`
- Useful sections: “Controller sequence”; “Artifact gates”; “Lifecycle and terminal states.”
- Summary: Gives the end-to-end state machine. Research, story editing, writing, and the four review lenses run sequentially. After each lens, the PM classifies findings. A validated must-fix stops later lenses, causes a revision when the candidate number is below 003, and restarts review at Evidence; human judgment or an exhausted budget blocks the run. It also specifies validation gates, timeout and cleanup behavior, final revalidation, atomic publication, and limitations such as no resume or parallel runs.
- Useful for: exact review routing, the significance of durable artifact validation, blocking conditions, and lifecycle guarantees.

## docs/roles.md

- Location: `context/source-hints/003-roles.md`
- Useful sections: “PM”; “Researcher”; “Story Editor”; “Writer”; “Reviewers.”
- Summary: Defines ownership boundaries. The Researcher produces sources and a classified claim ledger; the Story Editor produces an evidence-linked outline; the Writer alone produces candidates; reviewers produce lens-specific results and reports without editing drafts; and the persistent PM classifies findings but neither writes candidates nor reviews. It identifies the context each reviewer receives and states that all workflow transitions remain under Go’s control.
- Useful for: explaining each role, separation of duties, reviewer inputs and outputs, and the distinction between PM judgment and deterministic routing.

## docs/artifacts.md

- Location: `context/source-hints/004-artifacts.md`
- Useful sections: “Run layout”; “Review result”; “PM decision”; “workflow.json.”
- Summary: Documents the durable run tree and validation schemas. Earlier candidates, partial review sequences, reviews, and PM decisions survive revision or blockage. Review JSON binds findings to a lens and candidate hash; PM decisions bind classifications to request IDs and review digests while preserving prior lens outcomes. `workflow.json` records controller state and terminal reasons, `.control/` stores post-cleanup audit copies, and successful publication produces a byte-identical `article.md` only after final gates pass.
- Useful for: the concrete inspectable artifacts, revision binding, audit records, and terminal-state evidence.


</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

Classification meanings used here:

- **Fact**: directly stated by one or more allowed repository sources.
- **Firsthand observation**: learned through a performed test, run, or direct inspection beyond reading the supplied documents.
- **Inference**: a reasoned conclusion drawn from supported facts but not stated verbatim by a source.
- **Opinion**: an evaluative judgment, not an objectively established repository behavior.
- **Unresolved**: not established by the allowed sources and requiring qualification or omission.

No firsthand workflow run or behavioral test was performed, so this ledger makes no firsthand claims and no `evidence/firsthand.md` is warranted.

| ID | Classification | Claim | Support / reasoning | Article-use note |
| --- | --- | --- | --- | --- |
| C01 | Fact | write-uuter is a Go CLI that turns a Markdown brief into a reviewed article while preserving the evidence, outline, candidates, reviews, and PM decisions that produced it. | `001-README.md`, opening description. | Suitable as the concise description of the system. |
| C02 | Fact | Go owns state transitions, validation, revision hashes, timeouts, cleanup, and final routing; agent messages alone do not advance the workflow. | `001-README.md`, “Runtime model”; `002-workflow.md`, “Artifact gates.” | Explain “deterministic controller” in plain language: files must validate before state advances. |
| C03 | Fact | The initial editorial sequence is Researcher, Story Editor, then Writer, producing candidate 001. | `001-README.md`, “Runtime model”; `002-workflow.md`, controller flowchart; `003-roles.md`. | Useful for the opening workflow walkthrough. |
| C04 | Fact | The Researcher owns `evidence/sources.md`, optional firsthand evidence/assets, and `claim-ledger.md`; the ledger distinguishes Fact, Firsthand observation, Inference, Opinion, and Unresolved. | `003-roles.md`, “Researcher”; `002-workflow.md`, “Artifact gates.” | Establishes the evidence foundation and its validation gate. |
| C05 | Fact | The Story Editor owns `outline.md`, whose planned sections must state purpose, supporting evidence, and reader takeaway. | `003-roles.md`, “Story Editor”; `002-workflow.md`, “Artifact gates.” | Shows that structure is itself a checked artifact. |
| C06 | Fact | The Writer alone creates the assigned candidate; on revision it receives the prior candidate, PM decision, and reached review materials, but cannot classify findings, alter prior candidates, or create `article.md`. | `003-roles.md`, “Writer.” | Supports separation of authoring, judgment, and publication. |
| C07 | Fact | Evidence, Story, Clarity, and Copy reviews run in that order as fresh, sequential processes, and reviewers never edit candidates. | `001-README.md`, “Runtime model”; `002-workflow.md`, “Controller sequence”; `003-roles.md`, “Reviewers.” | State the fixed lens order explicitly. |
| C08 | Fact | Each reviewer receives the brief, exact candidate and revision, its durable lens prompt, and only lens-specific additional context: evidence materials, outline, audience/constraints, or an optional style guide. | `003-roles.md`, “Reviewers.” | Useful for explaining focused review and bounded context. |
| C09 | Fact | After each reached lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go validates the decision and applies routing. | `003-roles.md`, “PM”; `002-workflow.md`, flowchart and artifact gates. | Distinguish model judgment from controller-enforced transition rules. |
| C10 | Fact | A validated must-fix stops the remaining lenses for that candidate; if another candidate is available, the Writer revises and review restarts at Evidence. Optional and invalid findings do not consume a candidate, while human judgment blocks the run. | `002-workflow.md`, “Controller sequence.” | Core description of the sequential feedback loop. |
| C11 | Fact | Candidate 003 is the hard limit: a must-fix at that point blocks the run instead of creating candidate 004. | `001-README.md`, “Runtime model”; `002-workflow.md`, flowchart. | State the budget as at most three candidates. |
| C12 | Fact | Review artifacts must bind an allowed status and exact lens to the SHA-256 candidate revision and provide complete, unique findings mirrored in a report. | `002-workflow.md`, “Artifact gates”; `004-artifacts.md`, “Review result.” | Example of a durable gate preventing stale or malformed review output from advancing. |
| C13 | Fact | PM decision records cover each finding, bind to the current revision, request ID, and review digest, and preserve the accepted classifications and routing outcomes of earlier lenses. | `002-workflow.md`, “Artifact gates”; `003-roles.md`, “PM”; `004-artifacts.md`, “PM decision.” | Shows that later decisions cannot silently rewrite earlier review history. |
| C14 | Fact | Earlier candidates, partial lens sequences, reviews, and PM decisions remain in the run when revision occurs or the run blocks. | `004-artifacts.md`, “Run layout.” | Central support for inspectability after failure or revision. |
| C15 | Fact | `workflow.json` is the controller’s atomically rewritten source of truth and records status, phase, current candidate and revision, active role, artifact paths, reviewer attempt count, timestamps, and a terminal block reason when blocked. | `004-artifacts.md`, “workflow.json.” | Identify the compact machine-readable view of run state. |
| C16 | Fact | Success requires final revalidation and absence checks; only then is `article.md` atomically published as a byte-for-byte copy of the accepted candidate. | `001-README.md`, “Run”; `002-workflow.md`, “Lifecycle and terminal states”; `004-artifacts.md`, “Run layout.” | Explain why `article.md` is terminal output rather than another agent-authored draft. |
| C17 | Fact | Runtime failure, timeout, exhausted candidate budget, malformed or stale artifacts, cleanup failure, or need for human judgment can produce a blocked workflow with an actionable reason while retaining audit artifacts. | `001-README.md`, “Run”; `002-workflow.md`, “Lifecycle and terminal states”; `004-artifacts.md`. | Summarize blocked outcomes without implying every failure has identical cleanup evidence. |
| C18 | Fact | Role processes use separate controller-created workspaces, receive contracted context, and have validated regular-file outputs copied into the durable run; on macOS a native default-deny sandbox restricts unrelated reads. | `001-README.md`, “Runtime model”; `002-workflow.md`, “Lifecycle and terminal states”; `003-roles.md`, “Reviewers.” | Keep concise unless isolation is essential to the article’s explanation. |
| C19 | Inference | The workflow is inspectable because important handoffs are represented as preserved, revision-bound files rather than depending on chat transcripts or process exit messages. | Derived from C02, C12–C16; `004-artifacts.md` explicitly says editorial completion does not depend on tmux scrollback or chat transcripts. | Safe if framed as the reason the artifact design aids inspection, not as a universal guarantee of correctness. |
| C20 | Inference | The durable gates make failures diagnosable: the run can show which candidate, lens, finding, decision, or terminal condition stopped progress. | Derived from C12–C17 and the documented run layout. | Phrase as a design consequence, not a measured usability result. |
| C21 | Inference | Separating reviewers from the Writer and PM limits any one role’s authority over drafting, evaluation, and routing. | Derived from C06–C10 and role ownership rules. | “Limits authority” is supported; avoid claiming this eliminates bias or mistakes. |
| C22 | Opinion | A three-candidate budget is a sensible balance between iteration and bounded cost. | The sources document the limit but do not evaluate whether it is optimal. | Omit from a factual repository article unless clearly attributed as opinion. |
| C23 | Opinion | The workflow is easier to audit than conventional editorial collaboration. | No comparison or evaluation appears in the allowed sources. | Omit; comparative claims are out of scope and unsupported. |
| C24 | Unresolved | Whether the system improves article quality, reviewer accuracy, cost, or completion time in practice. | README and docs define behavior and tests but provide no outcome study or benchmark. | Do not claim practical performance or quality gains. |
| C25 | Unresolved | Whether the workflow provides complete containment against intentionally ancestry-escaping hostile processes. | The sources explicitly place such processes outside the current guarantee and defer complete containment to a future container/VM design (`001-README.md`; `002-workflow.md`; `003-roles.md`; `004-artifacts.md`). | If isolation is discussed, preserve this limitation; do not claim complete containment. |
| C26 | Unresolved | Whether a future version will support resuming a blocked run in place. | The allowed sources establish only that the current issue-1 controller is non-resumable and that retry uses a new run directory (`002-workflow.md`; `003-roles.md`); they make no future commitment. | State the current limitation, but do not predict a resume feature or schedule. |

</write-uuter-context>

## Provided context: outline.md

<write-uuter-context name="outline.md">
# Outline: From Brief to Inspectable Reviewed Article

Target length: 700–850 words (hard limit: under 900 words).

## 1. The workflow in one view

- **Purpose:** Introduce write-uuter as a Go CLI and establish the article's central explanation: a controller moves a Markdown brief through isolated editorial roles while preserving the artifacts produced at each handoff.
- **Supporting evidence:** C01 (CLI, reviewed article, preserved intermediate artifacts); C02 (Go owns transitions and validation); C03 (initial Researcher → Story Editor → Writer sequence); C19 (inspectability as a consequence of preserved, file-based handoffs). Sources: `001-README.md` introduction and “Runtime model”; `002-workflow.md` “Artifact gates.”
- **Reader takeaway:** The system is not a single prompt that returns an article; it is a controller-managed sequence whose work can be inspected through durable files.

## 2. Evidence and structure come before prose

- **Purpose:** Walk through the first artifact gates: the Researcher produces sourced evidence and a classified claim ledger, then the Story Editor turns that material into an evidence-linked outline before the Writer creates candidate 001.
- **Supporting evidence:** C03 (role order); C04 (Researcher-owned evidence and claim classifications); C05 (outline requires purpose, supporting evidence, and reader takeaway); C06 (Writer alone creates candidates). Sources: `002-workflow.md` “Controller sequence” and “Artifact gates”; `003-roles.md` “Researcher,” “Story Editor,” and “Writer.”
- **Reader takeaway:** Claims and article structure become explicit, validated inputs to writing rather than implicit context hidden inside an authoring session.

## 3. Four focused reviews, with judgment separated from routing

- **Purpose:** Explain the fixed Evidence, Story, Clarity, and Copy review order; show that reviewers assess rather than edit; and distinguish the PM's classification judgment from the Go controller's enforcement of the next state.
- **Supporting evidence:** C07 (sequential lens order and no reviewer edits); C08 (contracted reviewer context); C09 (PM classifications and controller routing); C21 (separation limits any one role's authority). Sources: `001-README.md` “Runtime model”; `002-workflow.md` “Controller sequence”; `003-roles.md` “PM” and “Reviewers.”
- **Reader takeaway:** Specialized roles produce bounded outputs: reviewers report findings, the PM classifies them, and only the controller advances or redirects the workflow.

## 4. Must-fixes restart the loop within a three-candidate budget

- **Purpose:** Describe the sequential feedback loop and its stopping rules: optional or invalid findings allow review to continue, a valid must-fix stops later lenses and sends an available revision back to Evidence review, human judgment blocks, and candidate 003 is the hard ceiling.
- **Supporting evidence:** C10 (routing for must-fix, optional, invalid, and human-judgment classifications); C11 (candidate 003 limit); C06 (revision inputs supplied to the Writer). Sources: `002-workflow.md` “Controller sequence”; `001-README.md` “Runtime model”; `003-roles.md` “Writer.”
- **Reader takeaway:** Review is ordered and bounded: revisions do not skip back into the middle of review, and the workflow can produce at most three candidates before an unresolved must-fix blocks it.

## 5. Durable gates make each handoff inspectable

- **Purpose:** Use review and PM records as concrete examples of validation gates, then show how preserved candidates, partial review sequences, decisions, and `workflow.json` expose both editorial history and controller state.
- **Supporting evidence:** C12 (reviews bind lens and candidate SHA-256 and require complete findings); C13 (PM decisions bind revision, request ID, and review digest while preserving earlier outcomes); C14 (earlier and partial artifacts survive); C15 (`workflow.json` fields and terminal reason); C20 (diagnosability as a design consequence). Sources: `002-workflow.md` “Artifact gates”; `004-artifacts.md` “Run layout,” “Review result,” “PM decision,” and “workflow.json.”
- **Reader takeaway:** A stale, malformed, or mismatched file cannot silently move the workflow forward, while the files left behind show which candidate, review, decision, or terminal condition determined the outcome.

## 6. Success publishes an accepted candidate; failure remains visible

- **Purpose:** Close with the two terminal outcomes. Explain final revalidation and atomic publication of a byte-identical `article.md`, contrast that with a blocked run retaining an actionable reason and audit artifacts, and connect both outcomes to the value of artifact-driven control.
- **Supporting evidence:** C16 (final revalidation, absence checks, atomic byte-for-byte publication); C17 (blocking causes and retained audit evidence); C01 (preserved artifacts); C19 (inspectability through preserved files). Sources: `001-README.md` “Run”; `002-workflow.md` “Lifecycle and terminal states”; `004-artifacts.md` “Run layout” and “workflow.json.”
- **Reader takeaway:** `article.md` is a controller-published terminal artifact, not another draft, and a run that cannot safely publish still leaves enough durable state to inspect why it stopped.

## Scope and phrasing guardrails

- Keep the explanation within this repository; do not discuss publishing integrations or web interfaces.
- Explain “deterministic” as controller-enforced validation and routing, not as a claim that model outputs are deterministic.
- Frame inspectability and diagnosability as design consequences supported by the artifact model, not measured guarantees of quality or usability.
- Do not claim improvements in quality, accuracy, cost, or completion time (C24), compare the workflow favorably with other editorial systems (C23), endorse the three-candidate limit as optimal (C22), or claim complete hostile-process containment (C25).
- Mention current non-resumability only if space permits, without predicting future support (C26).

</write-uuter-context>