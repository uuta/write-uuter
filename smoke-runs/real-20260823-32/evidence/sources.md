# Sources

Accessed 2026-08-23. All sources are controller-staged copies of the repository files named in the brief; no external sources were used.

## README.md

- Location: `context/source-hints/001-README.md`
- Useful sections: introduction; “Run”; “Runtime model.”
- Summary: Defines write-uuter as a Go CLI that turns a Markdown brief into a reviewed article while preserving intermediate evidence, outlines, candidates, reviews, and PM decisions. Describes success and blocked outcomes, the three-candidate hard limit, the controller’s ownership of state transitions and validation, isolated role workspaces, sequential roles and review lenses, and the requirement that a successful `article.md` exactly copy the accepted candidate.
- Useful for: high-level purpose, controller responsibilities, role order, candidate budget, and success/blocked behavior.

## docs/workflow.md

- Location: `context/source-hints/002-workflow.md`
- Useful sections: “Controller sequence”; “Artifact gates”; “Lifecycle and terminal states.”
- Summary: Gives the end-to-end state machine. Research, story editing, writing, and the four review lenses run sequentially. After each lens, the PM classifies findings. A validated must-fix stops later lenses, causes a revision when the candidate number is below 003, and restarts review at Evidence; human judgment or an exhausted budget blocks the run. It also specifies validation gates, timeout and cleanup behavior, final revalidation, atomic publication, and limitations such as no resume or parallel runs.
- Useful for: exact review routing, the significance of durable artifact validation, blocking conditions, and lifecycle guarantees.

## docs/roles.md

- Location: `context/source-hints/003-roles.md`
- Useful sections: “PM”; “Researcher”; “Story Editor”; “Writer”; “Reviewers.”
- Summary: Defines ownership boundaries. The Researcher produces sources and a classified claim ledger; the Story Editor produces an evidence-linked outline; the Writer alone produces candidates; reviewers produce lens-specific results and reports without editing drafts; and the persistent PM classifies findings but neither writes candidates nor reviews. It identifies the context each reviewer receives and states that all workflow transitions remain under Go’s control.
- Useful for: explaining each role, separation of duties, reviewer inputs and outputs, and the distinction between PM judgment and deterministic routing.

## docs/artifacts.md

- Location: `context/source-hints/004-artifacts.md`
- Useful sections: “Run layout”; “Review result”; “PM decision”; “workflow.json.”
- Summary: Documents the durable run tree and validation schemas. Earlier candidates, partial review sequences, reviews, and PM decisions survive revision or blockage. Review JSON binds findings to a lens and candidate hash; PM decisions bind classifications to request IDs and review digests while preserving prior lens outcomes. `workflow.json` records controller state and terminal reasons, `.control/` stores post-cleanup audit copies, and successful publication produces a byte-identical `article.md` only after final gates pass.
- Useful for: the concrete inspectable artifacts, revision binding, audit records, and terminal-state evidence.
