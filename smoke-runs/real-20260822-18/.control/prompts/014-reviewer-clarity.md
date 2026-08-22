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
Revision: `sha256:ce635ce987fd53b81f2be75015cc23b8314bc9c78ddb742a573d9aa650113990`

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

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article through a sequence of isolated editorial roles. Along the way, it retains the evidence, outline, article candidates, reviews, and product-manager decisions that produced the result. The Go controller—not an agent’s final message—decides when the workflow may advance. It does so through artifact gates: checks that require a role-owned file to exist and satisfy a defined contract.

That division of labor makes the run inspectable. Codex roles perform bounded editorial tasks, while deterministic controller code manages state transitions, validation, hashes, timeouts, and cleanup. Progress is therefore represented by durable files rather than only by a conversation or a successful process exit.

## From a validated brief to a candidate

Before invoking any editorial role, the controller parses the brief’s required level-two sections and verifies that the target run does not already exist. It initializes the run in a temporary sibling directory, then commits that directory with a rename that will not replace an existing target.

The editorial sequence begins with three distinct owners. The Researcher creates the source record and claim ledger, with optional firsthand evidence or assets when the assignment calls for them. The Story Editor turns that material into an outline. The Writer expands the outline into exactly one assigned, versioned candidate. The Writer does not create the terminal `article.md`.

These boundaries leave a trace from source material to structure to prose. Each role receives a contracted context in an isolated workspace and owns a specific output, so later inspection can identify which artifact entered and left each stage.

## Four review lenses, applied in order

Every candidate is reviewed through four sequential lenses: Evidence, Story, Clarity, and Copy. Each reviewer is a fresh process, receives the exact candidate revision plus lens-specific context, and cannot edit the candidate. Instead, the reviewer writes a machine-readable `result.json` and a matching human-readable `report.md`.

The review result is tied to both its lens and the candidate’s SHA-256 digest. Its report must reproduce every finding’s complete fields in the same order. These requirements let the controller check that the readable explanation and structured routing data refer to the same review of the same text.

A persistent, isolated PM process classifies every finding as a valid must-fix, valid optional issue, invalid finding with a reason, or a matter needing human judgment. The PM neither writes candidates nor performs reviews. Its decision files bind classifications to the candidate revision, request ID, and review digest and must cover each finding exactly once. Go independently validates those files and applies the route, rejecting stale, missing, duplicate, future, or altered routing data.

The order of the lenses has a practical consequence. A PM-validated must-fix stops the remaining reviews for that candidate. Optional or invalid findings allow the sequence to continue without consuming another candidate. A finding classified as needing human judgment blocks the run rather than letting the controller invent an answer.

## Revision has a hard budget

When a must-fix is validated and budget remains, the Writer creates the next versioned candidate. Review then restarts at Evidence and proceeds through the lenses in order unless another PM-validated must-fix stops the remaining reviews for that candidate. Candidate 003 is the hard ceiling. If that candidate still receives a validated must-fix, the run blocks instead of creating candidate 004.

This cap does not guarantee article quality. It does make revision effort bounded and turns unresolved must-fix work into an explicit blocked outcome. Earlier candidates, partial lens sequences, reviews, and PM decisions remain on disk, so revision does not erase the path that led to the terminal state.

## Why the artifact gates matter

A role’s successful exit is necessary but insufficient. Before advancing, the controller checks the existence and validity of required files, including schemas, identities, hashes, and revision bindings. Review artifacts connect findings to an exact candidate digest; PM artifacts connect routing decisions to the corresponding request and review. The atomically rewritten `workflow.json` is the controller’s source of truth, recording status, phase, current candidate and revision, active role, stable artifact paths, review-attempt count, timestamps, and any terminal block reason.

The `.control/` directory adds audit copies of generated assignments, invocation logs, and exit markers. Editorial completion therefore does not depend on chat transcripts or terminal scrollback. Taken together, these cross-checks reduce the chance that stale data, malformed output, or an agent’s unsupported assertion silently advances the run. That is an inference from the documented mechanics, not a measured claim about quality, speed, cost, or correctness.

## Success or preserved blockage

On success, `article.md` is created only after all four lenses pass PM routing. It is byte-for-byte identical to the accepted candidate. On blockage, the retained artifacts and `workflow.json` block reason show where and why the run stopped. Either outcome is available for inspection.

The scope is deliberately limited to the repository’s implemented workflow. Publishing integrations and web interfaces are outside it. Real runs currently require native macOS isolation, while parallel runs, resuming after a controller restart, and editing completed runs are not implemented. Within those limits, `write-uuter` demonstrates how a deterministic controller can coordinate isolated editorial roles while preserving the evidence and decisions behind the final—or blocked—result.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:ce635ce987fd53b81f2be75015cc23b8314bc9c78ddb742a573d9aa650113990

</write-uuter-context>