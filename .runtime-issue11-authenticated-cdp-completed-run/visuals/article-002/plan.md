# Visual plan — article-002

Source prose revision: `sha256:dc6bf537178b3ed4cea5d7b40b95be9b393f62e15b5a9d64e942c49bac00b1a2`

Five opportunities evaluated; two carry a visual, three are recorded as no-visual
decisions. The article is capped under 350 words, so every visual has to earn its
space against prose that already reads well.

---

## opp-001-page-screenshot — `existing_local_asset`

**Location:** Section "What the page shows", beside the paragraph describing the
heading, the body paragraph, and the "Learn more" link.

**Action:** place staged asset `shot-001`.

**Reason:** This is the only section making claims about what is *visible*, and it
is the one place where a reader benefits from seeing the pixels rather than
reading a report of them. I opened the capture before placing it. It shows a fully
rendered page: the bold heading "Example Domain", a paragraph reading "This domain
is for use in documentation examples without needing permission. Avoid use in
operations.", and a "Learn more" link — no blank or skeleton screen, login or
consent wall, bot challenge, regional-unavailable notice, or error page. That
visible content matches the request reason and independently corroborates
claim-004 at pixel level, which is exactly the check claim-007 leaves open. The
caption the Writer adds must name both the **requested** URL
(`https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F`) and the
**observed final** URL (`https://example.com/`) as distinct, per the brief; the
image must not be presented as a capture of the requested URL's own content.

---

## opp-002-redirect-chain — `mermaid`

**Location:** Section "Reaching the page", after the sentence naming the httpbingo
endpoint.

**Action:** inline Mermaid flowchart of the redirect chain.

**Reason:** The section's real content is a relationship between two distinct URLs
and the verification step that connects them — the single point the whole article
hinges on, since the brief supplies a redirect rather than the destination. Prose
has to carry that ordering serially; the diagram shows the requested URL, the
server-supplied `Location` value, the separate confirming fetch, and the
destination in one glance, so a reader can see at once that the destination was
confirmed rather than assumed. Every edge traces to claim-001 and claim-002, and
the label on the second edge preserves the qualifier that the redirect target was
the server's word only until fetched independently. The chain is linear in the
sources — no branch or failure outcome is documented — so no decision node is
drawn. Once the diagram is in place, the Writer should shorten the paragraph to
the *why* (a redirect target is only the server's claim) and let the shape carry
the sequence, rather than stating both in full.

---

## opp-003-visible-elements-list — `none`

**Location:** Section "What the page shows", first paragraph enumerating the three
visible elements.

**Action:** none.

**Reason:** Considered converting the three-item enumeration into a bullet list.
Rejected: the sentence is one line and already scans, and the screenshot placed
immediately alongside it shows all three elements directly. A bullet list here
would fragment short prose and restate what the image already carries, adding
vertical space to an article working under a 350-word cap.

---

## opp-004-limits-section — `none`

**Location:** Section "The limits of this description".

**Action:** none.

**Reason:** Considered a diagram separating firsthand observation from
tool-reported wording. Rejected: this section states a boundary on confidence, not
a relationship, sequence, or hierarchy. Drawing it would give a caveat the visual
weight of a structural claim and imply a taxonomy the claim ledger does not
assert. The prose is four sentences and needs no help.

---

## opp-005-opening — `none`

**Location:** Opening section, "What Example Domain Is For".

**Action:** none.

**Reason:** The opening is a conclusion-first paragraph of three sentences. There
is no relationship, comparison, or state change to show, and an image here would
only delay the answer the section exists to deliver.

---

## Screenshot outcomes

- `shot-001` — **usable**. Visible content matches the request reason and
  claim-004; placed at opp-001-page-screenshot.
