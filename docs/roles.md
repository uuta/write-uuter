# Roles

Every role has explicit inputs, owned artifacts, allowed actions, and a
completion condition. A role is complete only when its artifact satisfies the
contract described in [Artifacts](artifacts.md).

## Human Editor

- Owns: the initial `brief.md` and decisions that require editorial judgment.
- Complete when: the brief passes the PM intake gate, or the requested human
  decision is recorded.

## PM

- Owns: the complete run, workflow state, assignments, routing, and review
  decisions.
- Inputs: all workflow artifacts and external state.
- Complete when: the article is published or the run is explicitly blocked or
  cancelled.
- Must not: infer completion from an agent conversation alone.

The PM classifies each review finding as one of:

- `valid_must_fix`
- `valid_optional`
- `invalid`
- `needs_human_judgment`

An `invalid` decision must include a reason. Decisions about thesis, voice, or
claim strength may be routed to the Human Editor.

## Source Researcher

- Owns: `evidence/sources.md`.
- Complete when: each relevant source includes its location, supporting
  passage or summary, and access date.

## Firsthand Researcher

- Owns: `evidence/firsthand.md` and referenced screenshots or logs.
- Complete when: the conditions, procedure, result, and evidence location are
  recorded so that the observation can be checked.

Source and firsthand research may run in parallel.

## Fact Checker

- Owns: `claim-ledger.json`.
- Complete when: each material claim is classified as fact, observation,
  inference, opinion, or unresolved and points to its evidence.

## Story Editor

- Owns: `outline.md`.
- Complete when: headings form a coherent sequence and each heading has a
  purpose, evidence, and takeaway.

## Writer

- Owns: `article.md`.
- Complete when: the outline is expanded into publishable prose and no
  unresolved placeholders remain.
- Must not: accept or reject review findings on behalf of the PM.

## Reviewers

Reviewers are independent from the Writer and may run in parallel. They own
findings, not `article.md`.

| Reviewer | Owned artifact | Review lens |
| --- | --- | --- |
| Evidence Reviewer | `reviews/evidence.json` | Claims match the investigation |
| Story Reviewer | `reviews/story.json` | The narrative fulfills the article contract |
| Clarity Reviewer | `reviews/clarity.json` | The intended audience can understand it |
| Copy Reviewer | `reviews/copy.json` | Grammar, spelling, and consistency |

A reviewer is complete when the review artifact is tied to a specific article
revision and reports `clean`, `fix_required`, or `blocked`.

## Publisher

- Owns: `publish-report.md`.
- Complete when: the publication target, published URL, time, and exact article
  revision are recorded.

