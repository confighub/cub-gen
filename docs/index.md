# cub-gen

**Governance + traceability sidecar for GitOps.**

**gen = generator.** A generator is a function that maps DRY source (`values.yaml`, `score.yaml`, `application.yaml`) to WET rendered output (the manifests that reach your cluster). `cub-gen` detects which generators your repo uses, runs the mapping, and records provenance — so every deployed field traces back to a source file, line, and owner.

cub-gen works with what teams already run today:

- app/config in Git
- OCI artifacts
- Flux/Argo reconciliation to cluster

It adds what those layers do not provide by default:

- source-to-live field traceability (`which file/path controls this field?`)
- ownership-aware edit routing (`who should edit this?`)
- governed safety decisions before deploy (`ALLOW/ESCALATE/BLOCK`)

---

## Why cub-gen exists

Classical GitOps is strong at applying changes.

Teams still struggle to answer:

- Which source file/path controls this live field?
- Did the right team edit the right thing?
- Can we block unsafe edits before they hit cluster?

AI-assisted changes make this gap wider because more changes happen faster.

cub-gen adds the import/provenance layer that answers these questions while keeping Flux/Argo as reconciler.

## What cub-gen is not

- Not a Kubernetes reconciler — Flux/Argo still own WET→LIVE
- Not a Flux/Argo replacement
- Not an OCI replacement

DRY→WET is a one-way deterministic transform. There is no automatic LIVE→DRY path. But there is an outer loop: observe live state → decide what to change → edit DRY → re-render WET → reconcile to LIVE. cub-gen makes that outer loop safe by tracing every field back to its source and gating changes through governed decisions. See [Two loops, not a triangle](platform.md#two-loops-not-a-triangle).

---

## Supported generators

| Generator | Profile | DRY source | Status |
|-----------|---------|------------|--------|
| Helm | `helm-paas` | `Chart.yaml` + `values.yaml` | Stable |
| Score.dev | `scoredev-paas` | `score.yaml` | Stable |
| Spring Boot | `springboot-paas` | `application.yaml` | Stable |
| Backstage IDP | `backstage-idp` | `catalog-info.yaml` | v0.2 preview |
| No Config Platform | `no-config-platform` | Provider config | v0.2 preview |
| Ops Workflow | `ops-workflow` | Operations config (structural workflow governance) | v0.2 preview |
| C3 Agent | `c3agent` | Fleet config | v0.2 preview |
| Swamp | `swamp` | Workflow config (graph/model-method governance) | v0.2 preview |

---

## Part of the ConfigHub platform

cub-gen is the **local-first entry point** to the [ConfigHub platform](platform.md).

- Local mode: standalone, no backend login required.
- Connected mode: `cub auth login` + ConfigHub backend decision APIs.

ConfigHub backend OSS is available today:

- [confighubai/confighub](https://github.com/confighubai/confighub)

1. **DRY** app intent lives in Git (`Chart.yaml`, `score.yaml`, `application.yaml`, etc.)
2. **cub-gen** classifies DRY inputs + WET targets and emits provenance with field-origin tracing
3. **cub-gen publish** produces ConfigHub-ready change bundles with digest verification
4. **ConfigHub** ingests bundles, enforces governed decision state, manages units with revision history
5. **Bridge workers** connect ConfigHub to clusters via HTTP/2 SSE
6. **Flux/Argo** continue to reconcile WET to LIVE — unchanged

Teams can start with cub-gen locally today and connect to ConfigHub when they need cross-repo queries, policy at write time, and governed execution.

---

## Three invariants (never waived)

1. **Nothing implicit ever deploys** — every deployed artifact is explicit, diffable, traceable
2. **Nothing observed silently overwrites intent** — cluster changes produce governed proposals, not silent overwrites
3. **Configuration is data, not code** — output from generators is literal values, queryable and diffable

---

## Start here

<div class="grid cards" markdown>

-   **Understand the vision**

    Learn why GitOps needs a governance layer for the AI era.

    [What is Agentic GitOps?](agentic-gitops/01-vision/01-introducing-agentic-gitops.md)

-   **Try it in 10 minutes**

    Build, discover, import, and inspect provenance from a Helm repo.

    [Getting Started](getting-started.md)

-   **Interpret confidence quickly**

    Understand when to auto-route edits and when to escalate for review.

    [Confidence Scores](workflows/confidence-scores.md)

-   **Start with workflows (Ops + Swamp)**

    See structural workflow governance first: actions, schedules, approval gates, model/method bindings, and required-step checks.

    [Ops Workflow Generator](triple-styles/style-b-markdown/opsworkflow.md) · [Swamp Generator](triple-styles/style-b-markdown/swamp.md)

-   **Explore the architecture**

    DRY/WET model, field-origin maps, governed execution, contract triples.

    [Architecture](agentic-gitops/02-design/00-agentic-gitops-design.md)

-   **See the full platform**

    How cub-gen connects to ConfigHub, bridge workers, and Flux/ArgoCD.

    [The ConfigHub Platform](platform.md)

-   **Contribute**

    Deterministic behavior, proof-first delivery, test-backed PRs.

    [Contributing](contributing-guide.md)

</div>

---

## Terminology

| Term | Meaning |
|------|---------|
| **DRY source** | Human-editable app/platform intent (`values.yaml`, `score.yaml`, `application.yaml`) |
| **WET rendered units** | Explicit rendered deployment-facing units/manifests |
| **Provenance** | Record of DRY inputs, rendered outputs, field-origin map, inverse-edit pointers |
| **Inverse map** | Guidance from changed WET field to where to edit DRY safely |
| **Pre-sync** | cub-gen stops before WET-to-LIVE; Flux/Argo own reconciliation |
| **Contract triple** | GeneratorContract + ProvenanceRecord + InverseTransformPlan |

---

## Current status

**v0.2-preview-parity-locked** (2026-03-06)

- Core flow commands (`discover`, `import`, `cleanup`) frozen and golden-tested
- Bridge artifacts (`publish`, `verify`, `attest`, `verify-attestation`) symmetric across all 8 generators
- Generator catalog (`generators`) with filtering, details, and markdown output
- Local-first: works standalone, connects to [ConfigHub](platform.md) for governed execution
