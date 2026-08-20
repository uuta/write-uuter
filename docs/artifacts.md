# Artifacts

Agents hand work off through files with explicit ownership. Chat transcripts
may help diagnose a run, but they are not the source of truth for completion.

## Run layout

```text
article-run/
├── brief.md
├── evidence/
│   ├── sources.md
│   ├── firsthand.md
│   └── assets/
├── claim-ledger.json
├── outline.md
├── article.md
├── reviews/
│   ├── evidence.json
│   ├── story.json
│   ├── clarity.json
│   └── copy.json
├── pm-review-decision.json
├── workflow.json
└── publish-report.md
```

## PM intake gate

The PM may start a run when `brief.md` defines all of the following:

```yaml
question:
audience:
provisional_takeaway:
scope:
out_of_scope:
publication_target:
constraints:
done_when:
source_hints:
```

The takeaway is provisional and may change after investigation. If a required
field is missing or contradictory, the run is `blocked` instead of guessed.

## Review result

Each review result is machine-readable and tied to the exact revision that was
reviewed.

```json
{
  "status": "fix_required",
  "reviewed_revision": "sha256:...",
  "lens": "clarity",
  "findings": [
    {
      "id": "clarity-001",
      "severity": "must_fix",
      "location": "section: Introduction",
      "claim_id": null,
      "problem": "The audience cannot identify the recommended action.",
      "suggestion": "State the action before explaining the background."
    }
  ]
}
```

Allowed review statuses are:

- `running`
- `clean`
- `fix_required`
- `blocked`

## PM review decision

The PM validates findings before routing work back to the Writer or another
phase.

```json
{
  "finding_id": "clarity-001",
  "decision": "valid_must_fix",
  "route_to": "writer",
  "reason": "The article contract requires the action to be visible early."
}
```

## Completion rule

An agent message such as “done” is not sufficient. The PM advances the run only
when the required artifact exists, is valid, has a terminal status where
applicable, and references the current revision.

