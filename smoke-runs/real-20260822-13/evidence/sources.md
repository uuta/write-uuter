# Sources

Access date for all sources: 2026-08-22.

Only the controller-staged copies listed below were consulted. Line references in this research workspace refer to those copies.

## README

- Location: `context/source-hints/001-README.md`
- Original hint: `../README.md`
- Useful summary: Defines write-uuter as a Go CLI that converts a Markdown brief into a reviewed article while preserving its production artifacts (lines 1-5). Describes success and failure behavior, including exact-copy publication, blocked status, and preservation on runtime failure, timeout, candidate exhaustion, or human judgment (lines 31-46). Assigns workflow mechanics to Go—state transitions, validation, revision hashes, timeouts, cleanup, isolation, and sequential role execution—and establishes candidate 003 as the hard limit (lines 68-84).
- Best use: High-level product description, runtime ownership, operating constraints, and terminal outcomes.

## Workflow

- Location: `context/source-hints/002-workflow.md`
- Original hint: `../docs/workflow.md`
- Useful summary: Gives the exact controller sequence from brief validation through research, outline, candidate writing, four ordered review lenses, PM classification after each lens, revision, success, or blockage (lines 3-39). Explains that review lenses never run in parallel, a must-fix stops later lenses and restarts the next candidate at Evidence, optional/invalid findings do not consume candidates, and human judgment blocks (lines 41-43). Enumerates validation gates for research, outlines, candidates, reviews, and PM decisions (lines 45-62). Details lifecycle isolation, cleanup, timeouts, and terminal revalidation/persistence (lines 64-104), and states that resume, parallel runs, and editing completed runs are not implemented (lines 106-107).
- Best use: Sequential control flow, routing rules, validation rationale, isolation, and failure behavior.

## Roles

- Location: `context/source-hints/003-roles.md`
- Original hint: `../docs/roles.md`
- Useful summary: Explains that durable prompt files define roles while Go validates outputs and owns transitions (lines 1-9). Defines the Human Editor and persistent PM, including the PM's four finding classifications and the controller's independent enforcement of routing and candidate limits (lines 11-35). Defines Researcher, Story Editor, and Writer artifact ownership and revision context (lines 37-55). Specifies fresh, sequential Evidence, Story, Clarity, and Copy reviewers, their lens-specific context, immutable candidates, outputs, and filesystem isolation (lines 57-81).
- Best use: Role boundaries, artifact ownership, PM versus controller authority, and reviewer context.

## Artifacts

- Location: `context/source-hints/004-artifacts.md`
- Original hint: `../docs/artifacts.md`
- Useful summary: Shows the durable run-directory layout, including brief, state, evidence, ledger, outline, numbered drafts, per-lens reviews, PM decisions, success-only article, and post-cleanup audit copies (lines 3-28). States that earlier and partial artifacts survive revisions or blocked runs and that the final article is byte-for-byte identical to the accepted candidate (lines 30-33). Documents strict review and PM-decision schemas and rejection conditions (lines 35-100). Defines `workflow.json` as the atomically rewritten source of truth and lists its state fields (lines 102-114); explains what `.control/` retains, what remains private, and why completion does not depend on chat or tmux scrollback (lines 116-129).
- Best use: Inspectability, exact artifact paths, schema-level gates, state tracking, and audit boundaries.

## Source limits

- The brief requires factual support only from `README.md` and `docs/`; this record uses only their staged copies.
- No network research or firsthand investigation was performed.
- The sources describe the repository's implemented issue-1 workflow. They do not support general claims that this design is superior to other editorial systems, nor claims about publishing integrations or web interfaces.
