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

Accessed: 2026-08-20

The brief's relative source hints were resolved from
`/Users/yutaaoki/write-uuter/.worktrees/1/examples`, the directory containing
the supplied example brief. All four sources are repository documentation and
fall within the brief's allowed source set.

## S1 — Repository README

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/README.md`
- Repository-relative location: `README.md`
- Access date: 2026-08-20
- Useful summary: Defines write-uuter as a Go CLI that turns a Markdown brief
  into a reviewed article while retaining the evidence, outline, candidates,
  reviews, and PM decisions. It gives the runtime ordering, isolation model,
  success and blocked outcomes, and the hard limit of candidate 003.
- Especially useful passages: lines 3–5 (purpose); 36–41 (terminal behavior);
  52–61 (brief validation and source-hint resolution); 63–73 (controller
  responsibilities, isolated workspaces, sequential roles, and candidate
  budget).

## S2 — Workflow documentation

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/workflow.md`
- Repository-relative location: `docs/workflow.md`
- Access date: 2026-08-20
- Useful summary: Supplies the full controller sequence and routing rules. It
  explains that reviews are sequential, a validated must-fix ends the current
  candidate's remaining lenses and restarts the next candidate at Evidence,
  optional or invalid findings do not spend a candidate, and human judgment
  blocks. It also specifies artifact validation, process lifecycle, cleanup,
  and terminal revalidation.
- Especially useful passages: lines 5–10 (safe initialization); 12–43
  (sequence and routing); 45–61 (artifact gates); 63–81 (isolation, timeouts,
  cleanup, terminal checks); 83–84 (explicitly unimplemented features).

## S3 — Role contracts

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/roles.md`
- Repository-relative location: `docs/roles.md`
- Access date: 2026-08-20
- Useful summary: Assigns ownership and boundaries to the Human Editor, PM,
  Researcher, Story Editor, Writer, and four reviewers. It details PM finding
  classifications, the inputs and outputs of each role, and the deliberately
  limited context supplied to fresh reviewer processes.
- Especially useful passages: lines 3–7 (controller-owned transitions and
  durable prompts); 9–30 (human and PM duties); 32–48 (research, outline, and
  writing ownership); 50–69 (review order, inputs, outputs, and isolation).

## S4 — Artifact contracts

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/artifacts.md`
- Repository-relative location: `docs/artifacts.md`
- Access date: 2026-08-20
- Useful summary: Defines the inspectable run-directory layout and the
  validated formats for reviewer results, PM decisions, and workflow state. It
  states which intermediate artifacts remain after revision or blocking and
  that a successful `article.md` is byte-for-byte identical to the accepted
  candidate.
- Especially useful passages: lines 3–33 (layout, preservation, exact-copy
  publication); 35–59 (review result contract); 61–92 (accumulating PM
  decision contract); 94–115 (`workflow.json`, `.control/`, and independence
  from transient tmux/chat state).

## Source limitations

- These files describe the repository's shipped contracts; this research did
  not independently execute or benchmark the CLI.
- The sources do not establish comparative quality, productivity, or cost
  advantages over other editorial workflows.
- No external sources were consulted, in keeping with the brief's constraint
  to use only `README.md` and `docs/`.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

This ledger uses all five required classifications. `Fact` means directly
stated in the allowed repository sources. `Firsthand observation` is reserved
for behavior or output directly observed during an execution or inspection
beyond reading the supplied sources. `Inference` is a bounded conclusion drawn
from documented facts. `Opinion` is a value judgment that should be signaled
as such. `Unresolved` marks a question the supplied sources do not answer.

No firsthand execution was performed, so no `evidence/firsthand.md` or assets
were created.

| ID | Classification | Claim | Support or boundary | Safe use |
| --- | --- | --- | --- | --- |
| C01 | Fact | write-uuter is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | S1, lines 3–5. | Core description. |
| C02 | Fact | Go owns workflow state transitions, validation, revision hashes, timeouts, process cleanup, and routing, while Codex processes perform the editorial roles. | S1, lines 63–73; S3, lines 3–7. | Explain the deterministic controller/agent division. Avoid claiming that model output itself is deterministic. |
| C03 | Fact | The controller validates required brief sections and a new target, initializes through a temporary sibling directory, and uses a no-replace rename so it does not overwrite a concurrently created directory or symlink. | S1, lines 36–41 and 52–61; S2, lines 5–10. | Explain entry validation and safe initialization. |
| C04 | Fact | One persistent PM and at most one worker use a dedicated tmux session; each role works in a fresh private workspace outside the durable run directory, with only contracted inputs staged into it. | S1, lines 65–71; S2, lines 65–71; S3, lines 50–69. | Explain isolation and sequential execution. |
| C05 | Fact | The editorial sequence is Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewers, with PM classification after each reached review lens. | S1, lines 69–73; S2, lines 12–39; S3, lines 50–64. | Describe role order. |
| C06 | Fact | The Human Editor owns the brief and resolves `needs_human_judgment`; the PM classifies every reviewer finding as `valid_must_fix`, `valid_optional`, `invalid`, or `needs_human_judgment`. | S3, lines 9–30. | Explain responsibility for editorial decisions. |
| C07 | Fact | The Researcher owns sources, the claim ledger, and optional firsthand materials; the Story Editor owns an outline whose sections state purpose, supporting evidence, and reader takeaway; the Writer owns exactly one assigned candidate. | S2, lines 51–54; S3, lines 32–48. | Summarize artifact ownership without implying roles may edit each other's artifacts. |
| C08 | Fact | Review lenses never run in parallel. A must-fix stops the remaining lenses for that candidate; a replacement candidate restarts at Evidence. Optional and invalid findings do not consume another candidate, while a human-judgment decision blocks the run. | S2, lines 41–43. | Explain the sequential review loop. |
| C09 | Fact | Only a PM-validated must-fix creates another candidate, and candidate 003 is the hard limit. Exhausting that budget blocks the run and preserves its artifacts. | S1, lines 36–40 and 71–73; S2, lines 28–38 and 73–81. | Explain the candidate budget. “Three candidates maximum” is a faithful plain-language rendering. |
| C10 | Fact | Agent completion messages are not workflow completion: a worker must exit successfully and its owned files must exist and pass controller validation before the workflow advances. | S2, lines 45–61. | Explain why artifacts are gates rather than logs. |
| C11 | Fact | Review results are bound to an exact lens and SHA-256 candidate revision, require valid and internally consistent finding data, and must have a Markdown report matching the JSON findings. | S2, lines 54–61; S4, lines 35–59. | Explain review traceability and stale-review rejection. |
| C12 | Fact | PM decision records must cover every finding exactly once, carry the active request ID and review digest, retain prior reached lenses, exclude future lenses, and match the current revision. | S2, lines 57–58; S3, lines 26–30; S4, lines 61–92. | Explain controller validation of PM routing. |
| C13 | Fact | Earlier candidates, partial review sequences, reviews, and PM decisions remain available when revision occurs or a run blocks. | S4, lines 30–33. | Support “inspectable history” wording. |
| C14 | Fact | `workflow.json` is atomically rewritten as the controller's source of truth and records status, phase, current candidate and revision, active role, artifact paths, review attempts, timestamps, and a terminal block reason when blocked. | S4, lines 94–106. | Describe machine-readable run state. |
| C15 | Fact | On success, the controller revalidates the candidate hash, final reviews, and PM request bindings before writing `article.md`; the article is byte-for-byte identical to the accepted candidate. | S1, lines 36–38; S2, lines 73–81; S4, lines 30–33. | Explain the success gate and exact-copy terminal artifact. |
| C16 | Fact | The durable `.control/` directory retains prompt assignments, per-invocation logs, and natural exit markers only after cleanup; live launchers, PM requests, and agent workspaces stay in controller-private temporary paths and are not copied there. | S3, lines 3–7; S4, lines 108–115. | Explain the distinction between audit copies and live control state. |
| C17 | Fact | A blocked terminal state includes an actionable `workflow.json.block_reason`, and the controller verifies that the dedicated tmux session is gone before either success or blocked completion. | S2, lines 73–81; S4, lines 99–106. | Describe inspectable failure and cleanup behavior. |
| C18 | Fact | Parallel runs, resume after controller restart, and editing completed runs are not implemented; a blocked run is retried, if desired, in a new run directory after inspection. | S2, lines 83–84; S3, lines 9–13. | State current scope limits. Do not generalize beyond this implementation. |
| C19 | Firsthand observation | No runtime behavior was observed firsthand in this research assignment. | Research activity was limited to reading S1–S4; no CLI execution, artifact inspection from a real run, or benchmark was performed. | Do not use as a product claim. It records the evidence boundary. |
| C20 | Inference | Because transitions depend on validated, revision-bound files instead of agent chat or exit alone, a reader can reconstruct more of the controller's accepted path from the run artifacts than from transient conversation alone. | Derived from C10–C17, especially S2, lines 45–81 and S4, lines 94–115. The sources explicitly say completion never depends on tmux scrollback or chat transcripts, but “reconstruct more” is interpretive synthesis. | Use only with qualifying language such as “This makes…” or “Together, these records allow…”. |
| C21 | Inference | Fresh, context-limited reviewer processes and controller-owned routing separate critique from candidate editing and reduce the opportunity for one review lens to inherit another lens's conversation. | Derived from C04, C05, and S3, lines 50–69. Non-inheritance is documented; “reduce the opportunity” is the inferred consequence. | Suitable as a reason for role boundaries; do not claim elimination of bias or guaranteed independence. |
| C22 | Inference | The candidate cap converts an otherwise potentially open-ended revision cycle into a bounded outcome: acceptance by all gates, a human-judgment block, or a block after candidate 003. | Derived from C08–C09 and the flow in S2, lines 12–43. Other operational failures can also block, so this is a description of editorial routing, not every possible terminal path. | Use when explaining why the budget matters, with the stated scope. |
| C23 | Opinion | Durable artifact gates are a useful design choice for engineers who value auditability, because they leave concrete inputs, decisions, revisions, and terminal state to inspect. | Value judgment grounded in C10–C17; usefulness is not measured by the sources. | Clearly frame as an evaluation, not a proven performance result. |
| C24 | Unresolved | Whether write-uuter produces higher-quality articles, faster completion, or lower cost than other editorial workflows is not established. | S1–S4 provide contracts and behavior descriptions, not comparative studies or benchmarks. | Exclude comparative superiority claims. |
| C25 | Unresolved | The supplied documentation does not establish how the workflow performs under real Codex variability or how often runs reach candidate 003, block, or succeed. | No production-run data, success-rate data, or model-output evaluation appears in S1–S4. | Do not state reliability or success-rate claims. |
| C26 | Unresolved | Publishing integrations, web interfaces, and workflows outside this repository are outside the brief and are not supported by the allowed sources. | Brief scope boundary; S1–S4 do not document these capabilities. | Exclude from the article. |

## Writing guardrails for downstream roles

- Treat C01–C18 as document-supported facts and preserve their stated limits.
- Treat C20–C22 as interpretation, not direct quotations or measured outcomes.
- Signal C23 as an evaluative judgment if used.
- Do not convert C24–C26 into affirmative claims.
- Do not claim the whole system is deterministic: the sources assign
  deterministic state and validation responsibilities to Go, while Codex
  performs editorial work.

</write-uuter-context>