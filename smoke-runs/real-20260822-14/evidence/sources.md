# Sources

Research method: close reading of the controller-staged copies named by the brief. No network research or firsthand execution was performed. Accessed 2026-08-22.

## S1 — Repository README

- Location: `context/source-hints/001-README.md` (staged copy of `README.md`)
- Useful summary: Defines write-uuter as a Go CLI that converts a Markdown brief into a reviewed article while retaining evidence, outline, candidates, reviews, and PM decisions. Summarizes the runtime division of responsibility: Go owns state, validation, hashes, timeouts, cleanup, isolation, and artifact copying; isolated Codex roles perform editorial work. States the role order, sequential fresh reviewers, candidate-003 limit, success behavior, and blocked-run behavior.
- Particularly useful for: the high-level answer, system requirements, controller/agent boundary, success and failure outcomes.

## S2 — Workflow documentation

- Location: `context/source-hints/002-workflow.md` (staged copy of `docs/workflow.md`)
- Useful summary: Gives the exact controller sequence from brief validation and atomic run initialization through research, outline, candidate creation, four review lenses, PM classification, revision, and terminal publication or blocking. Defines artifact validation gates, explains that review lenses are sequential, and states that a must-fix stops later lenses and makes the replacement restart at Evidence. Documents lifecycle, isolation, timeout, cleanup, and terminal revalidation behavior.
- Particularly useful for: sequential review loop, routing rules, artifact gates, failure modes, and candidate budget.

## S3 — Role contracts

- Location: `context/source-hints/003-roles.md` (staged copy of `docs/roles.md`)
- Useful summary: Defines ownership and prohibited actions for the Human Editor, persistent PM, Researcher, Story Editor, Writer, and four reviewers. Specifies the PM's four finding classifications, the context supplied to each reviewer lens, and the fact that reviewers are fresh sequential processes that cannot edit candidates. Explains which durable files each role owns.
- Particularly useful for: role boundaries, PM-versus-controller responsibilities, reviewer inputs, and why artifacts remain attributable and inspectable.

## S4 — Artifact contracts

- Location: `context/source-hints/004-artifacts.md` (staged copy of `docs/artifacts.md`)
- Useful summary: Defines the run-directory layout and validated schemas for reviewer results, PM decisions, and `workflow.json`. States that earlier candidates and partial review sequences are retained, while `article.md` exists only on success and exactly matches the accepted candidate. Documents recursive JSON validation, atomic state/marker writes, audit copies, symlink and regular-file protections, and the rule that editorial completion does not depend on chat transcripts or tmux scrollback.
- Particularly useful for: inspectable terminal artifacts, schema-level validation, retained revision history, and exact publication semantics.

## Source-use boundaries

- These four staged files are the complete source set permitted by the brief.
- Claims about publishing integrations, web interfaces, other repositories, or editorial systems in general are unsupported and outside scope.
- The documentation describes the shipped issue-1 workflow; it explicitly says parallel runs, resume after controller restart, and editing completed runs are not implemented.
