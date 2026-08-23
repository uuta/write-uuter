# Evidence Reviewer lens contract

Review only whether material claims in the exact candidate are supported and
faithful to the supplied sources, firsthand record when present, and claim
ledger. Do not perform story, clarity, or copy review.


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

Lens: `evidence`
Candidate: `article-001`
Revision: `sha256:41600af0875720e5c11343daefeceed60ccd9c158c1442347b8e0d00f620d63e`

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

## Provided context: drafts/article-001.md

<write-uuter-context name="drafts/article-001.md">
# From Brief to Inspectable Reviewed Article

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article. Its defining output is not just the final prose: it also preserves the evidence, outline, candidate drafts, reviews, product-manager decisions, and workflow state that produced that prose. For engineers evaluating an editorial workflow, those durable files make the route from request to result inspectable without relying on an agent conversation.

## Go controls the workflow

The controller first validates the brief's required sections and requires a new run target. It initializes the run in a temporary sibling directory, then commits it with a rename that will not replace an existing target. From there, Go owns the state machine: role order, file validation, revision hashes, timeouts, routing, and cleanup. An agent's final message cannot advance the workflow. A worker must exit successfully, and the files it owns must pass the relevant artifact checks.

The editorial roles operate within those deterministic boundaries. The main sequence is a persistent PM, a Researcher, a Story Editor, and a Writer, followed by fresh reviewers for Evidence, Story, Clarity, and Copy. Roles work in separate controller-created workspaces outside the durable run directory. Each receives only its contracted context, and validated regular-file outputs are copied back into the run.

This produces a simple progression. The Researcher writes `evidence/sources.md` and `claim-ledger.md`, plus optional firsthand evidence or assets when available. The Story Editor turns the supported material into `outline.md`. The Writer expands that plan into one specifically assigned candidate under `drafts/`. Narrow ownership makes each handoff explicit: agents create editorial artifacts, while the controller determines whether those artifacts satisfy the contract required to continue.

## Findings, decisions, and revisions stay separate

Review runs one lens at a time in a fixed order: Evidence, Story, Clarity, then Copy. Each reviewer is fresh. It receives the brief, the exact candidate and its revision, a durable prompt for its lens, and only the additional context documented for that lens. Reviewers do not inherit earlier reviewers' conversations and cannot edit the candidate. Instead, each owns a lens-specific `result.json` and `report.md`.

Those results are tightly bound to the text reviewed. The declared lens and SHA-256 revision must match; findings must be complete and uniquely identified; and the Markdown report must reproduce the JSON findings in order. Stale metadata is rejected.

After every reached lens, the PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid`, or `needs_human_judgment`. An invalid classification requires a reason. The PM neither writes reviews nor changes candidates, and its authority is not the last word on workflow state: Go validates that the decision matches the request and review digest, covers every finding exactly once, retains earlier reached classifications, and does not invent decisions for future lenses. Go then applies the routing.

This separation leaves three distinct acts available for inspection: a reviewer identifies a problem, the PM classifies its significance, and the Writer changes the prose only when routed to a revision. No role can silently combine all three.

## Revision is sequential and bounded

A PM-validated must-fix stops the remaining review lenses for that candidate. If the workflow has used fewer than three candidates, the Writer creates the next version and review restarts at Evidence. Restarting from the first lens means factual support is checked again after the prose changes, before effort moves to Story, Clarity, or Copy. Optional or invalid findings do not consume another candidate.

Candidate 003 is the hard ceiling. If it still has a must-fix, the workflow blocks instead of iterating indefinitely. A decision requiring human judgment also blocks, as do operational or contract failures such as a timeout, malformed or stale artifacts, premature worker exit, or cleanup failure. The three-candidate budget therefore makes the revision loop finite without disguising an unresolved result as success.

## Terminal artifacts show what happened

Revision does not erase history. Earlier candidates, partial lens sequences, reviews, and PM decisions remain in the run. `workflow.json` records the status and phase, current candidate and revision, counts, artifact paths, timestamps, and, for a blocked run, an actionable terminal reason. Generated prompts, invocation logs, and exit markers remain as post-cleanup audit copies under `.control/`; editorial completion does not depend on chat transcripts or terminal scrollback.

Success has a similarly concrete definition. The final candidate must pass PM routing through all four lenses and then survive terminal revalidation. Only afterward does the controller write `article.md`, byte-for-byte identical to the accepted candidate. Thus the terminal state yields either an exact accepted article or a preserved explanation of why the run blocked.

The repository deliberately keeps this model small: runs are single-run and non-resumable, review lenses are not parallel, and completed runs are not edited. Within those limits, `write-uuter` answers the brief-to-article question with bounded role authority, deterministic routing, and durable artifact gates. The result is a reviewed article whose production trail remains visible on disk.

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

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:41600af0875720e5c11343daefeceed60ccd9c158c1442347b8e0d00f620d63e

</write-uuter-context>