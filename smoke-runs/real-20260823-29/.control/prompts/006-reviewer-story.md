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
Revision: `sha256:eee837c8fc243a063540ca1cc3a93c45a39fef1d9f27e740cdf3f97afc77646d`

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
# How write-uuter turns a brief into an inspectable reviewed article

write-uuter is a Go command-line tool that turns a Markdown brief into a reviewed article. It does not treat the job as one long prompt whose useful history disappears into a chat transcript. Instead, each stage produces durable files: evidence, a claim ledger, an outline, candidate drafts, reviews, and product-manager decisions. In this sense, the workflow is artifact-driven: validated files on disk represent progress and determine what may happen next.

That design makes a run inspectable. An engineer can examine the inputs, see which candidate each reviewer assessed, and trace the decisions that led to acceptance or blockage. The repository documentation specifies these behaviors as contracts; the account here is based on those documents rather than an independent execution of the CLI.

## The controller sets the boundaries

Before editorial work starts, the controller validates that the brief contains every required section and that the target run is new. It then initializes the run directory atomically. From that point, responsibilities are deliberately divided: Go owns state transitions, artifact validation, revision hashes, timeouts, routing, and cleanup, while each Codex role writes only its assigned outputs in an isolated workspace.

The controller records its source of truth in `workflow.json`, which it rewrites atomically. That file tracks such details as status, phase, current candidate and revision, active role, artifact paths, review attempts, timestamps, and, when necessary, the reason a run was blocked. Agents therefore contribute bounded editorial work; they do not decide for themselves that the workflow has advanced.

## From evidence to the first candidate

The production chain is sequential. First, the Researcher assembles the evidence and claim ledger. The Story Editor turns those materials into an outline. The Writer then uses the brief, evidence, ledger, and outline to create `article-001.md`.

These are separate handoffs rather than one continuing agent conversation. Each role owns a specific artifact that becomes an inspectable input to the next stage. This separation also keeps authorship attributable: a later reviewer judges the candidate but cannot silently repair it.

## Four fresh review lenses

Each candidate enters four reviews in a fixed order: Evidence, Story, Clarity, and Copy. Every lens runs in a fresh Codex process, sequentially. A reviewer receives the brief, the exact candidate and revision under review, durable instructions for that lens, and only the additional context permitted for that lens. Reviewers write review artifacts; they never edit the candidate.

Completion is not inferred merely because an agent prints a final message or exits. The expected owned file must exist and pass its role-specific validation gate. A review must use an allowed lens and is bound to the exact candidate revision by its SHA-256 hash. Those checks prevent a review of one revision from being applied as though it covered another.

After each review, a persistent PM classifies every finding using a constrained vocabulary. `valid_must_fix` means a correction is required; `valid_optional` records a legitimate but non-blocking improvement; `invalid` rejects a finding and requires a reason; and `needs_human_judgment` says automation should not decide. The PM supplies editorial judgment, but Go independently validates the decision before routing the run.

PM decisions are bound to the candidate revision, the request ID, and the review digest, while retaining decisions from previously reached lenses. In practical terms, these bindings let the controller detect stale or mismatched decision artifacts instead of trusting conversational assertions.

## Revision has a hard limit

A validated must-fix finding immediately stops the remaining review lenses for that candidate. If the candidate budget permits, the Writer creates the next numbered candidate using the prior draft and the reached review and decision. Review then restarts from Evidence, so every revised candidate must pass the entire lens sequence from the beginning.

Optional and invalid findings do not consume another candidate. A finding classified as needing human judgment blocks the run. Candidate 003 is the hard ceiling: if it receives a validated must-fix, the controller blocks rather than creating candidate 004. This budget turns revision into a bounded process with a deterministic terminal outcome instead of an open-ended rewriting loop.

## Terminal artifacts explain the outcome

Success has a precise on-disk meaning. After one candidate passes all four final lenses, `article.md` must be non-empty and byte-for-byte identical to that accepted candidate. The final file is not a separately regenerated version whose relationship to the reviewed text must be guessed.

Revision and blockage remain inspectable too. Earlier candidates, partial lens sequences, reviews, and PM decisions are retained. `workflow.json` records the terminal state and any block reason, while `.control/` keeps audit copies of generated assignments, logs, and exit markers after cleanup. Completion therefore does not depend on preserving chat transcripts or terminal scrollback.

The result is a small, deterministic editorial controller with visible boundaries. Codex roles research, structure, write, review, and classify; Go verifies their artifacts and controls movement through the workflow. Whether a run succeeds or blocks, the files left behind show what was reviewed, what was decided, and why the process stopped where it did.

</write-uuter-context>

## Provided context: outline.md

<write-uuter-context name="outline.md">
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

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:eee837c8fc243a063540ca1cc3a93c45a39fef1d9f27e740cdf3f97afc77646d

</write-uuter-context>