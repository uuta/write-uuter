# Story review

**Lens:** story
**Reviewed revision:** sha256:a4adaf177675f0fd9aba0587033d70f57e473f8f6eba503b88551d5d808c3272
**Status:** fix_required

## Summary

The candidate follows the outline's sequence logic exactly (declaration →
enforcement → the one tunable → why the roles split → what the run keeps →
close) and each section fulfills its stated purpose and reader takeaway:
Section 2 delivers the complete eight-role table and defines "prompt bundle"
and "profile"; Section 3 shows the policy is enforced rather than advisory;
Section 4 introduces "candidate" and "revision" at the natural moment
("nothing shifts mid-run"); Section 5 introduces "lens" and keeps the
role-separation claim appropriately modest (no invented rationale for why
these eight roles or these effort values were chosen, per the outline's
explicit instruction); Section 6 closes the loop opened in Section 1 with the
preserved-artifacts facts; Section 7 restates the takeaway without
introducing new facts. One finding survives review: a term-introduction
sequencing gap.

## Findings

- id: story-001
  severity: minor
  location: Section 1 (opening paragraph), sentence "There is no implicit default, no shared reviewer profile, no runtime routing, and no fallback."
  problem: The outline's term-introduction plan assigns "profile" its first, plain-language gloss in Section 2 ("Each role gets its own *profile*: one declared provider, one model, one reasoning effort"), but the article's Section 1 already uses "profile" ("no shared reviewer profile") one section earlier, before the term has been introduced or explained. This breaks the planned term-introduction sequence and briefly hands the reader an unglossed repository-specific term ahead of its defined first use.
  suggested_direction: Either drop "profile" from the Section 1 sentence and rephrase without the term (e.g. "no shared reviewer default"), or move a brief inline gloss of "profile" up to Section 1 so the term is explained at its true first use, keeping Section 2's fuller definition as reinforcement.
