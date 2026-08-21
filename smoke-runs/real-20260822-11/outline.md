# Outline: From brief to inspectable reviewed article

## 1. The workflow in one view

- **Purpose:** Answer the article's central question immediately: write-uuter is a Go CLI whose controller moves a Markdown brief through isolated editorial roles while preserving the materials behind the result. Define “inspectable” in concrete terms as retaining evidence, an outline, candidate drafts, reviews, decisions, workflow state, and audit records. Briefly set expectations that this is a documented, single-run repository workflow rather than a claim about editorial automation generally.
- **Supporting evidence:** F1 (S1); F2 (S1, S2); F12–F15 (S1, S4); F17 (S2). I1 may be used only as an explicitly framed synthesis of the retained artifacts, not as a measured result.
- **Reader takeaway:** The product is not just the final article: it is the final article plus a durable, traceable record of how the workflow reached it.

## 2. Go controls the sequence; roles own bounded artifacts

- **Purpose:** Explain the division of responsibility. Describe the controller validating the brief and target, initializing the run, starting a persistent PM, and then assigning Researcher, Story Editor, Writer, and reviewers in order. Contrast controller-owned state transitions, hashes, timeouts, cleanup, validation, and routing with each role's narrow output contract. Explain that roles work in separate private workspaces with only contracted context staged in; validated regular-file outputs are copied into the durable run.
- **Supporting evidence:** F2–F5 (S1, S2, S3). Use the concrete outputs from F5: sources and claim ledger, sectioned outline, and one assigned candidate. Avoid implying that agents can advance state themselves.
- **Reader takeaway:** Determinism comes from a Go controller enforcing the sequence and contracts, while isolated Codex roles contribute narrowly scoped editorial artifacts.

## 3. Review is sequential, lens-specific, and routed through the PM

- **Purpose:** Walk through the four fresh review lenses—Evidence, Story, Clarity, and Copy—and their sequential operation. Explain that reviewers do not edit the candidate and receive lens-specific context. After each reached lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go validates and enforces that decision. Show the core loop: a validated must-fix ends review of that candidate, sends the next candidate back to Evidence, and leaves optional or invalid findings without consuming candidate budget.
- **Supporting evidence:** F6–F8 (S2, S3); F11 (S2, S3, S4). A small linear sequence may be used: candidate → Evidence → PM → Story → PM → Clarity → PM → Copy → PM. Include the early exit from any PM step on a validated must-fix.
- **Reader takeaway:** Review is a controlled series of independent checks, and revision happens for a recorded, validated reason rather than through an opaque conversation among agents.

## 4. Artifact gates make each transition inspectable

- **Purpose:** Explain why files, not completion messages or clean process exits, determine progress. Describe artifact-specific validation and the bindings that reject stale or mismatched work: reviews identify the exact lens and candidate SHA-256 revision, while PM decisions identify the current revision, request, and review digest. Connect these checks to the durable history of earlier candidates, partial review sequences, reviews, decisions, and `workflow.json` state. Clearly label the claim that these gates reduce silent advancement from stale or incomplete work as an inference, not a measured reliability result.
- **Supporting evidence:** F3, F10–F13 (S1–S4); I1–I3 as reasoned synthesis with their stated caveats. Mention that `workflow.json` is atomically rewritten as the controller's source of truth and records status, phase, current candidate/revision, active role, artifact paths, review count, timestamps, and a block reason when applicable.
- **Reader takeaway:** An engineer can inspect what was produced, which exact candidate it refers to, what decision followed, and where the controller says the run currently stands.

## 5. The candidate budget turns failure into an explicit outcome

- **Purpose:** Describe the bounded revision policy and terminal states. State that candidate 003 is the hard limit: another validated must-fix at that point blocks the run instead of creating candidate 004. Note that human-judgment decisions, malformed or stale artifacts, timeouts, cleanup failures, and exhausted budget can also block with an actionable reason. Explain the design consequence—carefully as inference—that the cap replaces open-ended automated rewriting with a visible handoff point for a new human-directed run.
- **Supporting evidence:** F9, F16, F17 (S1, S2, S3); I4 as an explicitly framed inference. Do not speculate about how often any blocked condition occurs (U1) or about runtime and cost (U2).
- **Reader takeaway:** The workflow fails closed: when automation cannot safely continue within three candidates or needs judgment, it preserves the record and reports a blocked state.

## 6. What success preserves—and what the design does not claim

- **Purpose:** Close by defining successful completion: only after all four lenses on the final candidate pass PM routing does the controller publish `article.md`, byte-for-byte identical to the accepted candidate. Summarize the retained `.control/` audit copies of assignments, logs, and natural exit markers, alongside cleaned-up live workspaces. Reaffirm the practical tradeoff for the target audience without presenting it as fact: this documented design favors auditability and bounded control; it does not establish speed, accuracy gains, implementation conformance, or superiority over other workflows.
- **Supporting evidence:** F14–F17 (S1, S2, S4); O1 only if clearly labeled as editorial judgment. Use U1–U4 to bound the conclusion and avoid unsupported operational or comparative claims.
- **Reader takeaway:** A successful run ends with both an accepted article and enough linked artifacts to reconstruct its path, while the documentation leaves performance and comparative effectiveness unanswered.

## Writing guidance

- Keep the finished article below 900 words; target roughly 700–800 words across the six sections.
- Use plain language and define “candidate,” “lens,” “artifact gate,” and “PM” on first use.
- Prefer a chronological explanation, with the artifact-gate section supplying the rationale for the mechanics already introduced.
- Attribute behavior to the repository documentation; do not imply firsthand execution or code verification.
- Stay within the repository workflow. Exclude publishing integrations, web interfaces, and comparisons with external editorial systems.
