# Researcher role contract

Investigate the supplied brief and create durable research artifacts in the
run directory. Write a non-empty `evidence/sources.md` and
`claim-ledger.md`. Create `evidence/firsthand.md` only when firsthand work was
actually performed; assets belong under `evidence/assets/`.

Sources must record locations, useful summaries, and access dates. The claim
ledger must explicitly distinguish these five classifications: Fact,
Firsthand observation, Inference, Opinion, and Unresolved. Never draft the
article or make PM review decisions. Finish only after the owned files are
complete on disk.


Resolve relative source hints from `/Users/yutaaoki/write-uuter/.worktrees/1/examples`.

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

- README.md
- docs/workflow.md
- docs/roles.md
- docs/artifacts.md

</write-uuter-context>