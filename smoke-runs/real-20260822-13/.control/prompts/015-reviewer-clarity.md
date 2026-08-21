# Clarity Reviewer lens contract

Review only whether the supplied audience can understand and act on the exact
candidate within the supplied constraints. Do not perform evidence, story, or
copy review.


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

Lens: `clarity`
Candidate: `article-003`
Revision: `sha256:eee465edbaaedb1d6592deb1b2ba0bbccf1ede36b54ae69a19cfaff68073ae14`

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

## Provided context: clarity-fields.md

<write-uuter-context name="clarity-fields.md">
Audience:
Engineers evaluating a small, artifact-driven editorial workflow.

Constraints:
Use only facts supported by README.md and docs/. Keep the article under 900
words and explain terms in plain language.

</write-uuter-context>

## Provided context: drafts/article-003.md

<write-uuter-context name="drafts/article-003.md">
# From Brief to Inspectable Reviewed Article

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article. Its result is more than the final `article.md`: the run retains the evidence, outline, numbered drafts, reviews, and product-manager decisions that produced it. That durable production record is what makes the documented workflow inspectable. It does not prove that the article is better than one produced another way; it lets an engineer examine how this article moved from input to acceptance or blockage.

## Go controls the sequence; roles own the artifacts

The controller, written in Go, owns the mechanics. It advances workflow state, validates artifacts, tracks revision hashes, enforces timeouts, cleans up processes, and decides whether the run may continue or must terminate. Codex roles perform the bounded editorial work, but their conversation or exit status does not control the route.

The production order begins with a Researcher, which creates `evidence/sources.md` and `claim-ledger.md` and may also supply firsthand notes or supporting assets when its assignment permits them. A Story Editor turns that material into `outline.md`; every section of the outline must state its purpose, supporting evidence, and intended reader takeaway. A Writer then creates exactly one assigned numbered candidate under `drafts/`.

For each role, the controller creates an isolated workspace outside the durable run directory. It stages only the context allowed by that role's contract, then copies back only validated regular-file outputs. This separates specialized editorial responsibility from control-flow authority: roles own specific artifacts, while Go owns transitions.

## Artifact gates make progress verifiable

A role has not completed its task merely because its process exited successfully or it said it was finished. Before advancing, the controller checks that the role's owned files exist and meet their specific validation rules. Research, outline, candidate, review, and product-manager decision artifacts each have their own gate.

Reviews are bound to both their assigned lens and the SHA-256 digest of the exact candidate revision. Reviewers cannot edit that candidate. Product-manager decisions must cover every finding, match the current review request and digest, preserve earlier accepted classifications and routing, and contain no decision for a future lens. Malformed, incomplete, or stale data is rejected instead of silently changing the workflow. In this design, a durable artifact is not just documentation after the event; it is a machine-checked precondition for the next state.

## Four reviews form a serial routing loop

Each candidate passes through four review lenses in order: Evidence, Story, Clarity, and Copy. They never run in parallel, and every lens gets a fresh Codex invocation. After each review, the product-manager Codex role classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. Unlike the fresh reviewer invoked for each lens, this product-manager role persists across the run, retaining the prior review classifications and routing context. The Go controller separately validates each decision and applies the route.

An optional or invalid finding does not consume another candidate, so review can continue to the next lens. A request for human judgment blocks the run. A validated must-fix stops the remaining lenses for the current candidate. If budget remains, the Writer receives the prior candidate and the validated correction context, creates the next numbered candidate, and the review sequence starts again at Evidence. Restarting there means a revised draft does not inherit the previous candidate's partial path through later reviews.

This separation is important: the product-manager role supplies editorial classification, while deterministic controller logic enforces ordering, validation, and limits.

## The candidate budget makes termination explicit

Automated revision is capped at candidate 003. A must-fix can trigger a new candidate only while that budget remains; the controller never creates candidate 004. If candidate 003 still has a must-fix, the run becomes blocked. The repository documents the limit, not a claim that three is the optimal number. Its operational effect is to bound automated iteration and leave unresolved work in a visible terminal state.

Success is equally strict. Only after all four lenses clear product-manager routing does the controller create `article.md`, and that file is byte-for-byte identical to the accepted candidate. Runtime failure, timeout, candidate exhaustion, or a need for human judgment instead leaves a blocked or failed run with its available artifacts preserved.

## What an engineer can inspect

The durable run directory can contain the original brief, `workflow.json`, evidence and claim ledger, outline, numbered drafts, per-lens reviews, product-manager decisions, and the success-only article. Earlier candidates and partial review sequences remain available after revision or blockage. Atomically rewritten `workflow.json` is the controller's source of truth, recording such details as status, phase, current candidate and revision, active role, artifact paths, review-attempt count, timestamps, and any terminal block reason.

After verified cleanup, `.control/` retains audit copies of assignments, invocation logs, and exit markers. Live requests, temporary workspaces, sandbox profiles, and ownership state remain private and are removed. Consequently, reconstructing consequential decisions does not depend on agent chat or terminal scrollback.

The boundary is deliberately narrow. The implemented workflow is single-run and non-resumable: it does not provide parallel runs, restart resumption, or editing of completed runs. Nor does this repository claim a web interface or publishing integration. Within that scope, write-uuter answers the brief with an unchanged accepted article and a file-based record of how it got there—or an inspectable explanation of why it stopped.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:eee465edbaaedb1d6592deb1b2ba0bbccf1ede36b54ae69a19cfaff68073ae14

</write-uuter-context>
