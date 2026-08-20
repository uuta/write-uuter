# Sources

Accessed 2026-08-21. These are the controller-staged copies of the brief's local source hints; research used the staged copies rather than the source repository. No network or firsthand research was performed.

## README.md

- Location: `context/source-hints/001-README.md` (brief hint: `../README.md`)
- Useful summary: Defines write-uuter as a Go CLI that turns a Markdown brief into a reviewed article while preserving the evidence, outline, candidates, reviews, and PM decisions. Describes the runtime division of responsibility: Go owns transitions, validation, revision hashes, timeouts, and cleanup; isolated Codex roles do editorial work. Gives the sequential role order, says only PM-validated must-fix findings trigger revision, establishes candidate 003 as the hard limit, and states success/block outcomes.
- Particularly useful support: overview and requirements; run behavior; required brief headings; runtime model.

## docs/workflow.md

- Location: `context/source-hints/002-workflow.md` (brief hint: `../docs/workflow.md`)
- Useful summary: Gives the controller sequence from brief validation and atomic run initialization through Researcher, Story Editor, Writer, four fresh sequential review lenses, PM classification, revision, and publication. Explains that a must-fix stops later lenses and restarts the next candidate at Evidence, optional/invalid findings do not consume a candidate, and human judgment blocks. Specifies artifact gates, isolation/lifecycle behavior, final revalidation, and supported terminal states.
- Particularly useful support: controller flowchart; artifact validation requirements; timeout and cleanup behavior; non-resumable and non-parallel limitations.

## docs/roles.md

- Location: `context/source-hints/003-roles.md` (brief hint: `../docs/roles.md`)
- Useful summary: Defines ownership and prohibitions for the Human Editor, persistent PM, Researcher, Story Editor, Writer, and four reviewers. Records the PM classification vocabulary, Writer revision inputs, per-lens reviewer context, reviewer output ownership, and the rule that reviewers neither inherit prior-lens conversations nor edit candidates.
- Particularly useful support: role-to-artifact ownership; PM request binding and accumulated decisions; fresh isolated reviewer assignments.

## docs/artifacts.md

- Location: `context/source-hints/004-artifacts.md` (brief hint: `../docs/artifacts.md`)
- Useful summary: Documents the durable run tree and validation contracts for review results, PM decisions, and `workflow.json`. States that prior candidates, partial review sequences, reviews, and PM decisions remain when a run revises or blocks; `article.md` exists only on success and exactly matches the accepted candidate. Explains the post-cleanup `.control/` audit material and which live private state is deliberately not retained.
- Particularly useful support: inspectable terminal layout; exact candidate/article relationship; review and PM-decision integrity checks; atomic state and marker handling.

## Source-use boundaries

- The supplied brief limits factual support to README.md and `docs/`; every factual claim in the ledger below traces to one or more of the four staged sources above.
- No claims were derived from repository implementation files, tests, external sources, or workflows outside this repository.
- Documentation describes the shipped issue-1 workflow; it does not establish comparative effectiveness, editorial quality, or applicability to other systems.
