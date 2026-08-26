# write-uuter

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article
while preserving the evidence, outline, candidates, reviews, and PM decisions
that produced it.

## Requirements

- Go 1.26 or later to build
- cgo enabled plus the Xcode Command Line Tools to build for macOS; Darwin
  no-cgo builds are rejected because audit-token signaling is mandatory
- tmux on the host
- an installed and authenticated Codex CLI for real runs whose policy uses
  `codex`
- an installed Claude Code CLI logged in to a Claude Max subscription for real
  runs whose policy uses `claude_code`; API-key, Bedrock, Vertex, and Foundry
  sessions are refused
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

The tests build the CLI and execute it as a subprocess against deterministic
fake Codex and Claude Code executables inside real tmux sessions. The two
fixtures are selected separately through `--codex` and `--claude`. The tests do
not require network access, Codex authentication, or a Claude subscription.

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
--claude <path>      Claude Code executable (default: claude)
--tmux <path>        tmux executable (default: tmux)
--timeout <duration> Per-agent artifact timeout (default: 10m)
--prompts-dir <path> Checked-in prompt directory; an explicit value is binding
```

`--codex`, `--claude`, and `--tmux` fall back to a `PATH` lookup only when the
flag is omitted. Passing the flag with an empty value (`--codex=`) is rejected
before the run is initialized rather than silently using the default. Only the
providers the validated policy references are resolved and staged, so a policy
that never selects a provider does not require that CLI to be installed.

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

## Model policy

Every prompt bundle must contain `models.json`, which declares the agent
backend, model, and reasoning effort for every role. It is bound through the
same stable no-follow boundary as the role prompts, so the checked-in bundle
and an explicit `--prompts-dir` bundle are each a complete policy. There is no
implicit default, no shared reviewer profile, no runtime routing, and no
fallback.

| Role | Provider | Model | Effort |
| --- | --- | --- | --- |
| `pm` | `codex` | `gpt-5.6-sol` | `high` |
| `researcher` | `claude_code` | `claude-sonnet-5` | `medium` |
| `story_editor` | `claude_code` | `claude-opus-5` | `high` |
| `writer` | `claude_code` | `claude-opus-5` | `medium` |
| `reviewer_evidence` | `codex` | `gpt-5.6-sol` | `medium` |
| `reviewer_story` | `claude_code` | `claude-sonnet-5` | `medium` |
| `reviewer_clarity` | `claude_code` | `claude-sonnet-5` | `medium` |
| `reviewer_copy` | `codex` | `gpt-5.6-luna` | `low` |

`Human Editor` is a human role and has no profile. Schema version 1 supports
exactly the eight role keys above and the providers `claude_code` and `codex`.
Accepted `reasoning_effort` values are `minimal`, `low`, `medium`, and `high`
for `codex`, and `low`, `medium`, `high`, `xhigh`, and `max` for `claude_code`.
There is no global model allowlist: exact availability is decided by the
selected CLI, which blocks the run instead of substituting another model.

The following are rejected before the run directory is created and before tmux
starts: a missing or empty policy, an unsupported `schema_version`, a duplicate
JSON key, a missing or unknown role, an unknown field, an unsupported provider,
an empty model, an effort the provider does not accept, a `claude-*` model on
`codex`, and a `gpt-*` model on `claude_code`.

Evaluating a different policy means running a different version-controlled
prompt bundle. Per-run model overrides, automatic routing, and dynamic fallback
are not implemented.

### Claude Max preflight

When the validated policy references `claude_code`, the controller runs a
sanitized `claude auth status` before creating the run directory, starting
tmux, or launching any agent, using the same environment filtering as the
invocations themselves. The run continues only for a logged-in `claude.ai`
session on a `max` subscription. A missing executable, non-zero exit, malformed
response, logged-out state, API-key session, or non-Max subscription fails with
an actionable error and creates nothing. The account identity in the response
is never read into the run or recorded anywhere; the preflight is skipped
entirely when no role uses `claude_code`.

Claude invocations are non-interactive and use `--print`, `--safe-mode`,
`--dangerously-skip-permissions`, `--no-session-persistence`, and explicit
`--model`/`--effort`, with the prompt on stdin. `--bare` is never used: it
disables OAuth and keychain reads and accepts only an API key. Codex
invocations pass an explicit `--model` and
`--config model_reasoning_effort="<effort>"` alongside the existing
`exec --ephemeral --ignore-user-config` boundary.

`ANTHROPIC_API_KEY` and every alternative API, Bedrock, Vertex, Foundry, or
provider-selection credential is removed from provider child environments, so
an ambient credential cannot move a run to API billing or another provider.

Claude invocations keep the real `HOME`, because the Max session is resolved
from the user's account record; only `CLAUDE_CODE_TMPDIR` is redirected, to a
run-owned scratch directory that is removed with the run. The OS sandbox, not
the environment, is the boundary: the exact staged Claude client for that one
invocation is the only process path allowed to read `~/.claude.json` and to
start the system keychain client on the narrowly granted keychain path, and
`--safe-mode` stops the client loading non-managed customizations. Everything
else the user owns - the home directory, `~/.claude` and its settings, history,
plugins, skills, hooks, MCP configuration, sessions, and projects, and the
admin-managed settings tree - stays denied to the client. A model-invoked tool
is a different process path, so it can read neither the account record nor the
keychain, cannot start the keychain client, and cannot reach the user's home.

The brief requires these case-insensitive level-two headings; all except
`Source hints` need non-whitespace content:

```text
Question, Audience, Provisional takeaway, Scope, Out of scope,
Publication target, Constraints, Done when, Source hints
```

Relative source-hint paths are resolved from the directory containing the
input brief. The process working directory is the content root. If present,
`STYLE.md`, `style-guide.md`, or `docs/style-guide.md` under that root is staged
only for the Writer and Copy reviewer; prompt bundles may live elsewhere.

## Runtime model

Go owns state transitions, validation, revision hashes, timeouts, and process
cleanup. A provider-neutral runner gives Codex and Claude Code invocations the
same immutable role/task prompt, workspace boundary, timeout and cancellation
signal, and audit identity. One long-lived PM process and at most one worker
run in a dedicated tmux session. Each process receives a separate controller-created
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
