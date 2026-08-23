# Outline: How write-uuter turns a brief into an inspectable reviewed article

Target length: 700–850 words. Keep explanations in plain language and distinguish documented behavior from interpretation. Do not discuss publishing integrations, web interfaces, or workflows outside this repository.

## 1. An editorial workflow built around durable files

- **Purpose:** Answer the question directly and establish the article's central idea: write-uuter is a Go CLI whose controller coordinates specialized Codex roles while preserving the work products needed to inspect the run. Introduce “artifact-driven” in plain language as progress represented by validated files rather than by chat history.
- **Supporting evidence:** F1 (S1) establishes the CLI and preserved evidence, outline, candidates, reviews, and PM decisions. F10 and F15 (S2, S4) establish that completion depends on validated owned files, not a final message, transcript, or terminal scrollback. I1 may be used as explicitly interpretive framing: these durable files make routing and outcomes inspectable.
- **Reader takeaway:** The workflow is not a single prompt that returns prose; it is a controlled sequence whose intermediate and terminal outputs remain available for examination.

## 2. The controller sets the boundaries before writing begins

- **Purpose:** Explain the division of responsibility that makes the sequence deterministic. Cover brief validation and atomic run creation, then distinguish controller-owned state and validation from role-owned editorial output in isolated workspaces.
- **Supporting evidence:** F2 (S1, S2) supports required brief sections, a new target, and atomic initialization. F3 (S1–S3) supports Go ownership of state transitions, validation, revision hashes, timeouts, routing, and cleanup, while agents write only their assigned outputs. F14 (S4) supports `workflow.json` as the atomically rewritten source of truth. Optionally note the documented single-run, non-resumable limit from F16 (S2, S3), but omit platform/container details unless space permits because they are secondary to the brief.
- **Reader takeaway:** Go decides whether the workflow may advance; Codex roles contribute bounded artifacts rather than controlling the run themselves.

## 3. Research, outline, and candidate 001 form the production chain

- **Purpose:** Describe the first sequential handoff without overloading the article with role detail: the Researcher prepares evidence and a claim ledger, the Story Editor plans the structure, and the Writer produces the first candidate from the accumulated inputs. Emphasize separate ownership rather than portraying the agents as one continuing conversation.
- **Supporting evidence:** F4 (S2, S3) gives the Researcher → Story Editor → Writer order and the Writer's inputs: brief, evidence, ledger, and outline. F3 (S1–S3) supports isolated role workspaces and role-owned output. I2 may support a carefully qualified explanation that ownership separation helps keep evaluation and authorship attributable; do not state an empirically proven quality benefit.
- **Reader takeaway:** Each stage leaves an inspectable input for the next, so readers can trace how the brief and evidence became a candidate article.

## 4. Four fresh review lenses run as a gated sequence

- **Purpose:** Walk through the review order—Evidence, Story, Clarity, Copy—and explain what “fresh” and “sequential” mean. State that each reviewer sees the exact candidate and revision plus its durable lens instructions and permitted lens-specific context, writes a review artifact, and does not edit the candidate.
- **Supporting evidence:** F5 (S1–S3) establishes the fixed order and fresh Codex process for each lens. F6 (S3) establishes reviewer inputs, lens-specific context, and the prohibition on candidate edits. F10–F11 (S2–S4) establish schema validation and binding reviews to an allowed lens and the exact SHA-256 candidate revision.
- **Reader takeaway:** Review is a reproducible series of independent checks, not a shared discussion in which reviewers silently rewrite the text or inherit one another's conversations.

## 5. The PM classifies findings; Go applies the routing rules

- **Purpose:** Explain the decision loop precisely. The persistent PM classifies each finding, while the controller independently validates the decision and determines what happens next. Define the four classifications in plain language and show the routing consequences.
- **Supporting evidence:** F7 (S2, S3) supports `valid_must_fix`, `valid_optional`, `invalid` with a reason, and `needs_human_judgment`, plus controller validation. F11 (S2–S4) supports binding a decision to the revision, request ID, and review digest. I3 may be framed as an inference: those bindings allow stale or mismatched decisions to be rejected.
- **Reader takeaway:** The PM supplies editorial judgment within a constrained protocol, but validated artifacts—not conversational assertions—drive controller state changes.

## 6. Must-fix findings restart review, but the loop has a hard ceiling

- **Purpose:** Make the revision logic and candidate budget concrete. A validated must-fix stops the remaining lenses, produces the next candidate if budget remains, and restarts review at Evidence. Optional or invalid findings do not consume a candidate; a need for human judgment blocks immediately. Candidate 003 is the final allowed attempt.
- **Supporting evidence:** F8 (S2) supports early stopping, restart at Evidence, non-consuming optional/invalid findings, and blocking for human judgment. F9 (S1, S2) supports candidate 003 as the ceiling and blocking rather than creating candidate 004. I4 can provide the section's interpretation: the ceiling turns revision into a deterministic terminal choice instead of an open-ended loop.
- **Reader takeaway:** The system permits attributable revision without allowing endless automated rewriting: by the third candidate, the run must either pass or preserve a blocked outcome.

## 7. Success and blockage are both inspectable terminal states

- **Purpose:** Close by showing why the artifact gates matter. On success, `article.md` is an exact copy of the candidate accepted through all four lenses. On revision or blockage, earlier candidates, partial lens sequences, reviews, and PM decisions remain. Explain that `workflow.json` records current and terminal state and `.control/` retains audit materials after cleanup. Briefly acknowledge the evidence boundary: these are documented contracts, not independently observed runtime results.
- **Supporting evidence:** F12 (S1, S4) supports the non-empty, byte-for-byte identity of `article.md` and the accepted candidate. F13 (S4) supports retention of earlier candidates, partial reviews, reviews, and decisions. F14–F15 (S3, S4) support `workflow.json`, block reasons, assignment/log/exit-marker audit copies, and independence from transcripts. The absence of firsthand observations and U1 require wording such as “the documentation specifies” rather than a claim that this research executed and verified the CLI. O2 may inform restrained closing framing, but should not be presented as fact.
- **Reader takeaway:** A completed run yields either an exactly identified accepted article or a preserved explanation of why work stopped; in both cases, the decisive evidence is on disk and available for inspection.
