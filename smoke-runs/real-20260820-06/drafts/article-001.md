# From Brief to Inspectable Reviewed Article

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article while retaining the work that produced it: evidence, an outline, article versions, reviews, and product-manager decisions. Its important boundary is that Go makes workflow transitions deterministic; Codex processes still perform the editorial work, whose output is not deterministic.

Before that work begins, the controller checks that the brief contains the required sections and that the target is new. It initializes the run through a temporary sibling directory and a no-replace rename, preventing it from overwriting a directory or symlink created concurrently. The result is not a single agent being asked to write and critique its own answer, but a controller coordinating a series of limited editorial assignments.

## Durable handoffs between roles

The Human Editor owns the brief and later resolves questions that require human judgment. A Researcher creates the source record, claim ledger, and any firsthand material. A Story Editor turns that evidence into an outline in which each section identifies its purpose, support, and intended reader takeaway. A Writer then produces exactly one assigned *candidate*—one version of the article.

The controller runs one persistent PM and no more than one worker at a time in a dedicated tmux session. Each worker gets a fresh private workspace outside the durable run directory, populated only with the inputs named in that role’s contract. Roles therefore own distinct outputs instead of sharing a mutable conversation or editing one another’s artifacts. Reviewers are also fresh processes with deliberately limited context, reducing the opportunity for one lens to inherit another lens’s discussion.

## Files that control progress

An *artifact gate* is a required file whose contents the controller validates before advancing. A completion message is not enough, and neither is a successful process exit: the worker’s owned files must exist and satisfy their format and consistency rules.

That distinction becomes concrete during review. Each review result is tied to one review lens and to the exact SHA-256 revision of the candidate it examined. Its finding data must be valid and internally consistent, and its Markdown report must match the JSON findings. The PM’s decision record must classify every finding exactly once, match the active request and review digest, retain decisions from previously reached lenses, exclude future lenses, and refer to the current revision.

These checks reject stale or mismatched review material. More broadly, they make the controller’s accepted route easier to reconstruct from durable files than it would be from transient chat or tmux scrollback alone.

## A sequential, bounded review loop

Review never runs in parallel. Fresh reviewers examine a candidate in a fixed order: Evidence, Story, Clarity, then Copy. After each reached lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid`, or `needs_human_judgment`.

The classification determines the next state. Optional and invalid findings do not consume another candidate, so review continues to the next lens. A human-judgment decision blocks the run for the Human Editor. A validated must-fix stops the remaining lenses for that candidate, sends the approved required changes to the Writer, and starts the replacement candidate again at Evidence.

Only that last route spends the candidate budget. Candidate 003 is the hard limit: after at most three candidates, another required revision blocks the run rather than opening an unbounded loop. This makes the editorial route bounded, although operational failures can also block a run.

## Inspectable success and failure

Revision does not erase history. Earlier candidates, partial review sequences, reviews, and PM decisions remain in the run directory. The controller atomically rewrites `workflow.json` as its source of truth, recording status, phase, current candidate and revision, active role, artifact paths, review attempts, timestamps, and a terminal block reason when applicable.

On success, the controller revalidates the candidate hash, final reviews, and PM request bindings before writing `article.md`. That terminal article is byte-for-byte identical to the accepted candidate. On a blocked run, preserved intermediate work and an actionable `workflow.json.block_reason` expose where and why progress stopped. In either terminal state, the controller also verifies that its dedicated tmux session is gone.

The durable `.control/` directory keeps audit copies of prompts, invocation logs, and natural exit markers after cleanup. Live launchers, PM requests, and agent workspaces remain in controller-private temporary locations rather than being copied into that audit directory. This separation keeps durable evidence distinct from transient control state.

## What the design establishes

For engineers who value auditability, durable artifact gates are a useful design choice: they preserve concrete inputs, revisions, checks, decisions, and terminal state. That is the documented value of write-uuter’s approach—not evidence that it produces better articles, finishes faster, costs less, or behaves reliably under every variation in model output.

The implementation is deliberately narrow. Parallel runs, resuming after a controller restart, and editing completed runs are not implemented; retrying a blocked workflow means inspecting it and starting a new run directory. Within those limits, write-uuter answers the brief with a clear division of labor: isolated Codex roles create editorial artifacts, while a Go controller advances only when revision-specific, inspectable gates are satisfied.
