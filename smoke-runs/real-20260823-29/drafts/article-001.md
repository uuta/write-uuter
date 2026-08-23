# How write-uuter turns a brief into an inspectable reviewed article

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article. It does not treat the job as one long prompt whose useful history disappears into a chat transcript. Instead, each stage produces durable files: evidence, a claim ledger, an outline, candidate drafts, reviews, and product-manager decisions. In this sense, the workflow is artifact-driven: validated files on disk represent progress and determine what may happen next.

That design makes a run inspectable. An engineer can examine the inputs, see which candidate each reviewer assessed, and trace the decisions that led to acceptance or blockage. The repository documentation specifies these behaviors as contracts; the account here is based on those documents rather than an independent execution of the CLI.

## The controller sets the boundaries

Before editorial work starts, the controller validates that the brief contains every required section and that the target run is new. It then initializes the run directory atomically. From that point, responsibilities are deliberately divided: Go owns state transitions, artifact validation, revision hashes, timeouts, routing, and cleanup, while each Codex role writes only its assigned outputs in an isolated workspace.

The controller records its source of truth in `workflow.json`, which it rewrites atomically. That file tracks such details as status, phase, current candidate and revision, active role, artifact paths, review attempts, timestamps, and, when necessary, the reason a run was blocked. Agents therefore contribute bounded editorial work; they do not decide for themselves that the workflow has advanced.

## From evidence to the first candidate

The production chain is sequential. First, the Researcher assembles the evidence and claim ledger. The Story Editor turns those materials into an outline. The Writer then uses the brief, evidence, ledger, and outline to create `article-001.md`.

These are separate handoffs rather than one continuing agent conversation. Each role owns a specific artifact that becomes an inspectable input to the next stage. This separation also keeps authorship attributable: a later reviewer judges the candidate but cannot silently repair it.

## Four fresh review lenses

Each candidate enters four reviews in a fixed order: Evidence, Story, Clarity, and Copy. Every lens runs in a fresh Codex process, sequentially. A reviewer receives the brief, the exact candidate and revision under review, durable instructions for that lens, and only the additional context permitted for that lens. Reviewers write review artifacts; they never edit the candidate.

Completion is not inferred merely because an agent prints a final message or exits. The expected owned file must exist and pass its role-specific validation gate. A review must use an allowed lens and is bound to the exact candidate revision by its SHA-256 hash. Those checks prevent a review of one revision from being applied as though it covered another.

After each review, a persistent PM classifies every finding using a constrained vocabulary. `valid_must_fix` means a correction is required; `valid_optional` records a legitimate but non-blocking improvement; `invalid` rejects a finding and requires a reason; and `needs_human_judgment` says automation should not decide. The PM supplies editorial judgment, but Go independently validates the decision before routing the run.

PM decisions are bound to the candidate revision, the request ID, and the review digest, while retaining decisions from previously reached lenses. In practical terms, these bindings let the controller detect stale or mismatched decision artifacts instead of trusting conversational assertions.

## Revision has a hard limit

A validated must-fix finding immediately stops the remaining review lenses for that candidate. If the candidate budget permits, the Writer creates the next numbered candidate using the prior draft and the reached review and decision. Review then restarts from Evidence, so every revised candidate must pass the entire lens sequence from the beginning.

Optional and invalid findings do not consume another candidate. A finding classified as needing human judgment blocks the run. Candidate 003 is the hard ceiling: if it receives a validated must-fix, the controller blocks rather than creating candidate 004. This budget turns revision into a bounded process with a deterministic terminal outcome instead of an open-ended rewriting loop.

## Terminal artifacts explain the outcome

Success has a precise on-disk meaning. After one candidate passes all four final lenses, `article.md` must be non-empty and byte-for-byte identical to that accepted candidate. The final file is not a separately regenerated version whose relationship to the reviewed text must be guessed.

Revision and blockage remain inspectable too. Earlier candidates, partial lens sequences, reviews, and PM decisions are retained. `workflow.json` records the terminal state and any block reason, while `.control/` keeps audit copies of generated assignments, logs, and exit markers after cleanup. Completion therefore does not depend on preserving chat transcripts or terminal scrollback.

The result is a small, deterministic editorial controller with visible boundaries. Codex roles research, structure, write, review, and classify; Go verifies their artifacts and controls movement through the workflow. Whether a run succeeds or blocks, the files left behind show what was reviewed, what was decided, and why the process stopped where it did.
