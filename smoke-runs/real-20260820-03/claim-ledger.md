# Claim ledger

This ledger distinguishes the required classifications: **Fact**,
**Firsthand observation**, **Inference**, **Opinion**, and **Unresolved**.
Source IDs refer to `evidence/sources.md`.

## Fact

| ID | Claim | Support |
| --- | --- | --- |
| F01 | `write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | S1, opening description |
| F02 | The controller first validates all required level-two brief sections and that the target does not exist, then initializes the run via a temporary sibling directory renamed into place after initialization succeeds. | S2, **Controller sequence** |
| F03 | Go, rather than an agent conversation, owns workflow transitions, output validation, revision hashes, timeouts, and process cleanup. | S1, **Runtime model**; S3, introduction |
| F04 | One persistent PM starts before research, while at most one worker runs at a time in the dedicated tmux session. | S1, **Runtime model**; S2, **Lifecycle and terminal states** |
| F05 | The production order is Researcher, Story Editor, Writer, followed by fresh Evidence, Story, Clarity, and Copy reviewer processes in that order. | S1, **Runtime model**; S2, **Controller sequence** |
| F06 | Review lenses never run in parallel. After every reached lens, the PM classifies that lens's findings before the controller routes the run. | S2, **Controller sequence**; S3, **PM** and **Reviewers** |
| F07 | PM classifications are `valid_must_fix`, `valid_optional`, `invalid` with a reason, and `needs_human_judgment`; Go independently validates the decision file and applies routing. | S3, **PM** |
| F08 | A validated must-fix stops the remaining lenses for that candidate. If budget remains, the Writer creates the next candidate and review restarts at Evidence. Optional and invalid findings do not consume a candidate. | S2, **Controller sequence** |
| F09 | Candidate 003 is the hard limit. A validated must-fix after that budget is exhausted blocks the run, as does a human-judgment decision. | S1, **Runtime model**; S2, **Controller sequence** |
| F10 | The controller does not accept an agent's final message or process exit as completion; each role advances only after its owned artifacts exist and pass validation. | S2, **Artifact gates** |
| F11 | Research must include non-empty sources and a claim ledger naming all five claim classes; outlines, drafts, reviewer files, and PM decisions each have additional role-specific validation gates. | S2, **Artifact gates** |
| F12 | Review results must identify the assigned lens and exact SHA-256 candidate revision, contain structurally valid findings, and have a report containing the same finding fields. Stale lens or revision metadata is rejected. | S2, **Artifact gates**; S4, **Review result** |
| F13 | Reviewers do not edit candidates. Each reviewer is a fresh process with lens-specific durable context and no inherited conversation or reports from earlier reviewers. | S3, **Reviewers** |
| F14 | The run directory retains earlier candidates, partial lens sequences, reviews, and PM decisions after revision or blocking. | S4, **Run layout** |
| F15 | On success, `article.md` is written only after all four lenses pass PM routing and is byte-for-byte identical to the accepted candidate. | S1, **Run**; S4, **Run layout** |
| F16 | `workflow.json` is atomically rewritten as the controller's source of truth and records status, phase, candidate and revision, active role, artifact paths, review attempts, timestamps, and a terminal block reason when blocked. | S4, **workflow.json** |
| F17 | A timeout, premature agent exit, malformed artifact, stale review, human decision, or exhausted candidate budget blocks the workflow with an actionable reason. The controller kills its tmux session before returning on either success or failure. | S2, **Lifecycle and terminal states** |
| F18 | Editorial completion does not depend on tmux scrollback or chat transcripts; generated assignments, logs, lifecycle markers, and durable editorial artifacts remain available on disk. | S4, **workflow.json** and **Run layout** |
| F19 | The shipped controller is single-run and non-resumable; parallel runs, resume after restart, and editing completed runs are not implemented. | S2, **Controller sequence** and **Lifecycle and terminal states** |

## Firsthand observation

| ID | Claim | Support |
| --- | --- | --- |
| H01 | None recorded. No firsthand run, test, interface inspection, benchmark, or source-code investigation was performed for this research task. | Research method for this run; no `evidence/firsthand.md` or assets created |

## Inference

| ID | Claim | Basis and qualification |
| --- | --- | --- |
| I01 | Durable artifact gates make the workflow inspectable because intermediate evidence, candidates, reviews, decisions, and state survive independently of transient agent conversation. | Derived from F10, F14, F16, and F18. The sources document the persistence and validation mechanisms; “make inspectable” summarizes their combined effect. |
| I02 | Exact lens-and-revision checks keep a review tied to the candidate it actually evaluated, reducing the risk that stale output advances the workflow. | Derived from F12 and the explicit rejection of stale metadata in S2. “Reducing risk” is a consequence, not a measured result. |
| I03 | Separating reviewer findings, PM classification, and Go-controlled routing creates inspectable decision boundaries: reviewers report, the PM judges findings, and the controller enforces state changes. | Derived from F03, F06, F07, and F13. “Decision boundaries” is explanatory terminology, not wording used by the sources. |
| I04 | Restarting every revised candidate at the Evidence lens makes all four final approvals refer to the same candidate revision. | Derived from F08, F12, and F15. The final exact-copy condition confirms the accepted revision; the sources do not use this explanatory sentence verbatim. |
| I05 | Preserving blocked and superseded artifacts supports diagnosis after a run ends even though the current implementation cannot resume that run. | Derived from F14, F16, F17, and F19. The sources say preserved artifacts support a later inspected retry, but do not measure diagnostic effectiveness. |

## Opinion

| ID | Claim | Treatment |
| --- | --- | --- |
| O01 | The workflow is a good or superior way to produce technical articles. | Value judgment not established by the allowed sources; omit or clearly label as opinion. |
| O02 | Three candidates are the ideal revision budget. | Value judgment. The limit itself is Fact F09, but its optimality is unsupported. |
| O03 | The workflow is “small” or “simple.” | Editorial characterization from the brief's framing, not a property established by the four allowed factual sources. Use only as audience framing, not evidence-backed fact. |

## Unresolved

| ID | Claim or question | Why unresolved |
| --- | --- | --- |
| U01 | The overall article-production process is deterministic. | The sources say Go owns deterministic-style state machinery and that tests use a deterministic fake Codex executable, but they do not claim real Codex role outputs are deterministic. Safer supported wording: the Go controller owns and validates transitions. |
| U02 | The workflow improves article quality, accuracy, reviewer agreement, or throughput compared with another process. | No comparison, outcome data, benchmark, or quality study appears in the allowed sources. |
| U03 | The workflow works with publishing integrations, a web interface, parallel execution, or resume-in-place. | Publishing and web interfaces are out of scope in the brief; S2 explicitly says parallel runs and resume are not implemented. No positive claim is supportable. |
| U04 | The documented runtime behavior has been independently verified in this research run. | No firsthand execution or source-code inspection was performed; this ledger reports the four allowed documents' claims only. |
| U05 | Relative hints `README.md` and `docs/...` exist beneath the supplied resolution base `examples/`. | They do not exist under `examples/` in the inspected workspace. The correspondingly named allowed documents were located at the repository root and that resolution choice is disclosed in `evidence/sources.md`. |

## Safe synthesis boundary

The article may state F01–F19 as documented behavior, with citations to the
source notes. I01–I05 are useful explanations only if phrased as inferences
rather than measured benefits. O01–O03 and U01–U05 should not be converted into
unqualified factual claims.
