# Outline: From Brief to Inspectable Reviewed Article

Target length: under 900 words. Use plain language and keep all claims within the supplied README/docs evidence. Treat statements about inspectability, determinism, and boundedness as architectural explanations, not measured outcomes.

## 1. The workflow in one view

- **Purpose:** Answer the article's question up front: write-uuter is a Go CLI that turns a Markdown brief into a reviewed article while preserving the files that show how the result was produced. Establish the central division between deterministic controller work and editorial role work.
- **Supporting evidence:** F01 (CLI, input, output, and preserved artifacts); F03 (Go owns transitions, validation, hashes, timeouts, routing, cleanup, and publication); I05 (carefully labeled inference that this separation makes transitions more deterministic).
- **Reader takeaway:** This is a small, repository-specific workflow in which Go controls advancement and isolated Codex roles perform editorial tasks; the result is inspectable through files rather than agent conversation alone.

## 2. From validated brief to the first candidate

- **Purpose:** Explain the opening sequence and role ownership: the controller validates the brief and initializes a new run, then invokes Researcher, Story Editor, and Writer in order. Show how each role hands off a concrete artifact.
- **Supporting evidence:** F02 (brief validation, atomic initialization, single-run and non-resumable scope); F04 (role order); F05 (Researcher owns sources and claim ledger, Story Editor owns the outline, Writer owns its assigned candidate); F06 (private workspaces and staged context).
- **Reader takeaway:** The article emerges through explicit file handoffs—evidence, claim ledger, outline, and candidate—while each role sees only the context staged for its assignment.

## 3. Four sequential review lenses and one decision-maker

- **Purpose:** Describe review as a fixed Evidence → Story → Clarity → Copy sequence. Distinguish fresh reviewers, who report findings without editing the candidate, from the persistent PM, who classifies every finding and determines routing.
- **Supporting evidence:** F04 (fresh, sequential review lenses); F05 (reviewer and PM ownership boundaries); F08 (lens order and the four PM classifications); F11 (reviewers cannot edit candidates and reviews bind to lens and revision); F12 (PM decisions bind to request, review digest, and revision and accumulate without rewriting earlier outcomes).
- **Reader takeaway:** Review is not a parallel vote or an informal chat: each lens examines an exact candidate revision, and the PM records a classification for every reached finding before the controller advances.

## 4. Revision routing and the three-candidate budget

- **Purpose:** Explain exactly what causes revision, what does not, where review restarts, and how the candidate limit produces a terminal outcome. Make clear that only PM-validated must-fix findings consume another candidate.
- **Supporting evidence:** F09 (must-fix stops later lenses, next candidate restarts at Evidence, optional/invalid findings do not consume a candidate); F10 (candidate 003 hard limit and blocking conditions); I03 (labeled inference about why revisions restart at Evidence); I04 (labeled inference that the cap bounds candidate count, not total duration).
- **Reader takeaway:** A validated must-fix triggers a new candidate and a full review restart; the workflow permits at most candidates 001–003, then succeeds or blocks instead of revising indefinitely.

## 5. Artifact gates make advancement inspectable

- **Purpose:** Explain why a successful process exit or reassuring message is insufficient. Describe the controller's validation of owned files, revision hashes, review metadata, and PM bindings, and connect those checks to the workflow's inspectability without claiming measured reliability.
- **Supporting evidence:** F07 (valid owned files are required after successful exit); F11 (exact lens and SHA-256 revision binding; stale/mismatched artifacts rejected); F12 (decision integrity and accumulated history); F13 (final revalidation before publication); I01 and I02 (explicit architectural inferences about file-based inspectability and rejection of stale, incomplete, or changed inputs).
- **Reader takeaway:** Every transition depends on validated artifacts tied to the active candidate and request, so readers can inspect the same evidence the controller used; the documentation does not establish defect rates or quality gains.

## 6. What remains at success or failure

- **Purpose:** Close by walking through the durable terminal record. Contrast successful publication with a blocked run, while emphasizing that both retain useful history and that `article.md` is created only after final validation.
- **Supporting evidence:** F10 (`workflow.json.block_reason` for blocked outcomes); F13 (`article.md` is byte-for-byte identical to the accepted candidate); F14 (earlier candidates, partial reviews, decisions, status, paths, timestamps, and block reason remain); F15 (`.control/` audit material retained after cleanup and private live state omitted); F16 (completion does not depend on tmux scrollback or chat transcripts); I01 (inspectability as a documented-design inference).
- **Reader takeaway:** Success leaves the accepted article plus its evidence and decision trail; failure leaves an actionable status and the partial history. In either case, the durable run directory—not transient conversation—is the record to inspect.

## Boundaries for the writer

- Do not claim that the workflow improves editorial quality, catches defects at any measured rate, or is easier to audit in practice; U01–U03 remain unresolved.
- Do not speculate about resume support, parallel runs, Linux isolation, publishing, or web interfaces; these are unsupported, omitted, or outside scope (U04–U05 and the brief).
- Avoid presenting O01 or O02 as repository claims. If evaluative language is needed, attribute it as a judgment rather than a documented fact.
