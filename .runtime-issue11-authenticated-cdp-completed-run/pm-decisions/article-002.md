```json
{
  "reviewed_revision": "sha256:0ee2c2b661745bcb46b7453480c0062550288ec27b59a9ee2af1c499558b2bf5",
  "lenses": {
    "evidence": {
      "request_id": "7b2b6ae2b7020b2d896eada151680d05",
      "review_digest": "sha256:3909f50a944accb836885c1be872ffccd065a95214c3eae52a15e0a45e887785",
      "decisions": [
        {
          "finding_id": "evidence-001",
          "decision": "valid_must_fix",
          "reason": "The blank placed image does not show the heading, paragraph, or link asserted by the article and therefore fails the brief's required visible screenshot evidence; the validated capture must replace it and the revision-bound metadata must match the placed file."
        },
        {
          "finding_id": "evidence-002",
          "decision": "valid_must_fix",
          "reason": "The diagram explicitly labels a shortened URL with a different query-parameter value as the requested URL, conflicting with the exact supplied redirect URL and weakening the required provenance distinction; it must use the exact value or identify the rendering as schematic."
        }
      ]
    }
  }
}
```
