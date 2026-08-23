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
