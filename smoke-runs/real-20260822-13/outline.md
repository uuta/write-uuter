# Outline: From Brief to Inspectable Reviewed Article

Target length: 700–850 words. Use plain language and keep claims specific to the repository’s implemented workflow.

## 1. The workflow’s inspectable promise

- **Purpose:** Open with the direct answer: write-uuter is a Go CLI that turns a Markdown brief into a reviewed article while preserving the artifacts that explain how the result was produced. Frame inspectability as a property of the documented artifact design, not as a measured quality advantage.
- **Supporting evidence:** F1 (`001-README.md`, lines 1–5); F14–F18 (`001-README.md`, lines 41–46; `004-artifacts.md`, lines 30–33 and 102–129); I1. Mention the audience-relevant interpretation in O1 only as an explicit evaluation, if needed.
- **Reader takeaway:** The output is not just `article.md`; it is an article accompanied by a durable production record.

## 2. Go controls the sequence; isolated roles own artifacts

- **Purpose:** Explain the division of responsibility. Go owns state transitions, validation, hashes, timeouts, cleanup, and termination. In order, separate Codex roles research the brief, plan the story, write a numbered candidate, and review it. Each role works in an isolated controller-created workspace and may return only its contracted artifacts.
- **Supporting evidence:** F2–F5 (`001-README.md`, lines 68–84; `002-workflow.md`, lines 12–39 and 64–88; `003-roles.md`, lines 1–9 and 37–81). State artifact ownership: Researcher → `evidence/sources.md` and `claim-ledger.md`; Story Editor → `outline.md`; Writer → one numbered draft. Note that each outline section records purpose, evidence, and takeaway (F6).
- **Reader takeaway:** Specialized agents produce bounded outputs, but deterministic controller code—not agent conversation—decides what happens next.

## 3. Artifact gates make progress verifiable

- **Purpose:** Describe why merely exiting successfully or announcing completion is insufficient. Before advancing, the controller checks that each owned artifact exists and satisfies role-specific rules. Reviews and PM decisions are tied to the current request, lens, and candidate digest, so stale, malformed, incomplete, or future-lens data is rejected.
- **Supporting evidence:** F11–F13 (`002-workflow.md`, lines 45–62; `003-roles.md`, lines 28–35 and 57–73; `004-artifacts.md`, lines 35–100); I2. Include the boundary that reviewers cannot edit candidates (F12).
- **Reader takeaway:** Durable gates turn role completion into a machine-checkable state change and keep the record aligned with the exact candidate being reviewed.

## 4. Four reviews run as a sequential routing loop

- **Purpose:** Walk through the ordered Evidence, Story, Clarity, and Copy lenses. Each uses a fresh invocation and runs only after the previous lens and PM decision permit it. After every lens, the persistent PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`; Go validates that decision and enforces routing.
- **Supporting evidence:** F7–F9 (`002-workflow.md`, lines 19–43 and 87–88; `003-roles.md`, lines 17–35 and 57–60); I3. Clarify routes: optional and invalid findings continue without consuming a candidate; human judgment blocks; a must-fix stops later lenses and sends the next candidate back to Evidence.
- **Reader takeaway:** Review is deliberately serial: every lens sees a candidate whose earlier routing decisions are already recorded, and revision invalidates the prior review path.

## 5. The candidate budget bounds automated revision

- **Purpose:** Explain the revision ceiling and terminal outcomes. Candidates are numbered through 003; a validated must-fix may create the next candidate only while budget remains. Exhaustion blocks instead of producing candidate 004. On success, `article.md` appears only after all four lenses clear routing and is an exact copy of the accepted candidate. Runtime failure, timeout, human judgment, and exhaustion also end in inspectable preserved state.
- **Supporting evidence:** F9–F10 and F14–F16 (`001-README.md`, lines 31–46 and 81–84; `002-workflow.md`, lines 28–38 and 90–104; `004-artifacts.md`, lines 30–33 and 102–114); I5. Do not claim that three is optimal (U3, O2).
- **Reader takeaway:** Automation is finite and explicit: success publishes an unchanged accepted candidate, while unresolved work becomes a visible blocked run rather than an unbounded loop.

## 6. What remains available for inspection—and the limits

- **Purpose:** Close by inventorying the audit trail: brief, workflow state, evidence, claim ledger, outline, numbered drafts, lens reviews, PM decisions, and success-only article, plus post-cleanup assignments, invocation logs, and exit markers in `.control/`. Explain that earlier and partial artifacts survive revision or blockage, while temporary requests, workspaces, sandbox profiles, and ownership state are removed after cleanup. State the implementation boundary: no resume, parallel runs, completed-run editing, publishing integration, or web interface is claimed.
- **Supporting evidence:** F15–F19 (`002-workflow.md`, lines 3–10 and 106–107; `004-artifacts.md`, lines 3–33 and 102–129); brief scope and out-of-scope constraints. Use I1 to connect preservation to inspectability without claiming comparative quality, cost, latency, or consistency (U1–U2).
- **Reader takeaway:** Engineers can reconstruct consequential workflow decisions from durable files rather than chat or terminal history, while recognizing that this is a bounded, single-run repository workflow—not a publishing platform.
