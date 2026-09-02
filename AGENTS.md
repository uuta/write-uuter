# Repository Instructions

## Authoring briefs

Start new article briefs from `examples/brief-template.md`. The brief
validator requires `Audience` and `Publication target` sections; a brief
without them fails before any agent runs. The series plan document referenced
by the brief is the source of truth for series-level media design (reader
persona, value, strengths, freshness rules); reusable prose rules live in
`STYLE.md`. A brief may quote the applicable persona, value, and strengths so
the run is self-contained — agents read the brief, not the plan — but keep the
quote in sync with the plan rather than diverging or inventing new
series-level design there.

## Editorial guidance

`STYLE.md` is the source of truth for reusable article-writing conventions.
Changes to the writing workflow must keep it available to both the Writer and
the Copy Reviewer as read-only context.

Human review files such as `r.md` are run-specific. When incorporating their
feedback into the workflow:

1. Keep corrections that depend on one article, sentence, or topic in the run.
2. Promote only feedback that should apply across future articles to
   `STYLE.md`.
3. Express promoted feedback as a general principle. Do not turn a single
   suggested replacement into a mandatory phrase.
