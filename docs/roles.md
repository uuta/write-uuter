# Roles

Role instructions are durable files under `prompts/`. Generated assignments
are created in a controller-private runtime outside the run and combine those
instructions with the allowed artifact context. They are copied to the run's
`.control/prompts/` only after all processes are stopped. Go validates role
output and owns every workflow transition.

## Human Editor

The human owns `brief.md` and resolves decisions classified as
`needs_human_judgment`. Issue 1 does not resume a blocked run; the preserved
artifacts support a later inspected retry in a new run directory.

## PM

The PM is one long-lived Codex process in its own isolated workspace. It watches
private, request-ID-specific review requests; Go validates and records the
result as `pm-decisions/article-00N.md`. It classifies every finding as:

- `valid_must_fix`
- `valid_optional`
- `invalid` with a non-empty reason
- `needs_human_judgment`

The PM never writes a candidate or review. Each decision record must repeat the
active request ID and a digest of that review, preserve and revalidate every
previously reached lens, and contain no future lens. Go independently validates
the decision file and applies routing and the three-candidate limit. Human
judgment takes precedence if a response also contains a must-fix decision.

## Researcher

The Researcher owns `evidence/sources.md`, optional
`evidence/firsthand.md`/assets, and `claim-ledger.md`. The ledger distinguishes
fact, firsthand observation, inference, opinion, and unresolved claims.

## Story Editor

The Story Editor owns `outline.md`. Every planned section records purpose,
supporting evidence, and reader takeaway.

## Writer

The Writer owns exactly one assigned `drafts/article-00N.md`. The initial
assignment uses the brief, evidence, ledger, and outline. A revision also gets
the prior candidate and its PM decision. The Writer cannot classify findings,
change earlier candidates, or create `article.md`.

## Reviewers

Evidence, Story, Clarity, and Copy reviewers run in that order as four fresh,
sequential Codex processes. They never edit candidates. Every reviewer gets
only its durable lens prompt, full brief, exact candidate, and candidate
revision, plus:

| Lens | Additional context |
| --- | --- |
| Evidence | sources, optional firsthand evidence, claim ledger |
| Story | outline |
| Clarity | extracted audience and constraints |
| Copy | optional repository style guide |

Each owns `reviews/article-00N/<lens>/result.json` and `report.md`. The report
must contain the same findings as JSON. The process's filesystem contains only
the listed inputs under `context/` and its output root; it cannot browse the
durable run or another review. This implementation omits prior-lens
conversation and reports from reviewer assignments; a reviewer process has no
inherited conversation from another lens.
