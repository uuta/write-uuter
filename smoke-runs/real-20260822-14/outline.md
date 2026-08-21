# Outline: From Brief to Inspectable Reviewed Article

Target length: 700–850 words. The article should answer the brief through the implemented workflow, using “inspectable” as a synthesis grounded in retained artifacts rather than as a measured quality claim.

## 1. The design in one view: Go controls, Codex edits

- **Purpose:** Open with the direct answer: write-uuter is a Go CLI that converts a Markdown brief into a reviewed article by having a deterministic controller coordinate isolated editorial roles. Establish the key division of responsibility before introducing workflow detail.
- **Supporting evidence:** F1 (S1); F3 (S1, S2, S3); F5–F6 (S1, S2, S3). Use I2 only as an explicitly framed interpretation: retained PM judgments plus controller-enforced routing make the responsibility split auditable.
- **Reader takeaway:** The system is neither one free-running agent nor a conventional in-memory pipeline: Go owns control and validation, while separate Codex processes own bounded editorial tasks.

## 2. The forward path: brief, evidence, outline, candidate

- **Purpose:** Walk through the first half of a run in sequence: validate the brief and initialize a new run atomically; have the Researcher produce sources and a classified claim ledger; have the Story Editor turn that evidence into an outline; then have the Writer produce the first candidate. Explain role ownership in plain language.
- **Supporting evidence:** F2 (S1, S2); F4 (S1, S2, S3); F8–F9 (S2, S3). Mention the required claim classes and the outline’s purpose/evidence/takeaway fields as concrete examples of artifact contracts.
- **Reader takeaway:** Each editorial handoff becomes a named file with minimum content requirements, so later work begins from validated inputs rather than an informal conversation.

## 3. Why artifact gates, not agent messages, advance the run

- **Purpose:** Explain the mechanism that makes the workflow inspectable: roles work in isolated workspaces; the controller stages only allowed context; after a successful process exit it validates owned regular-file outputs and copies them into the durable run. Clarify that a final chat message cannot advance state.
- **Supporting evidence:** F6–F7 (S1, S2, S3); F11 (S2, S4); F17 (S4). Use I1 as a labeled synthesis: inspectability comes from retained, validated files rather than transient transcripts. Include representative checks such as revision hashes and consistent review JSON/report content without turning the section into a schema inventory.
- **Reader takeaway:** Progress is evidenced by durable artifacts that satisfy contracts; apparent completion by an agent is insufficient.

## 4. The sequential review and decision loop

- **Purpose:** Describe the fixed Evidence → Story → Clarity → Copy sequence. Distinguish reviewers, which report findings without editing candidates, from the persistent PM, which classifies each finding, and from Go, which validates decisions and applies routing.
- **Supporting evidence:** F5, F10 (S1, S2, S3); F11 (S2, S4); F12 (S2, S3, S4). Name all four PM classifications: `valid_must_fix`, `valid_optional`, `invalid` with a reason, and `needs_human_judgment`.
- **Reader takeaway:** Fresh reviewers supply lens-specific judgments, the PM records editorial decisions, and deterministic controller rules decide what happens next.

## 5. Revision routing and the three-candidate budget

- **Purpose:** Show the loop’s branching behavior. A validated must-fix stops later lenses, sends the Writer to the next candidate, and restarts that candidate at Evidence; optional or invalid findings do not consume a candidate. Explain that candidate 003 is the hard limit and identify the conditions that block a run.
- **Supporting evidence:** F13 (S2); F14 (S1, S2, S3). I3 may explain, as an inference, why review restarts at Evidence after content changes. I4 may frame the candidate cap as bounding automated revision, but do not claim that three is an optimal number. State that must-fix at candidate 003, human judgment, and runtime or artifact failure produce a blocked run.
- **Reader takeaway:** Revision is bounded and predictable: the workflow cannot revise indefinitely or silently skip unresolved problems.

## 6. Terminal artifacts: exact success or inspectable blockage

- **Purpose:** Close by contrasting the two terminal outcomes. On success, all four lenses have passed PM routing and `article.md` exactly matches the accepted candidate. On blockage, the reason and accumulated work remain. Tie both outcomes back to `workflow.json` and the retained revision/review history.
- **Supporting evidence:** F15 (S1, S2, S4); F16–F17 (S2, S4). Mention earlier candidates, partial review sequences, reviews, PM decisions, workflow state, and applicable audit copies as retained evidence. If limits are noted, use F18 (S2, S3): parallel runs, restart-resume, and editing completed runs are not implemented.
- **Reader takeaway:** The terminal result is inspectable in either case: readers can trace an accepted article to its candidate and reviews, or a blocked run to its preserved state and reason.

## Scope and language guardrails

- Do not claim measured improvements in accuracy, quality, reliability, or usability; U1–U5 remain unresolved.
- Do not call the three-candidate limit “reasonable” or “optimal” (O2).
- Keep publishing integrations, web interfaces, other repositories, and unsupported resume behavior out of the article.
- Explain “artifact gate” on first use as a required file plus validation that must pass before the controller advances.
