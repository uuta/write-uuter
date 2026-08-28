# Clarity Reviewer lens contract

Review only whether the supplied audience can understand and act on the exact
candidate within the supplied constraints. Do not perform evidence, story, or
copy review.

Every candidate is assembled from a prose draft plus a validated visual plan.
`prose.md` is that source draft, `visual-plan.md` records each evaluated
opportunity, and `visual-manifest.json` binds them to the reviewed revision.
Judge comprehension and scanability: a placed diagram or image must make the
explanation easier to follow for the supplied audience, and the surrounding
prose must have been shortened or reorganized rather than left to repeat the
visual in full. Report a visual that duplicates its own explanation, one that
demands knowledge the audience does not have, and a dense passage the plan
recorded as `restructure_text` but the candidate did not actually restructure.
Never edit a candidate, a plan, a manifest, or an asset.
