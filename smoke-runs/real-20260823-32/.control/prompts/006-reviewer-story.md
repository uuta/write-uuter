# Story Reviewer lens contract

Review only whether the exact candidate follows a coherent narrative and
fulfills the supplied outline's section purposes and reader takeaways. Do not
perform evidence, clarity, or copy review.


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

Lens: `story`
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

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:2ae1558e4ad3e273e274c33050531d33d4036bc25bfb88c613d81e096640db80

</write-uuter-context>