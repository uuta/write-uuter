# Issue 11 authenticated runtime verification

- Outcome: succeeded on candidate 3 at `2026-08-30T10:45:47.708340Z`.
- Controller: the locally built `bin/write-uuter`.
- External runner: the locally built real `bin/write-uuter-cloudflare-capture` with ambient authenticated Cloudflare credentials; no test endpoint or test capture scenario was set.
- Agents: the installed real Codex and Claude CLIs, not repository fixtures.
- Request manifest: `evidence/screenshot-requests.json` requests `https://httpbingo.org/redirect-to?url=https%3A%2F%2Fexample.com%2F` as `shot-001`.
- Result manifest: `evidence/screenshots.json` records observed `final_url` `https://example.com/`, backend `cloudflare-chromium`, attempt 1, and a request-keyed `usable` editorial outcome.
- Screenshot: `evidence/assets/screenshots/shot-001.png`, 1280x800, 19,303 bytes, SHA-256 `410dd462fa1cc2a7ae84c0feef288195963439a520d826b5a5d44429555d0a59`.
- Pixel inspection: the retained image visibly shows the heading “Example Domain”, the documentation-use paragraph, and “Learn more”; it corroborates the observed destination.
- Placement: `visuals/article-003/assets/shot-001.png` is byte-identical to the evidence image. `visuals/article-003/manifest.json`, all four candidate-3 review results, and `workflow.json` bind revision `sha256:efa0440b31351a5db3a46d5c408d68220e6a9f38d42cd9463ec62dc06f665007`.
- Cleanup: after completion, no `.write-uuter-capture-*` workspace and no `write-uuter-cloudflare-capture` process remained.
- Credential audit: fixed-string scans using the ambient account ID and token found neither value anywhere in this retained run; invocation audits also showed no Cloudflare credential variable present in article-agent environments.

The earlier sibling `.runtime-issue11-authenticated-cdp-run` is retained separately and labeled as a genuinely authenticated but blocked run; it is not cited as the completed verification.
