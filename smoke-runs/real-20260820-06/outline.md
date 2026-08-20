# Outline: From Brief to Inspectable Reviewed Article

## Article direction

- **Audience:** Engineers evaluating a small, artifact-driven editorial workflow.
- **Central answer:** write-uuter uses a Go controller to enforce workflow state, validation, routing, and cleanup while isolated Codex roles produce research, an outline, article candidates, reviews, and decisions. Progress depends on validated files, leaving the accepted path—or the reason for blocking—available for inspection.
- **Target length:** 750–850 words, leaving margin below the 900-word limit.
- **Terminology:** Introduce a *candidate* as one version of the article and an *artifact gate* as a required file whose contents the controller validates before advancing.

## 1. A controller around editorial work

- **Purpose:** Open with the system’s promise and establish the crucial boundary: write-uuter is a Go CLI, but the controller—not the model output—provides deterministic transitions and checks. Briefly note that it validates the brief and safely creates a new run target before editorial work begins.
- **Supporting evidence:** C01 (S1, lines 3–5); C02 (S1, lines 63–73; S3, lines 3–7); C03 (S1, lines 36–41 and 52–61; S2, lines 5–10).
- **Reader takeaway:** The workflow is not a single agent asked to “write and review.” A deterministic program coordinates model-driven editorial work and refuses to begin with an invalid brief or unsafe target.

## 2. Each role owns a durable handoff

- **Purpose:** Walk through the production sequence: Human Editor supplies the brief; Researcher produces sources and a claim ledger; Story Editor produces this structured outline; Writer creates exactly one assigned candidate; fresh reviewers later examine it. Explain that a persistent PM and at most one worker run sequentially, each worker receiving only contracted inputs in a private workspace outside the durable run directory.
- **Supporting evidence:** C04 (S1, lines 65–71; S2, lines 65–71; S3, lines 50–69); C05 (S1, lines 69–73; S2, lines 12–39; S3, lines 50–64); C06 (S3, lines 9–30); C07 (S2, lines 51–54; S3, lines 32–48).
- **Reader takeaway:** Ownership is explicit: roles create their assigned artifacts rather than sharing one mutable conversation or editing one another’s work. Isolation also limits the context carried from one review lens to the next.

## 3. Files are gates, not merely logs

- **Purpose:** Explain the mechanism that makes the workflow inspectable. A successful process exit or completion message is insufficient; the expected owned files must exist and pass schema and consistency checks. Show this concretely with revision-bound reviews and PM decisions that must correspond to every finding and the active request.
- **Supporting evidence:** C10 (S2, lines 45–61); C11 (S2, lines 54–61; S4, lines 35–59); C12 (S2, lines 57–58; S3, lines 26–30; S4, lines 61–92). Use C20 only as a clearly signaled inference from C10–C17.
- **Reader takeaway:** The controller advances on validated, revision-specific evidence, not on transient agent chat. That rejects stale or internally inconsistent review material and leaves concrete handoffs a reader can inspect.

## 4. Sequential review makes routing explicit

- **Purpose:** Trace the four lenses in order—Evidence, Story, Clarity, Copy—and the PM classification after each reached lens. Describe all routes: `valid_optional` and `invalid` continue without another candidate; `needs_human_judgment` blocks for the Human Editor; `valid_must_fix` ends the remaining lenses and starts the next candidate again at Evidence. State the hard limit of candidate 003.
- **Supporting evidence:** C05 (S1, lines 69–73; S2, lines 12–39; S3, lines 50–64); C06 (S3, lines 9–30); C08 (S2, lines 41–43); C09 (S1, lines 36–40 and 71–73; S2, lines 28–38 and 73–81). C22 may support a qualified interpretation of the editorial loop as bounded, while acknowledging that operational failures can also block.
- **Reader takeaway:** Reviews are intentionally serial, and only a PM-validated must-fix spends one of the three available candidates. The budget turns editorial revision into a bounded route rather than an open-ended loop.

## 5. Success and failure both leave inspectable state

- **Purpose:** Describe what remains at the end. Earlier candidates, partial review sequences, reviews, and PM decisions survive revision or blocking. `workflow.json` records current and terminal state. On success, the controller revalidates the accepted candidate and bindings before copying it byte-for-byte to `article.md`; on failure, it records an actionable block reason. Mention retained prompt/log/exit audit copies and verified tmux cleanup without confusing them with live control files.
- **Supporting evidence:** C13 (S4, lines 30–33); C14 (S4, lines 94–106); C15 (S1, lines 36–38; S2, lines 73–81; S4, lines 30–33); C16 (S3, lines 3–7; S4, lines 108–115); C17 (S2, lines 73–81; S4, lines 99–106).
- **Reader takeaway:** A completed run exposes both the article and the checked path that led to it; a blocked run exposes preserved intermediate work and a machine-readable reason instead of erasing the failed path.

## 6. Why the artifact gates matter—and what they do not prove

- **Purpose:** Conclude by answering the brief directly. Frame auditability as an engineering evaluation, not a measured advantage: durable, validated artifacts make the controller’s accepted path easier to reconstruct. Delimit the implementation: no parallel runs, restart resume, or editing of completed runs, and no evidence here of superior quality, speed, cost, reliability, or real-world success rates.
- **Supporting evidence:** C18 (S2, lines 83–84; S3, lines 9–13); C20 (inference derived from C10–C17); C23 (explicitly signaled opinion grounded in C10–C17). Respect the unresolved boundaries in C24–C26 and the lack of firsthand execution recorded in C19.
- **Reader takeaway:** write-uuter’s documented value is inspectability and bounded control within this repository. The documentation supports how the workflow operates, not comparative performance claims about editorial systems in general.

## Drafting guardrails

- Keep the explanation in workflow order and use plain-language definitions before filenames or status values.
- Attribute determinism only to controller-owned transitions, validation, hashes, routing, timeouts, and cleanup; never to Codex output as a whole.
- Present C20 and C22 as interpretations and C23 as an evaluation, not as measured facts.
- Do not claim firsthand verification, benchmarking, comparative superiority, production reliability, publishing support, or web-interface support.
- Preserve the distinction between durable audit artifacts in the run directory and transient live workspaces or control files.
