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
Candidate: `article-001`
Revision: `sha256:41600af0875720e5c11343daefeceed60ccd9c158c1442347b8e0d00f620d63e`

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

## Provided context: drafts/article-001.md

<write-uuter-context name="drafts/article-001.md">
# From Brief to Inspectable Reviewed Article

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article. Its defining output is not just the final prose: it also preserves the evidence, outline, candidate drafts, reviews, product-manager decisions, and workflow state that produced that prose. For engineers evaluating an editorial workflow, those durable files make the route from request to result inspectable without relying on an agent conversation.

## Go controls the workflow

The controller first validates the brief's required sections and requires a new run target. It initializes the run in a temporary sibling directory, then commits it with a rename that will not replace an existing target. From there, Go owns the state machine: role order, file validation, revision hashes, timeouts, routing, and cleanup. An agent's final message cannot advance the workflow. A worker must exit successfully, and the files it owns must pass the relevant artifact checks.

The editorial roles operate within those deterministic boundaries. The main sequence is a persistent PM, a Researcher, a Story Editor, and a Writer, followed by fresh reviewers for Evidence, Story, Clarity, and Copy. Roles work in separate controller-created workspaces outside the durable run directory. Each receives only its contracted context, and validated regular-file outputs are copied back into the run.

This produces a simple progression. The Researcher writes `evidence/sources.md` and `claim-ledger.md`, plus optional firsthand evidence or assets when available. The Story Editor turns the supported material into `outline.md`. The Writer expands that plan into one specifically assigned candidate under `drafts/`. Narrow ownership makes each handoff explicit: agents create editorial artifacts, while the controller determines whether those artifacts satisfy the contract required to continue.

## Findings, decisions, and revisions stay separate

Review runs one lens at a time in a fixed order: Evidence, Story, Clarity, then Copy. Each reviewer is fresh. It receives the brief, the exact candidate and its revision, a durable prompt for its lens, and only the additional context documented for that lens. Reviewers do not inherit earlier reviewers' conversations and cannot edit the candidate. Instead, each owns a lens-specific `result.json` and `report.md`.

Those results are tightly bound to the text reviewed. The declared lens and SHA-256 revision must match; findings must be complete and uniquely identified; and the Markdown report must reproduce the JSON findings in order. Stale metadata is rejected.

After every reached lens, the PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid`, or `needs_human_judgment`. An invalid classification requires a reason. The PM neither writes reviews nor changes candidates, and its authority is not the last word on workflow state: Go validates that the decision matches the request and review digest, covers every finding exactly once, retains earlier reached classifications, and does not invent decisions for future lenses. Go then applies the routing.

This separation leaves three distinct acts available for inspection: a reviewer identifies a problem, the PM classifies its significance, and the Writer changes the prose only when routed to a revision. No role can silently combine all three.

## Revision is sequential and bounded

A PM-validated must-fix stops the remaining review lenses for that candidate. If the workflow has used fewer than three candidates, the Writer creates the next version and review restarts at Evidence. Restarting from the first lens means factual support is checked again after the prose changes, before effort moves to Story, Clarity, or Copy. Optional or invalid findings do not consume another candidate.

Candidate 003 is the hard ceiling. If it still has a must-fix, the workflow blocks instead of iterating indefinitely. A decision requiring human judgment also blocks, as do operational or contract failures such as a timeout, malformed or stale artifacts, premature worker exit, or cleanup failure. The three-candidate budget therefore makes the revision loop finite without disguising an unresolved result as success.

## Terminal artifacts show what happened

Revision does not erase history. Earlier candidates, partial lens sequences, reviews, and PM decisions remain in the run. `workflow.json` records the status and phase, current candidate and revision, counts, artifact paths, timestamps, and, for a blocked run, an actionable terminal reason. Generated prompts, invocation logs, and exit markers remain as post-cleanup audit copies under `.control/`; editorial completion does not depend on chat transcripts or terminal scrollback.

Success has a similarly concrete definition. The final candidate must pass PM routing through all four lenses and then survive terminal revalidation. Only afterward does the controller write `article.md`, byte-for-byte identical to the accepted candidate. Thus the terminal state yields either an exact accepted article or a preserved explanation of why the run blocked.

The repository deliberately keeps this model small: runs are single-run and non-resumable, review lenses are not parallel, and completed runs are not edited. Within those limits, `write-uuter` answers the brief-to-article question with bounded role authority, deterministic routing, and durable artifact gates. The result is a reviewed article whose production trail remains visible on disk.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:41600af0875720e5c11343daefeceed60ccd9c158c1442347b8e0d00f620d63e

</write-uuter-context>