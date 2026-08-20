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
Revision: `sha256:ea1d076018910c4f9e768f5878beb399a5c2b3fa7b5f17af2068d9680113bb17`

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

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article while preserving the evidence, outline, candidates, reviews, and product-manager decisions behind it. Its central design choice is a division of responsibility: isolated Codex roles do the editorial work, while Go controls transitions, validation, revision hashes, routing, timeouts, cleanup, and final publication. Architecturally, that makes advancement more deterministic than leaving agents to decide when their own work is complete.

## From brief to candidate

A run starts only after the controller validates the brief's required sections and confirms that the target is new. It then initializes the run atomically. The issue-1 workflow handles one run at a time and does not resume an interrupted run or edit a completed one.

The editorial sequence begins with three explicit handoffs. The Researcher produces `evidence/sources.md` and `claim-ledger.md`, along with any optional firsthand evidence or assets. The Story Editor turns that material into `outline.md`. The Writer then expands the outline into one assigned candidate, beginning with `drafts/article-001.md`.

Each role works in a private workspace created by the controller. Go stages only the context allowed for that assignment, validates the role's output after a successful exit, and copies validated regular files into the durable run. On macOS, sandboxing also prevents a role from reading the durable run, other roles' workspaces, controller-private state, or unrelated host files. The resulting article therefore moves between roles through concrete files, not a shared conversation.

## Four reviews, in order

Every candidate enters four fresh review lenses in a fixed sequence: Evidence, Story, Clarity, then Copy. They are deliberately sequential rather than parallel. Each reviewer receives the context for its own lens, writes a lens-specific `result.json` and `report.md`, and cannot edit the candidate.

After every reached lens, a persistent PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. The PM records decisions but does not write candidates or reviews. Those decisions are tied to the active request, the review digest, and the exact candidate revision. As review progresses, the accumulated decision record must retain earlier lenses without changing their classifications or routing outcomes.

This is not a parallel vote or an informal approval exchange. Each reviewer examines a specific revision, and the controller advances only after the PM has classified every finding from the reached lens.

## Revision has a hard budget

Only a PM-validated must-fix finding triggers revision. It stops the remaining review lenses for that candidate. If the current candidate is earlier than 003, the Writer produces the next numbered candidate and review starts again at Evidence. Restarting there ensures, by construction, that changed text does not bypass evidence review on the strength of later reviews of an older revision.

Optional and invalid findings do not consume another candidate. A `needs_human_judgment` decision blocks the run. Candidate 003 is the hard limit: if it still receives a validated must-fix, the run blocks instead of opening an indefinite revision loop. The cap bounds the number of candidates, although total duration also depends on timeouts and cleanup.

Other failures can also block a run, including a timeout, malformed or stale artifacts, premature process exit, or cleanup failure. The controller records an actionable reason in `workflow.json`.

## Files are the gates

A worker's reassuring message—or even a successful process exit—is not enough to move the workflow forward. The files owned by that role must exist and pass their specific validation rules. Review metadata must name the correct lens and include the SHA-256 hash of the exact candidate revision. Stale or mismatched artifacts are rejected. PM records must match the active request, review digest, and revision while preserving all earlier reached decisions.

Before publication, the controller revalidates the candidate hash, final reviews, PM request bindings, and accepted classification lists. Only then does it write `article.md`, which is byte-for-byte identical to the candidate accepted through all four lenses.

These gates make the workflow inspectable in an architectural sense: important handoffs and decisions survive as validated files rather than existing only in transient agent conversation. They also provide checks against treating stale reviews, incomplete decisions, or silently changed candidates as approval. The repository documentation specifies those checks; it does not establish defect rates or comparative improvements in editorial quality.

## The terminal record

Success leaves the accepted article and the trail that produced it. Revision or failure does not erase earlier candidates, partial lens sequences, reviews, or PM decisions. `workflow.json` records the status, phase, current candidate and revision, active role, artifact paths, review count, timestamps, and any terminal block reason.

After verified cleanup, `.control/` retains audit copies of generated assignments, per-invocation logs, and natural exit markers. It deliberately omits live runner executables, sandbox profiles, process-group records, PM requests, and agent workspaces.

The durable run directory is therefore the record of both outcomes: on success, it contains `article.md` and its evidence and decision trail; on failure, it preserves the partial history and the reason the controller stopped. Inspecting the workflow does not depend on tmux scrollback or chat transcripts.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:ea1d076018910c4f9e768f5878beb399a5c2b3fa7b5f17af2068d9680113bb17

</write-uuter-context>
