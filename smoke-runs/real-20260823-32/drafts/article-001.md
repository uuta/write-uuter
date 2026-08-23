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
