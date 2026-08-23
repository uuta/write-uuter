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
