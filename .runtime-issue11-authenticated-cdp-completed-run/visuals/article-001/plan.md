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
