# Workflow

## Overview

The overview contains the cross-phase return paths. Keep this diagram small so
that the complete process remains readable at a glance.

```mermaid
flowchart TD
    t["Define question, audience, and provisional takeaway"]
    a["Build material and evidence"]
    ms["Make a story (draft)"]
    w["Write and review the article"]
    pu["Publish"]

    t --> a
    a --> ms
    ms -->|Evidence missing| a
    ms --> w
    w -->|Story problem| ms
    w -->|Evidence missing| a
    w -->|Passed| pu
```

## Detailed flow

The detailed diagram shows the normal direction and local correction loops.
Cross-phase returns terminate at an explicit routing result; the PM follows the
return path defined in the overview.

```mermaid
flowchart TD
    t["Define question, audience, and provisional takeaway"]

    subgraph a["Build material and evidence"]
        a-c["Make a plan"]
        a-cs["Collect sources"]
        a-v["Collect firsthand evidence"]
        a-f["Fact-check claims"]
        a-l["Create claim ledger<br/>Fact / Observation / Inference / Opinion / Unresolved"]

        a-c --> a-cs
        a-c --> a-v
        a-cs --> a-f
        a-v --> a-f
        a-f --> a-l
    end

    subgraph ms["Make a story (draft)"]
        ms-h["Put headings in order"]
        ms-d["Add purpose, evidence, and takeaway under each heading"]
        ms-r{"Story contract satisfied?"}

        ms-h --> ms-d
        ms-d --> ms-r
        ms-r -->|Story problem| ms-h
    end

    subgraph w["Write an article"]
        w-w["Expand the outline into polished prose"]
        w-r-c["Evidence review"]
        w-r-s["Story review"]
        w-r-e["Clarity review"]
        w-r-p["Copy review"]
        w-r-g{"Passed?"}

        w-w --> w-r-c
        w-w --> w-r-s
        w-w --> w-r-e
        w-w --> w-r-p
        w-r-c --> w-r-g
        w-r-s --> w-r-g
        w-r-e --> w-r-g
        w-r-p --> w-r-g
        w-r-g -->|Writing problem| w-w
    end

    back-e["Return to evidence phase<br/>(see overview)"]
    back-s["Return to story phase<br/>(see overview)"]
    pu["Publish"]

    t --> a-c
    a-l --> ms-h
    ms-r -->|Evidence missing| back-e
    ms-r -->|Passed| w-w
    w-r-g -->|Evidence missing| back-e
    w-r-g -->|Story problem| back-s
    w-r-g -->|Passed| pu
```

## Routing rule

The PM validates review findings and routes them to the role that owns the
problem. A reviewer never changes the workflow phase or edits the article
directly.

