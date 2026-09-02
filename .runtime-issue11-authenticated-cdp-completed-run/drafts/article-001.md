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
