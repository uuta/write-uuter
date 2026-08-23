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
