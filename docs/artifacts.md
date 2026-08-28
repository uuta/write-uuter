# Artifacts

## Run layout

```text
<run-dir>/
├── brief.md
├── workflow.json
├── model-policy.json                 # exact validated policy for this run
├── evidence/
│   ├── sources.md
│   ├── firsthand.md                 # optional
│   ├── screenshot-requests.json     # optional, Researcher-owned
│   ├── screenshots.json             # generated, only with captured images
│   └── assets/
│       └── screenshots/
│           └── shot-001.png         # controller-owned captured evidence
├── claim-ledger.md
├── outline.md
├── visual-inputs.json                # generated, only when an input was staged
├── drafts/
│   ├── article-001-prose.md          # Writer prose draft
│   ├── article-001.md                # Writer assembly output, reviewed
│   └── article-00N.md
├── visuals/
│   └── article-00N/
│       ├── plan.md                   # Visual Editor rationale
│       ├── manifest.json             # generated revision binding
│       └── assets/
│           └── vin-001.png           # placed local image, read-only
├── reviews/
│   └── article-00N/
│       ├── evidence/{result.json,report.md}
│       ├── story/{result.json,report.md}
│       ├── clarity/{result.json,report.md}
│       └── copy/{result.json,report.md}
├── pm-decisions/
│   └── article-00N.md
├── article.md                        # success only
└── .control/
    ├── invocations/                  # published before each process is ready
    ├── prompts/                      # post-cleanup audit copies
    ├── logs/
    └── exits/
```

Earlier candidates, their prose drafts, visual plans and assets, partial lens
sequences, reviews, and PM decisions are kept when a revision occurs or a run
blocks. `article.md` is written only after all
four final-candidate lenses pass PM routing and is byte-for-byte identical to
that candidate.

## Model policy and invocation audit

`model-policy.json` is a byte-exact copy of the `models.json` the controller
validated for this run, and `workflow.json.model_policy_digest` is its SHA-256
digest. Both are written during initialization, so a blocked run preserves them
too.

Before any launched process is considered ready, the controller atomically
publishes one immutable record per invocation under `.control/invocations/`:

```json
{
  "invocation": "008-reviewer-copy",
  "role": "reviewer",
  "lens": "copy",
  "candidate": 1,
  "provider": "codex",
  "model": "gpt-5.6-luna",
  "reasoning_effort": "low",
  "model_policy_digest": "sha256:..."
}
```

The recorded values come from the same validated profile that built the process
arguments, so the artifacts and the launched command cannot disagree. Records
are retained for successful, blocked, timed-out, and non-zero invocations, and
never contain authentication values, environment values, prompts, or secrets.
A run that blocks because a provider, model, or quota is unavailable records the
effective provider, model, and reasoning effort in `block_reason` and never
retries with a different profile.

## Screenshot evidence

`evidence/screenshot-requests.json` is optional and Researcher-owned. It holds
zero to five entries; `id`, `url`, `reason`, and at least one `supports` claim
ID are required, and `selector` is the only optional page-targeting field:

```json
{
  "screenshots": [
    {
      "id": "shot-001",
      "url": "https://example.com/report",
      "reason": "Shows the interface described by claim-004",
      "supports": ["claim-004"],
      "selector": "main"
    }
  ]
}
```

IDs are filename-safe (letters, digits, `-`, `_`) and unique when compared
case-insensitively, because they become file names and the run may live on a
case-insensitive file system. Unknown fields
and duplicate object keys are rejected recursively. Every `supports` entry must
appear in `claim-ledger.md` as a whole token, so `claim-004` never matches
`claim-0041`. A URL must be a public `https://` page with a DNS hostname on the
default port; embedded credentials, `localhost`, `.local`, `.internal`, `.lan`,
`.home`, `.intranet`, `.corp`, IP literals, single-label hosts, non-default
ports, and non-HTTPS schemes are rejected. An explicitly empty `screenshots`
list means the same as no artifact: no capture, and no Cloudflare credential.
The Researcher may not create `evidence/assets/screenshots/`; that directory is
controller-owned. The artifact is copied into the run only when it asks for at
least one capture, and it is stored read-only (`0444`).

`evidence/screenshots.json` is controller-generated. No agent writes it:

```json
{
  "schema_version": 1,
  "engine": "cloudflare-chromium",
  "viewport": { "width": 1280, "height": 800 },
  "screenshots": [
    {
      "id": "shot-001",
      "path": "evidence/assets/screenshots/shot-001.png",
      "requested_url": "https://example.com/report",
      "selector": "main",
      "captured_at": "2026-08-26T10:00:00Z",
      "supports": ["claim-004"],
      "reason": "Shows the interface described by claim-004",
      "engine": "cloudflare-chromium",
      "media_type": "image/png",
      "byte_size": 51234,
      "width": 1280,
      "height": 800,
      "sha256": "sha256:..."
    }
  ]
}
```

The manifest and the images are read-only (`0444`) immutable inputs to later
roles. Their bytes are exactly the accepted capture response, and the visual
pass consumes them in place: every capture joins the visual input pool with
origin `screenshot`, and placing one binds those exact bytes into the candidate
revision without re-acquiring anything.

## Visual inputs

`brief.md` may carry an optional level-two `## Visual inputs` section. Each
non-empty list item names one file relative to the content root:

```md
## Visual inputs

- images/current-workflow.png
- evidence/browser-result.webp
```

An absent or empty section is valid and stages nothing. Supported formats are
PNG, JPEG, and WebP, matched on both the extension and the file signature, with
a 10 MiB per-file limit. Absolute paths, parent traversal, symlinked files,
symlinked path components, directories, special files, missing files, and
unsupported formats are rejected before the run directory is created and before
any agent starts. The controller keeps a private copy of the accepted bytes, so
replacing a source file afterwards changes nothing inside the run.

`visual-inputs.json` is controller-generated and written read-only (`0444`)
only when the run staged at least one image. Every screenshot the controller
captured joins the same pool with origin `screenshot`:

```json
{
  "schema_version": 1,
  "inputs": [
    {
      "id": "vin-001",
      "origin": "brief",
      "source": "images/current-workflow.png",
      "media_type": "image/png",
      "byte_size": 105393,
      "sha256": "sha256:...",
      "staged_path": "visual-inputs/vin-001.png"
    }
  ]
}
```

## Visual plan and manifest

`visuals/article-00N/plan.md` is the Visual Editor's human record. It names
each evaluated opportunity, its location, the selected action, and the concrete
reason that action improves explanation or reading flow, including the
opportunities it rejected. The controller requires it to record every
opportunity ID and its action, so the artifact stays inspectable.

The Visual Editor also writes a machine-readable `plan.json`, but only inside
its own isolated workspace: the controller validates it and does not retain it
in the run directory. The validated decisions are preserved durably in the
`actions` array of `manifest.json` below, so `visuals/article-00N/` holds
`plan.md`, `manifest.json`, and `assets/` and nothing else.

That workspace plan uses this validated shape. Supported actions are exactly
`mermaid`, `existing_local_asset`, `restructure_text`, and `none`; unknown
fields and duplicate object keys are rejected recursively:

```json
{
  "schema_version": 1,
  "source_revision": "sha256:...",
  "opportunities": [
    {
      "id": "vis-001",
      "location": "section: How the stages connect",
      "action": "mermaid",
      "rationale": "The paragraph describes one sequence of stages.",
      "mermaid": "flowchart TD\n    a[Brief] --> b[Researcher]"
    },
    {
      "id": "vis-002",
      "location": "section: opening",
      "action": "none",
      "rationale": "The opening is two sentences long."
    }
  ]
}
```

`source_revision` must equal the SHA-256 of the exact prose draft, so a plan
bound to another revision cannot advance. A `mermaid` entry carries a fenceless
diagram and no asset or alt text; an `existing_local_asset` entry names a
staged asset placed at most once plus meaningful alt text and no diagram; a
`restructure_text` or `none` entry carries none of them. At least one evaluated
opportunity is required and at most twenty are accepted. `location` is at most
512 bytes, `rationale` and `alt_text` at most 1024 bytes each, and a `mermaid`
body at most 8192 bytes. Every bound is repeated in the Visual Editor
assignment, so the contract is knowable before the plan is written.

`visuals/article-00N/manifest.json` is controller-generated and read-only
(`0444`). It binds the plan, the source prose revision, the assembled
candidate, and every referenced local asset to the revision the four lenses
review:

```json
{
  "schema_version": 1,
  "candidate": 1,
  "source_prose": { "path": "drafts/article-001-prose.md", "sha256": "sha256:..." },
  "plan": { "path": "visuals/article-001/plan.md", "sha256": "sha256:..." },
  "actions": [],
  "assets": [
    {
      "id": "vin-001",
      "opportunity_id": "vis-001",
      "path": "visuals/article-001/assets/vin-001.png",
      "origin": "brief",
      "source": "images/current-workflow.png",
      "media_type": "image/png",
      "byte_size": 105393,
      "sha256": "sha256:...",
      "alt_text": "Diagram of the current workflow stages"
    }
  ],
  "article": { "path": "drafts/article-001.md", "sha256": "sha256:..." },
  "reviewed_revision": "sha256:...",
  "prose_characters_before": 964,
  "prose_characters_after": 421
}
```

`prose_characters_before` and `prose_characters_after` count explanation
characters outside fenced blocks and image references, so a candidate that
placed a diagram or an image can be checked to have shortened the prose rather
than duplicated it.

The controller revalidates the manifest and every regular file it names before
each lens starts, at the publication boundary, and again after the succeeded
state is persisted. A malformed or stale manifest, a stale source prose
revision, a changed plan, a missing, unsafe, non-regular, or replaced asset, an
asset outside the candidate asset directory, and a recorded revision that
disagrees with the bound bytes all block the run.

## Candidate revision

`workflow.json.current_revision` and every `reviewed_revision` name the
canonical candidate revision, which covers the assembled Markdown and the bytes
of every referenced local asset:

- with no referenced asset it is the SHA-256 of the assembled Markdown, so runs
  without visual assets are unchanged;
- with at least one referenced asset it is the SHA-256 of this exact block,
  with assets in lexicographic path order:

```text
write-uuter/candidate-revision/v1
article sha256:<article digest>
assets <count>
<asset path> sha256:<asset digest>
```

Each line ends with a single newline. Because the digest covers the asset
bytes, a visual cannot be replaced after its candidate passed review without
invalidating every review binding and the publication check.

## Review result

`result.json` has this validated minimum shape:

```json
{
  "status": "fix_required",
  "lens": "clarity",
  "reviewed_revision": "sha256:...",
  "findings": [
    {
      "id": "clarity-001",
      "severity": "must_fix",
      "location": "section: Introduction",
      "problem": "The requested action is not visible.",
      "suggested_direction": "State the action before the background."
    }
  ]
}
```

Allowed statuses are `clean`, `fix_required`, and `blocked`. A clean result has
no findings; `fix_required` has at least one. IDs are non-empty and unique
within the result, all finding fields contain non-whitespace text,
lens/revision must match the assignment, and `report.md` contains one exact,
complete five-field entry per JSON finding in the same order.
The controller rejects unknown fields and duplicate object keys recursively,
including keys inside every finding object; JSON's usual last-value-wins
behavior is never used for routing artifacts.

## PM decision

`pm-decisions/article-00N.md` contains one fenced JSON object. Its `lenses` map
accumulates only the lenses reached for that candidate:

```json
{
  "reviewed_revision": "sha256:...",
  "lenses": {
    "evidence": {
      "request_id": "request-token",
      "review_digest": "sha256:...",
      "decisions": []
    },
    "story": {
      "request_id": "request-token",
      "review_digest": "sha256:...",
      "decisions": [
        {
          "finding_id": "story-001",
          "decision": "invalid",
          "reason": "The candidate already satisfies the outline."
        }
      ]
    }
  }
}
```

Each reached lens covers every finding exactly once. Unknown, duplicate, or
missing IDs, stale revisions, mismatched request IDs or review digests, dropped
prior lenses, changed prior classification lists, changed prior routing
outcomes, multiple fenced documents, and prepopulated future lenses fail the
contract.
Unknown fields and duplicate keys are rejected recursively in the top-level
document, each lens record, and every decision object.

## workflow.json

`workflow.json` is atomically rewritten and is the controller's source of
truth. Schema version 1 records:

- `status`: `running`, `succeeded`, or `blocked`;
- `phase`: current controller phase;
- `current_candidate` and `current_revision` (`sha256:<hex>`);
- `active_role`;
- `current_prose_revision` (`sha256:<hex>`), the revision of the prose draft the
  accepted visual plan was bound to;
- stable relative `artifact_paths`, including `model_policy`, `invocations`,
  `screenshot_requests`, `screenshots`, `screenshot_assets`, `visual_inputs`,
  and `visuals` (the capture paths and `visual_inputs` name stable paths that
  exist only when a capture or a staged input happened);
- `model_policy_digest`: SHA-256 of the validated policy copied into the run;
- `review_attempt_count` (one per launched reviewer process; it is incremented
  only after tmux has been asked to start that reviewer, so a failure before
  the launch request never claims a reviewer process, while a launch request
  that timed out ambiguously is counted conservatively);
- `started_at`, `updated_at`, and terminal `completed_at` timestamps;
- terminal `block_reason` when blocked.

The durable `.control/` directory is controller-owned and preserves audit
copies of generated prompt assignments, per-invocation logs, and exit
markers. The live runner executable, sandbox profiles, ready/ownership records,
PM requests, and agent workspaces exist only in a controller-private sibling
directory. Worker `0` markers record natural successful exits. The persistent
PM is controller-terminated after final validation, so its durable `143`
marker is explicitly controller-synthesized cleanup evidence rather than a
natural PM exit; interrupted workers likewise retain a nonzero terminal marker.
Private state is removed after verified cleanup of controller-launched and
controller-trackable tmux/process identities and is never copied into
`.control/`. An intentionally ancestry-escaping hostile process is outside
this slice's guarantee; complete containment is deferred to a future
container/VM design. Completion markers are atomically renamed into
place; a partial marker cannot advance the workflow. Controller artifact access
rejects symlink path components, symlinked tree roots/directories, and
non-regular files. Editorial completion never depends on tmux scrollback or
chat transcripts.
