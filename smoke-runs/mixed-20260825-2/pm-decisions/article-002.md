```json
{
  "reviewed_revision": "sha256:de5cf22227acee3d99de7f8fe6ee985a0fe473d3843a49a66b71d5e076d5b67d",
  "lenses": {
    "evidence": {
      "request_id": "db77cb19e4e1571a3e67e92d6187d0ff",
      "review_digest": "sha256:225da1095aa217add46cfcaa35d815a5902b15dd0ef8d98a83b1fd0b3282ac95",
      "decisions": []
    },
    "story": {
      "request_id": "23420b72174f4c68a679d333c362fe00",
      "review_digest": "sha256:0fed74b67f9acdd2a4fefe5cf614ecc9a077dcae9e2dd3c43fd43d192f133bc3",
      "decisions": [
        {
          "finding_id": "story-001",
          "decision": "invalid",
          "reason": "Defining profile at its actual first body use satisfies the brief's explicit first-use requirement, and the concise definition does not disrupt the lead's framing or the article's sequence."
        }
      ]
    },
    "clarity": {
      "request_id": "8ab43b896b0b89ac42b0a42717f68137",
      "review_digest": "sha256:861966a0357d942dd51cffdbe2ae3c7112b8c0e9123bb392c1e8fc112ce4c5f2",
      "decisions": [
        {
          "finding_id": "clarity-001",
          "decision": "valid_must_fix",
          "reason": "The sentence conflates per-role profile fields with the run-level prompt bundle, creating ambiguity in the article's central explanation of what is fixed at each scope."
        },
        {
          "finding_id": "clarity-002",
          "decision": "valid_optional",
          "reason": "Technical readers can infer a non-zero exit status from the outcome list, but naming the exit explicitly would improve precision."
        },
        {
          "finding_id": "clarity-003",
          "decision": "invalid",
          "reason": "The complete table already maps the code-form provider and role identifiers, and the later capitalized prose names are conventional, unambiguous labels rather than a separate terminology system."
        }
      ]
    }
  }
}
```
