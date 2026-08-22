# Outline: From Brief to Inspectable Reviewed Article

Target length: 700–850 words. Keep the explanation specific to the repository, define workflow terms in plain language, and distinguish documented facts from inferences about inspectability.

## 1. The workflow in one view

- **Purpose:** Answer the article's central question up front: `write-uuter` is a Go CLI whose controller moves a Markdown brief through isolated editorial roles and retains the artifacts that explain the resulting article. Introduce an "artifact gate" as a controller check that requires a role-owned file to exist and satisfy its contract before the workflow advances.
- **Supporting evidence:** C01 (README, lines 3–5); C03 (README, lines 91–94; workflow documentation, lines 12–43); C04 (workflow documentation, lines 45–62). Use C17 only as an explicitly reasoned synthesis of these documented mechanics.
- **Reader takeaway:** The system is not a single opaque generation step: control belongs to deterministic Go code, editorial work belongs to bounded Codex roles, and progress is represented by files that can be inspected.

## 2. From validated brief to candidate

- **Purpose:** Establish the first half of the sequence. Explain that the controller validates required brief sections and safely initializes a run, then invokes the Researcher, Story Editor, and Writer in order. Clarify ownership: research produces sources and a claim ledger, the Story Editor produces the outline, and the Writer produces exactly one assigned candidate rather than the terminal article.
- **Supporting evidence:** C02 (workflow documentation, lines 5–10); C03 (workflow documentation, lines 12–27); C05 (roles documentation, lines 40–58). Mention initialization details only briefly so the article remains centered on the editorial flow.
- **Reader takeaway:** Each stage has one bounded responsibility and a durable output, making it possible to trace how the brief and evidence became a particular candidate.

## 3. Four sequential review lenses and PM routing

- **Purpose:** Describe the review loop precisely. A fresh Evidence, Story, Clarity, and Copy reviewer examines the exact candidate revision in sequence and cannot edit it. Each produces a validated JSON result plus a matching human-readable report. The persistent PM classifies every finding, while Go validates the decision data and applies the route. Explain why sequential order matters operationally: a validated must-fix stops the remaining lenses for that candidate, while optional or invalid findings allow review to continue; a need for human judgment blocks the run.
- **Supporting evidence:** C03 and C07 (workflow documentation, lines 12–43); C06 and C09 (roles documentation, lines 20–38 and 60–84); C10–C11 (artifacts documentation, lines 35–100). Avoid suggesting that the PM writes reviews or candidates.
- **Reader takeaway:** Review is a controlled series of lens-specific checks, and the PM interprets findings without being allowed to silently rewrite the work or determine controller state on its own.

## 4. Revision is bounded by a three-candidate budget

- **Purpose:** Explain candidate consumption and termination. A PM-validated must-fix sends the Writer to the next candidate, after which review restarts at Evidence. Optional and invalid findings do not spend the budget. Candidate 003 is the hard ceiling: another must-fix blocks the run instead of creating candidate 004. Earlier candidates, completed or partial reviews, and PM decisions remain available.
- **Supporting evidence:** C07–C08 (README, lines 43–47 and 91–94; workflow documentation, lines 28–43 and 93–96); C12 (artifacts documentation, lines 30–33). C19 may frame the consequence as an inference: the cap bounds revision effort and turns unresolved work into an inspectable blocked outcome; do not claim that it guarantees quality.
- **Reader takeaway:** The loop can revise, but it cannot continue indefinitely or hide exhaustion; success and blockage are both explicit, reviewable outcomes.

## 5. Artifact gates make the run inspectable

- **Purpose:** Tie the mechanics to the provisional takeaway. Explain that a successful process exit or an agent's final message is insufficient: Go checks required files and their schemas, identities, hashes, and revision bindings. Show the inspection trail: `result.json` and `report.md` bind findings to a candidate digest; PM decisions bind classifications to the request and review; `workflow.json` atomically records current and terminal state; `.control/` retains assignments, logs, and exit markers. Phrase the benefit carefully as an inference: these cross-checks reduce the chance that stale, malformed, or merely asserted results advance unnoticed.
- **Supporting evidence:** C04 and C10–C13 (workflow documentation, lines 45–62; artifacts documentation, lines 35–114); C15 (artifacts documentation, lines 116–129); inferential support C17–C18. Do not claim measured improvements in correctness, quality, speed, or cost (C22).
- **Reader takeaway:** Inspectability comes from durable, cross-linked artifacts and controller validation, not from chat history or trust in an agent's self-report.

## 6. Terminal artifacts: accepted article or preserved blockage

- **Purpose:** Close with the two terminal states. On success, `article.md` appears only after all four lenses pass PM routing and exactly matches the accepted candidate. On blockage, the retained run state and block reason show where and why work stopped. Briefly bound the claim to the implemented repository: publishing and web interfaces are outside scope, and parallel runs, resume after controller restart, and editing completed runs are not implemented.
- **Supporting evidence:** C13–C16 (workflow documentation, lines 96–112; artifacts documentation, lines 30–33 and 102–129); scope guardrail C23. Include limitations compactly rather than opening a broader product comparison.
- **Reader takeaway:** A reader can inspect either the exact candidate promoted to `article.md` or the preserved evidence of a blocked run, while understanding the implementation's documented limits.
