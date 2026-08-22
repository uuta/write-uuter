# Writer role contract

Write only the assigned versioned candidate under `drafts/`. Expand the
supplied outline into publishable prose supported by the supplied evidence and
brief. Do not leave TODOs or unresolved placeholders.

For a revision, apply every PM-validated must-fix decision using the prior
candidate and the reached review result/report as input. Use the matching
finding's problem, location, and suggested direction to make the correction,
then verify that the revised wording actually resolves it. Do not accept or
reject findings yourself. Never edit a review result, PM decision, earlier
draft, or final `article.md`. Finish only after the assigned candidate is
complete on disk.


## Assignment

Write candidate 003 to `drafts/article-003.md` in this isolated workspace.

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

## Provided context: evidence/sources.md

<write-uuter-context name="evidence/sources.md">
# Sources

Access date for all sources: 2026-08-22. All locations below are controller-staged copies of the brief's local source hints; no source-repository files were read.

## 1. README

- Location: `context/source-hints/001-README.md`
- Original hint: `../README.md`
- Useful summary: Defines `write-uuter` as a Go CLI that converts a Markdown brief into a reviewed article while retaining evidence, outline, candidates, reviews, and PM decisions. Describes runtime ownership: Go controls state transitions, validation, hashes, timeouts, and cleanup; roles receive isolated workspaces and contracted context. States the sequential role/reviewer order, the four review lenses, the hard limit of candidate 003, and success/blocked behavior.
- Particularly useful passages: lines 3-5 (purpose); 43-48 (terminal behavior); 74-94 (runtime model, isolation, sequence, and candidate limit).

## 2. Workflow documentation

- Location: `context/source-hints/002-workflow.md`
- Original hint: `../docs/workflow.md`
- Useful summary: Gives the controller sequence from brief validation and atomic run initialization through research, outlining, drafting, four sequential review lenses, and final publication. Explains that a PM-validated must-fix stops the remaining lenses and restarts the next candidate at Evidence; optional or invalid findings do not consume a candidate; human judgment blocks. Defines artifact gates and terminal revalidation/cleanup behavior.
- Particularly useful passages: lines 5-10 (initialization); 12-39 (full sequence); 41-43 (routing and candidate consumption); 45-62 (artifact gates); 90-112 (timeouts, terminal validation, blocked-state preservation, and implementation limits).

## 3. Roles documentation

- Location: `context/source-hints/003-roles.md`
- Original hint: `../docs/roles.md`
- Useful summary: Assigns ownership and boundaries to the Human Editor, PM, Researcher, Story Editor, Writer, and reviewers. The PM classifies findings but does not write candidates or reviews. The Writer creates exactly one assigned candidate and cannot create the terminal article. Fresh Evidence, Story, Clarity, and Copy reviewers run sequentially, cannot edit candidates, and receive only lens-specific inputs.
- Particularly useful passages: lines 14-18 (human); 20-38 (PM); 40-58 (research, outline, and writing); 60-84 (review order, inputs, outputs, and isolation).

## 4. Artifacts documentation

- Location: `context/source-hints/004-artifacts.md`
- Original hint: `../docs/artifacts.md`
- Useful summary: Specifies the durable run layout and validation contracts for review results, PM decisions, `workflow.json`, and `.control/` audit copies. Earlier candidates, partial review sequences, reviews, and PM decisions remain available after revision or blockage. `article.md` exists only on success and exactly matches the accepted candidate. `workflow.json` is the atomically rewritten controller source of truth and records status, phase, current candidate/revision, paths, attempts, timestamps, and block reason.
- Particularly useful passages: lines 3-33 (run tree and retained artifacts); 35-63 (review result contract); 65-100 (PM decision contract); 102-129 (`workflow.json`, audit artifacts, and independence from chat/tmux scrollback).

## Source-boundary note

These documents describe the repository's implemented workflow and are sufficient for the scoped article. They do not support claims that the design is generally superior to other editorial systems, works outside the documented macOS real-run environment, supports parallel/resumable runs, or includes publishing/web integrations.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

The ledger uses all five required classifications: **Fact**, **Firsthand observation**, **Inference**, **Opinion**, and **Unresolved**. No firsthand work was performed, so no `evidence/firsthand.md` was created.

| ID | Classification | Claim | Support / handling |
| --- | --- | --- | --- |
| C01 | Fact | `write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | `context/source-hints/001-README.md`, lines 3-5. |
| C02 | Fact | Before running roles, the controller parses every required level-two brief section, verifies the target does not exist, initializes in a temporary sibling directory, and commits it with a no-replace rename. | `context/source-hints/002-workflow.md`, lines 5-10. |
| C03 | Fact | The role sequence is Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewers, with the review lenses run sequentially rather than in parallel. | `context/source-hints/001-README.md`, lines 91-94; `context/source-hints/002-workflow.md`, lines 12-43. |
| C04 | Fact | Go, not an agent's final message or process exit alone, decides advancement by checking successful exit plus the existence and validity of role-owned files. | `context/source-hints/002-workflow.md`, lines 45-62. |
| C05 | Fact | The Researcher owns sources and the claim ledger (plus optional firsthand evidence/assets); the Story Editor owns the outline; the Writer owns exactly one assigned candidate. | `context/source-hints/003-roles.md`, lines 40-58. |
| C06 | Fact | The PM is a persistent isolated Codex process that classifies each reviewer finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go independently validates and applies routing. | `context/source-hints/003-roles.md`, lines 20-38. |
| C07 | Fact | A PM-validated must-fix ends review of that candidate; if the candidate budget remains, the Writer creates the next candidate and review restarts at Evidence. Optional and invalid findings do not consume a candidate, while a human-judgment decision blocks the run. | `context/source-hints/002-workflow.md`, lines 28-43. |
| C08 | Fact | Candidate 003 is the hard limit. Exhausting it blocks the run rather than creating a fourth candidate. | `context/source-hints/001-README.md`, lines 43-47 and 91-94; `context/source-hints/002-workflow.md`, lines 28-38 and 93-96. |
| C09 | Fact | Each reviewer is a fresh process, cannot edit the candidate, and receives the brief, exact candidate/revision, its lens prompt, and only the additional context specified for that lens. | `context/source-hints/003-roles.md`, lines 60-84. |
| C10 | Fact | Review outputs consist of a validated `result.json` and matching `report.md`; the JSON binds the lens and candidate SHA-256 revision, and each report reproduces the complete fields of each finding in order. | `context/source-hints/003-roles.md`, lines 74-76; `context/source-hints/004-artifacts.md`, lines 35-63. |
| C11 | Fact | PM decision files bind decisions to a candidate revision, request ID, and review digest, cover every finding exactly once, retain earlier reached lenses, and reject stale, missing, duplicate, future, or altered routing data. | `context/source-hints/003-roles.md`, lines 31-38; `context/source-hints/004-artifacts.md`, lines 65-100. |
| C12 | Fact | Earlier candidates, partial lens sequences, reviews, and PM decisions are retained when revision occurs or a run blocks. | `context/source-hints/004-artifacts.md`, lines 30-33. |
| C13 | Fact | `workflow.json` is atomically rewritten as the controller's source of truth and records status, phase, candidate/revision, active role, stable artifact paths, review-attempt count, timestamps, and a terminal block reason when blocked. | `context/source-hints/004-artifacts.md`, lines 102-114. |
| C14 | Fact | On success, `article.md` is written only after all four lenses pass PM routing and is byte-for-byte identical to the accepted candidate. | `context/source-hints/004-artifacts.md`, lines 30-33; `context/source-hints/002-workflow.md`, lines 96-103. |
| C15 | Fact | Durable `.control/` audit copies include generated assignments, invocation logs, and exit markers; editorial completion does not depend on chat transcripts or tmux scrollback. | `context/source-hints/004-artifacts.md`, lines 116-129. |
| C16 | Fact | Real runs currently require macOS native isolation; parallel runs, resume after controller restart, and editing completed runs are not implemented. | `context/source-hints/001-README.md`, lines 9-20; `context/source-hints/002-workflow.md`, lines 111-112. |
| C17 | Inference | The workflow is inspectable because major inputs, intermediate candidates, reviews, PM classifications, controller state, and terminal output are represented as durable files with cross-checked identities and revisions. | Synthesis of C01, C04, C10-C15. Phrase as an explanation derived from documented mechanics, not as a measured quality claim. |
| C18 | Inference | Artifact gates reduce the chance that a stale review, malformed response, or agent assertion silently advances the workflow. | Supported mechanism: `context/source-hints/002-workflow.md`, lines 45-62, and `context/source-hints/004-artifacts.md`, lines 56-63 and 94-100. “Reduce the chance” is an inference; no comparative failure-rate data is supplied. |
| C19 | Inference | The three-candidate cap makes revision effort bounded and converts unresolved must-fix work into an inspectable blocked outcome. | Derived from C08, C12, and C13. Do not claim the cap guarantees article quality. |
| C20 | Firsthand observation | No firsthand execution, test run, or direct artifact inspection beyond the supplied documentation copies was performed for this research assignment. | Method note; therefore `evidence/firsthand.md` is intentionally absent. |
| C21 | Opinion | Calling the workflow “small,” “clear,” or “useful” is evaluative unless tied to a defined criterion; such wording should be framed as the author's judgment, not repository fact. | Editorial caution; the supplied sources do not measure these qualities. |
| C22 | Unresolved | Whether the workflow improves correctness, editorial quality, speed, or cost compared with another workflow is unresolved. | No benchmarks, comparative study, or outcome measurements appear in the allowed sources; keep out of factual claims. |
| C23 | Unresolved | Behavior outside this repository's documented implementation—especially publishing integrations, web interfaces, parallel execution, and resumability—is not established for the scoped article. | Brief marks publishing/web integrations out of scope; `context/source-hints/002-workflow.md`, lines 111-112, says parallel runs and resume are not implemented. |

## Recommended factual spine

For a concise technical explanation, prioritize C01, C03-C08, C10-C15, then use C17-C19 only with inferential wording. Treat C21-C23 as guardrails rather than affirmative article claims.

</write-uuter-context>

## Provided context: outline.md

<write-uuter-context name="outline.md">
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

</write-uuter-context>

## Provided context: drafts/article-002.md

<write-uuter-context name="drafts/article-002.md">
# From Brief to Inspectable Reviewed Article

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article through a sequence of isolated editorial roles. Along the way, it retains the evidence, outline, article candidates, reviews, and product-manager decisions that produced the result. The Go controller—not an agent’s final message—decides when the workflow may advance. It does so through artifact gates: checks that require a role-owned file to exist and satisfy a defined contract.

That division of labor makes the run inspectable. Codex roles perform bounded editorial tasks, while deterministic controller code manages state transitions, validation, hashes, timeouts, and cleanup. Progress is therefore represented by durable files rather than only by a conversation or a successful process exit.

## From a validated brief to a candidate

Before invoking any editorial role, the controller parses the brief’s required level-two sections and verifies that the target run does not already exist. It initializes the run in a temporary sibling directory, then commits that directory with a rename that will not replace an existing target.

The editorial sequence begins with three distinct owners. The Researcher creates the source record and claim ledger, with optional firsthand evidence or assets when the assignment calls for them. The Story Editor turns that material into an outline. The Writer expands the outline into exactly one assigned, versioned candidate. The Writer does not create the terminal `article.md`.

These boundaries leave a trace from source material to structure to prose. Each role receives a contracted context in an isolated workspace and owns a specific output, so later inspection can identify which artifact entered and left each stage.

## Four review lenses, applied in order

Every candidate is reviewed through four sequential lenses: Evidence, Story, Clarity, and Copy. Each reviewer is a fresh process, receives the exact candidate revision plus lens-specific context, and cannot edit the candidate. Instead, the reviewer writes a machine-readable `result.json` and a matching human-readable `report.md`.

The review result is tied to both its lens and the candidate’s SHA-256 digest. Its report must reproduce every finding’s complete fields in the same order. These requirements let the controller check that the readable explanation and structured routing data refer to the same review of the same text.

A persistent, isolated PM process classifies every finding as a valid must-fix, valid optional issue, invalid finding with a reason, or a matter needing human judgment. The PM neither writes candidates nor performs reviews. Its decision files bind classifications to the candidate revision, request ID, and review digest and must cover each finding exactly once. Go independently validates those files and applies the route, rejecting stale, missing, duplicate, future, or altered routing data.

The order of the lenses has a practical consequence. A PM-validated must-fix stops the remaining reviews for that candidate. Optional or invalid findings allow the sequence to continue without consuming another candidate. A finding classified as needing human judgment blocks the run rather than letting the controller invent an answer.

## Revision has a hard budget

When a must-fix is validated and budget remains, the Writer creates the next versioned candidate. Review then restarts at Evidence and proceeds through the lenses in order unless another PM-validated must-fix stops the remaining reviews for that candidate. Candidate 003 is the hard ceiling. If that candidate still receives a validated must-fix, the run blocks instead of creating candidate 004.

This cap does not guarantee article quality. It does make revision effort bounded and turns unresolved must-fix work into an explicit blocked outcome. Earlier candidates, partial lens sequences, reviews, and PM decisions remain on disk, so revision does not erase the path that led to the terminal state.

## Why the artifact gates matter

A role’s successful exit is necessary but insufficient. Before advancing, the controller checks the existence and validity of required files, including schemas, identities, hashes, and revision bindings. Review artifacts connect findings to an exact candidate digest; PM artifacts connect routing decisions to the corresponding request and review. The atomically rewritten `workflow.json` is the controller’s source of truth, recording status, phase, current candidate and revision, active role, stable artifact paths, review-attempt count, timestamps, and any terminal block reason.

The `.control/` directory adds audit copies of generated assignments, invocation logs, and exit markers. Editorial completion therefore does not depend on chat transcripts or terminal scrollback. Taken together, these cross-checks reduce the chance that stale data, malformed output, or an agent’s unsupported assertion silently advances the run. That is an inference from the documented mechanics, not a measured claim about quality, speed, cost, or correctness.

## Success or preserved blockage

On success, `article.md` is created only after all four lenses pass PM routing. It is byte-for-byte identical to the accepted candidate. On blockage, the retained artifacts and `workflow.json` block reason show where and why the run stopped. Either outcome is available for inspection.

The scope is deliberately limited to the repository’s implemented workflow. Publishing integrations and web interfaces are outside it. Real runs currently require native macOS isolation, while parallel runs, resume after a controller restart, and editing completed runs are not implemented. Within those limits, `write-uuter` demonstrates how a deterministic controller can coordinate isolated editorial roles while preserving the evidence and decisions behind the final—or blocked—result.

</write-uuter-context>

## Provided context: pm-decisions/article-002.md

<write-uuter-context name="pm-decisions/article-002.md">
```json
{
  "reviewed_revision": "sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab",
  "lenses": {
    "evidence": {
      "request_id": "35ee95e2066dc5aa16773a5e83de4118",
      "review_digest": "sha256:bfc8386d12ba8173d0f32bba26a4c58fe0714c5ca9360e33315aa43dec5f6aa5",
      "decisions": []
    },
    "story": {
      "request_id": "be4cbe88cefad07aa55e470fb5927016",
      "review_digest": "sha256:7098c9db30d3d3616072af2087eac1e263d99a241e25667c96b39003f8c2df14",
      "decisions": []
    },
    "clarity": {
      "request_id": "fc37c93a5e9a381481fbf63949f6575d",
      "review_digest": "sha256:0c66a0c94b0d4c5e35f930b23f913dc0beba7b4b139e683ac6d6449cfb00c12c",
      "decisions": []
    },
    "copy": {
      "request_id": "fef5800c138dd40c2a5ba2f416e9659d",
      "review_digest": "sha256:490241e8a2bbcee1e6371df1427b6f1f91b9dccb8517eb2659e409a0bc9851d5",
      "decisions": [
        {
          "finding_id": "copy-001",
          "decision": "valid_must_fix",
          "reason": "The coordinated list mixes noun and verb forms, creating a genuine grammatical defect; making the three items parallel is required for publication-ready copy."
        }
      ]
    }
  }
}
```

</write-uuter-context>

## Provided context: reviews/article-002/evidence/result.json

<write-uuter-context name="reviews/article-002/evidence/result.json">
{
  "status": "clean",
  "lens": "evidence",
  "reviewed_revision": "sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab",
  "findings": []
}

</write-uuter-context>

## Provided context: reviews/article-002/evidence/report.md

<write-uuter-context name="reviews/article-002/evidence/report.md">
# Evidence Review

status: clean

lens: evidence

reviewed_revision: sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab

findings: []

</write-uuter-context>

## Provided context: reviews/article-002/story/result.json

<write-uuter-context name="reviews/article-002/story/result.json">
{
  "status": "clean",
  "lens": "story",
  "reviewed_revision": "sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab",
  "findings": []
}

</write-uuter-context>

## Provided context: reviews/article-002/story/report.md

<write-uuter-context name="reviews/article-002/story/report.md">
# Story Review

status: clean

lens: story

reviewed_revision: sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab

findings: []

</write-uuter-context>

## Provided context: reviews/article-002/clarity/result.json

<write-uuter-context name="reviews/article-002/clarity/result.json">
{
  "status": "clean",
  "lens": "clarity",
  "reviewed_revision": "sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab",
  "findings": []
}

</write-uuter-context>

## Provided context: reviews/article-002/clarity/report.md

<write-uuter-context name="reviews/article-002/clarity/report.md">
# Clarity Review

status: clean

lens: clarity

reviewed_revision: sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab

findings: []

</write-uuter-context>

## Provided context: reviews/article-002/copy/result.json

<write-uuter-context name="reviews/article-002/copy/result.json">
{
  "status": "fix_required",
  "lens": "copy",
  "reviewed_revision": "sha256:767c2e26620d33a46d4dca80e52b5468022f5b2150450ca82a866c8e7c40f4ab",
  "findings": [
    {
      "id": "copy-001",
      "severity": "minor",
      "location": "Final paragraph, sentence beginning \"Real runs currently require\"",
      "problem": "The list is not grammatically parallel: the noun phrase \"parallel runs\" and the gerund phrase \"editing completed runs\" are paired with the verb phrase \"resume after a controller restart.\"",
      "suggested_direction": "Change \"resume after a controller restart\" to \"resuming after a controller restart\" so all three listed items function as noun phrases."
    }
  ]
}

</write-uuter-context>

## Provided context: reviews/article-002/copy/report.md

<write-uuter-context name="reviews/article-002/copy/report.md">
# Copy Review

id: copy-001

severity: minor

location: Final paragraph, sentence beginning "Real runs currently require"

problem: The list is not grammatically parallel: the noun phrase "parallel runs" and the gerund phrase "editing completed runs" are paired with the verb phrase "resume after a controller restart."

suggested_direction: Change "resume after a controller restart" to "resuming after a controller restart" so all three listed items function as noun phrases.

</write-uuter-context>