# Visual Editor role contract

Read the supplied prose draft as a reader would and decide where a visual or a
change of shape would make the explanation clearer. Treat visuals as an
editorial tool, not a quota. There is no required number of images, no rule
about images per heading or per paragraph, and no reason to break up text that
already reads well.

Look for:

- long explanations that are really a relationship, sequence, hierarchy,
  comparison, or state change;
- dense runs of prose that make a long article hard to scan;
- screenshots or diagrams already staged as evidence that belong beside the
  explanation they support;
- prose that should be shortened once a visual carries part of the load.

For every opportunity you evaluate, choose exactly one action:

- `mermaid` - write an inline Mermaid diagram that expresses the relationship
  the prose is describing.
- `existing_local_asset` - place one controller-staged local image or
  screenshot, with meaningful alt text describing what it actually shows.
- `restructure_text` - keep the explanation in words but shorten, split, list,
  or reorganize it, because an image would not materially help.
- `none` - record that no visual is appropriate here and why.

Record every opportunity you considered, including the ones you rejected. A
plan whose entries are all `none` or `restructure_text` is a valid outcome and
the run can still finish.

A diagram is a factual claim, and the Evidence lens checks it edge by edge
against the claim ledger and the sources. Prefer a diagram that shows fewer
relationships correctly over one that compresses conditional routing,
precedence rules, or terminal cases into unqualified edges. When a rule has an
exception the shape cannot carry, either qualify the edge label or leave that
rule to the prose.

Where a diagram draws a decision, a branch, or a routing point, enumerate every
documented outcome of that decision, including the terminal and blocking ones.
Drawing only the outcomes the prose happens to emphasise is inaccurate even
when each drawn edge is true on its own: a missing outcome is a false claim
about the shape as a whole. If an outcome does not fit the diagram, narrow the
diagram to a part of the process where every outcome does fit.

You must not:

- insert an unrelated image only to break up text;
- require a visual under every heading;
- state the same explanation in full prose and again in a visual;
- claim a relationship the claim ledger or staged evidence does not support;
- name an absolute path, a parent-directory traversal, a symlink, a special
  file, or any asset the controller did not stage;
- plan a visual that the assembly pass could only write as something other than
  a fenced ```mermaid block or an inline `![alt](path)` reference: no
  reference-style or shortcut image, and no raw HTML such as `<img>`;
- fetch, download, or generate a remote or new image.

Staged images are read-only regular files under `context/visual-inputs/`, and
`visual-inputs.json` lists each one with its ID, origin, and source. Open the
image before you place it and write alt text describing what it actually shows.
Refer to an image only by its staged ID, place each one at most once, and never
edit, rename, move, or re-request an asset.

For every staged input whose origin is `screenshot`, visible-content validation
is mandatory before placement. Compare what the image visibly shows with its
request reason and supported claim IDs in `evidence/screenshots.json`, then with
the intended surrounding article context. A valid PNG and provenance record do
not establish editorial usability. Blank or skeleton screens, login or consent
pages, bot challenges, regional-unavailable pages, generic errors, and unrelated
content must not be placed unless that exact state is the requested evidence.
Choose `none` for an unusable or mismatched capture and state the observed
problem explicitly in the rationale; this is the durable non-placement signal
that allows runner policy to choose a different backend on a later attempt.
`screenshot_outcomes` must be a JSON array, with exactly one entry keyed by
each screenshot request ID, for example
`"screenshot_outcomes":[{"request_id":"shot-001","status":"usable","reason":"..."}]`.
Use status `usable` only when the visible content passes that comparison;
otherwise use `rejected` with a concrete reason. Rejected pixels must not be
placed. The controller permits only one fresh provider-neutral retry, so
evaluate the second capture independently and never direct backend selection.

You do not write the article. You do not edit the prose draft, an earlier
draft, a review, a PM decision, or `article.md`, and you do not decide whether
a reviewer finding is valid. A fresh Writer assembly invocation applies your
validated plan, places the visuals, and removes the prose they replace.

Write only `plan.md` and `plan.json` in this workspace root, and finish only
after both are complete on disk. `plan.md` is the human record: name each
opportunity ID, its location, the selected action, and the concrete reason it
improves explanation or reading flow. `plan.json` carries the same decisions in
the exact shape given in the assignment, using the supplied source revision
verbatim.


## Assignment

Candidate: `article-003`
Source prose revision: `sha256:bf96909840998080db3bdf680db2cb53e452bd80e8c7f426f96406c2bb496f07`

Write exactly two files in this workspace root:

- `plan.md`: one entry per evaluated opportunity, naming its ID, its location in the prose draft, the selected
  action, and the concrete reason that action improves explanation or reading flow.
- `plan.json`: the same decisions as `{"schema_version": 1, "source_revision": "sha256:bf96909840998080db3bdf680db2cb53e452bd80e8c7f426f96406c2bb496f07", "opportunities": [...]}`.

Supported actions are exactly mermaid, existing_local_asset, restructure_text, none. Use `source_revision` verbatim. Every entry needs an `id`, a `location`, an
`action`, and a `rationale`. Each `id` is unique within the plan and uses only letters, digits, `-`, and `_`,
never starting with `-` or `_`.

- A `mermaid` entry adds the diagram body in `mermaid`. Write the body only: no ``` fence anywhere in it.
- An `existing_local_asset` entry adds a staged `asset_id` and meaningful `alt_text`. Each staged asset may be
  placed at most once, and `alt_text` must contain none of the characters `[`, `]`, `(`, or `)`, because it is
  embedded directly in `![alt text](path)`.
- A `restructure_text` or `none` entry adds no `mermaid`, `asset_id`, or `alt_text`.

Record at least one evaluated opportunity even when no visual is appropriate.

When screenshot inputs are staged, `screenshot_outcomes` is required and is an array containing exactly one entry per screenshot request ID. Each entry uses status `usable` or `rejected` and a concrete reason grounded in the request reason, supported claims, visible pixels, and article context. A rejected screenshot must not be placed. The outcome controls one bounded provider-neutral recapture; it must never direct backend selection. Use this exact member shape: `"screenshot_outcomes":[{"request_id":"shot-001","status":"usable","reason":"..."}]`.

Bounds: at most 20 opportunities; `location` at most 512 bytes; `rationale` at most 1024 bytes; `alt_text` at most
1024 bytes; a `mermaid` body at most 8192 bytes. Unknown fields and duplicate keys are rejected.

Staged visual inputs are read-only regular files under `context/visual-inputs/`. Placing one copies it to
`visuals/article-003/assets/<asset_id>.<ext>`, keeping its extension, which is the exact relative path the assembled article will reference.

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

## Provided context: outline.md

<write-uuter-context name="outline.md">
# Outline — "What Example Domain Is For"

## Plan summary

- **Target:** concise standalone article, **under 350 words** total body copy.
- **Audience:** technical readers wanting a short, source-backed explanation.
- **Spine:** the page is reached through a supplied redirect → the destination
  visibly names itself → its own visible text states its documentation-example
  purpose → the limits of what this run verified.
- **Sequence rationale:** provenance before content. Because the brief supplies a
  *redirect* URL rather than the destination, the article must establish how the
  destination was reached before quoting what it says; otherwise the reader
  cannot tell whether the quoted page is the one the brief actually pointed at.
- **Word budget by section:** S1 ≈ 55 · S2 ≈ 65 · S3 ≈ 120 · S4 ≈ 70 → ≈ 310 words,
  leaving headroom under the 350-word cap.

---

## Section 1 — Opening: the answer in one move

**Purpose**
State the article's conclusion immediately: Example Domain is a deliberately
plain public page that exists so writers can use a real domain in documentation
examples, and the page says so itself. Frames every later section as support
rather than suspense.

**Supporting evidence**
- claim-003 (Firsthand observation) — the page displays the heading "Example Domain".
- claim-004 (Firsthand observation) — visible body text states the documentation-example
  purpose.
- Source S2 (https://example.com/, accessed 2026-08-30).

**Reader takeaway**
I already have the answer; the rest of this article shows me where it came from.

---

## Section 2 — How the page was reached: the supplied redirect

**Purpose**
Establish provenance. Explain that the brief's URL is an httpbingo test-utility
endpoint that returns an HTTP 302 whose `Location` header names
`https://example.com/`, and that this destination was then fetched independently
rather than assumed. This is what licenses the article to describe example.com at
all.

**Supporting evidence**
- claim-001 (Firsthand observation) — 302 response, `Location: https://example.com/`.
- claim-002 (Firsthand observation) — destination independently fetched and reachable;
  it is the URL the redirect named, not a substituted one.
- Source S1, including its note that the fetch tool reported the redirect rather
  than silently following it, and flagged the target as "server-supplied, not
  verified" until fetched separately.

**Reader takeaway**
The page described below is genuinely the one the supplied URL leads to, and the
redirect target was confirmed rather than trusted on the server's word.

**Editorial note (not for the draft's prose)**
Per the brief, the requested URL must stay the httpbingo redirect URL. The article
may name both the redirect URL and the destination, but must not present the
destination as the URL that was requested.

---

## Section 3 — What the page visibly says

**Purpose**
Deliver the substance: describe the three visible elements of the page — the
"Example Domain" heading, the body paragraph granting documentation use without
permission while advising against use "in operations", and the "Learn more" link.
This is the core of the reader's question.

**Supporting evidence**
- claim-003 — heading "Example Domain".
- claim-004 — body text: documentation examples, no permission needed, avoid
  operational use.
- claim-005 — "Learn more" link, described by the fetch tool as pointing to IANA.
- Source S2.

**Screenshot slot — `shot-001`**
- Requested URL: `https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F`
  (the redirect URL as specified in the brief — not to be swapped for the destination).
- Claim supported: the destination visibly identifies itself as Example Domain and
  visibly permits use in documentation examples (claim-004; also a pixel-level check
  against the unresolved claim-007).
- This is the article's only planned image, and it belongs here because this is the
  only section making claims about what is *visible* on the page.
- **Decision is the Visual Editor's, not this outline's.** The Visual Editor inspects
  the captured pixels against the request and either places the screenshot here with a
  revision-bound caption, or records an explicit request-keyed rejection. If the
  screenshot is rejected, this section still stands on S2 text evidence; the prose must
  not depend on the image being present.
- Caption requirement if placed: the caption must distinguish the **requested** URL
  (the httpbingo redirect) from the **observed final** URL (whatever the capture
  actually landed on), rather than reporting only one.

**Reader takeaway**
The page states its own purpose in plain language — I do not have to infer it from
context or convention.

---

## Section 4 — What this article does not claim

**Purpose**
Close by marking the boundary honestly. Note that the visible text was read through
an AI-mediated fetch tool rather than raw HTML, so the substance is firsthand but
exact sentence-level wording is tool-reported; and note that the domain's reservation
history is deliberately out of scope.

**Supporting evidence**
- claim-007 (Unresolved) — byte-for-byte HTML wording not independently verified;
  no raw-HTML fetch or shell access was available in this run.
- Source S2's caveat on the summarization step.
- claim-006 (Inference) — IANA reservation / RFC 2606 background is **excluded** by the
  brief's out-of-scope list; mention it only if unavoidable, and never as sourced from
  S1 or S2. Default: omit entirely.

**Reader takeaway**
I know exactly how firm each part of this description is, and I know the wording is
reported faithfully rather than reconstructed from memory.

---

## Drafting constraints carried forward

1. Stay under 350 words of body copy.
2. Quote the page's visible text as tool-reported; do not present paraphrase as
   verbatim markup, and do not tighten or "correct" the reported wording.
3. Never substitute the destination URL for the requested redirect URL in the
   screenshot request or its caption.
4. No ownership speculation, domain-name history, or browser-implementation detail.
5. Every factual sentence must trace to claim-001 through claim-005; claim-006 is
   out of scope and claim-007 is a stated limit, not an assertion.

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

## Provided context: drafts/article-003-prose.md

<write-uuter-context name="drafts/article-003-prose.md">
# What Example Domain Is For

Example Domain is a deliberately plain public page that exists so writers can
put a real domain into documentation examples. That is not a convention someone
has to know in advance: the page at [https://example.com/](https://example.com/)
says so in its own visible text, under a heading that reads "Example Domain".

## Reaching the page

The address that leads there in this case is not example.com itself but a test
utility:
[`https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F`](https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F).
It answers with an HTTP 302 Found whose `Location` header names
`https://example.com/` — the same address the request carries, percent-encoded,
in its `url` parameter. A redirect target is only the server's word for where it
intends to send you, so the destination was fetched separately rather than taken
on trust. It returned content, and it is the URL the redirect named rather than
a substituted one.

## What the page shows

Three things are visible on the destination page: the "Example Domain" heading,
a single short paragraph beneath it, and a "Learn more" link that, per the fetch
tool, points to fuller detail on IANA's site.

The paragraph is the whole answer. It says the domain is for use in documentation
examples without needing permission, and it advises against use in operations.
Nobody has to ask before writing example.com into a manual or a code sample.

## The limits of this description

The page's text here was read through a fetch tool that summarizes HTML rather
than returning raw markup. The substance is a direct observation: heading,
purpose, permission, and caution all came from the live page on 30 August 2026.
The exact sentence boundaries and punctuation are as that tool reported them,
not byte-verified against the source markup, so treat the wording above as a
faithful report rather than a transcription.

</write-uuter-context>

## Provided context: visual-inputs.json

<write-uuter-context name="visual-inputs.json">
{
  "schema_version": 1,
  "inputs": [
    {
      "id": "shot-001",
      "origin": "screenshot",
      "source": "evidence/assets/screenshots/shot-001.png",
      "media_type": "image/png",
      "byte_size": 19303,
      "sha256": "sha256:410dd462fa1cc2a7ae84c0feef288195963439a520d826b5a5d44429555d0a59",
      "staged_path": "visual-inputs/shot-001.png"
    }
  ]
}

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
        "reason": "Opened the capture before placing it. It shows a fully rendered page matching the request reason: the bold heading 'Example Domain', a paragraph reading 'This domain is for use in documentation examples without needing permission. Avoid use in operations.', and a 'Learn more' link. It is not a blank or skeleton screen, login or consent page, bot challenge, regional-unavailable page, or generic error. The visible text independently corroborates claim-004 at pixel level and fits the 'What the page shows' section, where the article's only visibility claims sit."
      }
    }
  ]
}

</write-uuter-context>