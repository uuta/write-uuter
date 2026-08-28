# Roles

## Model policy

`prompts/models.json` is part of the durable bundle and declares the agent
backend, model, and reasoning effort of every role. The controller validates it
completely before creating a run and resolves each invocation by its role key,
so no role inherits another role's profile and no invocation reaches a CLI
without an explicit profile.

| Role | Lifecycle | Provider | Model | Effort |
| --- | --- | --- | --- | --- |
| PM | One persistent process per run | Codex | `gpt-5.6-sol` | `high` |
| Researcher | Once before the first candidate | Claude Code | `claude-sonnet-5` | `medium` |
| Story Editor | Once before the first candidate | Claude Code | `claude-opus-5` | `high` |
| Visual Editor | Fresh process per candidate | Claude Code | `claude-opus-5` | `high` |
| Writer | Twice per candidate: prose draft, then assembly | Claude Code | `claude-opus-5` | `medium` |
| Evidence Reviewer | Fresh process per candidate | Codex | `gpt-5.6-sol` | `medium` |
| Story Reviewer | Fresh process per candidate | Claude Code | `claude-sonnet-5` | `medium` |
| Clarity Reviewer | Fresh process per candidate | Claude Code | `claude-sonnet-5` | `medium` |
| Copy Reviewer | Fresh process per candidate | Codex | `gpt-5.6-luna` | `low` |

Both Writer invocations of a candidate resolve the same `writer` profile: the
prose draft and the assembly pass are one role doing one job in two stages, and
they are told apart by their assignment and by their sequential invocation IDs.

Reviewer profiles are selected by the combined `reviewer_<lens>` key, so the
four lenses are independent. Retries and revised candidates reuse the same
declared profile; a different policy means a different version-controlled
prompt bundle. Human Editor has no profile.

A provider-neutral runner gives both providers the same immutable role and task
prompt, workspace boundary, timeout and cancellation signal, and audit
identity. Codex invocations add explicit `--model` and
`--config model_reasoning_effort=...` to the existing
`exec --ephemeral --ignore-user-config` boundary. Claude Code invocations are
non-interactive (`--print` with the prompt on stdin) and use `--safe-mode`,
`--dangerously-skip-permissions`, `--no-session-persistence`, and explicit
`--model`/`--effort`. `--bare` is never used, because it disables the OAuth and
keychain reads the Max session depends on.

Claude processes keep the real `HOME`: the Max session is resolved from the
user's account record, so that record has to stay reachable. Only
`CLAUDE_CODE_TMPDIR` is run-owned, pointing at a per-invocation scratch
directory that is removed with the run. The OS sandbox is what enforces the
rest. The exact staged Claude client for a single invocation is the only
process path permitted to read `~/.claude.json` and to start the system
keychain client on the narrowly granted keychain path; `--safe-mode`
additionally stops it loading non-managed customizations. User Claude
configuration under `~/.claude`, history, plugins, skills, hooks, MCP
configuration, session and project state, the rest of the home directory, and
the admin-managed settings tree are denied to the client as well. A
model-invoked tool runs from a different process path, so it can read neither
the account record nor the keychain, cannot start the keychain client, and
cannot reach the user's home.

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

The PM is one long-lived Codex process (`gpt-5.6-sol`, `high` effort) in its
own isolated workspace. It watches
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

It may also write the optional `evidence/screenshot-requests.json` described in
[artifacts](artifacts.md). The Researcher never receives Cloudflare
credentials, never calls a capture API, and never writes
`evidence/assets/screenshots/`; the controller performs, validates, and records
every capture.

## Story Editor

The Story Editor owns `outline.md`. Every planned section records purpose,
supporting evidence, and reader takeaway.

## Visual Editor

The Visual Editor runs once per candidate as a fresh sequential worker, between
the Writer prose draft and the first review lens. It receives the brief, the
outline, the claim ledger, the current prose draft, the generated
`visual-inputs.json`, the screenshot manifest when one exists, and the staged
images themselves as read-only files under `context/visual-inputs/`.

It owns `plan.md` and `plan.json` in its own workspace and nothing else. It
never writes a durable candidate, never edits the prose draft or any other run
artifact, and never decides whether a reviewer finding is valid. Supported
actions are exactly `mermaid`, `existing_local_asset`, `restructure_text`, and
`none`. Visuals are an editorial tool rather than a quota: there is no image
count, no per-heading rule, and a plan that records only `none` and
`restructure_text` is a valid, publishable outcome. It may not place an
unrelated image to break up text, state the same explanation twice, claim a
relationship the ledger or staged evidence does not support, name an unstaged
or unsafe path, or fetch or generate a remote image.

The controller validates the plan completely, copies `plan.md` to
`visuals/article-00N/plan.md`, and places each referenced asset as a read-only
regular file under `visuals/article-00N/assets/`.

## Writer

The Writer owns exactly one assigned `drafts/article-00N.md`. The initial
assignment uses the brief, evidence, ledger, and outline. A revision also gets
the prior candidate, its PM decision, and the reached review result/report so
the validated finding's exact problem and suggested direction are available.
When present, the repository style guide is also supplied as read-only context
for both initial candidates and revisions, and for the assembly pass.

The Writer runs twice per candidate. The prose draft
(`drafts/article-00N-prose.md`) is explanation only: a Mermaid block or an image
reference in it fails the artifact contract, because the Visual Editor pass
evaluates visual opportunities next. The assembly pass
(`drafts/article-00N.md`) receives the validated `plan.md` and
`visual-plan.json` plus each planned asset staged at the exact relative path
the article will use, and it reproduces each planned diagram, references each
planned image once with the plan's alt text, applies each `restructure_text`
entry, and shortens the explanation the visual now carries. It does not revisit
the substance of the prose draft and cannot edit the plan. When the controller
captured screenshots, `evidence/screenshots.json` and the validated images are
supplied read-only as well, so prose can refer to available evidence without
inventing assets. Which image is placed, where it goes, and what its alt text
says are decided by the validated visual plan, not by the Writer: the prose
draft never places one, and the assembly pass follows the plan.
The Writer cannot classify findings, change earlier candidates, or create
`article.md`.

## Reviewers

Evidence, Story, Clarity, and Copy reviewers run in that order as four fresh,
sequential processes. Evidence and Copy run on Codex and Story and Clarity run
on Claude Code, so a candidate is not reviewed only by the Writer's own
provider. They never edit candidates. Every reviewer gets
only its durable lens prompt, full brief, exact candidate, and candidate
revision, plus:

| Lens | Additional context |
| --- | --- |
| Evidence | sources, optional firsthand evidence, claim ledger, optional screenshot manifest and images, placed visual assets |
| Story | outline |
| Clarity | extracted audience and constraints, source prose draft |
| Copy | optional repository style guide (also supplied to the Writer) |

Every lens also receives `visual-plan.md` and `visual-manifest.json`, because
every lens reviews the same visual decisions:

- Evidence checks the factual claims a Mermaid diagram asserts and what a
  placed image visibly shows.
- Story checks that a visual serves its section's purpose and takeaway.
- Clarity checks that a visual improves comprehension and scanability and that
  the prose it replaced was actually shortened rather than duplicated.
- Copy checks Markdown syntax, relative asset paths, meaningful alt text,
  captions or attribution, and formatting around each visual.

A visual or layout defect follows the ordinary PM validation and candidate
revision loop. Reviewers never edit a candidate, a plan, a manifest, or an
asset; editing a staged asset is an artifact-contract failure like editing the
candidate.

The Evidence Reviewer is the only lens that receives screenshot or placed-asset
image bytes. A valid PNG is not evidence on its own: the reviewer must reject a
blank, loading, skeleton, error, or unrelated image, and any screenshot the
candidate describes inaccurately.

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
