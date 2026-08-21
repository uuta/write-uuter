# From Brief to Inspectable Reviewed Article

write-uuter is a Go CLI that turns a Markdown brief into a reviewed article while preserving the work that produced it. Its central design choice is a clear division of responsibility: a deterministic Go controller owns workflow state, validation, routing, revision hashes, timeouts, cleanup, and publication, while isolated Codex processes perform bounded editorial tasks.

This is not one agent carrying an article from prompt to publication. The project assigns research, structure, writing, review, and editorial judgment to distinct roles. Workers and reviewers run separately, each reviewer starts as a fresh process, and one persistent PM makes editorial classifications throughout the run. Because those judgments are retained while Go enforces the resulting routes, engineers can inspect both the editorial decision and the control flow that followed it.

## From brief to first candidate

A run begins only after the controller validates the brief's required sections and confirms that the target is new. It then initializes the run directory atomically. From there, work proceeds in a fixed order.

First, the Researcher creates a non-empty source record and a claim ledger. The ledger must explicitly account for five kinds of material: Fact, Firsthand observation, Inference, Opinion, and Unresolved. That classification gives later roles a concrete boundary between supported claims and material that needs qualification or cannot yet be established.

Next, the Story Editor converts the evidence into an outline. Each section must state its purpose, supporting evidence, and intended reader takeaway. The Writer then expands that outline into candidate 001. A candidate must contain substantive text and cannot contain unresolved drafting markers.

These requirements are examples of artifact gates: a required file and the validation it must pass before the controller advances. Each handoff is therefore a named, durable artifact with a minimum contract, not an informal assurance that the work is done.

## Files, not final messages, prove progress

Each role works in an isolated workspace rather than directly in the durable run directory or another role's workspace. Go stages only the context that role is allowed to receive. After the process exits successfully, the controller checks the role's owned outputs, requires valid regular files, and copies validated results back into the run.

An agent's final message cannot advance the workflow on its own. The expected files must exist and satisfy their contracts. Review output, for example, is tied to a specific candidate by its SHA-256 revision; the declared lens and revision must match the assignment, and the structured findings must agree with the accompanying report.

The controller records the live state in `workflow.json`, which is rewritten atomically. It includes the run's status and phase, current candidate and revision, active role, artifact paths, review attempts, timestamps, and—when applicable—a terminal blocking reason. In this repository, “inspectable” is best understood as a consequence of this design: state transitions depend on retained, validated files rather than transient chat transcripts or terminal scrollback.

## Four reviews, one lens at a time

Every candidate enters a fixed, sequential review sequence: Evidence, Story, Clarity, then Copy. Reviewers report findings but never edit the candidate. After each reached lens, the persistent PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`.

The separation matters. A fresh reviewer supplies a lens-specific assessment, the PM records the editorial judgment about each finding, and Go validates that decision artifact and applies predetermined routing rules. Later lenses do not run concurrently or get ahead of an unresolved must-fix.

If the PM validates a must-fix, review of that candidate stops immediately. The Writer receives the prior candidate and the reached review result and produces the next numbered candidate. Review then restarts at Evidence. One rationale supported by the design is that changed content should not inherit later-lens results attached to an earlier revision. Optional and invalid findings do not consume another candidate.

Automated revision is bounded at candidate 003. A must-fix against that candidate blocks the run instead of creating candidate 004. A need for human judgment, a runtime failure, or an artifact-validation failure also blocks the run. The limit does not establish that three attempts are optimal; it simply prevents indefinite automated revision and turns unresolved work into an explicit terminal state.

## Two inspectable endings

Success requires the final candidate to pass routing at all four review lenses. Only then does the controller create `article.md`, as a byte-for-byte copy of the accepted candidate. The published artifact can therefore be matched exactly to the revision that was reviewed.

A blocked run keeps its reason and the work accumulated before the stop. Depending on how far it progressed, that record can include earlier candidates, partial review sequences, reviewer reports, PM decisions, workflow state, and audit copies of prompts, logs, and exit markers. Failure does not collapse into an unexplained absence of an article.

That durable record is the practical answer to how write-uuter produces an inspectable reviewed article. The controller does not merely ask roles to follow a process; it requires evidence of each completed step, validates the evidence, and preserves the route to either exact publication or explicit blockage. The documented workflow does have firm boundaries: parallel runs, resuming after a controller restart, and editing completed runs are not implemented. Within those boundaries, its artifacts make the outcome traceable without treating agent conversation as workflow state.
