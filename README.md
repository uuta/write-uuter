# write-uuter

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article
while preserving the evidence, outline, candidates, reviews, and PM decisions
that produced it.

## Requirements

- Go 1.26 or later to build
- cgo enabled plus the Xcode Command Line Tools to build for macOS; Darwin
  no-cgo builds are rejected because audit-token signaling is mandatory
- tmux on the host
- an installed and authenticated Codex CLI for real runs
- macOS for real runs; the controller uses the native Seatbelt sandbox to
  enforce role filesystem isolation

No third-party Go packages are used.

Linux cross-builds are supported, but Linux execution fails closed until an
equivalent native read-isolation backend is implemented.

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
--prompts-dir <path> Checked-in prompt directory; an explicit value is binding
```

`--codex` and `--tmux` fall back to a `PATH` lookup only when the flag is
omitted. Passing the flag with an empty value (`--codex=`) is rejected before
the run is initialized rather than silently using the default.

The prompt bundle is resolved once, before the run starts, in this order:

```text
1. --prompts-dir <path>          explicit and binding; an invalid or empty
                                 value fails the run and never falls back
2. $WRITE_UUTER_PROMPTS_DIR      ambient override; outranks both bundles below
3. <working directory>/prompts
4. <executable directory>/prompts, ../prompts,
   ../share/write-uuter/prompts
```

The first candidate that contains every required prompt as a regular
no-follow file wins. That directory and each prompt file are then held open
for the whole run, so replacing the directory, one of its ancestors, or a
prompt file afterwards cannot change the content the controller uses.

The brief requires these case-insensitive level-two headings; all except
`Source hints` need non-whitespace content:

```text
Question, Audience, Provisional takeaway, Scope, Out of scope,
Publication target, Constraints, Done when, Source hints
```

Relative source-hint paths are resolved from the directory containing the
input brief. The process working directory is the content root. If present,
`STYLE.md`, `style-guide.md`, or `docs/style-guide.md` under that root is staged
only for the Copy reviewer; prompt bundles may live elsewhere.

## Runtime model

Go owns state transitions, validation, revision hashes, timeouts, and process
cleanup. One long-lived PM Codex process and at most one worker run in a
dedicated tmux session. Each process receives a separate controller-created
workspace outside the durable run directory; Go copies in only the role's
contracted context and copies validated regular-file outputs back. A native
sandbox denies agents access to the durable run, other role workspaces, and
controller-private launch state. Codex's inner sandbox is disabled because
macOS does not support nesting it inside the stricter controller sandbox. Go
owns each invocation with native stable process identities (pidfds on Linux and
audit tokens on macOS). The macOS sandbox permits forks only from the staged
single-use Codex client. Only the original host `sandbox-exec` transition may
enter that client; sandboxed descendants cannot execute either the privileged
client or `sandbox-exec` again. Model-invoked runtimes therefore cannot reacquire
client authority or double-fork out of controller ownership; a
controller-private manifest tracks the remaining descendants.
The controller requires a live PM/worker handshake and publishes completion
through an atomic marker after controller-launched and controller-trackable
descendants are gone. An intentionally ancestry-escaping hostile process is
outside this slice's guarantee; complete containment is deferred to a future
container/VM design.
Researcher, Story Editor, Writer, then fresh Evidence, Story, Clarity, and Copy
reviewer processes run sequentially. Reviewers never receive the run directory
or edit candidates. Only PM-validated must-fix findings create a new candidate;
candidate 003 is the hard limit.

See [workflow](docs/workflow.md), [roles](docs/roles.md), and
[artifacts](docs/artifacts.md) for the exact shipped contracts.
