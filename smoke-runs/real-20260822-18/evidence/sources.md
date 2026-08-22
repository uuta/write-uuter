# Sources

Access date for all sources: 2026-08-22. All locations below are controller-staged copies of the brief's local source hints; no source-repository files were read.

## 1. README

- Location: `context/source-hints/001-README.md`
- Original hint: `../README.md`
- Useful summary: Defines `write-uuter` as a Go CLI that converts a Markdown brief into a reviewed article while retaining evidence, outline, candidates, reviews, and PM decisions. Describes runtime ownership: Go controls state transitions, validation, hashes, timeouts, and cleanup; roles receive isolated workspaces and contracted context. States the sequential role/reviewer order, the four review lenses, the hard limit of candidate 003, and success/blocked behavior.
- Particularly useful passages: lines 3-5 (purpose); 43-48 (terminal behavior); 74-94 (runtime model, isolation, sequence, and candidate limit).

## 2. Workflow documentation

- Location: `context/source-hints/002-workflow.md`
- Original hint: `../docs/workflow.md`
- Useful summary: Gives the controller sequence from brief validation and atomic run initialization through research, outlining, drafting, four sequential review lenses, and final publication. Explains that a PM-validated must-fix stops the remaining lenses and restarts the next candidate at Evidence; optional or invalid findings do not consume a candidate; human judgment blocks. Defines artifact gates and terminal revalidation/cleanup behavior.
- Particularly useful passages: lines 5-10 (initialization); 12-39 (full sequence); 41-43 (routing and candidate consumption); 45-62 (artifact gates); 90-112 (timeouts, terminal validation, blocked-state preservation, and implementation limits).

## 3. Roles documentation

- Location: `context/source-hints/003-roles.md`
- Original hint: `../docs/roles.md`
- Useful summary: Assigns ownership and boundaries to the Human Editor, PM, Researcher, Story Editor, Writer, and reviewers. The PM classifies findings but does not write candidates or reviews. The Writer creates exactly one assigned candidate and cannot create the terminal article. Fresh Evidence, Story, Clarity, and Copy reviewers run sequentially, cannot edit candidates, and receive only lens-specific inputs.
- Particularly useful passages: lines 14-18 (human); 20-38 (PM); 40-58 (research, outline, and writing); 60-84 (review order, inputs, outputs, and isolation).

## 4. Artifacts documentation

- Location: `context/source-hints/004-artifacts.md`
- Original hint: `../docs/artifacts.md`
- Useful summary: Specifies the durable run layout and validation contracts for review results, PM decisions, `workflow.json`, and `.control/` audit copies. Earlier candidates, partial review sequences, reviews, and PM decisions remain available after revision or blockage. `article.md` exists only on success and exactly matches the accepted candidate. `workflow.json` is the atomically rewritten controller source of truth and records status, phase, current candidate/revision, paths, attempts, timestamps, and block reason.
- Particularly useful passages: lines 3-33 (run tree and retained artifacts); 35-63 (review result contract); 65-100 (PM decision contract); 102-129 (`workflow.json`, audit artifacts, and independence from chat/tmux scrollback).

## Source-boundary note

These documents describe the repository's implemented workflow and are sufficient for the scoped article. They do not support claims that the design is generally superior to other editorial systems, works outside the documented macOS real-run environment, supports parallel/resumable runs, or includes publishing/web integrations.
