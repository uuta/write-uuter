# Evidence Reviewer lens contract

Review only whether material claims in the exact candidate are supported and
faithful to the supplied sources, firsthand record when present, and claim
ledger. Do not perform story, clarity, or copy review.


# Shared reviewer output contract

Use only the files under the supplied `context/` directory. Do not inspect its
parent, another workspace, the source repository, `.control/`, `reviews/`, PM
decisions, logs, or another lens's output. Never edit `context/article.md`.
The `context/` directory contains every permitted input and no other run
artifact. Write only `result.json` and `report.md` in the workspace root. Use
status `clean`, `fix_required`, or `blocked`; the exact supplied lens and
revision; and an array of findings. Every finding requires a stable ID,
severity, location, problem, and `suggested_direction`. The report must repeat
every machine finding field verbatim.

For each finding, use these five labels in this order (bullets and blank lines
between fields are optional): `id`, `severity`, `location`, `problem`, and
`suggested_direction`. Do not split a field value across lines.

The JSON field name for the revision is exactly `reviewed_revision` (never
`revision`). Use this exact shape, retaining the finding objects only when
there are findings:

```json
{
  "status": "clean",
  "lens": "evidence",
  "reviewed_revision": "sha256:the-exact-assigned-revision",
  "findings": []
}
```

Before exiting, re-read `result.json` and verify that it contains all four
top-level keys: `status`, `lens`, `reviewed_revision`, and `findings`.


## Assignment

Lens: `evidence`
Candidate: `article-001`
Revision: `sha256:2ae1558e4ad3e273e274c33050531d33d4036bc25bfb88c613d81e096640db80`

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

## Provided context: drafts/article-001.md

<write-uuter-context name="drafts/article-001.md">
# From Brief to Inspectable Reviewed Article

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article. It does not treat that job as one long prompt. Instead, a Go controller moves the brief through a sequence of editorial roles and preserves the evidence, outline, drafts, reviews, and product-manager decisions produced along the way.

That distinction explains what “deterministic” means here. It does not mean that model-generated prose or judgments are deterministic. It means that Go, rather than an agent message, owns state transitions, validates the required files, tracks revisions, and decides which contracted step may run next. The resulting handoffs are durable files, so the path to an article can be inspected without relying on a chat transcript or process output.

## Evidence and structure before prose

The sequence begins with a Researcher. This role creates a source record and a claim ledger, with claims classified as facts, firsthand observations, inferences, opinions, or unresolved. Optional firsthand evidence and assets can also be included. These artifacts make the proposed factual basis explicit before drafting begins.

Next, the Story Editor creates an evidence-linked outline. Each planned section must identify its purpose, supporting evidence, and intended reader takeaway. Only after those inputs pass their gates does the Writer create candidate 001. The Writer alone owns candidate prose: other roles cannot quietly rewrite a draft, and the Writer cannot classify review findings or publish the terminal article.

The roles run in controller-created, separate workspaces with only their contracted context. Validated regular-file outputs are copied into the durable run. This keeps role responsibilities narrow while allowing the controller to retain the work that crosses each boundary.

## Four reviews, one ordered loop

Every candidate enters four fresh, sequential review processes in a fixed order: Evidence, Story, Clarity, and Copy. Reviewers assess the candidate and produce findings; they never edit it. Each receives the brief, the exact candidate revision, and a durable prompt for its lens. Additional context is lens-specific—for example, the Evidence reviewer receives the evidence materials, while the Story reviewer receives the outline.

After each reached lens, a persistent PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. This is a deliberate separation between judgment and routing. The PM decides how findings should be classified, but Go validates that decision and enforces the next transition. No single editorial role controls drafting, evaluation, and workflow movement.

Optional and invalid findings do not consume another candidate, so review continues to the next lens. A need for human judgment blocks the run. A validated must-fix immediately stops the remaining lenses for that candidate. If the revision budget allows it, the Writer receives the prior candidate, the PM decision, and the review materials reached so far, then creates the next numbered candidate. Review restarts at Evidence rather than resuming midway through the lens sequence.

The budget is a hard maximum of three candidates. A must-fix against candidate 003 blocks the workflow instead of producing candidate 004. The limit therefore bounds iteration without implying that an unresolved issue may be ignored.

## Why the artifact gates matter

The controller checks more than whether a file exists. A review result must use an allowed status, identify the exact lens, bind itself to the candidate’s SHA-256 revision, and contain complete, unique findings that are mirrored in its report. A PM decision must cover those findings and bind itself to the current revision, request ID, and review digest. It must also preserve classifications and routing outcomes already accepted for earlier lenses.

These checks prevent a stale, malformed, or mismatched artifact from silently advancing the run. They also leave a concrete explanation of what happened. Earlier candidates, incomplete lens sequences, reviews, and PM decisions remain present after a revision or blockage. `workflow.json`, atomically rewritten by the controller, provides the compact machine-readable view: status, phase, current candidate and revision, active role, artifact paths, reviewer attempts, timestamps, and—when blocked—the terminal reason.

That history makes failures diagnosable at the workflow’s own level. An inspector can locate the candidate, lens, finding, decision, or terminal condition that stopped progress rather than reconstructing the sequence from transient agent output.

## Publication is a terminal controller action

A run succeeds only after final revalidation and checks for artifacts that should not be present. The controller then atomically publishes `article.md` as a byte-for-byte copy of the accepted candidate. In other words, `article.md` is not a fresh draft created by another role; it records the candidate that cleared the gates.

Runs can instead block because of runtime failures, timeouts, an exhausted candidate budget, stale or malformed artifacts, cleanup failure, or a decision requiring human judgment. In that case, the workflow retains an actionable reason and its audit artifacts. Success and blockage therefore share the same central property: the terminal state and the editorial path to it remain available as files that engineers can inspect.

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

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:2ae1558e4ad3e273e274c33050531d33d4036bc25bfb88c613d81e096640db80

</write-uuter-context>