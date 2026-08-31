# Visual plan — article-003

Source prose revision: `sha256:bf96909840998080db3bdf680db2cb53e452bd80e8c7f426f96406c2bb496f07`

Five opportunities were evaluated across the four sections of the prose draft.
Two produce a visual; three are recorded as `none`. The draft is a ~330-word
article that already reads cleanly, so the bar for adding a visual was whether it
carries a relationship the prose has to spend sentences on, or whether it is
evidence the reader would otherwise have to take on trust.

---

## opp-001-redirect-provenance — `mermaid`

**Location:** Section "Reaching the page", the paragraph beginning "The address
that leads there in this case is not example.com itself but a test utility".

**Action:** `mermaid` — a three-node linear flow from the requested httpbingo URL,
through the 302 response and its `Location` header, to the separately fetched
destination.

**Reason:** This paragraph is the article's provenance spine, and it is doing two
jobs at once in continuous prose: naming a chain of three distinct URLs states,
and marking the point where the server's word stopped being trusted. Those are a
sequence, not an argument, and a sequence is easier to hold as a shape than as a
sentence that has to keep re-introducing long percent-encoded URLs. The diagram
also fixes the requested-versus-destination distinction that the brief's "done
when" depends on: the requested URL is the httpbingo endpoint, the destination is
`https://example.com/`, and the two are drawn as separate nodes so neither can be
read as the other. Edges are labelled with the qualification the prose supplies —
the redirect was reported rather than auto-followed, and the target was
server-supplied and therefore verified independently — so no edge asserts more
than claim-001 and claim-002 support. Once the diagram carries the chain, the
Writer can shorten the paragraph to the interpretive point about not trusting a
redirect target, rather than restating each hop in words.

**Evidence basis:** claim-001 (302 with `Location: https://example.com/`),
claim-002 (destination independently fetched, not substituted). No edge asserts
anything about why the domain is reserved, which is out of scope.

---

## opp-002-page-screenshot — `existing_local_asset` (`shot-001`)

**Location:** Section "What the page shows", immediately after the paragraph
beginning "The paragraph is the whole answer."

**Action:** `existing_local_asset` placing `shot-001`.

**Reason:** This is the only section of the article making claims about what is
*visible*, and it is also the section standing on the weakest evidence: the
draft's own closing section concedes the page text was read through a summarizing
fetch tool rather than raw markup, so exact wording is tool-reported. The
screenshot is the independent pixel-level check against that limit — it lets the
reader see the heading and the permission sentence rather than accept a report of
them. Placing it here converts the article's shakiest passage into something the
reader can verify at a glance, which no amount of restructured prose can do.

**Visible-content validation (mandatory, origin `screenshot`):** I opened the
staged file before planning the placement. It shows a fully rendered page on a
plain light grey background: a bold heading reading "Example Domain", a paragraph
reading "This domain is for use in documentation examples without needing
permission. Avoid use in operations.", and a blue underlined "Learn more" link.
It is not a blank or skeleton screen, a login or consent page, a bot challenge, a
regional-unavailable notice, or a generic error. The visible pixels match the
request reason in `evidence/screenshots.json` and independently corroborate
claim-004, and they are also a direct pixel-level answer to the unresolved
claim-007. Recorded as `usable`.

**Caption requirement carried to the Writer:** the caption must name both the
**requested** URL (`https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F`)
and the **observed final** URL (`https://example.com/`), not one in place of the
other. The alt text describes the rendered page only; it does not assert the URL.

---

## opp-003-visible-elements-list — `none`

**Location:** Section "What the page shows", the sentence beginning "Three things
are visible on the destination page".

**Action:** `none`.

**Reason:** Considered converting this to a three-item bullet list. Rejected: it
is already one short, scannable sentence, and the section is only two short
paragraphs, so a list would fragment it without improving scanning. Considered
also whether the sentence becomes redundant once `shot-001` is placed above it.
It does not — the draft's own framing requires the prose to stand without the
image, and the enumeration is what makes the section readable for anyone not
seeing the screenshot. Keeping both is not duplicated *explanation*; the sentence
names the elements, the image evidences them.

---

## opp-004-limits-section — `none`

**Location:** Section "The limits of this description", entire section.

**Reason:** This section is an epistemic caveat about how the text was obtained —
firsthand substance, tool-reported punctuation. That is a claim about confidence,
not a relationship, sequence, hierarchy, or state change, and there is no shape a
diagram could draw that would not overstate it. Drawing "raw HTML → tool →
summary" as a pipeline would imply a verified processing chain the run never
observed. Three sentences of plain prose is the honest form.

---

## opp-005-opening — `none`

**Location:** Opening section, from "Example Domain is a deliberately plain public
page" through the heading quote.

**Reason:** The opening is a single paragraph stating the conclusion, and its
whole job is to be read straight through in about fifteen seconds. It is not
dense, not long, and not a relationship. A visual here would delay the answer the
paragraph exists to deliver, and the screenshot is deliberately held back to the
section that actually makes visibility claims.
