# Evidence Reviewer lens contract

Review only whether material claims in the exact candidate are supported and
faithful to the supplied sources, firsthand record when present, and claim
ledger. Do not perform story, clarity, or copy review.

When `evidence/screenshots.json` is supplied, the validated images are staged
read-only under `context/evidence/assets/screenshots/`. A valid PNG proves
nothing about content. Open each screenshot the candidate relies on and reject
it when the image is blank, a loading or skeleton state, an error or consent
page, or otherwise does not visibly contain the information named by its
`supports` claim and the surrounding prose. Also reject a screenshot the
candidate describes inaccurately. Never edit a screenshot or the manifest.

Every candidate is assembled from a prose draft plus a validated visual plan.
`visual-plan.md` records each evaluated opportunity and `visual-manifest.json`
binds the plan, the source prose, the article, and every placed asset to the
reviewed revision. Verify the factual claims a visual asserts: read each
Mermaid diagram as a statement about relationships, sequences, hierarchies, or
state changes, and reject one whose steps, direction, or labels contradict the
claim ledger, the supplied sources, or the candidate's own prose. Placed images
are staged read-only at the same relative path the candidate references; open
each one and reject an image that does not visibly show what the surrounding
explanation and its alt text claim. Never edit a candidate, a plan, a manifest,
or an asset.


# Shared reviewer output contract

Use only the files under the supplied `context/` directory. Do not inspect its
parent, another workspace, the source repository, `.control/`, `reviews/`, PM
decisions, logs, or another lens's output. Never edit `context/article.md`.
The `context/` directory contains every permitted input and no other run
artifact. Write only `result.json` and `report.md` in the workspace root. Use
status `clean`, `fix_required`, or `blocked`; the exact supplied lens and
revision; and an array of findings. Every finding requires a stable ID,
severity, location, problem, and `suggested_direction`. The report must repeat
every machine finding field verbatim.

For each finding, use these five labels in this order (bullets and blank lines
between fields are optional): `id`, `severity`, `location`, `problem`, and
`suggested_direction`. Do not split a field value across lines.

The JSON field name for the revision is exactly `reviewed_revision` (never
`revision`). Use this exact shape, retaining the finding objects only when
there are findings:

```json
{
  "status": "clean",
  "lens": "evidence",
  "reviewed_revision": "sha256:the-exact-assigned-revision",
  "findings": []
}
```

Before exiting, re-read `result.json` and verify that it contains all four
top-level keys: `status`, `lens`, `reviewed_revision`, and `findings`.


## Assignment

Lens: `evidence`
Candidate: `article-001`
Revision: `sha256:bd6977253b6a09af21f278ae75afb6facd9e1e5a0f724a0cc6e7e18a5df5f47e`

## Provided context: brief.md

<write-uuter-context name="brief.md">
# Brief

## Question

What is Example Domain for, and what does its public page visibly say?

## Audience

Technical readers who need a short, source-backed explanation of the reserved example page.

## Provisional takeaway

Example Domain is a deliberately simple public page intended for documentation examples, and its visible wording makes that purpose clear.

## Scope

Describe only the purpose and visible content of the Example Domain page reached through the supplied redirect URL.

## Out of scope

Domain-name history, ownership speculation, browser-provider implementation details, and claims not visible in the supplied source.

## Publication target

A concise standalone article under 350 words.

## Constraints

Use the supplied public sources. Screenshot evidence is required: the Researcher must request exactly one screenshot using ID `shot-001` at `https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F`, supporting the claim that the destination visibly identifies itself as Example Domain and permits use in documentation examples. Do not replace that requested URL with the destination URL. The Visual Editor must inspect the captured pixels against that request and either place the screenshot where it supports the explanation or record an explicit request-keyed rejection.

## Done when

The article accurately states what the page visibly says, the retained screenshot evidence truthfully distinguishes requested and observed final URLs, and the visual decision is explicit and revision-bound if placed.

## Source hints

- https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F
- https://example.com/

</write-uuter-context>

## Provided context: claim-ledger.md

<write-uuter-context name="claim-ledger.md">
# Claim Ledger

Classifications used: Fact, Firsthand observation, Inference, Opinion, Unresolved.

## claim-001
- Statement: The supplied URL `https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F` returns an HTTP 302 redirect whose Location header is `https://example.com/`.
- Classification: Firsthand observation
- Basis: S1. Observed directly via the fetch tool, which reported the redirect status and target rather than silently following it.

## claim-002
- Statement: Following that redirect leads to the page at https://example.com/.
- Classification: Firsthand observation
- Basis: S1 + S2. The redirect target was independently fetched at S2 and returned content, confirming the destination is reachable and is the one named in the redirect, not a substituted URL.

## claim-003
- Statement: The example.com page displays a heading reading "Example Domain".
- Classification: Firsthand observation
- Basis: S2.

## claim-004
- Statement: The page's visible body text states that the domain is for use in documentation examples and does not require permission to use, and it advises against use in "operations."
- Classification: Firsthand observation
- Basis: S2. Caveat: extracted through an AI-mediated fetch/summarization step rather than raw HTML retrieval, so exact sentence boundaries/punctuation are tool-reported. The substance (documentation-example use, no permission needed) is corroborated by the domain's long-established, widely-documented public purpose.
- Screenshot support: shot-001 (requested) is intended to let the Visual Editor independently confirm this claim against captured pixels.

## claim-005
- Statement: The page includes a "Learn more" link to further information (described by the fetch tool as pointing to IANA).
- Classification: Firsthand observation
- Basis: S2.

## claim-006
- Statement: example.com is a domain reserved by IANA for illustrative/documentation use, historically associated with RFC 2606 reserved names.
- Classification: Inference
- Basis: Not independently verified in this run (out of scope per brief: "domain-name history" is excluded). Included only as background context if needed; must not be presented as directly sourced from S1/S2.

## claim-007
- Statement: The exact byte-for-byte HTML wording of the example.com body paragraph, beyond the tool-reported summary in S2.
- Classification: Unresolved
- Basis: No raw-HTML fetch capability was available in this environment (only an AI-mediated WebFetch tool); shell/curl access was not available. The screenshot requested as shot-001 is intended to give the Visual Editor an independent, pixel-level check against this same claim.

</write-uuter-context>

## Provided context: drafts/article-001.md

<write-uuter-context name="drafts/article-001.md">
# What Example Domain Is For

Example Domain is a deliberately plain public page that exists so writers can
put a real domain into documentation examples. That is not a convention someone
has to know in advance: the page at [https://example.com/](https://example.com/)
says so in its own visible text, under a heading that reads "Example Domain".

## Reaching the page

The address that leads there in this case is not example.com itself but a test
utility:
[`https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F`](https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F).
That endpoint answers with an HTTP 302 Found, and its `Location` header names
`https://example.com/`. A redirect target is only the server's word for where it
intends to send you, so the destination was fetched separately rather than taken
on trust. It resolved, returned content, and turned out to be exactly the URL the
header named — no substitution along the way.

## What the page shows

The page answers the question in one short paragraph: the domain is for use in
documentation examples without needing permission, and should be kept out of
operations. The page grants the permission it describes and sets that one limit
on it; a "Learn more" link offers fuller detail on IANA's site.

![Screenshot of the example.com page: a bold heading reading Example Domain, below it one short paragraph reading This domain is for use in documentation examples without needing permission. Avoid use in operations. and below that a Learn more link, all on a plain light grey background.](visuals/article-001/assets/shot-001.png)

Requested URL: `https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F`.
Observed final URL: `https://example.com/`.

## The limits of this description

The page's text here was read through a fetch tool that summarizes HTML rather
than returning raw markup. The substance is a direct observation: heading,
purpose, permission, and caution all came from the live page on 30 August 2026.
The exact sentence boundaries and punctuation are as that tool reported them,
not byte-verified against the source markup, so treat the wording above as a
faithful report rather than a transcription.

</write-uuter-context>

## Provided context: evidence/firsthand.md

<write-uuter-context name="evidence/firsthand.md">
# Firsthand Work

Two live fetches were performed via the WebFetch tool during this research session (2026-08-30):

1. `https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F` — the tool did not auto-follow the redirect and instead reported the 302 status and the server-supplied `Location` header (`https://example.com/`), explicitly noting it was "not verified" until independently fetched.
2. `https://example.com/` — fetched directly as a follow-up, confirming the redirect target is live and returning the visible heading, body paragraph, and link text described in evidence/sources.md (S2).

No browser rendering, screenshotting, or credentialed access was performed directly by the Researcher; the one requested screenshot (shot-001) is delegated to the controller's capture-runner per the brief's constraints, using the exact redirect URL rather than the resolved destination URL.

</write-uuter-context>

## Provided context: evidence/screenshots.json

<write-uuter-context name="evidence/screenshots.json">
{
  "schema_version": 3,
  "screenshots": [
    {
      "id": "shot-001",
      "path": "evidence/assets/screenshots/shot-001.png",
      "requested_url": "https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F",
      "final_url": "https://example.com/",
      "captured_at": "2026-08-30T10:22:33.371676Z",
      "supports": [
        "claim-004"
      ],
      "reason": "Shows the destination page visibly identifying itself as Example Domain and stating it permits use in documentation examples, reached via the exact supplied redirect URL",
      "backend": "cloudflare-chromium",
      "media_type": "image/png",
      "viewport": {
        "width": 1280,
        "height": 800
      },
      "full_page": false,
      "byte_size": 19303,
      "width": 1280,
      "height": 800,
      "sha256": "sha256:410dd462fa1cc2a7ae84c0feef288195963439a520d826b5a5d44429555d0a59",
      "action_summary": [
        "navigate to the requested public URL",
        "observe the final URL from the capture session",
        "capture the visible viewport as a PNG"
      ],
      "attempt": 1,
      "editorial_outcome": {
        "request_id": "shot-001",
        "status": "usable",
        "reason": "The capture shows a fully rendered page matching the request reason: the bold heading \"Example Domain\", a paragraph reading \"This domain is for use in documentation examples without needing permission. Avoid use in operations.\", and a \"Learn more\" link. It is not a blank or skeleton screen, login or consent page, bot challenge, regional-unavailable page, or error. The visible text independently corroborates claim-004 and fits the \"What the page shows\" section; provenance lists the requested httpbingo redirect URL and the observed final URL https://example.com/ as distinct, which the caption must preserve."
      }
    }
  ]
}

</write-uuter-context>

## Provided context: evidence/sources.md

<write-uuter-context name="evidence/sources.md">
# Sources

## S1 — httpbingo.org redirect endpoint
- Location: https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F
- Accessed: 2026-08-30 (via WebFetch tool)
- Summary: A test-utility endpoint (httpbingo, an httpbin-compatible service) that issues an HTTP 302 Found redirect. The `Location` response header points to `https://example.com/`, matching the `url` query parameter supplied in the request. The tool used to access this page reported the redirect rather than auto-following it, and explicitly flagged the target as "server-supplied, not verified" until independently fetched.
- Use: Establishes that the brief's supplied redirect URL leads to https://example.com/ via a standard 302 redirect, and that this is the destination actually reached, not a substituted URL.

## S2 — example.com landing page
- Location: https://example.com/
- Accessed: 2026-08-30 (via WebFetch tool; fetched separately after following the redirect reported by S1)
- Summary: A minimal single-page site. Visible content extracted by the fetch tool:
  - Heading: "Example Domain"
  - Body paragraph: "This domain is for use in documentation examples without needing permission. Avoid use in operations."
  - A "Learn more" link, pointing (per the tool's description) to further detail on IANA's website.
- Use: Primary evidentiary basis for the article's description of what the page visibly says and what it identifies its own purpose as.
- Caveat: The fetch tool processes raw HTML through an intermediate summarization step rather than returning raw markup verbatim. The heading and general meaning ("for documentation examples," "no permission needed") are consistent with the well-known, long-standing content of this reserved domain, but the exact sentence-level wording returned should be treated as tool-reported rather than independently byte-verified. See claim-ledger for how this caveat is classified.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:bd6977253b6a09af21f278ae75afb6facd9e1e5a0f724a0cc6e7e18a5df5f47e

</write-uuter-context>

## Provided context: visuals/article-001/manifest.json

<write-uuter-context name="visuals/article-001/manifest.json">
{
  "schema_version": 1,
  "candidate": 1,
  "source_prose": {
    "path": "drafts/article-001-prose.md",
    "sha256": "sha256:2182d470bbe1c6dc15061f05fbe23e0f842c1e3651de129442cb7053742fcc67"
  },
  "plan": {
    "path": "visuals/article-001/plan.md",
    "sha256": "sha256:76ceedceb5857d81df901df39c6e17a3fe185199a303c5a3612fef41e86f1b78"
  },
  "actions": [
    {
      "id": "opp-001",
      "location": "Untitled opening section, before the \"Reaching the page\" heading.",
      "action": "none",
      "rationale": "The opening is three lines stating the conclusion and attributing it to the page's own visible text. There is no relationship, sequence, or comparison underneath it to draw, and placing the article's only image here would spend it before the section that actually makes claims about what is visible."
    },
    {
      "id": "opp-002",
      "location": "\"Reaching the page\" section, the paragraph covering the httpbingo endpoint, the HTTP 302 response, and the independent fetch of the destination.",
      "action": "none",
      "rationale": "A flow diagram would restate a five-sentence paragraph edge for edge without compressing anything, and would lose the section's point: a Location header is only the server's word for the destination until it is fetched separately. An unqualified requested-URL to 302 to example.com arrow asserts the confirmed destination as if the header alone established it, so this stays in prose where it can be qualified."
    },
    {
      "id": "opp-003",
      "location": "\"What the page shows\" section, immediately after the paragraph naming the heading, the body paragraph, and the \"Learn more\" link.",
      "action": "existing_local_asset",
      "rationale": "This is the only section making claims about what is visible, and the one place where the reader must otherwise take tool-reported wording on trust. The capture makes the heading, the permission sentence, and the link directly checkable, corroborating claim-004 at pixel level and giving the pixel check the ledger anticipated against the unresolved claim-007. It sits after the prose so the text still stands if the image is absent. The caption must distinguish the requested URL, the httpbingo redirect endpoint, from the observed final URL https://example.com/.",
      "asset_id": "shot-001",
      "alt_text": "Screenshot of the example.com page: a bold heading reading Example Domain, below it one short paragraph reading This domain is for use in documentation examples without needing permission. Avoid use in operations. and below that a Learn more link, all on a plain light grey background."
    },
    {
      "id": "opp-004",
      "location": "\"What the page shows\" section, the paragraphs beginning \"Three things are visible on the destination page\" and \"That paragraph is the whole answer\".",
      "action": "restructure_text",
      "rationale": "With shot-001 placed below, the inventory of heading, paragraph, and link is carried twice. Tightening the two paragraphs into one - naming what the page says and the single limit it sets, without walking the reader element by element down a layout they can see - removes the duplication and buys back words against the 350-word cap. A diagram of the same three elements would be a third statement of the same fact."
    },
    {
      "id": "opp-005",
      "location": "\"The limits of this description\" section.",
      "action": "none",
      "rationale": "Three sentences distinguishing firsthand substance from tool-reported wording. This is a caveat about evidential strength rather than a structure; drawing it would invite a confidence hierarchy the claim ledger does not support, and the paragraph already reads cleanly at this length."
    }
  ],
  "assets": [
    {
      "id": "shot-001",
      "opportunity_id": "opp-003",
      "path": "visuals/article-001/assets/shot-001.png",
      "origin": "screenshot",
      "source": "evidence/assets/screenshots/shot-001.png",
      "media_type": "image/png",
      "byte_size": 19303,
      "sha256": "sha256:410dd462fa1cc2a7ae84c0feef288195963439a520d826b5a5d44429555d0a59",
      "alt_text": "Screenshot of the example.com page: a bold heading reading Example Domain, below it one short paragraph reading This domain is for use in documentation examples without needing permission. Avoid use in operations. and below that a Learn more link, all on a plain light grey background."
    }
  ],
  "article": {
    "path": "drafts/article-001.md",
    "sha256": "sha256:0382351bef1851ca656e54fbc40d8851a452e565e0cb569f534873f9a3225cd4"
  },
  "reviewed_revision": "sha256:bd6977253b6a09af21f278ae75afb6facd9e1e5a0f724a0cc6e7e18a5df5f47e",
  "prose_characters_before": 1678,
  "prose_characters_after": 1607
}

</write-uuter-context>

## Provided context: visuals/article-001/plan.md

<write-uuter-context name="visuals/article-001/plan.md">
# Visual plan — article-001

Source prose revision: `sha256:2182d470bbe1c6dc15061f05fbe23e0f842c1e3651de129442cb7053742fcc67`

The draft is a ~320-word article with four short sections and one staged
screenshot. One visual is placed; the remaining opportunities were evaluated and
rejected on the record.

## Screenshot validation — shot-001

Before placement I opened `context/visual-inputs/shot-001.png` and compared the
visible pixels with the request reason in `evidence/screenshots.json` and with
the article context.

- Request reason: show the destination visibly identifying itself as Example
  Domain and stating it permits use in documentation examples, reached via the
  exact supplied redirect URL. Supports `claim-004`.
- Observed pixels: a 1280x800 viewport on a light grey background showing the
  bold heading "Example Domain", one short paragraph reading "This domain is for
  use in documentation examples without needing permission. Avoid use in
  operations.", and a "Learn more" link below it.
- Verdict: **usable**. The capture is a fully rendered page, not a blank or
  skeleton screen, a login or consent wall, a bot challenge, a regional-block
  page, or a generic error. Every element the request named is legible, and the
  visible sentence independently corroborates `claim-004` at pixel level, which
  is also the pixel check the ledger anticipated against the unresolved
  `claim-007`.

The provenance record lists `requested_url` as the httpbingo redirect and
`final_url` as `https://example.com/`. Both are true and distinct, so the
Writer's caption must report the requested redirect URL and the observed final
URL rather than only one of them.

## Opportunities

### opp-001 — Opening paragraph

- **Location:** Untitled opening section, before the "Reaching the page"
  heading.
- **Action:** `none`
- **Reason:** The opening is three lines that state the article's conclusion and
  then immediately attribute it to the page's own visible text. There is no
  relationship, sequence, or comparison underneath it to draw, and putting the
  screenshot here would spend the article's only image before the section that
  actually makes claims about what is visible.

### opp-002 — Redirect provenance

- **Location:** "Reaching the page" section, the paragraph covering the
  httpbingo endpoint, the 302 response, and the independent fetch.
- **Action:** `none`
- **Reason:** A flow diagram of this hop would restate a five-sentence paragraph
  edge for edge rather than compress anything. It would also lose the point of
  the section: a `Location` header is the server's *word* for the destination
  until the destination is fetched separately, and an unqualified
  `requested URL --> 302 --> example.com` arrow asserts the confirmed
  destination as if the header alone established it. That distinction is the
  whole reason the section exists, so it stays in prose where it can be
  qualified.

### opp-003 — Evidence for what the page visibly says

- **Location:** "What the page shows" section, immediately after the paragraph
  that names the heading, the body paragraph, and the "Learn more" link.
- **Action:** `existing_local_asset` — `shot-001`
- **Reason:** This is the only section making claims about what is *visible*,
  and it is the one place where the reader has to take the tool-reported wording
  on trust. The screenshot converts that into something the reader can check
  directly: the heading, the permission sentence, and the link are all legible
  in the capture. It sits after the prose so the text still stands on its own if
  the image is unavailable. The Writer's caption must distinguish the requested
  URL, the httpbingo redirect endpoint, from the observed final URL,
  `https://example.com/`.

### opp-004 — The three visible elements, stated twice

- **Location:** "What the page shows" section, the two paragraphs beginning
  "Three things are visible on the destination page" and "That paragraph is the
  whole answer".
- **Action:** `restructure_text`
- **Reason:** With `shot-001` placed directly below, the inventory of heading,
  paragraph, and link is now carried in two places at once. Tightening these two
  paragraphs into one — naming what the page says and the single limit it sets,
  without walking the reader element by element down a layout they can see —
  removes the duplication and buys back words against the 350-word cap. An
  additional diagram of the same three elements would be a third statement of
  the same fact, so no visual is warranted here.

### opp-005 — Limits of the description

- **Location:** "The limits of this description" section.
- **Action:** `none`
- **Reason:** Three sentences distinguishing firsthand substance from
  tool-reported wording. The content is a caveat about evidential strength, not
  a structure; drawing it would invite a confidence hierarchy the claim ledger
  does not support, and the paragraph already reads cleanly at this length.

</write-uuter-context>