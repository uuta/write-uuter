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

Write candidate 001 to `drafts/article-001.md` in this isolated workspace.

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

Accessed 2026-08-23. These are controller-staged copies of the brief's local
source hints; no source-repository paths were read directly and no network
sources were used.

## S1 — README.md

- Location: `context/source-hints/001-README.md`
- Original hint: `../README.md`
- Useful summary: Defines write-uuter as a Go CLI that turns a Markdown brief
  into a reviewed article while preserving evidence, outline, candidates,
  reviews, and PM decisions. States that Go owns state transitions,
  validation, revision hashes, timeouts, and cleanup. Gives the role order,
  the three-candidate ceiling, success behavior (an exact copy of the accepted
  candidate), and blocked-run behavior.
- Most useful sections: opening description; **Run**; **Runtime model**.

## S2 — docs/workflow.md

- Location: `context/source-hints/002-workflow.md`
- Original hint: `../docs/workflow.md`
- Useful summary: Gives the controller sequence and review loop. Review lenses
  are sequential; a validated must-fix stops later lenses for that candidate,
  sends work back to the Writer if the candidate is below 003, and restarts
  review at Evidence. Optional or invalid findings do not use a candidate;
  human judgment blocks. Documents artifact gates, isolation/lifecycle checks,
  final revalidation, and unsupported resume/parallel-run behavior.
- Most useful sections: **Controller sequence**; **Artifact gates**;
  **Lifecycle and terminal states**.

## S3 — docs/roles.md

- Location: `context/source-hints/003-roles.md`
- Original hint: `../docs/roles.md`
- Useful summary: Defines ownership and authority for the Human Editor, PM,
  Researcher, Story Editor, Writer, and four reviewer lenses. The PM classifies
  every finding but does not write candidates or reviews; Go independently
  validates PM decisions and applies routing. Reviewers are fresh, sequential,
  read only lens-specific context, and cannot edit candidates.
- Most useful sections: **PM**; **Researcher**; **Story Editor**; **Writer**;
  **Reviewers**.

## S4 — docs/artifacts.md

- Location: `context/source-hints/004-artifacts.md`
- Original hint: `../docs/artifacts.md`
- Useful summary: Specifies the durable run layout and schemas for review
  results, PM decisions, and workflow state. Earlier candidates and partial
  review histories remain available after revision or blockage. `article.md`
  exists only on success and is byte-for-byte identical to the accepted final
  candidate. `.control/` holds post-cleanup audit copies; editorial completion
  does not depend on chat transcripts or tmux scrollback.
- Most useful sections: **Run layout**; **Review result**; **PM decision**;
  **workflow.json**.

## Source boundaries

The supplied brief permits only README.md and docs/ facts. Accordingly, this
research does not make claims about publishing integrations, web interfaces,
or workflows outside this repository. The documents describe the shipped
contracts and also name limitations: runs are single-run and non-resumable,
review lenses are not parallel, completed runs are not edited, Linux execution
fails closed pending a native read-isolation backend, and intentional ancestry
escape is outside the current containment guarantee.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

Classification vocabulary used explicitly in this ledger: **Fact**,
**Firsthand observation**, **Inference**, **Opinion**, and **Unresolved**.
Source IDs refer to `evidence/sources.md`.

## Facts

| ID | Classification | Claim | Support |
| --- | --- | --- | --- |
| F1 | Fact | `write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article and preserves the evidence, outline, candidates, reviews, and PM decisions that produced it. | S1, opening description |
| F2 | Fact | Before role work begins, the controller validates all required brief sections, requires a new target, initializes the run in a temporary sibling directory, and commits it with a no-replace rename. | S2, Controller sequence |
| F3 | Fact | Go—not an agent final message—owns state transitions and validates files, revisions, timeouts, routing, and cleanup. A worker must exit successfully and its owned files must pass artifact validation before the workflow advances. | S1, Runtime model; S2, Artifact gates |
| F4 | Fact | The main production order is persistent PM, Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewers. The reviewer lenses execute sequentially. | S1, Runtime model; S2, Controller sequence; S3, Reviewers |
| F5 | Fact | The Researcher owns `evidence/sources.md`, optional firsthand evidence/assets, and `claim-ledger.md`; the Story Editor owns `outline.md`; the Writer owns one assigned candidate; each reviewer owns a lens-specific `result.json` and `report.md`. | S3, role sections |
| F6 | Fact | Roles run in separate controller-created workspaces outside the durable run directory, receive only contracted context, and have validated regular-file outputs copied back into the run. | S1, Runtime model; S2, Lifecycle and terminal states |
| F7 | Fact | Evidence, Story, Clarity, and Copy reviewers receive the brief, exact candidate and revision, their durable lens prompt, and only the documented lens-specific additional context. They cannot edit candidates and do not inherit prior-lens conversations. | S3, Reviewers |
| F8 | Fact | After each reached lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` (with a reason), or `needs_human_judgment`; the PM cannot write candidates or reviews, and Go validates its decisions and applies routing. | S3, PM |
| F9 | Fact | A PM-validated must-fix stops the remaining lenses for the current candidate. If fewer than three candidates have been used, the Writer creates the next candidate and review restarts at Evidence. Optional and invalid findings do not consume a candidate. | S2, Controller sequence |
| F10 | Fact | Candidate 003 is the hard limit. An exhausted third candidate, a human-judgment decision, timeout, malformed or stale artifacts, premature exit, or cleanup failure produces a blocked workflow with an actionable reason. | S1, Run/Runtime model; S2, Lifecycle and terminal states |
| F11 | Fact | Review artifacts are revision-bound and schema-validated: lens and SHA-256 revision must match, findings must be complete and unique, and the Markdown report must reproduce the JSON findings in order. Stale metadata is rejected. | S2, Artifact gates; S4, Review result |
| F12 | Fact | PM decisions are request- and review-digest-bound, must cover every finding exactly once, retain previously reached lenses and classifications, and cannot prepopulate future lenses. | S3, PM; S4, PM decision |
| F13 | Fact | Earlier candidates, partial lens sequences, reviews, and PM decisions remain in the run when revision occurs or the workflow blocks. `workflow.json` records running/succeeded/blocked status, phase, current candidate/revision, counts, paths, timestamps, and a terminal block reason when applicable. | S4, Run layout/workflow.json |
| F14 | Fact | Success requires all four final-candidate lenses to pass PM routing plus terminal revalidation. Only then is `article.md` written, byte-for-byte identical to the accepted candidate. | S1, Run; S2, Lifecycle and terminal states; S4, Run layout |
| F15 | Fact | Generated prompts, invocation logs, and exit markers are retained as post-cleanup audit copies under `.control/`; completion does not depend on chat transcripts or tmux scrollback. | S3, opening; S4, workflow.json |
| F16 | Fact | The shipped workflow is single-run and non-resumable; parallel review lenses, resume after controller restart, and editing completed runs are not implemented. | S2, Controller sequence/Lifecycle and terminal states |

## Firsthand observations

No firsthand work was performed. No command execution, live workflow run,
artifact inspection from an actual run, interview, or original measurement is
claimed as a **Firsthand observation**. Therefore `evidence/firsthand.md` and
`evidence/assets/` were not created.

## Inferences

| ID | Classification | Claim | Basis and boundary |
| --- | --- | --- | --- |
| I1 | Inference | The workflow is inspectable because consequential inputs, intermediate candidates, reviews, classifications, state, and terminal output are represented as durable files rather than existing only in agent conversation. | Derived from F1, F5, F11–F15. “Inspectable” is an interpretation of the documented artifact retention and validation model. |
| I2 | Inference | The artifact gates reduce ambiguity at handoffs by requiring each role to produce a narrowly specified, machine-checkable output before Go advances. | Derived from F3, F5, F11, and F12. The sources specify the checks; “reduce ambiguity” is the inferred benefit. |
| I3 | Inference | Restarting every revised candidate at the Evidence lens prioritizes rechecking factual support before spending work on later story, clarity, or copy checks. | Derived from F4 and F9. The priority rationale is inferred; the documents specify the order but do not state this motivation verbatim. |
| I4 | Inference | The three-candidate budget converts an otherwise potentially open-ended agent revision loop into a bounded workflow with a visible blocked outcome. | Derived from F9, F10, and F13. “Bounded” follows directly from the hard ceiling; the design intention is interpretive. |
| I5 | Inference | Keeping review and PM classification separate from candidate writing creates inspectable separation between detecting a problem, deciding whether it must be fixed, and changing prose. | Derived from F5, F7, F8, F12, and F13. “Separation” describes documented authority boundaries; its inspectability benefit is inferred. |

## Opinions

| ID | Classification | Claim | Treatment |
| --- | --- | --- | --- |
| O1 | Opinion | For engineers evaluating editorial automation, explicit artifact contracts are generally preferable to relying on chat history alone. | Editorial judgment, not a repository fact. If used, present as analysis and ground it in F11–F15. |
| O2 | Opinion | A three-candidate ceiling is a pragmatic budget for a small workflow. | Value judgment; the sources establish the ceiling but not that it is optimal or pragmatic. Avoid presenting it as fact. |

## Unresolved

| ID | Classification | Question or claim not established | Why unresolved |
| --- | --- | --- | --- |
| U1 | Unresolved | How often the workflow succeeds, blocks, or reaches candidate 003 in real use. | No operational dataset or run history is included in the permitted sources. |
| U2 | Unresolved | Whether this workflow improves article quality, reviewer agreement, cost, or elapsed time compared with another process. | The permitted sources define implementation and contracts, not comparative outcomes. |
| U3 | Unresolved | Whether the three-candidate budget is the best limit. | The documents state the limit but provide no optimization evidence. |
| U4 | Unresolved | How a future resume mechanism, parallel-run support, Linux isolation backend, container/VM containment, publishing integration, or web interface would work. | These capabilities are absent, deferred, or outside the brief's scope. |

## Safe synthesis boundary

A repository article can state F1–F16 as documented facts and may use I1–I5
when clearly framed as interpretation. O1–O2 must remain opinion, and U1–U4
must not be resolved without additional allowed evidence. No firsthand claims
are available.

</write-uuter-context>

## Provided context: outline.md

<write-uuter-context name="outline.md">
# Outline: From Brief to Inspectable Reviewed Article

## Article direction

- **Audience:** Engineers evaluating a small, artifact-driven editorial workflow.
- **Provisional thesis:** `write-uuter` uses a deterministic Go controller to coordinate narrowly scoped Codex roles, validate their file-based handoffs, and preserve the evidence, candidates, reviews, decisions, and terminal state needed to inspect how an article was produced.
- **Target length:** 750–850 words, leaving margin below the 900-word limit.
- **Scope boundary:** Describe only the workflow and contracts documented by the repository. Do not claim measured improvements in quality, cost, speed, or reviewer agreement, and do not speculate about publishing, web interfaces, or unimplemented capabilities.

## 1. The workflow’s central promise: an article with a visible trail

- **Purpose:** Open with the reader’s question and establish the repository’s defining idea: the product is not merely final prose, but final prose accompanied by inspectable production artifacts.
- **Supporting evidence:** F1 establishes that the Go CLI turns a Markdown brief into a reviewed article while preserving evidence, outline, candidates, reviews, and PM decisions. F14 establishes that success produces `article.md` only after review and terminal revalidation. I1 may be presented explicitly as interpretation: durable files make the process inspectable in a way that conversation-only state would not.
- **Reader takeaway:** `write-uuter` makes the path from brief to accepted article visible on disk, so engineers can examine both the outcome and the steps that led to it.

## 2. Go controls the sequence; roles own bounded artifacts

- **Purpose:** Explain the architecture and role sequence in plain language, distinguishing deterministic workflow control from agent-authored editorial work.
- **Supporting evidence:** F2 covers initial brief validation and safe run initialization. F3 states that Go owns state transitions, validation, revision hashes, timeouts, routing, and cleanup, and advances only after successful exit plus artifact validation. F4 gives the order: persistent PM, Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewers. F5 assigns each role its owned files. F6 documents isolated controller-created workspaces and copying validated outputs into the durable run. I2 may be framed as analysis: narrow, machine-checked handoffs reduce ambiguity between roles.
- **Reader takeaway:** Agents do specialized editorial tasks, but the Go controller—not an agent’s final message—decides whether the workflow may advance.

## 3. Research, outline, and candidate form the production path

- **Purpose:** Walk through the pre-review stages without turning the section into a role catalog: research establishes supported claims, the Story Editor organizes them, and the Writer produces one assigned candidate.
- **Supporting evidence:** F4 supplies the production order. F5 identifies the Researcher’s ownership of sources and claim ledger, the Story Editor’s ownership of `outline.md`, and the Writer’s ownership of a single assigned candidate. F6 supports the claim that each role receives contracted context in an isolated workspace and returns validated regular-file output.
- **Reader takeaway:** The workflow converts the brief into progressively more concrete artifacts—evidence, outline, then candidate—while keeping ownership and handoffs explicit.

## 4. Sequential review separates findings, decisions, and revisions

- **Purpose:** Explain how the four review lenses operate and why reviewer findings do not directly change prose or routing.
- **Supporting evidence:** F4 establishes the sequential Evidence, Story, Clarity, and Copy order. F7 states that reviewers are fresh, receive the exact candidate and revision plus lens-specific context, cannot edit candidates, and do not inherit prior-lens conversations. F8 states that the PM classifies every reached finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`, while Go validates the decisions and routes the workflow. F11 and F12 establish revision-, digest-, and request-bound validation and complete finding coverage. I5 may be labeled as interpretation: separate authority makes detection, classification, and prose changes independently inspectable.
- **Reader takeaway:** A reviewer reports a problem, the PM classifies it, and only the controller routes a revision; no single role both judges and silently rewrites the candidate.

## 5. Must-fixes restart review within a three-candidate budget

- **Purpose:** Describe the revision loop, the early stop on a must-fix, and the hard ceiling that prevents indefinite iteration.
- **Supporting evidence:** F9 states that a validated must-fix stops later lenses, sends candidates below 003 back to the Writer, and restarts review at Evidence; optional and invalid findings do not consume a candidate. F10 establishes candidate 003 as the hard limit and lists human judgment, exhaustion, timeout, malformed or stale artifacts, premature exit, and cleanup failure as blocked outcomes. I3 may be framed as interpretation: restarting at Evidence rechecks factual support before later lenses. I4 may be framed as interpretation: the ceiling turns an open-ended revision loop into a bounded workflow with a visible blocked result. Avoid O2 and U3; do not call three candidates optimal.
- **Reader takeaway:** Revision is bounded and conservative: each changed candidate starts review from the first lens, and unresolved must-fixes after candidate 003 block rather than loop forever.

## 6. Artifact gates make success and blockage auditable

- **Purpose:** Close by showing what remains available for inspection and how terminal states are earned, tying durable files back to the article’s central promise.
- **Supporting evidence:** F11 specifies schema and revision checks for review JSON and Markdown reports. F12 specifies complete, digest-bound PM decisions. F13 states that earlier candidates, partial reviews, and decisions remain available and that `workflow.json` records status, phase, candidate, revision, counts, paths, timestamps, and block reason. F14 states that all four final-candidate lenses and terminal revalidation must pass before `article.md` is written byte-for-byte from the accepted candidate. F15 documents audit copies under `.control/` and independence from chat transcripts or tmux scrollback. F16 can supply a concise limitation note: runs are single-run and non-resumable; parallel lenses and editing completed runs are not implemented.
- **Reader takeaway:** Success is a validated terminal state, blockage leaves an actionable record, and either outcome preserves enough durable context to inspect what happened.

## Closing emphasis

- End by restating the answer in one compact paragraph: `write-uuter` combines bounded role authority with deterministic routing and durable artifact gates, yielding either an exact accepted article or an inspectable blocked run.
- Keep “inspectable,” “reduce ambiguity,” and design-rationale language clearly framed as synthesis from the documented contracts, not measured outcomes.
- Do not resolve U1–U4 or imply that this workflow outperforms alternatives.

</write-uuter-context>