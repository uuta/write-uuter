# Story Reviewer lens contract

Review only whether the exact candidate follows a coherent narrative and
fulfills the supplied outline's section purposes and reader takeaways. Do not
perform evidence, clarity, or copy review.


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

Lens: `story`
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

## Provided context: outline.md

<write-uuter-context name="outline.md">
# Outline: From Brief to Inspectable Reviewed Article

## Article direction

- **Audience:** Engineers evaluating a small, artifact-driven editorial workflow.
- **Provisional thesis:** `write-uuter` uses a deterministic Go controller to coordinate narrowly scoped Codex roles, validate their file-based handoffs, and preserve the evidence, candidates, reviews, decisions, and terminal state needed to inspect how an article was produced.
- **Target length:** 750–850 words, leaving margin below the 900-word limit.
- **Scope boundary:** Describe only the workflow and contracts documented by the repository. Do not claim measured improvements in quality, cost, speed, or reviewer agreement, and do not speculate about publishing, web interfaces, or unimplemented capabilities.

## 1. The workflow’s central promise: an article with a visible trail

- **Purpose:** Open with the reader’s question and establish the repository’s defining idea: the product is not merely final prose, but final prose accompanied by inspectable production artifacts.
- **Supporting evidence:** F1 establishes that the Go CLI turns a Markdown brief into a reviewed article while preserving evidence, outline, candidates, reviews, and PM decisions. F14 establishes that success produces `article.md` only after review and terminal revalidation. I1 may be presented explicitly as interpretation: durable files make the process inspectable in a way that conversation-only state would not.
- **Reader takeaway:** `write-uuter` makes the path from brief to accepted article visible on disk, so engineers can examine both the outcome and the steps that led to it.

## 2. Go controls the sequence; roles own bounded artifacts

- **Purpose:** Explain the architecture and role sequence in plain language, distinguishing deterministic workflow control from agent-authored editorial work.
- **Supporting evidence:** F2 covers initial brief validation and safe run initialization. F3 states that Go owns state transitions, validation, revision hashes, timeouts, routing, and cleanup, and advances only after successful exit plus artifact validation. F4 gives the order: persistent PM, Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewers. F5 assigns each role its owned files. F6 documents isolated controller-created workspaces and copying validated outputs into the durable run. I2 may be framed as analysis: narrow, machine-checked handoffs reduce ambiguity between roles.
- **Reader takeaway:** Agents do specialized editorial tasks, but the Go controller—not an agent’s final message—decides whether the workflow may advance.

## 3. Research, outline, and candidate form the production path

- **Purpose:** Walk through the pre-review stages without turning the section into a role catalog: research establishes supported claims, the Story Editor organizes them, and the Writer produces one assigned candidate.
- **Supporting evidence:** F4 supplies the production order. F5 identifies the Researcher’s ownership of sources and claim ledger, the Story Editor’s ownership of `outline.md`, and the Writer’s ownership of a single assigned candidate. F6 supports the claim that each role receives contracted context in an isolated workspace and returns validated regular-file output.
- **Reader takeaway:** The workflow converts the brief into progressively more concrete artifacts—evidence, outline, then candidate—while keeping ownership and handoffs explicit.

## 4. Sequential review separates findings, decisions, and revisions

- **Purpose:** Explain how the four review lenses operate and why reviewer findings do not directly change prose or routing.
- **Supporting evidence:** F4 establishes the sequential Evidence, Story, Clarity, and Copy order. F7 states that reviewers are fresh, receive the exact candidate and revision plus lens-specific context, cannot edit candidates, and do not inherit prior-lens conversations. F8 states that the PM classifies every reached finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`, while Go validates the decisions and routes the workflow. F11 and F12 establish revision-, digest-, and request-bound validation and complete finding coverage. I5 may be labeled as interpretation: separate authority makes detection, classification, and prose changes independently inspectable.
- **Reader takeaway:** A reviewer reports a problem, the PM classifies it, and only the controller routes a revision; no single role both judges and silently rewrites the candidate.

## 5. Must-fixes restart review within a three-candidate budget

- **Purpose:** Describe the revision loop, the early stop on a must-fix, and the hard ceiling that prevents indefinite iteration.
- **Supporting evidence:** F9 states that a validated must-fix stops later lenses, sends candidates below 003 back to the Writer, and restarts review at Evidence; optional and invalid findings do not consume a candidate. F10 establishes candidate 003 as the hard limit and lists human judgment, exhaustion, timeout, malformed or stale artifacts, premature exit, and cleanup failure as blocked outcomes. I3 may be framed as interpretation: restarting at Evidence rechecks factual support before later lenses. I4 may be framed as interpretation: the ceiling turns an open-ended revision loop into a bounded workflow with a visible blocked result. Avoid O2 and U3; do not call three candidates optimal.
- **Reader takeaway:** Revision is bounded and conservative: each changed candidate starts review from the first lens, and unresolved must-fixes after candidate 003 block rather than loop forever.

## 6. Artifact gates make success and blockage auditable

- **Purpose:** Close by showing what remains available for inspection and how terminal states are earned, tying durable files back to the article’s central promise.
- **Supporting evidence:** F11 specifies schema and revision checks for review JSON and Markdown reports. F12 specifies complete, digest-bound PM decisions. F13 states that earlier candidates, partial reviews, and decisions remain available and that `workflow.json` records status, phase, candidate, revision, counts, paths, timestamps, and block reason. F14 states that all four final-candidate lenses and terminal revalidation must pass before `article.md` is written byte-for-byte from the accepted candidate. F15 documents audit copies under `.control/` and independence from chat transcripts or tmux scrollback. F16 can supply a concise limitation note: runs are single-run and non-resumable; parallel lenses and editing completed runs are not implemented.
- **Reader takeaway:** Success is a validated terminal state, blockage leaves an actionable record, and either outcome preserves enough durable context to inspect what happened.

## Closing emphasis

- End by restating the answer in one compact paragraph: `write-uuter` combines bounded role authority with deterministic routing and durable artifact gates, yielding either an exact accepted article or an inspectable blocked run.
- Keep “inspectable,” “reduce ambiguity,” and design-rationale language clearly framed as synthesis from the documented contracts, not measured outcomes.
- Do not resolve U1–U4 or imply that this workflow outperforms alternatives.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:41600af0875720e5c11343daefeceed60ccd9c158c1442347b8e0d00f620d63e

</write-uuter-context>