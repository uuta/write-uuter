# Sources

Accessed: 2026-08-20 (Asia/Tokyo)

The brief's source hints name repository-root files even though the supplied
base for relative hints is `examples/`. There are no `README.md` or `docs/`
files inside `examples/`; the four named files were therefore located at the
repository root, one directory above that base. Only these four allowed files
were used as factual sources.

## S1 — README.md

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/README.md`
- Repository-relative location: `README.md`
- Useful sections: opening description, **Run**, **Runtime model**
- Summary: Defines `write-uuter` as a Go CLI that turns a Markdown brief into a
  reviewed article while retaining the evidence, outline, candidates, reviews,
  and PM decisions behind it. Documents successful output as a non-empty exact
  copy of the candidate accepted by all four reviewers. Documents blocked
  outcomes for runtime failure, timeout, an exhausted third candidate, or need
  for human judgment. States that Go owns transitions, validation, revision
  hashes, timeouts, and cleanup; that one persistent PM and no more than one
  worker run in the dedicated tmux session; that roles and review lenses run
  sequentially; and that candidate 003 is the hard limit.
- Best use: High-level product behavior, operational requirements, success and
  failure semantics, role order, and candidate budget.

## S2 — docs/workflow.md

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/workflow.md`
- Repository-relative location: `docs/workflow.md`
- Useful sections: **Controller sequence**, **Artifact gates**, **Lifecycle and
  terminal states**
- Summary: Describes the shipped controller as single-run and non-resumable.
  Validation precedes atomic run initialization. The PM starts before research;
  Researcher, Story Editor, and Writer produce the inputs and first candidate;
  fresh Evidence, Story, Clarity, and Copy reviewers then run one at a time.
  After each reached lens, the PM classifies findings. A validated must-fix
  stops later lenses, causes a replacement candidate when the budget permits,
  and restarts review at Evidence. Optional or invalid findings do not consume
  a candidate, while human judgment blocks the run. Go advances workers only
  after their owned files exist and satisfy role-specific validation. It
  rejects malformed or stale artifacts, records actionable block reasons,
  kills the dedicated tmux session on terminal exit, and leaves no agent
  process behind. Parallel runs, resume, and edits to completed runs are not
  implemented.
- Best use: Exact sequence, revision routing, validation gates, lifecycle, and
  implementation limits.

## S3 — docs/roles.md

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/roles.md`
- Repository-relative location: `docs/roles.md`
- Useful sections: all role sections, especially **PM**, **Writer**, and
  **Reviewers**
- Summary: Separates ownership among the human editor, a persistent PM,
  Researcher, Story Editor, Writer, and four fresh reviewers. The human owns the
  brief and resolves `needs_human_judgment`. The PM classifies every review
  finding but cannot write candidates or reviews. The Writer owns one assigned
  candidate and cannot classify findings or create the terminal article.
  Reviewers cannot edit candidates and receive only the durable context allowed
  for their lens. Each reviewer process is fresh and inherits neither prior
  reviewers' conversations nor reports. Go validates role output and owns all
  transitions.
- Best use: Role boundaries, artifact ownership, reviewer isolation, and the
  distinction between editorial classification and controller routing.

## S4 — docs/artifacts.md

- Location: `/Users/yutaaoki/write-uuter/.worktrees/1/docs/artifacts.md`
- Repository-relative location: `docs/artifacts.md`
- Useful sections: **Run layout**, **Review result**, **PM decision**,
  **workflow.json**
- Summary: Enumerates the durable run tree, including the brief, workflow
  state, evidence, claim ledger, outline, numbered drafts, per-candidate and
  per-lens review files, PM decisions, final article, and controller-owned
  control files. Earlier candidates, partial review sequences, reviews, and PM
  decisions survive revision or blocking. `article.md` appears only after all
  four lenses pass PM routing and is byte-for-byte identical to the accepted
  candidate. Review JSON, matching reports, PM decision coverage, and workflow
  state all have validated structure. `workflow.json` is atomically rewritten
  as the controller's source of truth. Editorial completion does not depend on
  tmux scrollback or chat transcripts.
- Best use: Inspectable layout, persistence across revisions and blocked runs,
  exact terminal artifact semantics, and machine-validated metadata.

## Cross-source reliability notes

- The four sources agree on the review order, one-worker-at-a-time execution,
  PM classification after each lens, and the three-candidate ceiling.
- `README.md` is the concise behavior overview; the three `docs/` files provide
  the exact workflow, role, and artifact contracts.
- No firsthand execution, source-code inspection, external research, or web
  research was performed. Accordingly, there is no `evidence/firsthand.md` and
  no asset was created.
