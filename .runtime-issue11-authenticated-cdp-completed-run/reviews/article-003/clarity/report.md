# Clarity review — article-003

Status: fix_required
Lens: clarity
Reviewed revision: sha256:efa0440b31351a5db3a46d5c408d68220e6a9f38d42cd9463ec62dc06f665007

## Findings

id: clarity-001
severity: major
location: drafts/article-003.md, section 'What the page shows': the paragraph beginning 'The paragraph is the whole answer.' immediately followed by the shot-001 screenshot
problem: The screenshot (opp-002-page-screenshot) is placed directly after a paragraph that already paraphrases its full content in prose ('the domain is for use in documentation examples without needing permission, and it advises against use in operations'), which is a close restatement of the exact visible text the screenshot shows ('This domain is for use in documentation examples without needing permission. Avoid use in operations.'). Comparing drafts/article-003-prose.md to drafts/article-003.md, this paragraph is byte-for-byte unchanged between the pre-visual prose and the version with the screenshot placed — the manifest's character reduction (1638→1509) is entirely attributable to opp-001's diagram, not to any shortening here. The visual therefore duplicates its own explanation instead of letting the reader verify claims the prose only summarizes, and the surrounding prose was left to repeat the visual in full rather than being shortened or reorganized as the lens contract requires.
suggested_direction: Shorten the 'The paragraph is the whole answer' sentence to its interpretive point (e.g. that permission is granted without asking, and operational use is discouraged) and let the screenshot carry the literal wording, or move the exact quoted sentence into the screenshot's caption instead of restating it in running prose beforehand.

## Notes

The opp-001 mermaid diagram (redirect provenance) is a positive example of the required pattern: the paragraph it accompanies was measurably shortened (the 302/Location-header sentence and the percent-encoding detail were removed from prose and moved into the diagram), and the diagram's edge labels stay within the audience's technical vocabulary (302, Location header) appropriate for the stated technical audience. No issues found there.

The opp-003 'none' decision (enumeration sentence not converted to a list, not treated as redundant with the screenshot) is reasonable as reasoned in the plan, since it names elements while the image evidences them, and the sentence sits at a different point in the section than the paragraph flagged above.
