# Artifacts

## Run layout

```text
<run-dir>/
├── brief.md
├── workflow.json
├── evidence/
│   ├── sources.md
│   ├── firsthand.md                 # optional
│   └── assets/
├── claim-ledger.md
├── outline.md
├── drafts/
│   ├── article-001.md
│   └── article-00N.md
├── reviews/
│   └── article-00N/
│       ├── evidence/{result.json,report.md}
│       ├── story/{result.json,report.md}
│       ├── clarity/{result.json,report.md}
│       └── copy/{result.json,report.md}
├── pm-decisions/
│   └── article-00N.md
├── article.md                        # success only
└── .control/                         # post-cleanup audit copies
```

Earlier candidates, partial lens sequences, reviews, and PM decisions are kept
when a revision occurs or a run blocks. `article.md` is written only after all
four final-candidate lenses pass PM routing and is byte-for-byte identical to
that candidate.

## Review result

`result.json` has this validated minimum shape:

```json
{
  "status": "fix_required",
  "lens": "clarity",
  "reviewed_revision": "sha256:...",
  "findings": [
    {
      "id": "clarity-001",
      "severity": "must_fix",
      "location": "section: Introduction",
      "problem": "The requested action is not visible.",
      "suggested_direction": "State the action before the background."
    }
  ]
}
```

Allowed statuses are `clean`, `fix_required`, and `blocked`. A clean result has
no findings; `fix_required` has at least one. IDs are non-empty and unique
within the result, all finding fields contain non-whitespace text,
lens/revision must match the assignment, and `report.md` contains one exact,
complete five-field entry per JSON finding in the same order.
The controller rejects unknown fields and duplicate object keys recursively,
including keys inside every finding object; JSON's usual last-value-wins
behavior is never used for routing artifacts.

## PM decision

`pm-decisions/article-00N.md` contains one fenced JSON object. Its `lenses` map
accumulates only the lenses reached for that candidate:

```json
{
  "reviewed_revision": "sha256:...",
  "lenses": {
    "evidence": {
      "request_id": "request-token",
      "review_digest": "sha256:...",
      "decisions": []
    },
    "story": {
      "request_id": "request-token",
      "review_digest": "sha256:...",
      "decisions": [
        {
          "finding_id": "story-001",
          "decision": "invalid",
          "reason": "The candidate already satisfies the outline."
        }
      ]
    }
  }
}
```

Each reached lens covers every finding exactly once. Unknown, duplicate, or
missing IDs, stale revisions, mismatched request IDs or review digests, dropped
prior lenses, changed prior classification lists, changed prior routing
outcomes, multiple fenced documents, and prepopulated future lenses fail the
contract.
Unknown fields and duplicate keys are rejected recursively in the top-level
document, each lens record, and every decision object.

## workflow.json

`workflow.json` is atomically rewritten and is the controller's source of
truth. Schema version 1 records:

- `status`: `running`, `succeeded`, or `blocked`;
- `phase`: current controller phase;
- `current_candidate` and `current_revision` (`sha256:<hex>`);
- `active_role`;
- stable relative `artifact_paths`;
- `review_attempt_count` (one per reviewer process);
- `started_at`, `updated_at`, and terminal `completed_at` timestamps;
- terminal `block_reason` when blocked.

The durable `.control/` directory is controller-owned and preserves audit
copies of generated prompt assignments, per-invocation logs, and exit
markers. The live runner executable, sandbox profiles, ready/ownership records,
PM requests, and agent workspaces exist only in a controller-private sibling
directory. Worker `0` markers record natural successful exits. The persistent
PM is controller-terminated after final validation, so its durable `143`
marker is explicitly controller-synthesized cleanup evidence rather than a
natural PM exit; interrupted workers likewise retain a nonzero terminal marker.
Private state is removed after verified cleanup of controller-launched and
controller-trackable tmux/process identities and is never copied into
`.control/`. An intentionally ancestry-escaping hostile process is outside
this slice's guarantee; complete containment is deferred to a future
container/VM design. Completion markers are atomically renamed into
place; a partial marker cannot advance the workflow. Controller artifact access
rejects symlink path components, symlinked tree roots/directories, and
non-regular files. Editorial completion never depends on tmux scrollback or
chat transcripts.
