# Sources

Accessed 2026-08-22. All research used the controller-staged copies in
`context/source-hints/`; no source-repository files were read directly and no
network sources were used.

## S1 — README

- Staged location: `context/source-hints/001-README.md`
- Source hint represented: `../README.md`
- Useful summary: Defines write-uuter as a Go CLI that turns a Markdown brief
  into a reviewed article while preserving the evidence, outline, candidates,
  reviews, and PM decisions. Describes brief validation, atomic run creation,
  the runtime division of responsibility, isolated role workspaces, the
  sequential role/reviewer order, the three-candidate ceiling, success as an
  exact copy of the accepted candidate, and blocked terminal behavior.
- Especially useful for: the top-level workflow, controller/agent boundary,
  platform constraints, success and failure semantics, and candidate budget.

## S2 — Workflow

- Staged location: `context/source-hints/002-workflow.md`
- Source hint represented: `../docs/workflow.md`
- Useful summary: Gives the controller sequence from brief validation through
  research, outlining, drafting, four fresh review lenses, PM classification,
  revision, and publication or blocking. Specifies that lenses are sequential,
  a must-fix stops the remaining lenses and restarts review at Evidence on the
  next candidate, optional/invalid findings do not consume a candidate, and
  human judgment blocks. Enumerates validation gates and lifecycle/cleanup
  requirements, including hash binding and failure preservation.
- Especially useful for: sequence, artifact gates, revision routing, terminal
  validation, timeouts, non-resumability, and failure modes.

## S3 — Roles

- Staged location: `context/source-hints/003-roles.md`
- Source hint represented: `../docs/roles.md`
- Useful summary: Defines ownership and prohibitions for the Human Editor, PM,
  Researcher, Story Editor, Writer, and Evidence/Story/Clarity/Copy reviewers.
  Explains the persistent PM protocol, its four finding classifications, the
  context supplied to each reviewer, and the rule that reviewers neither edit
  candidates nor inherit prior reviewers' conversations.
- Especially useful for: explaining each role, separation of duties, reviewer
  context, and why revisions remain attributable.

## S4 — Artifacts

- Staged location: `context/source-hints/004-artifacts.md`
- Source hint represented: `../docs/artifacts.md`
- Useful summary: Defines the durable run layout and validated schemas for
  reviews, PM decisions, and `workflow.json`. States that earlier candidates,
  partial reviews, reviews, and decisions remain available after revision or
  blocking; `article.md` exists only on success and is byte-for-byte identical
  to the accepted candidate. Describes `.control/` audit copies and emphasizes
  that completion does not depend on chat transcripts or tmux scrollback.
- Especially useful for: inspectability, schema-level gates, terminal artifact
  meanings, audit records, and exact-copy publication.

## Source-use boundaries

- The brief limits support to README and `docs/`; S1–S4 are the complete source
  set used here.
- Claims about publishing integrations, web interfaces, or workflows outside
  this repository are unsupported and excluded.
- The documentation describes the shipped contracts; this research did not run
  the CLI or independently test implementation behavior.
