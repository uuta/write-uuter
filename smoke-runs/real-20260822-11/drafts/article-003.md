# From Brief to Inspectable Article

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article while preserving the work behind it. The repository documentation describes a single-run workflow in which evidence, an outline, draft versions, reviews, decisions, workflow state, and audit records remain available for inspection. The result is therefore more than `article.md`: it includes a traceable account of how that article was produced.

## Go controls the sequence

Before editorial work begins, the Go controller validates the brief's required sections and checks that the target is new. It initializes the run atomically and starts a persistent project manager, or PM. It then coordinates the Researcher, Story Editor, Writer, and reviewers in order.

The division of responsibility is strict. Go owns state transitions, artifact validation, revision hashes, timeouts, cleanup, and final routing. The editorial roles own narrow outputs. The Researcher creates a source record and a ledger that classifies claims. The Story Editor turns that material into an outline whose sections identify their purpose, evidence, and intended reader takeaway. The Writer produces one assigned candidate—the workflow's term for a versioned draft.

Each role works in a separate private workspace and receives only its contracted context. A role cannot advance the workflow by declaring itself finished. Its output must be a valid regular file, and the controller copies validated output into the durable run. This arrangement gives agents bounded editorial jobs while leaving control of the sequence with deterministic code.

## Four reviews, one lens at a time

Each candidate passes through four fresh reviewers in sequence: Evidence, Story, Clarity, and Copy. A lens is the particular concern a reviewer examines. Reviewers receive context specific to that lens and report findings; they do not edit the candidate.

After every reached lens, the PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. Go validates the PM's recorded decision and enforces the route it specifies. The normal path is:

`candidate → Evidence → PM → Story → PM → Clarity → PM → Copy → PM`

A validated must-fix stops the remaining reviews for that candidate. If the candidate budget permits, the Writer creates the next version and review begins again at Evidence. Optional and invalid findings do not consume another candidate. This makes a revision attributable to a recorded finding and decision instead of an informal exchange among roles.

## Files are gates, not incidental output

An artifact gate is a file requirement that must pass structural and contextual checks before the controller changes state. A successful process exit or persuasive completion message is insufficient. Reviews are bound to both the named lens and the SHA-256 revision of the exact candidate they examined. PM decisions are bound to the current revision, request ID, and review digest, and they preserve decisions from earlier lenses already reached.

These bindings matter when several versions coexist. Earlier candidates, partial review sequences, reviews, and PM decisions remain in the run after revision or blockage. The controller atomically rewrites `workflow.json` as its source of truth. That file records such details as status, phase, current candidate and revision, active role, artifact paths, review count, timestamps, and—when blocked—the reason.

Taken together, these documented checks should reduce the chance that stale, mismatched, or incomplete work silently advances a run. That is a design inference, not a measured reliability result. More concretely, the retained files let an engineer inspect what a role produced, which candidate a review addressed, what the PM decided, and what state the controller recorded next.

## Revision is deliberately bounded

The workflow permits at most three candidates. Candidate 003 is a hard limit: if it still receives a validated must-fix, the run blocks rather than creating candidate 004. A need for human judgment can also block progress, as can malformed or stale artifacts, a timeout, cleanup failure, or another enforced failure condition. The blocked state includes an actionable reason and retains the work already completed.

In design terms, the cap turns potentially open-ended automated rewriting into a visible handoff point for a new, human-directed run. It also means the system fails closed when its contracts cannot justify continuing. The current documented implementation is single-run and non-resumable; it does not support parallel runs, resuming after a restart, or editing completed runs.

## Success includes the trail

Only after all four lenses on the final candidate pass PM routing does the controller publish `article.md`. The published file is byte-for-byte identical to the accepted candidate. At terminal completion, `.control/` retains copies of role assignments, logs, and markers recording that processes ended naturally. These files show what work was assigned and what the processes reported while running and when exiting. The controller removes the temporary execution state used to manage active processes and each role's isolated working directory; those short-lived files are not part of the retained audit record.

For engineers evaluating this repository, the central tradeoff is clear: the documented design favors auditability and bounded control. Its durable artifacts connect editorial inputs, versions, reviews, decisions, and controller state closely enough to reconstruct the path to success or blockage. The documentation does not, however, establish typical speed or cost, success rates, implementation conformance, or advantages over other editorial workflows. Its supported claim is narrower: write-uuter specifies an inspectable route from a brief to a reviewed article, and makes the files—not agent assertions—the authority at every step.
