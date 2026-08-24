# Clarity review — article-002

Status: fix_required
Lens: clarity
Reviewed revision: sha256:de5cf22227acee3d99de7f8fe6ee985a0fe473d3843a49a66b71d5e076d5b67d

## Findings

### clarity-001

id: clarity-001
severity: major
location: article-002.md:48-49
problem: The sentence "Provider, model, and prompt bundle are fixed per role" lists prompt bundle alongside provider and model as if it were a per-role attribute. But the article's own earlier definition (lines 13-15) describes the prompt bundle as one version-controlled set of role prompts plus models.json that an entire run is launched from, not something assigned separately per role. A reader who just learned that definition cannot reconcile it with this sentence: is there one bundle per role, or one bundle for the whole run that happens to be fixed? The sentence undermines the reader's mental model of what varies at the role level versus the run level.
suggested_direction: Rephrase to keep the per-role claim limited to provider and model, and state separately that the run as a whole draws from one fixed prompt bundle, so the two scopes (per-role profile vs. per-run bundle) stay distinct.

### clarity-002

id: clarity-002
severity: major
location: article-002.md:80-81
problem: "Records are kept for successful, blocked, timed-out, and non-zero invocations" uses "non-zero" as a bare modifier with no noun to attach to. The reader cannot tell what is non-zero (an exit code is the likely intent, given the surrounding process-outcome list) versus, say, a count of invocations, so the fourth outcome category in this list is not actually understandable as written.
suggested_direction: Name what is non-zero explicitly, e.g. "invocations that exit non-zero" or "non-zero-exit invocations," so the outcome category reads the same way as "blocked" and "timed-out."

### clarity-003

id: clarity-003
severity: minor
location: article-002.md:61-65
problem: This section switches to capitalized prose names — "Codex", "Claude Code", "PM", "Researcher", "Story Editor", "Writer" — without tying them back to the schema identifiers introduced earlier: the provider values `codex`/`claude_code` (lines 16-17) and the role keys `pm`/`researcher`/`story_editor`/`writer` (lines 21-24). The mapping is inferable but not stated, adding avoidable friction in a piece whose stated point is that the policy is precisely inspectable.
suggested_direction: On first use in this section, connect the prose names to their schema identifiers (e.g. "Writer (the `writer` role)" or "Codex (the `codex` provider)") so the reader isn't left to infer the correspondence.
