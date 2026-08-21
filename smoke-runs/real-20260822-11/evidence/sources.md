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
