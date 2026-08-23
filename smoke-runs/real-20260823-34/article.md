# How write-uuter turns a brief into an inspectable reviewed article

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article. Its notable output is not only the finished prose: it also preserves the evidence, outline, candidate drafts, reviews, product-manager decisions, and workflow state that led to success or failure. In this sense, the process is inspectable: a reader can relate the durable artifacts to one another after the participating processes have finished. That describes the record the system keeps, not a measured claim about how easy that record is to use.

The Go controller owns the workflow. It validates state transitions, checks artifacts, binds reviews to revisions, enforces routing and timeouts, and cleans up processes. Editorial roles produce content and judgments, but they do not advance the workflow themselves. The controller records the current state in `workflow.json`, which is rewritten atomically and includes the phase, candidate and revision, active role, artifact paths, review-attempt count, timestamps, and any block reason.

## From brief to candidate

A run begins only after the controller validates all required sections of the brief and confirms a new target. It then initializes the run directory atomically. The roles proceed in a fixed order. First, the Researcher produces a source record and claim ledger. Next, the Story Editor turns that material into an outline in which each section identifies its purpose, supporting evidence, and intended reader takeaway. The Writer then produces exactly the candidate assigned to it.

Each role operates in a fresh private workspace outside the durable run directory and receives only the context defined by its contract. When the role finishes, Go validates its owned regular-file outputs before copying them into the run directory. The handoff is therefore a set of explicit artifacts, rather than an informal shared conversation.

## Review is sequential and recorded

Every candidate is reviewed through four lenses in order: Evidence, Story, Clarity, and Copy. They never run in parallel. Each lens gets a fresh reviewer process, and reviewers cannot edit the candidate. Instead, each writes a structured JSON result and a corresponding Markdown report, both of which must pass validation.

After every lens that is reached, a persistent PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. The separation of responsibilities is deliberate: reviewers identify problems, the PM classifies them, and the Go controller validates the decision and applies the resulting route. Neither a reviewer's message nor a process exit is enough to move the workflow forward.

A validated must-fix ends the remaining reviews for that candidate. The Writer receives a revision assignment, produces the next candidate, and review restarts at Evidence. Optional and invalid findings do not consume another candidate. A finding that needs human judgment blocks the run because the automated workflow does not have authority to resolve it.

Revision is also bounded. Candidate 003 is the hard ceiling; if a validated must-fix remains after that budget is exhausted, the run blocks. This does not guarantee article quality, but it does make the automated rewrite loop finite and leaves its stopping point visible.

## Why artifact gates matter

The controller treats durable files as gates. A role completes only when its worker exits successfully and its owned files have the required structure. Chat messages, transcripts, terminal scrollback, and process exit alone are not editorial completion evidence.

Reviews and PM decisions are also bound to the current candidate revision. The controller rejects stale or mismatched lenses, revisions, requests, digests, findings, and prior-decision data. These checks are designed to keep a decision about one version from silently governing another and to preserve the history of earlier routing. The artifact trail is therefore not just a collection of files; it is a sequence whose links are checked before the next transition.

## Inspectable endings

Success requires the final candidate to clear all four review lenses and their PM routing, followed by final revision and decision validation and verified process cleanup. Only then does the controller atomically publish `article.md`, byte for byte identical to the accepted candidate. The published article therefore cannot drift from the version that passed the workflow.

Failure remains explicit as well. Timeouts, malformed artifacts, premature exits, stale reviews, cleanup failures, requests for human judgment, and an exhausted candidate budget can all block a run with an actionable reason. Earlier candidates, partial review sequences, reports, PM decisions, and post-cleanup audit copies remain available for inspection.

The documented controller is intentionally limited: it handles one non-resumable run, while parallel runs and resumption after a controller restart are not implemented. Publishing integrations and web interfaces are outside its scope. Within those boundaries, `write-uuter` demonstrates the central benefit of an artifact-driven editorial workflow: both the accepted article and the path—or blockage—that produced it remain concrete and checkable.
