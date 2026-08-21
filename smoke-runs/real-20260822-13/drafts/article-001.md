# From Brief to Inspectable Reviewed Article

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article. Its result is more than the final `article.md`: the run retains the evidence, outline, numbered drafts, reviews, and product-manager decisions that produced it. That durable production record is what makes the documented workflow inspectable. It does not prove that the article is better than one produced another way; it lets an engineer examine how this article moved from input to acceptance or blockage.

## Go controls the sequence; roles own the artifacts

The controller, written in Go, owns the mechanics. It advances workflow state, validates artifacts, tracks revision hashes, enforces timeouts, cleans up processes, and decides whether the run may continue or must terminate. Codex roles perform the bounded editorial work, but their conversation or exit status does not control the route.

The production order begins with a Researcher, which creates `evidence/sources.md` and `claim-ledger.md` and may also supply contracted firsthand evidence or assets. A Story Editor turns that material into `outline.md`; every section of the outline must state its purpose, supporting evidence, and intended reader takeaway. A Writer then creates exactly one assigned numbered candidate under `drafts/`.

For each role, the controller creates an isolated workspace outside the durable run directory. It stages only the context allowed by that role's contract, then copies back only validated regular-file outputs. This separates specialized editorial responsibility from control-flow authority: roles own specific artifacts, while Go owns transitions.

## Artifact gates make progress verifiable

A role has not completed its task merely because its process exited successfully or it said it was finished. Before advancing, the controller checks that the role's owned files exist and meet their specific validation rules. Research, outline, candidate, review, and PM-decision artifacts each have their own gate.

Reviews are bound to both their assigned lens and the SHA-256 digest of the exact candidate revision. Reviewers cannot edit that candidate. PM decisions must cover every finding, match the current review request and digest, preserve earlier accepted classifications and routing, and contain no decision for a future lens. Malformed, incomplete, or stale data is rejected instead of silently changing the workflow. In this design, a durable artifact is not just documentation after the event; it is a machine-checked precondition for the next state.

## Four reviews form a serial routing loop

Each candidate passes through four review lenses in order: Evidence, Story, Clarity, and Copy. They never run in parallel, and every lens gets a fresh Codex invocation. After each review, the persistent PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. Go independently validates the decision and applies the route.

An optional or invalid finding does not consume another candidate, so review can continue to the next lens. A request for human judgment blocks the run. A validated must-fix stops the remaining lenses for the current candidate. If budget remains, the Writer receives the prior candidate and the validated correction context, creates the next numbered candidate, and the review sequence starts again at Evidence. Restarting there means a revised draft does not inherit the previous candidate's partial path through later reviews.

This separation is important: the PM supplies editorial classification, while deterministic controller logic enforces ordering, validation, and limits.

## The candidate budget makes termination explicit

Automated revision is capped at candidate 003. A must-fix can trigger a new candidate only while that budget remains; the controller never creates candidate 004. If candidate 003 still has a must-fix, the run becomes blocked. The repository documents the limit, not a claim that three is the optimal number. Its operational effect is to bound automated iteration and leave unresolved work in a visible terminal state.

Success is equally strict. Only after all four lenses clear PM routing does the controller create `article.md`, and that file is byte-for-byte identical to the accepted candidate. Runtime failure, timeout, candidate exhaustion, or a need for human judgment instead leaves a blocked or failed run with its available artifacts preserved.

## What an engineer can inspect

The durable run directory can contain the original brief, `workflow.json`, evidence and claim ledger, outline, numbered drafts, per-lens reviews, PM decisions, and the success-only article. Earlier candidates and partial review sequences remain available after revision or blockage. Atomically rewritten `workflow.json` is the controller's source of truth, recording such details as status, phase, current candidate and revision, active role, artifact paths, review-attempt count, timestamps, and any terminal block reason.

After verified cleanup, `.control/` retains audit copies of assignments, invocation logs, and exit markers. Live requests, temporary workspaces, sandbox profiles, and ownership state remain private and are removed. Consequently, reconstructing consequential decisions does not depend on agent chat or terminal scrollback.

The boundary is deliberately narrow. The implemented workflow is single-run and non-resumable: it does not provide parallel runs, restart resumption, or editing of completed runs. Nor does this repository claim a web interface or publishing integration. Within that scope, write-uuter answers the brief with an unchanged accepted article and a file-based record of how it got there—or an inspectable explanation of why it stopped.
