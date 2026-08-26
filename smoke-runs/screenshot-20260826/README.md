# Authenticated screenshot capture smoke run

Date: 2026-08-26. Purpose: prove the issue #7 capture path end to end against
the real Cloudflare Browser Rendering Chromium quick action.

## What was real and what was deterministic

- Real: `CLOUDFLARE_ACCOUNT_ID`/`CLOUDFLARE_API_TOKEN` supplied only to the
  controller process, the live `POST
  https://api.cloudflare.com/client/v4/accounts/{account_id}/browser-rendering/screenshot`
  call, and the returned PNG.
- Deterministic: the role agents, so the run exercises the controller-owned
  capture, validation, persistence, staging, and cleanup rather than model
  behaviour. The Researcher fixture emitted the request below.

Command shape (credential values were never printed or recorded):

```sh
CLOUDFLARE_ACCOUNT_ID=… CLOUDFLARE_API_TOKEN=… \
  write-uuter run --brief examples/brief.md --run-dir <run> \
    --codex <fixture> --claude <fixture> --timeout 60s --prompts-dir prompts
```

## Result

Exit 0, `workflow.json` `status: succeeded`.

- `evidence/screenshot-requests.json` - the Researcher request, stored `0444`.
- `evidence/screenshots.json` - the controller-generated manifest, stored
  `0444`: `cloudflare-chromium`, `image/png`, 1280x800 viewport, 126650 bytes,
  1280x800 image, SHA-256 recomputed from the stored bytes and confirmed equal.
- `evidence/assets/screenshots/shot-001.png` - the captured image, stored
  `0444`, byte-identical to the accepted response and therefore reusable in
  place by the visual pass planned in #3.
- `reviews/article-001/evidence/result.json` - the Evidence lens result for the
  candidate produced with that screenshot context.
- `role-context.txt` - which roles received the manifest and image, and the
  Cloudflare variable probe for every launched agent.

## Visual confirmation

The PNG was opened and inspected. It is a fully rendered public documentation
page (Cloudflare "Browser Run" overview, matching `requested_url`), not a
blank, loading, skeleton, consent, or error state.

## Credential and process boundary

- `grep -rlF` for the account ID and API token over the whole retained run
  directory - including `.control/prompts` and `.control/logs` - and over every
  captured agent invocation record (prompt, argv, environment): no match.
- Every launched agent reported `CLOUDFLARE_ACCOUNT_ID=ABSENT` and
  `CLOUDFLARE_API_TOKEN=ABSENT`.
- After exit: no controller-private root beside the run directory, and no
  process referencing the run remained.
