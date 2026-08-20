# write-uuter

`write-uuter` is an agent workflow for turning a question into a publishable
article.

The project is currently in the design phase. The first goal is to define a
durable workflow contract before choosing an execution framework.

## Workflow

1. Define the question, audience, and provisional takeaway.
2. Build material and evidence.
3. Make the story as an outline.
4. Expand the outline into an article.
5. Review the article from independent perspectives.
6. Publish only after the required gates pass.

See:

- [Workflow](docs/workflow.md)
- [Roles](docs/roles.md)
- [Artifacts](docs/artifacts.md)

## Principles

- One PM owns the workflow from intake to publication.
- Writers and reviewers are separate roles.
- Reviewers report findings; they do not edit the article directly.
- Agents hand work off through durable artifacts instead of chat messages.
- Completion is determined from artifact state, not from an agent saying that
  it is done.
- Evidence, story, and writing problems return to the phase that owns them.

