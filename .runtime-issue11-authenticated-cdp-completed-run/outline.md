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
