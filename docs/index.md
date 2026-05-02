# cub-gen

**Start from a repo, see what it renders, and know what to edit.**

`cub-gen` is the repo-side traceability CLI for GitOps config. It detects which
Generators your repo uses, renders them locally, and records provenance so
every deployed field traces back to a source file, path, and owner.

The user model is intentionally small:

```text
Component
  -> Variant
      -> Base Variant
      -> Deployment Variant
          -> Target
          -> Connections
          -> Change
          -> Proof
```

A **Component** is the reusable base. A **Variant** is a member of that
Component family. A **Base Variant** is not deployed. A **Deployment Variant**
is the concrete deployed copy or context of that Component. A **Target** is
where a Deployment Variant runs or reconciles. **Connections** are what it is
wired to. A **Change** is what someone wants to alter. **Proof** shows where it
came from, who owns it, what changed, and whether the change is allowed.

Today ConfigHub answers "base or deployment?" from Target presence: no Target
means Base Variant, and Target means Deployment Variant. A Variant is the whole
config context for that Component, implemented as a space containing units.
Base Variants may contain placeholders. Newly cloned Deployment Variants may
also contain placeholders until they are adapted for their Target, and apply
gates should prevent applying them too early.

An **AI Variant** is a Variant whose delta, wiring, or operation is
AI-assisted and governed.

`cub-gen` exists to make those variants explainable before Flux, Argo, or a
human applies them.

Install with Homebrew:

```bash
brew install confighub/tap/cub-gen
```

Or download a release artifact from [GitHub release `v0.3.0`](https://github.com/confighub/cub-gen/releases/tag/v0.3.0).

The ConfigHub plugin form is `cub gen`. From source, you can stage it locally:

```bash
make build-plugin
CUB_CONFIG="$PWD/.tmp/cub-plugin/config.yaml" cub gen --help
```

**gen = Generator.** A Generator is a function on config data. It maps source
config such as `values.yaml`, `score.yaml`, or `application.yaml` to rendered
config: the manifests or other deployable output that reach your cluster.

Many app platforms are Generators in this sense. A Spring platform, Score.dev,
Helm platform, OpenChoreo-style component model, Argo ApplicationSet,
app-of-apps catalog, or app-config manager all take source config plus
environment/platform context and produce deployable config. `cub-gen` starts by
importing and explaining that shape, not by replacing the platform.

cub-gen works with what teams already run today:

- app/config in Git
- OCI artifacts
- Flux/Argo reconciliation to cluster

It adds what those layers do not provide by default:

- source-to-live field traceability (`which file/path controls this field?`)
- ownership-aware edit routing (`who should edit this?`)
- multi-repo platform graphs (`which Components and Variants exist, and which Variants deploy?`)
- PR-friendly proof sidecars (`enrich preview` / `enrich write`)
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

## Import And Adapt Paths

ConfigHub now has three complementary import stories, plus a deployment
adaptation preview:

- `cub gitops import` imports existing ArgoCD/Flux application resources from a cluster or worker target into ConfigHub.
- `cub-gen gitops import` reads source-side Generators such as Helm, Score.dev, Spring Boot, and workflow config, then emits provenance, inverse-edit guidance, and evidence.
- `cub-gen platform import` reads a local multi-repo manifest and emits a read-only Component -> Variant -> Target graph before rewrites.
- `cub-gen platform adapt` reads explicit placeholder/context data for a cloned Deployment Variant and emits a review-only adaptation plan before apply.

Use them for different jobs:

- brownfield GitOps app onboarding -> ConfigHub GitOps import
- source-to-runtime traceability and governed edits -> `cub-gen`
- platform-estate discovery across apps, platform contracts, envs, and rendered repos -> `cub-gen platform import`
- cloned deployment adaptation before `vet-placeholders` clears -> `cub-gen platform adapt`

## What cub-gen is not

- Not a Kubernetes reconciler -- Flux/Argo still reconcile rendered config to the live cluster
- Not a Flux/Argo replacement
- Not an OCI replacement

Source config -> rendered config is a one-way deterministic transform. There is
no automatic live-cluster -> source-config path. There is an outer loop: observe
live state, decide what to change, edit the source config, re-render, and let
Flux or Argo reconcile. cub-gen makes that loop safer by tracing every field
back to its source and gating changes through governed decisions. See [Two
loops, not a triangle](platform.md#two-loops-not-a-triangle).

---

## Supported Generators

| Generator | Profile | Source config | Status |
|-----------|---------|---------------|--------|
| Helm | `helm-paas` | `Chart.yaml` + `values.yaml` | Stable |
| ApplicationSet | `applicationset` | `applicationset.yaml` + pinned inventory | v0.2 preview |
| App-of-apps | `app-of-apps` | root `Application` + child app catalog | v0.4 fixture-backed |
| Score.dev | `scoredev-paas` | `score.yaml` | Stable |
| Spring Boot | `springboot-paas` | `application.yaml` | Stable |
| Backstage IDP | `backstage-idp` | `catalog-info.yaml` | v0.2 preview |
| No Config Platform | `no-config-platform` | Provider config | v0.2 preview |
| OpenChoreo-style | `openchoreo` | Workload + environment/platform bindings | v0.4 fixture-backed |
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

1. App source config lives in Git (`Chart.yaml`, `score.yaml`, `application.yaml`, etc.)
2. **cub-gen** classifies source inputs and rendered targets, then emits provenance with field-origin tracing
3. **cub-gen enrich** can write reviewable sidecar proof under `.cub-gen/enrichment/`
4. **cub-gen publish** produces ConfigHub-ready change bundles with digest verification
5. **ConfigHub** ingests bundles, enforces governed decision state, manages units with revision history
6. **Bridge workers** connect ConfigHub to clusters via HTTP/2 SSE
7. **Flux/Argo** continue to reconcile rendered config to the live cluster -- unchanged

This is why the import surfaces both exist:

- ConfigHub cluster import starts from Argo/Flux objects that already exist,
- `cub-gen` starts from the source repo before those objects become opaque cluster state.

Teams can start with cub-gen locally today and connect to ConfigHub when they need cross-repo queries, policy at write time, and governed execution.

---

## Three invariants (never waived)

1. **Nothing implicit ever deploys** — every deployed artifact is explicit, diffable, traceable
2. **Nothing observed silently overwrites source config** — cluster changes produce governed proposals, not silent overwrites
3. **Configuration is data, not code** — output from Generators is literal values, queryable and diffable

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

-   **Run AI-First Safely**

    Use the current checklist and worked prompt-as-DRY story for safe
    AI-assisted change.

    [AI Example Hygiene Checklist](workflows/ai-example-hygiene-checklist.md) · [Prompt as DRY](workflows/prompt-as-dry.md)

-   **Check what is really proven**

    Use the derived example matrix to see which examples are source-chain verified, in the connected smoke lane, AI-first, or backed by real live proof.

    [Example Truth Matrix](testing/example-truth-matrix.md)

-   **Add Your Generator**

    If your platform has its own framework or layered Generator chain, start
    with the user-facing onboarding path.

    [Platform Generators Manifesto](agentic-gitops/02-design/90-platform-generators-manifesto.md) · [Custom Generator Onboarding](workflows/custom-generator-onboarding.md)

-   **Study Clean Generator Examples**

    See how OpenChoreo, ApplicationSet, and app-of-apps map into the same
    source-to-rendered model.

    [OpenChoreo](agentic-gitops/03-worked-examples/05-openchoreo-generator-worked-example.md) · [Argo Generators](agentic-gitops/03-worked-examples/06-argo-generators-worked-example.md)

-   **Start with workflows (Ops + Swamp)**

    See structural workflow governance first: actions, schedules, approval gates, model/method bindings, and required-step checks.

    [Ops Workflow Generator](triple-styles/style-b-markdown/opsworkflow.md) · [Swamp Generator](triple-styles/style-b-markdown/swamp.md)

-   **Explore the architecture**

    Source/rendered model, field-origin maps, governed execution, contract triples.

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
| **Source config** | Human-editable app/platform config (`values.yaml`, `score.yaml`, `application.yaml`) |
| **Rendered config** | Explicit deployment-facing units/manifests |
| **Provenance** | Record of source inputs, rendered outputs, field-origin map, inverse-edit pointers |
| **Inverse map** | Guidance from changed rendered field to the source file/path to edit safely |
| **Pre-sync** | cub-gen stops before rendered config reaches the live cluster; Flux/Argo own reconciliation |
| **Contract triple** | GeneratorContract + ProvenanceRecord + InverseTransformPlan |

---

## Current status

**Latest shipped:** `v0.3.0` (2026-04-12)

**Current target:** working `v0.4` roadmap: make `cub-gen`'s role obvious as
Component -> Variant -> Base/Deployment -> Target/Connections/Change/Proof.

- repo-first CLI and contract coverage remain green,
- the shipped release includes a first-class standalone `applicationset`
  Generator family,
- flagship Helm, Score, Spring, and Swamp examples carry explicit
  `AI_START_HERE.md`, `prompts.md`, and `contracts.md` bundles,
- connected release status is backed by the repo's `ConfigHub Smoke` lane,
- the active roadmap is tracked by
  [GitHub issue #287](https://github.com/confighub/cub-gen/issues/287) and
  [milestone 4](https://github.com/confighub/cub-gen/milestone/4).

See the [v0.3.0 release notes](releases/v0.3.0.md) for the shipped summary. The
[v0.2-preview.2 Release Plan](releases/v0.2-preview.2-plan.md) and
[v0.2-preview.2 Ship Checklist](releases/v0.2-preview.2-ship-checklist.md) are
kept as archived preview-planning references for how the release bar was closed.
For the active sequence, see the
[v0.4 Working Roadmap](plans/2026-04-30-v0.4-obvious-value-roadmap.md).

- Core flow commands (`discover`, `import`, `cleanup`) frozen and golden-tested
- Platform graph command (`platform import`) emits read-only multi-repo Component/Variant graphs with diagnostics
- Platform adaptation command (`platform adapt`) plans placeholder replacement for cloned Deployment Variants without writing hidden changes
- Enrichment commands (`enrich preview`, `enrich write`) produce sidecar proof without manifest rewrites
- Bridge artifacts (`publish`, `verify`, `attest`, `verify-attestation`) symmetric across all 11 Generators
- Generator catalog (`generators`) with filtering, details, and markdown output
- Local-first: works standalone, connects to [ConfigHub](platform.md) for governed execution
