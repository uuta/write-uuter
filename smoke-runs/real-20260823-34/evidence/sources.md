# Sources

Accessed 2026-08-23. These are controller-staged copies of the repository sources named by the brief; no outside sources were used.

## 1. Repository README

- Location: `context/source-hints/001-README.md` (staged from `README.md`)
- Type: Repository documentation
- Useful summary: Defines `write-uuter` as a Go CLI that turns a Markdown brief into a reviewed article while preserving intermediate evidence, outline, candidates, reviews, and PM decisions. Documents prerequisites, brief validation, prompt resolution, isolated per-role workspaces, controller ownership of transitions and cleanup, sequential role order, the four review lenses, the three-candidate ceiling, and success/blocked outcomes. It states that successful `article.md` is an exact copy of the candidate accepted through all four reviews.
- Most useful sections: opening description; `Run`; `Runtime model`.

## 2. Workflow documentation

- Location: `context/source-hints/002-workflow.md` (staged from `docs/workflow.md`)
- Type: Repository documentation
- Useful summary: Gives the controller sequence from brief validation and atomic run initialization through Researcher, Story Editor, Writer, Evidence/Story/Clarity/Copy review, PM classification, revision, and terminal publication or blocking. Explains that review lenses are sequential, a must-fix stops the remaining lenses and restarts the next candidate at Evidence, candidate 003 is the limit, and human-judgment routing blocks. Defines artifact gates and terminal checks, including worker exit, file validation, hash/revision binding, process cleanup, and atomic publication.
- Most useful sections: `Controller sequence`; `Artifact gates`; `Lifecycle and terminal states`.

## 3. Role contracts

- Location: `context/source-hints/003-roles.md` (staged from `docs/roles.md`)
- Type: Repository documentation
- Useful summary: Assigns ownership and limits for the Human Editor, persistent PM, Researcher, Story Editor, Writer, and four fresh reviewers. The PM classifies findings but does not write candidates or reviews; Go validates decisions and applies routing. The Writer owns one candidate at a time. Reviewers receive lens-specific context, write `result.json` and `report.md`, and cannot edit candidates. The Researcher owns the sources and claim ledger, while the Story Editor owns an outline with purpose, evidence, and reader takeaway.
- Most useful sections: `PM`; `Researcher`; `Story Editor`; `Writer`; `Reviewers`.

## 4. Artifact documentation

- Location: `context/source-hints/004-artifacts.md` (staged from `docs/artifacts.md`)
- Type: Repository documentation
- Useful summary: Specifies the durable run layout and validation contracts for reviewer results, PM decisions, `workflow.json`, and `.control/` audit copies. States that earlier candidates, partial review sequences, reviews, and PM decisions remain available after revision or blocking. Defines strict JSON validation and says `article.md` appears only on success and is byte-for-byte identical to the accepted candidate.
- Most useful sections: `Run layout`; `Review result`; `PM decision`; `workflow.json`.

## Source boundaries

- The brief requires facts to come only from `README.md` and `docs/`; all factual entries in the claim ledger below trace to the four staged copies above.
- No firsthand testing, execution of `write-uuter`, interviews, or independent code inspection was performed. Therefore `evidence/firsthand.md` and `evidence/assets/` were not created.
