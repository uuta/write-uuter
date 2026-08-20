# Outline: From brief to inspectable reviewed article

## Planned article shape

- Target length: 750–850 words, staying below the 900-word limit.
- Narrative spine: introduce the artifact-driven promise, follow one run through its role sequence and review loop, then show how validation and retained terminal artifacts make the result inspectable.
- Terminology: describe the controller as Go-owned and validation-driven, not as making all agent output deterministic.

## 1. A brief becomes a controlled run

**Purpose**

Open with the repository's central behavior: `write-uuter` takes a Markdown brief toward a reviewed article, while preserving the materials and decisions behind that article. Establish that Go—not an agent conversation—controls transitions, validation, timeouts, revision hashes, and cleanup. Briefly explain that validation happens before the run directory is atomically initialized.

**Supporting evidence**

- F01: the CLI turns a brief into a reviewed article and preserves evidence, outline, candidates, reviews, and PM decisions (S1).
- F02: required brief sections and target availability are checked before atomic run initialization (S2).
- F03: Go owns state transitions, validation, revision hashes, timeouts, and cleanup (S1, S3).
- F10: a final chat message or process exit is not accepted as completion without a valid owned artifact (S2).

**Reader takeaway**

The workflow begins as a controller-managed run whose progress is defined by validated files, not by claims made in transient agent conversation.

## 2. Narrow roles create explicit handoffs

**Purpose**

Introduce the actors in production order and explain their boundaries in plain language. The persistent PM begins before research; one worker at a time serves as Researcher, Story Editor, Writer, or a fresh reviewer. Distinguish reporting, editorial classification, and state routing: reviewers identify findings, the PM classifies them, and Go validates the decision and moves the run.

**Supporting evidence**

- F04–F05: one persistent PM, at most one worker, and the order Researcher → Story Editor → Writer → Evidence → Story → Clarity → Copy (S1, S2).
- F06–F07: the PM classifies findings after each reached lens, while Go validates decisions and applies routing (S2, S3).
- F13: reviewers cannot edit candidates and each starts fresh with only lens-specific durable context (S3).
- I03: the separation among reviewer findings, PM classification, and controller routing creates inspectable decision boundaries.

**Reader takeaway**

Each role has a limited job and a durable output, so readers can tell who produced a finding, who judged it, and what mechanism changed workflow state.

## 3. Sequential review spends a bounded candidate budget

**Purpose**

Walk through the review loop. The four lenses run one at a time in a fixed order. After every reached lens, the PM classifies its findings. A validated must-fix stops later lenses and sends the Writer to a replacement candidate when budget remains; that candidate restarts at Evidence. Optional or invalid findings do not consume a candidate, while a need for human judgment blocks the run. Make clear that candidate 003 is the hard ceiling without arguing that three is an ideal number.

**Supporting evidence**

- F06: lenses are sequential and PM classification occurs after each reached lens (S2, S3).
- F07: the four PM classifications are `valid_must_fix`, `valid_optional`, `invalid` with a reason, and `needs_human_judgment` (S3).
- F08–F09: must-fix routing, restart at Evidence, non-consuming optional/invalid findings, human block, and the candidate 003 limit (S1, S2).
- I04: restarting review at Evidence means the final four approvals can all refer to the same candidate revision.

**Reader takeaway**

Revision is controlled and finite: every replacement candidate receives the full ordered review sequence, and unresolved must-fix work cannot continue beyond candidate 003.

## 4. Artifact gates turn files into workflow checkpoints

**Purpose**

Explain what an artifact gate is: the controller advances only when the current role's required file exists and satisfies role-specific rules. Use research and review artifacts as concrete examples. Research needs non-empty sources and a ledger containing all five claim classes; review data must name the assigned lens and exact candidate SHA-256 revision, use valid finding structure, and match its human-readable report. Stale or malformed output is rejected.

**Supporting evidence**

- F10–F11: role advancement depends on validated owned artifacts, including required research, outline, draft, review, and PM-decision structures (S2).
- F12: review results bind the lens and exact candidate revision, and JSON findings must match report fields (S2, S4).
- I02: lens-and-revision checks reduce the risk that stale review output advances the workflow.

**Reader takeaway**

Files are not merely a record made after the fact; their validated content is the gate that permits each transition and ties every review to the candidate it evaluated.

## 5. Success and failure both remain inspectable

**Purpose**

Close by showing what remains on disk. Earlier candidates, partial review sequences, findings, and PM decisions survive revision or blocking. `workflow.json` is the atomically rewritten source of truth for current and terminal state. On success, `article.md` appears only after all four lenses pass routing and exactly matches the accepted candidate. On failure, the run retains an actionable block reason and diagnostic artifacts. Note the current boundary: runs are single-run and non-resumable, with no parallel execution or edits to completed runs.

**Supporting evidence**

- F14: superseded candidates, partial reviews, reviews, and decisions remain available (S4).
- F15: successful `article.md` is byte-for-byte identical to the candidate accepted after all four lenses (S1, S4).
- F16–F18: atomic workflow state, actionable terminal reasons, cleanup, and independence from tmux scrollback or chat transcripts (S2, S4).
- F19: parallel runs, resume, and completed-run edits are not implemented (S2).
- I01 and I05: durable artifacts make intermediate and blocked outcomes inspectable, even though a blocked run cannot be resumed.

**Reader takeaway**

The terminal article is traceable to one accepted candidate, and unsuccessful runs are still explainable from retained state and artifacts; inspectability does not imply resume support.
