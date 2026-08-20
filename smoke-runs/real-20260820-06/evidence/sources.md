# Sources

Accessed: 2026-08-20

The brief's relative source hints were resolved from
`/Users/yutaaoki/write-uuter/.worktrees/1/examples`, the directory containing
the supplied example brief. All four sources are repository documentation and
fall within the brief's allowed source set.

## S1 — Repository README

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/README.md`
- Repository-relative location: `README.md`
- Access date: 2026-08-20
- Useful summary: Defines write-uuter as a Go CLI that turns a Markdown brief
  into a reviewed article while retaining the evidence, outline, candidates,
  reviews, and PM decisions. It gives the runtime ordering, isolation model,
  success and blocked outcomes, and the hard limit of candidate 003.
- Especially useful passages: lines 3–5 (purpose); 36–41 (terminal behavior);
  52–61 (brief validation and source-hint resolution); 63–73 (controller
  responsibilities, isolated workspaces, sequential roles, and candidate
  budget).

## S2 — Workflow documentation

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/workflow.md`
- Repository-relative location: `docs/workflow.md`
- Access date: 2026-08-20
- Useful summary: Supplies the full controller sequence and routing rules. It
  explains that reviews are sequential, a validated must-fix ends the current
  candidate's remaining lenses and restarts the next candidate at Evidence,
  optional or invalid findings do not spend a candidate, and human judgment
  blocks. It also specifies artifact validation, process lifecycle, cleanup,
  and terminal revalidation.
- Especially useful passages: lines 5–10 (safe initialization); 12–43
  (sequence and routing); 45–61 (artifact gates); 63–81 (isolation, timeouts,
  cleanup, terminal checks); 83–84 (explicitly unimplemented features).

## S3 — Role contracts

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/roles.md`
- Repository-relative location: `docs/roles.md`
- Access date: 2026-08-20
- Useful summary: Assigns ownership and boundaries to the Human Editor, PM,
  Researcher, Story Editor, Writer, and four reviewers. It details PM finding
  classifications, the inputs and outputs of each role, and the deliberately
  limited context supplied to fresh reviewer processes.
- Especially useful passages: lines 3–7 (controller-owned transitions and
  durable prompts); 9–30 (human and PM duties); 32–48 (research, outline, and
  writing ownership); 50–69 (review order, inputs, outputs, and isolation).

## S4 — Artifact contracts

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/artifacts.md`
- Repository-relative location: `docs/artifacts.md`
- Access date: 2026-08-20
- Useful summary: Defines the inspectable run-directory layout and the
  validated formats for reviewer results, PM decisions, and workflow state. It
  states which intermediate artifacts remain after revision or blocking and
  that a successful `article.md` is byte-for-byte identical to the accepted
  candidate.
- Especially useful passages: lines 3–33 (layout, preservation, exact-copy
  publication); 35–59 (review result contract); 61–92 (accumulating PM
  decision contract); 94–115 (`workflow.json`, `.control/`, and independence
  from transient tmux/chat state).

## Source limitations

- These files describe the repository's shipped contracts; this research did
  not independently execute or benchmark the CLI.
- The sources do not establish comparative quality, productivity, or cost
  advantages over other editorial workflows.
- No external sources were consulted, in keeping with the brief's constraint
  to use only `README.md` and `docs/`.
