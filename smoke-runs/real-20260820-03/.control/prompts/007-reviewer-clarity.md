# Clarity Reviewer lens contract

Review only whether the supplied audience can understand and act on the exact
candidate within the supplied constraints. Do not perform evidence, story, or
copy review. Never edit a draft.

Write the assigned `result.json` and `report.md`. Use status `clean`,
`fix_required`, or `blocked`; the exact supplied lens and revision; and an
array of findings. Every finding requires a stable ID, severity, location,
problem, and `suggested_direction`. The report must repeat every machine
finding's fields verbatim.


## Assignment

Lens: `clarity`
Candidate: `article-001`
Revision: `sha256:a0a75f9bb3b5dc0a60c500b14ba315b82204c384472f966b1a709cebc2839bb7`
Write only `reviews/article-001/clarity/result.json` and `reviews/article-001/clarity/report.md`; never edit a draft.

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

- README.md
- docs/workflow.md
- docs/roles.md
- docs/artifacts.md

</write-uuter-context>

## Provided context: drafts/article-001.md

<write-uuter-context name="drafts/article-001.md">
# From Brief to Inspectable Reviewed Article

`write-uuter` is a Go command-line tool that turns a Markdown brief into a reviewed article while retaining the materials and decisions that produced it. Evidence, the claim ledger, outline, numbered candidates, reviews, and product-manager decisions remain available as files. The article is therefore not just an endpoint: it can be traced through the run that created it.

The important qualifier is that the Go controller, not an agent conversation, owns the workflow. Before starting, it checks the brief's required sections and confirms that the target run does not already exist. It initializes the run in a temporary sibling directory and renames that directory into place only after initialization succeeds. From then on, Go controls state transitions, validates outputs, tracks candidate revision hashes, enforces timeouts, and cleans up processes.

An agent saying that it has finished is not enough. The controller advances only when the file owned by the active role exists and passes that role's validation. This makes the on-disk artifacts part of the control mechanism rather than a transcript assembled after the work.

## Roles with explicit handoffs

One persistent product manager, or PM, starts before research. At most one other worker runs at a time in the dedicated tmux session. Those workers proceed in a fixed production order: Researcher, Story Editor, Writer, then Evidence, Story, Clarity, and Copy reviewers.

Each role has narrow ownership. The Researcher prepares sources and a claim ledger. The Story Editor creates the outline. The Writer produces only its assigned numbered candidate. Reviewers report findings but cannot edit that candidate. Each reviewer is a fresh process given the durable context appropriate to its lens; it does not inherit earlier reviewers' conversations or reports.

Review, judgment, and routing are also separate. A reviewer records findings. After every lens that runs, the PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. The PM does not write candidates or reviews. Go then validates the PM's decision file and applies the corresponding state transition. These boundaries leave a visible chain from finding, to editorial judgment, to controller action.

## A sequential, bounded review loop

The four review lenses never run in parallel. A candidate begins with Evidence review and, when permitted to continue, moves through Story, Clarity, and Copy. The PM classifies the result after each reached lens before the controller decides what happens next.

A validated must-fix stops the remaining lenses for that candidate. If revision budget remains, the Writer creates the next numbered candidate, and review starts again at Evidence. Restarting from the first lens ensures that the approvals leading to success concern the same candidate revision. Optional or invalid findings do not spend a candidate. A finding that needs human judgment blocks the run for the human editor to resolve.

The budget is a firm operational limit: candidate 003 is the last candidate. If it still has a validated must-fix, the workflow blocks rather than creating candidate 004. The limit bounds automated revision without claiming that three attempts are universally ideal.

## Files that act as gates

Every handoff has content checks specific to its artifact. Research, for example, must include non-empty sources and a claim ledger containing all five defined classes: fact, firsthand observation, inference, opinion, and unresolved. Outlines, drafts, reviews, and PM decisions have their own structural requirements.

Review validation is especially strict. A review result must identify its assigned lens and the exact SHA-256 revision of the candidate it examined. Its findings must be structurally valid, and the same finding fields must appear in the accompanying readable report. A malformed review, a mismatched lens, or stale revision metadata cannot advance the workflow. Binding a review to the candidate's hash reduces the risk that feedback for an older draft is accidentally treated as approval of a newer one.

## Inspectable endings

Revision does not erase history. Earlier candidates, partial lens sequences, review results, and PM decisions remain in the run directory. The controller atomically rewrites `workflow.json` as its source of truth, recording the current or terminal status, phase, candidate and revision, active role, artifact paths, review attempts, timestamps, and any block reason.

On success, `article.md` appears only after all four lenses pass PM routing. It is a byte-for-byte copy of the accepted candidate, so the published terminal artifact can be matched exactly to the reviews and decisions behind it. On failure, an actionable reason and the artifacts reached so far remain available for diagnosis. A timeout, premature process exit, malformed or stale output, human-judgment decision, or exhausted candidate budget can all produce such a blocked state. In either terminal case, the controller kills its dedicated tmux session, so completion does not depend on lingering agents, scrollback, or chat transcripts.

This inspectability should not be confused with resumability. The shipped controller handles one run at a time and does not implement parallel runs, restart-and-resume, or edits to completed runs. What it does provide is a durable account of the run: the accepted article is traceable to one candidate, and a blocked run remains explainable from validated files left on disk.

</write-uuter-context>

## Provided context: clarity-fields

<write-uuter-context name="clarity-fields">
Audience:
Engineers evaluating a small, artifact-driven editorial workflow.

Constraints:
Use only facts supported by README.md and docs/. Keep the article under 900
words and explain terms in plain language.
</write-uuter-context>