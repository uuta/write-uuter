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
└── .control/                         # generated prompts and lifecycle files
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
within the result, all finding fields are non-empty, lens/revision must match
the assignment, and `report.md` repeats every finding field.

## PM decision

`pm-decisions/article-00N.md` contains one fenced JSON object. Its `lenses` map
accumulates only the lenses reached for that candidate:

```json
{
  "reviewed_revision": "sha256:...",
  "lenses": {
    "evidence": [],
    "story": [
      {
        "finding_id": "story-001",
        "decision": "invalid",
        "reason": "The candidate already satisfies the outline."
      }
    ]
  }
}
```

Each reached lens covers every finding exactly once. Unknown, duplicate, or
missing IDs and stale revisions fail the contract.

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

The `.control/` directory is controller-owned. It preserves generated prompt
assignments for audit, per-invocation logs, the launcher, exit markers when
agents exit naturally, and the transient PM request while one is active.
Editorial completion never depends on tmux scrollback or chat transcripts.
