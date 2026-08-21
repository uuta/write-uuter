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
Revision: `sha256:94b953876ab2019d9308404fdbefe8dc64506aa5b25aebf30abca863bc0f551c`

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
# From Brief to Inspectable Reviewed Article

write-uuter is a Go CLI that turns a Markdown brief into a reviewed article while preserving the work that produced it. Its central design choice is a clear division of responsibility: a deterministic Go controller owns workflow state, validation, routing, revision hashes, timeouts, cleanup, and publication, while isolated Codex processes perform bounded editorial tasks.

This is not one agent carrying an article from prompt to publication. The project assigns research, structure, writing, review, and editorial judgment to distinct roles. Workers and reviewers run separately, each reviewer starts as a fresh process, and one persistent PM makes editorial classifications throughout the run. Because those judgments are retained while Go enforces the resulting routes, engineers can inspect both the editorial decision and the control flow that followed it.

## From brief to first candidate

A run begins only after the controller validates the brief's required sections and confirms that the target is new. It then initializes the run directory atomically. From there, work proceeds in a fixed order.

First, the Researcher creates a non-empty source record and a claim ledger. The ledger must explicitly account for five kinds of material: Fact, Firsthand observation, Inference, Opinion, and Unresolved. That classification gives later roles a concrete boundary between supported claims and material that needs qualification or cannot yet be established.

Next, the Story Editor converts the evidence into an outline. Each section must state its purpose, supporting evidence, and intended reader takeaway. The Writer then expands that outline into candidate 001. A candidate must contain substantive text and cannot contain unresolved drafting markers.

These requirements are examples of artifact gates: a required file and the validation it must pass before the controller advances. Each handoff is therefore a named, durable artifact with a minimum contract, not an informal assurance that the work is done.

## Files, not final messages, prove progress

Each role works in an isolated workspace rather than directly in the durable run directory or another role's workspace. Go stages only the context that role is allowed to receive. After the process exits successfully, the controller checks the role's owned outputs, requires valid regular files, and copies validated results back into the run.

An agent's final message cannot advance the workflow on its own. The expected files must exist and satisfy their contracts. Review output, for example, is tied to a specific candidate by its SHA-256 revision; the declared lens and revision must match the assignment, and the structured findings must agree with the accompanying report.

The controller records the live state in `workflow.json`, which is rewritten atomically. It includes the run's status and phase, current candidate and revision, active role, artifact paths, review attempts, timestamps, and—when applicable—a terminal blocking reason. In this repository, “inspectable” is best understood as a consequence of this design: state transitions depend on retained, validated files rather than transient chat transcripts or terminal scrollback.

## Four reviews, one lens at a time

Every candidate enters a fixed, sequential review sequence: Evidence, Story, Clarity, then Copy. Reviewers report findings but never edit the candidate. After each reached lens, the persistent PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`.

The separation matters. A fresh reviewer supplies a lens-specific assessment, the PM records the editorial judgment about each finding, and Go validates that decision artifact and applies predetermined routing rules. Later lenses do not run concurrently or get ahead of an unresolved must-fix.

If the PM validates a must-fix, review of that candidate stops immediately. The Writer receives the prior candidate and the reached review result and produces the next numbered candidate. Review then restarts at Evidence. One rationale supported by the design is that changed content should not inherit later-lens results attached to an earlier revision. Optional and invalid findings do not consume another candidate.

Automated revision is bounded at candidate 003. A must-fix against that candidate blocks the run instead of creating candidate 004. A need for human judgment, a runtime failure, or an artifact-validation failure also blocks the run. The limit does not establish that three attempts are optimal; it simply prevents indefinite automated revision and turns unresolved work into an explicit terminal state.

## Two inspectable endings

Success requires the final candidate to pass routing at all four review lenses. Only then does the controller create `article.md`, as a byte-for-byte copy of the accepted candidate. The published artifact can therefore be matched exactly to the revision that was reviewed.

A blocked run keeps its reason and the work accumulated before the stop. Depending on how far it progressed, that record can include earlier candidates, partial review sequences, reviewer reports, PM decisions, workflow state, and audit copies of prompts, logs, and exit markers. Failure does not collapse into an unexplained absence of an article.

That durable record is the practical answer to how write-uuter produces an inspectable reviewed article. The controller does not merely ask roles to follow a process; it requires evidence of each completed step, validates the evidence, and preserves the route to either exact publication or explicit blockage. The documented workflow does have firm boundaries: parallel runs, resuming after a controller restart, and editing completed runs are not implemented. Within those boundaries, its artifacts make the outcome traceable without treating agent conversation as workflow state.

</write-uuter-context>

## Provided context: outline.md

<write-uuter-context name="outline.md">
# Outline: From Brief to Inspectable Reviewed Article

Target length: 700–850 words. The article should answer the brief through the implemented workflow, using “inspectable” as a synthesis grounded in retained artifacts rather than as a measured quality claim.

## 1. The design in one view: Go controls, Codex edits

- **Purpose:** Open with the direct answer: write-uuter is a Go CLI that converts a Markdown brief into a reviewed article by having a deterministic controller coordinate isolated editorial roles. Establish the key division of responsibility before introducing workflow detail.
- **Supporting evidence:** F1 (S1); F3 (S1, S2, S3); F5–F6 (S1, S2, S3). Use I2 only as an explicitly framed interpretation: retained PM judgments plus controller-enforced routing make the responsibility split auditable.
- **Reader takeaway:** The system is neither one free-running agent nor a conventional in-memory pipeline: Go owns control and validation, while separate Codex processes own bounded editorial tasks.

## 2. The forward path: brief, evidence, outline, candidate

- **Purpose:** Walk through the first half of a run in sequence: validate the brief and initialize a new run atomically; have the Researcher produce sources and a classified claim ledger; have the Story Editor turn that evidence into an outline; then have the Writer produce the first candidate. Explain role ownership in plain language.
- **Supporting evidence:** F2 (S1, S2); F4 (S1, S2, S3); F8–F9 (S2, S3). Mention the required claim classes and the outline’s purpose/evidence/takeaway fields as concrete examples of artifact contracts.
- **Reader takeaway:** Each editorial handoff becomes a named file with minimum content requirements, so later work begins from validated inputs rather than an informal conversation.

## 3. Why artifact gates, not agent messages, advance the run

- **Purpose:** Explain the mechanism that makes the workflow inspectable: roles work in isolated workspaces; the controller stages only allowed context; after a successful process exit it validates owned regular-file outputs and copies them into the durable run. Clarify that a final chat message cannot advance state.
- **Supporting evidence:** F6–F7 (S1, S2, S3); F11 (S2, S4); F17 (S4). Use I1 as a labeled synthesis: inspectability comes from retained, validated files rather than transient transcripts. Include representative checks such as revision hashes and consistent review JSON/report content without turning the section into a schema inventory.
- **Reader takeaway:** Progress is evidenced by durable artifacts that satisfy contracts; apparent completion by an agent is insufficient.

## 4. The sequential review and decision loop

- **Purpose:** Describe the fixed Evidence → Story → Clarity → Copy sequence. Distinguish reviewers, which report findings without editing candidates, from the persistent PM, which classifies each finding, and from Go, which validates decisions and applies routing.
- **Supporting evidence:** F5, F10 (S1, S2, S3); F11 (S2, S4); F12 (S2, S3, S4). Name all four PM classifications: `valid_must_fix`, `valid_optional`, `invalid` with a reason, and `needs_human_judgment`.
- **Reader takeaway:** Fresh reviewers supply lens-specific judgments, the PM records editorial decisions, and deterministic controller rules decide what happens next.

## 5. Revision routing and the three-candidate budget

- **Purpose:** Show the loop’s branching behavior. A validated must-fix stops later lenses, sends the Writer to the next candidate, and restarts that candidate at Evidence; optional or invalid findings do not consume a candidate. Explain that candidate 003 is the hard limit and identify the conditions that block a run.
- **Supporting evidence:** F13 (S2); F14 (S1, S2, S3). I3 may explain, as an inference, why review restarts at Evidence after content changes. I4 may frame the candidate cap as bounding automated revision, but do not claim that three is an optimal number. State that must-fix at candidate 003, human judgment, and runtime or artifact failure produce a blocked run.
- **Reader takeaway:** Revision is bounded and predictable: the workflow cannot revise indefinitely or silently skip unresolved problems.

## 6. Terminal artifacts: exact success or inspectable blockage

- **Purpose:** Close by contrasting the two terminal outcomes. On success, all four lenses have passed PM routing and `article.md` exactly matches the accepted candidate. On blockage, the reason and accumulated work remain. Tie both outcomes back to `workflow.json` and the retained revision/review history.
- **Supporting evidence:** F15 (S1, S2, S4); F16–F17 (S2, S4). Mention earlier candidates, partial review sequences, reviews, PM decisions, workflow state, and applicable audit copies as retained evidence. If limits are noted, use F18 (S2, S3): parallel runs, restart-resume, and editing completed runs are not implemented.
- **Reader takeaway:** The terminal result is inspectable in either case: readers can trace an accepted article to its candidate and reviews, or a blocked run to its preserved state and reason.

## Scope and language guardrails

- Do not claim measured improvements in accuracy, quality, reliability, or usability; U1–U5 remain unresolved.
- Do not call the three-candidate limit “reasonable” or “optimal” (O2).
- Keep publishing integrations, web interfaces, other repositories, and unsupported resume behavior out of the article.
- Explain “artifact gate” on first use as a required file plus validation that must pass before the controller advances.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:94b953876ab2019d9308404fdbefe8dc64506aa5b25aebf30abca863bc0f551c

</write-uuter-context>