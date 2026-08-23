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
