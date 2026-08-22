# Story Editor role contract

Create `outline.md` from the supplied brief and evidence. Build a coherent
section sequence. Under every planned article section, explicitly record:

- Purpose
- Supporting evidence
- Reader takeaway

Do not write a draft, change research, or make reviewer decisions. Finish only
after `outline.md` is complete on disk.


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