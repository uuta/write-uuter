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

Accessed 2026-08-22. These are controller-staged copies of the brief's local source hints; no source-repository paths were read directly and no network sources were used.

## S1 — Repository README

- Staged location: `context/source-hints/001-README.md`
- Original hint: `../README.md`
- Useful sections: opening description (lines 3–5), run behavior (41–46), brief requirements (57–66), and runtime model (68–84).
- Summary: Defines write-uuter as a Go CLI that preserves the materials behind a reviewed article. Documents successful and blocked outcomes, the required brief structure, controller ownership of state and validation, isolated role workspaces, sequential roles and review lenses, PM-routed revision, and candidate 003 as the hard limit.

## S2 — Workflow documentation

- Staged location: `context/source-hints/002-workflow.md`
- Original hint: `../docs/workflow.md`
- Useful sections: Controller sequence (lines 3–43), Artifact gates (45–62), and Lifecycle and terminal states (64–98).
- Summary: Provides the end-to-end state sequence and revision loop. Explains that review lenses are sequential, a validated must-fix stops later lenses and restarts the next candidate at Evidence, optional/invalid findings do not use candidate budget, and human judgment blocks. Enumerates validation gates and describes cleanup, timeout, failure, and success behavior.

## S3 — Role contracts

- Staged location: `context/source-hints/003-roles.md`
- Original hint: `../docs/roles.md`
- Useful sections: role-instruction model (lines 3–9), Human Editor and PM (11–35), Researcher through Writer (37–55), and Reviewers (57–81).
- Summary: Assigns ownership and prohibitions to the human editor, persistent PM, Researcher, Story Editor, Writer, and four fresh reviewers. Specifies PM classifications, writer revision inputs, reviewer order and lens-specific context, output locations, and filesystem isolation.

## S4 — Artifact contracts

- Staged location: `context/source-hints/004-artifacts.md`
- Original hint: `../docs/artifacts.md`
- Useful sections: Run layout and retention (lines 3–33), Review result (35–60), PM decision (62–95), and `workflow.json` plus audit data (97–120).
- Summary: Lists the durable run tree and states what remains inspectable after revisions or blockage. Defines validated review and PM-decision structures, identifies `workflow.json` as the controller's source of truth, describes `.control/` audit copies, and says successful `article.md` is byte-for-byte identical to the accepted candidate.

## Source limitations

- The brief restricts support to README.md and `docs/`; this record therefore makes no claims from implementation code, tests, runtime execution, or external material.
- The documentation describes the shipped contracts and issue-1 behavior, but this research did not independently execute the CLI or verify implementation conformance.

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim ledger

## Classification key

- **Fact** — directly stated by one or more allowed repository sources.
- **Firsthand observation** — produced by direct execution, inspection, measurement, or other firsthand work during this research.
- **Inference** — a reasoned synthesis that is not stated verbatim by a source.
- **Opinion** — an evaluative judgment rather than a verifiable description.
- **Unresolved** — a relevant question the allowed evidence does not settle.

Source IDs refer to `evidence/sources.md`. No firsthand investigation was performed.

## Facts

| ID | Claim | Support |
| --- | --- | --- |
| F1 | write-uuter is a Go CLI that turns a Markdown brief into a reviewed article while preserving evidence, outline, candidates, reviews, and PM decisions. | S1, lines 3–5 |
| F2 | The controller validates required brief sections and a new target, initializes the run atomically, then starts a persistent PM before sequential Researcher, Story Editor, Writer, and reviewer work. | S2, lines 5–39; S1, lines 81–84 |
| F3 | Go, rather than an agent, owns workflow transitions, artifact validation, revision hashes, timeouts, cleanup, and final routing enforcement. | S1, lines 70–80; S3, lines 3–9, 28–35 |
| F4 | Each role runs in a separate private workspace; only contracted context is staged in, and validated regular-file outputs are copied into the durable run. | S1, lines 71–80; S2, lines 66–79 |
| F5 | The Researcher produces sources and a classified claim ledger; the Story Editor produces an outline whose sections include purpose, supporting evidence, and reader takeaway; the Writer produces one assigned candidate. | S2, lines 51–54; S3, lines 37–55 |
| F6 | Evidence, Story, Clarity, and Copy reviewers run fresh and sequentially, do not edit candidates, and receive lens-specific additional context. | S3, lines 57–81 |
| F7 | After each reached lens, the PM classifies every finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go validates the decision and applies routing. | S3, lines 17–35 |
| F8 | A validated must-fix stops the remaining lenses for that candidate; if budget remains, the Writer creates a replacement and review restarts at Evidence. Optional and invalid findings do not consume a candidate. | S2, lines 28–43 |
| F9 | Candidate 003 is the hard limit. Exhausting it blocks the run rather than producing another revision. | S1, lines 81–84; S2, lines 28–34, 84–87 |
| F10 | Agent completion messages and process exit alone do not advance the workflow: owned files must exist and pass artifact-specific validation. | S2, lines 45–62 |
| F11 | Review artifacts bind findings to an exact lens and SHA-256 candidate revision; PM records bind decisions to the current revision, request ID, and review digest and preserve prior reached-lens decisions. | S2, lines 55–62; S3, lines 28–35; S4, lines 56–60, 91–95 |
| F12 | Earlier candidates, partial review sequences, reviews, and PM decisions remain in the run when revision occurs or the run blocks. | S4, lines 30–33 |
| F13 | `workflow.json` is atomically rewritten as the controller's source of truth and records status, phase, current candidate/revision, active role, artifact paths, review count, timestamps, and a terminal block reason when blocked. | S4, lines 97–109 |
| F14 | On success, `article.md` is published only after all four final-candidate lenses pass PM routing and is byte-for-byte identical to that candidate. | S1, lines 41–46; S4, lines 30–33 |
| F15 | On terminal completion, `.control/` retains audit copies of assignments, logs, and natural exit markers, while live private runner state and agent workspaces are cleaned up and are not copied there. | S4, lines 111–120 |
| F16 | Human-judgment decisions, malformed or stale artifacts, timeout, cleanup failure, and exhausted candidate budget are among conditions that produce a blocked state with an actionable reason. | S1, lines 41–46; S2, lines 81–95 |
| F17 | The documented implementation is single-run and non-resumable; parallel runs, restart resume, and editing completed runs are not implemented. | S2, lines 5–10, 97–98 |

## Firsthand observations

None. The CLI was not executed and repository behavior was not independently measured. Accordingly, `evidence/firsthand.md` is intentionally absent.

## Inferences

| ID | Claim | Basis and caveat |
| --- | --- | --- |
| I1 | The workflow is inspectable because editorial inputs, candidate versions, lens outputs, PM classifications, state metadata, and audit copies are durable and linked by candidate/revision identifiers. | Synthesis of F11–F15. “Inspectable” is an interpretation of the retained, cross-bound artifacts, not a measured usability result. |
| I2 | Artifact gates reduce the chance that a persuasive agent message, stale review, or incomplete file silently advances the run. | Synthesis of F3, F10, and F11. The sources specify rejection behavior but provide no comparative error-rate measurement. |
| I3 | Sequential lenses make the provenance of a revision easier to trace because the first PM-validated must-fix ends that candidate's remaining review sequence and the next candidate restarts from Evidence. | Synthesis of F7–F9 and F12. “Easier” is a reasoned workflow consequence, not a tested reader outcome. |
| I4 | The three-candidate budget converts otherwise open-ended automated revision into an explicit blocked terminal state requiring a new human-directed run. | Synthesis of F9, F16, and the human role in S3 lines 11–15. |

## Opinions

| ID | Claim | Use constraint |
| --- | --- | --- |
| O1 | The durable artifact gates are a sensible tradeoff for engineers who value auditability over maximum throughput. | Editorial evaluation only; not established by the sources. If used, label clearly as judgment. |

## Unresolved

| ID | Question | Why unresolved |
| --- | --- | --- |
| U1 | How often do real runs succeed, revise, block, or exhaust the candidate budget? | README/docs provide contracts, not operational statistics. |
| U2 | How much time or compute does a typical run consume? | No benchmarks or run measurements are in the allowed sources. |
| U3 | How reliably does the implementation match every documented contract? | No code/test inspection or firsthand execution was performed, and the brief permits only README/docs support. |
| U4 | Does the retained artifact set make review faster or more accurate than another editorial workflow? | No comparative evaluation is present, and workflows outside this repository are out of scope. |

</write-uuter-context>