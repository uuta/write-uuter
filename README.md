# write-uuter

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article
while preserving the evidence, outline, candidates, reviews, and PM decisions
that produced it.

## Requirements

- Go 1.26 or later to build
- tmux on the host
- an installed and authenticated Codex CLI for real runs

No third-party Go packages are used.

## Build and test

```sh
make build
make verify
```

The tests build the CLI and execute it as a subprocess against a deterministic
fake Codex executable inside real tmux sessions. They do not require network
access or Codex authentication.

## Run

From the repository root:

```sh
./bin/write-uuter run \
  --brief examples/brief.md \
  --run-dir smoke-runs/my-run
```

The run directory must not already exist. Its parent is created when needed.
On success the command exits 0 and `smoke-runs/my-run/article.md` is a non-empty
exact copy of the candidate accepted by all four reviewers. A runtime failure,
timeout, exhausted third candidate, or need for human judgment exits non-zero
and preserves a `workflow.json` whose status is `blocked`. Invalid briefs and
existing targets exit non-zero without creating or changing a run.

Operational options:

```text
--codex <path>       Codex executable (default: codex)
--tmux <path>        tmux executable (default: tmux)
--timeout <duration> Per-agent artifact timeout (default: 10m)
--prompts-dir <path> Checked-in prompt directory (normally auto-detected)
```

The brief requires these case-insensitive level-two headings; all except
`Source hints` need non-whitespace content:

```text
Question, Audience, Provisional takeaway, Scope, Out of scope,
Publication target, Constraints, Done when, Source hints
```

Relative source-hint paths are resolved from the directory containing the
input brief.

## Runtime model

Go owns state transitions, validation, revision hashes, timeouts, and process
cleanup. One long-lived PM Codex process and at most one worker run in a
dedicated tmux session. Each process receives a separate controller-created
workspace outside the durable run directory; Go copies in only the role's
contracted context and copies validated regular-file outputs back. Researcher,
Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy reviewer
processes run sequentially. Reviewers never receive the run directory or edit
candidates. Only PM-validated must-fix findings create a new candidate;
candidate 003 is the hard limit.

See [workflow](docs/workflow.md), [roles](docs/roles.md), and
[artifacts](docs/artifacts.md) for the exact shipped contracts.
