# Story review — article-002

status: fix_required
lens: story
reviewed_revision: sha256:de5cf22227acee3d99de7f8fe6ee985a0fe473d3843a49a66b71d5e076d5b67d

## Summary

The candidate follows the outline's sequence (declaration → enforcement → the
one tunable → why the roles split → what the finished run keeps → close) and
its section-by-section purposes and reader takeaways are fulfilled almost
throughout, including the planned first-use locations for *prompt bundle*,
*candidate*, *revision*, and *lens*. One term-placement deviation from the
outline's explicit plan survives review.

## Findings

id: story-001
severity: low
location: article-002.md lines 6-7 (unheaded lead paragraph, before the "Eight role keys, eight profiles" heading)
problem: The outline's term-introduction plan and Section 2 purpose explicitly reserve the formal, italicized definition of *profile* for Section 2 ("Define prompt bundle and profile at first use"), with Section 1's purpose limited to framing the question and stating that the answer is a file in version control. The candidate instead defines *profile* ("That trio ... is a role's *profile*") inside the lead paragraph, before the "Eight role keys, eight profiles" section. This pulls Section 2's designed definitional work into Section 1, blurring the planned boundary between the lead's framing role and the declaration section's defining role, so Section 2 no longer performs the "first use" introduction the outline assigned it.
suggested_direction: Move the sentence that names and defines *profile* out of the lead paragraph and into the "Eight role keys, eight profiles" section, alongside the *prompt bundle* definition, so both terms are introduced together at their planned location; keep the lead limited to framing the question and stating that model choice is declared rather than decided at runtime.
