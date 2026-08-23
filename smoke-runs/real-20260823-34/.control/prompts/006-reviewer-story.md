# Story Reviewer lens contract

Review only whether the exact candidate follows a coherent narrative and
fulfills the supplied outline's section purposes and reader takeaways. Do not
perform evidence, clarity, or copy review.


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

Lens: `story`
Candidate: `article-001`
Revision: `sha256:6cbeb5c667416cf523bba4494e3b235a3ebc8f41a299443ed069087af7cca18c`

## Provided context: brief.md

<write-uuter-context name="brief.md">
# Brief

## Question

How does write-uuter turn a brief into an inspectable reviewed article?

## Audience

Engineers evaluating a small, artifact-driven editorial workflow.

## Provisional takeaway

A deterministic Go controller can coordinate isolated Codex roles while
leaving evidence, drafts, reviews, and decisions available for inspection.

## Scope

Describe the workflow implemented by this repository and the reason its
durable artifact gates matter.

## Out of scope

Publishing integrations, web interfaces, and claims about workflows outside
this repository.

## Publication target

A concise repository article for technical readers.

## Constraints

Use only facts supported by README.md and docs/. Keep the article under 900
words and explain terms in plain language.

## Done when

The article accurately explains the roles, sequential review loop, candidate
budget, and inspectable terminal artifacts.

## Source hints

- ../README.md
- ../docs/workflow.md
- ../docs/roles.md
- ../docs/artifacts.md

</write-uuter-context>

## Provided context: drafts/article-001.md

<write-uuter-context name="drafts/article-001.md">
# How write-uuter turns a brief into an inspectable reviewed article

`write-uuter` is a Go CLI that turns a Markdown brief into a reviewed article. Its notable output is not only the finished prose: it also preserves the evidence, outline, candidate drafts, reviews, product-manager decisions, and workflow state that led to success or failure. In this sense, the process is inspectable: a reader can relate the durable artifacts to one another after the participating processes have finished. That describes the record the system keeps, not a measured claim about how easy that record is to use.

The Go controller owns the workflow. It validates state transitions, checks artifacts, binds reviews to revisions, enforces routing and timeouts, and cleans up processes. Editorial roles produce content and judgments, but they do not advance the workflow themselves. The controller records the current state in `workflow.json`, which is rewritten atomically and includes the phase, candidate and revision, active role, artifact paths, review-attempt count, timestamps, and any block reason.

## From brief to candidate

A run begins only after the controller validates all required sections of the brief and confirms a new target. It then initializes the run directory atomically. The roles proceed in a fixed order. First, the Researcher produces a source record and claim ledger. Next, the Story Editor turns that material into an outline in which each section identifies its purpose, supporting evidence, and intended reader takeaway. The Writer then produces exactly the candidate assigned to it.

Each role operates in a fresh private workspace outside the durable run directory and receives only the context defined by its contract. When the role finishes, Go validates its owned regular-file outputs before copying them into the run directory. The handoff is therefore a set of explicit artifacts, rather than an informal shared conversation.

## Review is sequential and recorded

Every candidate is reviewed through four lenses in order: Evidence, Story, Clarity, and Copy. They never run in parallel. Each lens gets a fresh reviewer process, and reviewers cannot edit the candidate. Instead, each writes a structured JSON result and a corresponding Markdown report, both of which must pass validation.

After every lens that is reached, a persistent PM classifies each finding as `valid_must_fix`, `valid_optional`, `invalid` with a reason, or `needs_human_judgment`. The separation of responsibilities is deliberate: reviewers identify problems, the PM classifies them, and the Go controller validates the decision and applies the resulting route. Neither a reviewer's message nor a process exit is enough to move the workflow forward.

A validated must-fix ends the remaining reviews for that candidate. The Writer receives a revision assignment, produces the next candidate, and review restarts at Evidence. Optional and invalid findings do not consume another candidate. A finding that needs human judgment blocks the run because the automated workflow does not have authority to resolve it.

Revision is also bounded. Candidate 003 is the hard ceiling; if a validated must-fix remains after that budget is exhausted, the run blocks. This does not guarantee article quality, but it does make the automated rewrite loop finite and leaves its stopping point visible.

## Why artifact gates matter

The controller treats durable files as gates. A role completes only when its worker exits successfully and its owned files have the required structure. Chat messages, transcripts, terminal scrollback, and process exit alone are not editorial completion evidence.

Reviews and PM decisions are also bound to the current candidate revision. The controller rejects stale or mismatched lenses, revisions, requests, digests, findings, and prior-decision data. These checks are designed to keep a decision about one version from silently governing another and to preserve the history of earlier routing. The artifact trail is therefore not just a collection of files; it is a sequence whose links are checked before the next transition.

## Inspectable endings

Success requires the final candidate to clear all four review lenses and their PM routing, followed by final revision and decision validation and verified process cleanup. Only then does the controller atomically publish `article.md`, byte for byte identical to the accepted candidate. The published article therefore cannot drift from the version that passed the workflow.

Failure remains explicit as well. Timeouts, malformed artifacts, premature exits, stale reviews, cleanup failures, requests for human judgment, and an exhausted candidate budget can all block a run with an actionable reason. Earlier candidates, partial review sequences, reports, PM decisions, and post-cleanup audit copies remain available for inspection.

The documented controller is intentionally limited: it handles one non-resumable run, while parallel runs and resumption after a controller restart are not implemented. Publishing integrations and web interfaces are outside its scope. Within those boundaries, `write-uuter` demonstrates the central benefit of an artifact-driven editorial workflow: both the accepted article and the path—or blockage—that produced it remain concrete and checkable.

</write-uuter-context>

## Provided context: outline.md

<write-uuter-context name="outline.md">
# Outline: How write-uuter turns a brief into an inspectable reviewed article

## 1. A controller that leaves an editorial trail

- **Purpose:** Introduce `write-uuter` as a small, artifact-driven workflow and state the article's central idea: a deterministic Go controller coordinates isolated editorial roles while preserving the work that leads to success or failure. Define “inspectable” carefully as the ability to relate durable sources, plans, candidates, reviews, decisions, and terminal state—not as a measured usability claim.
- **Supporting evidence:** F1, F2, F13, F14; I1 with its stated caution.
- **Reader takeaway:** The system produces more than an article: it retains a traceable record of how the article moved through the workflow.

## 2. From validated brief to one candidate

- **Purpose:** Explain the opening sequence: Go validates the required brief sections and atomically initializes a new run; the Researcher creates sources and a claim ledger, the Story Editor creates an evidence-linked outline, and the Writer creates exactly one assigned candidate. Note that roles use fresh private workspaces, receive contracted context, and return validated files to the durable run directory.
- **Supporting evidence:** F3, F4, F5, F6.
- **Reader takeaway:** Work advances through explicit role-owned artifacts rather than through an informal shared conversation.

## 3. Four sequential review lenses and explicit routing

- **Purpose:** Describe the Evidence, Story, Clarity, and Copy reviews in order. Explain that each lens uses a fresh reviewer process, cannot alter the candidate, and must produce a validated JSON result plus a matching report. After every reached lens, the persistent PM classifies each finding, while Go validates the decision and owns the route.
- **Supporting evidence:** F4, F7, F8, F11.
- **Reader takeaway:** Review is a serial, recorded decision loop: reviewers identify issues, the PM classifies them, and the controller—not either role—changes workflow state.

## 4. Revisions are bounded and restart from Evidence

- **Purpose:** Show how findings affect progress. A validated must-fix stops later lenses for the current candidate, gives the Writer a revision assignment, and sends the next candidate back through review starting at Evidence. Optional and invalid findings do not consume a candidate; a human-judgment classification blocks the run. Explain that candidate 003 is the ceiling and characterize the bound as making the automated loop finite and visible, not as a proven quality guarantee.
- **Supporting evidence:** F9, F10; I3 with its stated caution.
- **Reader takeaway:** The workflow permits revision without allowing an indefinite rewrite cycle, and it stops when automation lacks authority to decide.

## 5. Artifact gates make each transition checkable

- **Purpose:** Explain why durable gates matter: a role is complete only after successful exit and validation of its owned regular files. Reviews and PM decisions are tied to the current candidate revision, and stale or mismatched identifiers, digests, findings, or prior decisions are rejected. Mention that `workflow.json` is atomically rewritten as the controller's source of truth and records the active and terminal state.
- **Supporting evidence:** F2, F11, F12, F13; I2 with its stated caution.
- **Reader takeaway:** Files, validation, and revision binding prevent process chatter or stale decisions from masquerading as completed editorial work.

## 6. Success and failure both remain inspectable

- **Purpose:** Close by contrasting the terminal states. Success requires all four final-candidate lenses to clear PM routing, final revision and decision checks, and process cleanup; only then is `article.md` atomically published as an exact copy of the accepted candidate. Blocking conditions produce an actionable reason, while earlier candidates, partial review sequences, reviews, decisions, and control records remain available. Briefly delimit the claim: the documented controller is single-run and non-resumable, and publishing integrations and web interfaces are outside this article.
- **Supporting evidence:** F14, F15, F16, F17; brief scope and out-of-scope constraints.
- **Reader takeaway:** The terminal article cannot drift from the reviewed candidate, and a blocked run still leaves enough durable context to inspect where and why progress stopped.

</write-uuter-context>

## Provided context: revision.txt

<write-uuter-context name="revision.txt">
sha256:6cbeb5c667416cf523bba4494e3b235a3ebc8f41a299443ed069087af7cca18c

</write-uuter-context>