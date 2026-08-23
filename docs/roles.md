# Roles

Role instructions are durable files under `prompts/`. Generated assignments
are created in a controller-private runtime outside the run and combine those
instructions with the allowed artifact context. They are copied to the run's
`.control/prompts/` after the terminal cleanup attempt. Success and ordinary
blocked runs verify process absence first; an exceptional cleanup-verification
failure archives the available audit, keeps the staged Codex credentials until
every owned identity has exited and only then removes and verifies them,
retains the non-secret ownership state afterwards, and explicitly does not
claim every process is gone. Go validates role
output and owns every workflow transition. The durable prompt set includes the
PM polling protocol and the shared reviewer output/filesystem contract; those
protocols are not embedded as Go string literals.

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
previously reached lens without changing its accepted classification list, and
contain no future lens. The PM response must be exactly one complete fenced
JSON document. Go independently validates the decision file and applies routing
and the three-candidate limit. Human judgment takes precedence if a response
also contains a must-fix decision. A response is accepted only while the
persistent PM process, tmux window, and ready/ownership identity are all live.

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
the prior candidate, its PM decision, and the reached review result/report so
the validated finding's exact problem and suggested direction are available.
The Writer cannot classify findings, change earlier candidates, or create
`article.md`.

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
must contain one complete five-field entry for every JSON finding in the same
order. The process's filesystem contains only the listed inputs under
`context/` and its output root. Default-deny macOS Seatbelt enforcement denies
unrelated host paths as well as reads of the
durable run and prior-lens/PM/controller workspaces. The
controller uses this external boundary instead of nesting Codex's own sandbox,
which macOS does not permit. The controller tracks and audits the tmux session,
process groups, and descendants it launches; intentional ancestry escapes are
outside this slice's guarantee and complete containment is deferred to a future
container/VM design. This
implementation omits prior-lens conversation and reports from reviewer
assignments; a reviewer process has no inherited conversation from another
lens.
