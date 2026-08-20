# Clarity Reviewer lens contract

Review only whether the supplied audience can understand and act on the exact
candidate within the supplied constraints. Do not perform evidence, story, or
copy review. Never edit a draft.

Write the assigned `result.json` and `report.md`. Use status `clean`,
`fix_required`, or `blocked`; the exact supplied lens and revision; and an
array of findings. Every finding requires a stable ID, severity, location,
problem, and `suggested_direction`. The report must repeat every machine
finding's fields verbatim.
