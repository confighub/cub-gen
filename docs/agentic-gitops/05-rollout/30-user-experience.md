# User Experience: Surfaces, Personas, and Workflows

**Part of:** [AI and GitOps v7 Document Set](../00-index/00-gitops7-index.md)
**Status:** Planning doc (v7)
**Date:** 2026-02-28
**Audience:** Product team, UX designers, developer advocates
**Purpose:** How the agentic GitOps model feels to users — four UI surfaces, two personas, import flow, generator UX, AI tooling, and the Claude skill concept

---

Qualification rule:

Use `Agentic GitOps` only when an active inner reconciliation loop
(`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that
loop, classify the flow as `governed config automation`.

---

## Table of Contents

1. [Four Surfaces, Two Personas](#1-four-surfaces-two-personas)
2. [cub-scout TUI](#2-cub-scout-tui)
3. [ConfigHub GUI](#3-confighub-gui)
4. [CLI Experience](#4-cli-experience)
5. [Import from Git](#5-import-from-git)
6. [Using Generators (The Platform Engineer Experience)](#6-using-generators-the-platform-engineer-experience)
7. [Using Generators (The App Developer Experience)](#7-using-generators-the-app-developer-experience)
8. [How A Leads to B (Import to Generators)](#8-how-a-leads-to-b-import-to-generators)
9. [AI Tooling and the Claude Skill](#9-ai-tooling-and-the-claude-skill)
10. [Surface Connection Diagram](#10-surface-connection-diagram)
11. [Day 2 Scenarios](#11-day-2-scenarios)
12. [Cross-References](#12-cross-references)

---

## 1. Four Surfaces, Two Personas

ConfigHub, cub-scout, the CLI, and AI tooling are not competing interfaces.
Each has a natural entry point and they hand off to each other. Two personas
cut across all four surfaces: the Platform Engineer and the App Developer.

### The Four Surfaces

| Surface | Strength | When |
|---------|----------|------|
| **cub-scout TUI** | Explore one cluster interactively | "What's running? Who owns this?" |
| **ConfigHub GUI** | Fleet view, comparison, editing, governance | "What's deployed where? Change this. Who approved that?" |
| **CLI** | Scripting, CI, power users, Git hooks | Pipelines, automation, terminal workflows |
| **AI Tooling** | Natural language, contextual routing, agentic execution | "Scale payments to 5 in prod", "Why is staging different?", "What broke?" |

### The Two Personas

**Platform Engineer**

- **Owns:** generators, constraints, platform context, the "how" of deployment
- **Thinks in:** charts, schemas, constraint rules, fleet-wide consistency
- **Worries about:** version skew, constraint violations, teams doing unsafe things, generator adoption

**App Developer**

- **Owns:** the app — code, DRY config (score.yaml, values.yaml, application.yaml)
- **Thinks in:** "my app", environments, replicas, image tags, feature flags
- **Worries about:** "is my thing working?", "how do I change this value?", "why is prod different from staging?"

### Surface x Persona Matrix

| View | Platform Engineer | App Developer |
|------|-------------------|---------------|
| **cub-scout TUI** | Full cluster map, all namespaces, ownership overview | Filtered to their app, cross-env status, link to edit |
| **ConfigHub GUI** | Generator catalog, repo view with annotations, fleet-wide constraints, stale render dashboard | App view with env cards, app graph (DRY to WET to resources), field-origin edit view, change timeline |
| **CLI** | Fleet queries, bulk re-render, constraint validation, CI integration | App queries, env diff, `cub edit`, provenance trace |
| **AI tooling** | "Find violations", "What's on old chart versions?", "Pre-check this upgrade" | "Why is prod different?", "Change replicas to 5", "What would break if...?" |

---

## 2. cub-scout TUI

The TUI is the interactive exploration layer. It shows what is running in a
cluster, who owns each resource, and how resources relate to each other. It is
read-only — observation, not mutation.

### What the Platform Engineer Sees

```
┌─ cub-scout map ─────────────────────────────────────────────────────┐
│                                                                      │
│  Namespace: payments                                                 │
│                                                                      │
│  ▶ Deployment/payments-api          Flux    helm@33.2.1    ✓ synced │
│    ├─ ReplicaSet/payments-api-7f8   3/3 ready                       │
│    ├─ Service/payments-api          ClusterIP                        │
│    ├─ Ingress/payments-api          pay.example.com  TLS ✓          │
│    └─ HPA/payments-api              2-10 replicas                    │
│                                                                      │
│  ▶ Deployment/orders-service        Argo   kustomize      ✓ synced  │
│  ▶ CronJob/cert-rotation            Native  (no owner)    ⚠ orphan  │
│                                                                      │
│  [Tab] Details  [t] Trace  [p] Provenance  [d] Drift  [g] Graph    │
└──────────────────────────────────────────────────────────────────────┘
```

They see the whole cluster. Ownership is front and center (Flux, Argo, Native).
Generator info is inline (helm@33.2.1). They can spot orphans and unmanaged
resources immediately.

**Key interactions:**

- **Drill down:** Expand any resource to see its children (ReplicaSet, Service, Ingress, HPA)
- **Trace ownership:** Press `[t]` to trace the full ownership lineage for any resource
- **View evidence:** Press `[p]` to see provenance — generator, inputs, field origins

**Press [p] for provenance on a resource:**

```
┌─ Provenance: Deployment/payments-api ────────────────────────────────┐
│                                                                       │
│  Generator:     helm@3.14.0 / traefik/traefik@33.2.1                │
│  Inputs digest: sha256:a1b2c3...                                     │
│  Rendered:      2h ago (✓ fresh)                                     │
│                                                                       │
│  Field origins:                                                       │
│    spec.replicas           → values-prod.yaml:14     (app-team)      │
│    spec.resources.limits   → platform-context.yaml   (platform-team) │
│    spec.containers[0].image → values-prod.yaml:7     (app-team)      │
│    ingress.tls             → platform constraint     (platform-team) │
│                                                                       │
│  [Enter] Open in ConfigHub   [e] Edit field   [h] History            │
└───────────────────────────────────────────────────────────────────────┘
```

This is the bridge between surfaces. The Platform Engineer sees provenance in
the TUI and can drill into ConfigHub GUI for editing and governance. The TUI is
the exploration layer; the GUI is the action layer.

### What the App Developer Sees

Same TUI, but filtered to their app:

```
$ cub-scout map --app payments-api

┌─ payments-api ──────────────────────────────────────────────────────┐
│                                                                      │
│  Environments:                                                       │
│    dev       1 replica    1.2.3    ✓ healthy    rendered 2h ago     │
│    staging   2 replicas   1.2.3    ✓ healthy    rendered 2h ago     │
│    prod      5 replicas   1.2.1    ⚠ image behind staging           │
│                                                                      │
│  Owner: Flux (helm@33.2.1)                                          │
│  DRY source: acme/payments-api/score.yaml                           │
│                                                                      │
│  [Enter] Details  [d] Diff envs  [e] Edit  [o] Open in ConfigHub   │
└──────────────────────────────────────────────────────────────────────┘
```

App-centric, not cluster-centric. They see their app across all environments
in one view. The "image behind staging" flag catches their attention. They
press `[d]` to diff.

---

## 3. ConfigHub GUI

The GUI is the primary surface for viewing, comparing, editing, and governing
configuration across the fleet. Different personas see different entry points.

### Generator Catalog View (Platform Engineer Perspective)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  ConfigHub > Generators                                                  │
│                                                                          │
│  ┌─ helm ────────────────────────────────────────────────────────────┐  │
│  │  Versions in use:                                                  │  │
│  │    33.2.1  ████████████████░░  12 apps (dev, staging)             │  │
│  │    32.1.0  ████░░░░░░░░░░░░░░   3 apps (prod — upgrade pending)  │  │
│  │                                                                    │  │
│  │  Helm CLI versions:                                                │  │
│  │    3.14.0  ██████████████████  14 apps                            │  │
│  │    3.13.2  ██░░░░░░░░░░░░░░░░   1 app (prod/orders — pinned)     │  │
│  │                                                                    │  │
│  │  Stale renders: 2 (payments-api/prod, billing/prod)               │  │
│  │  Constraint violations: 1 (orders-service missing resource limits) │  │
│  │                                                                    │  │
│  │  [View apps] [View constraints] [View field-origin coverage]      │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌─ score-gen ───────────────────────────────────────────────────────┐  │
│  │  Version: 1.0.0                                                    │  │
│  │  Apps: 4 (payments-api, checkout, notifications, user-service)     │  │
│  │  Field-origin coverage: 100% (all fields traced)                   │  │
│  │  Stale renders: 0                                                  │  │
│  │  Constraint violations: 0                                          │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌─ identity (raw imports) ──────────────────────────────────────────┐  │
│  │  Units: 7 (CronJobs, one-off CRDs, legacy manifests)              │  │
│  │  No generator provenance (Level 0)                                 │  │
│  │  Recommendation: register generators for 3 units with Helm origin  │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  Repos connected: 6                                                      │
│  Total Units: 47  │  Generators: 3  │  Constraints: 12                  │
└─────────────────────────────────────────────────────────────────────────┘
```

This is their catalog. They see:
- Which generators are in use and version skew across the fleet
- Which apps are on old versions (upgrade pending)
- Stale renders and constraint violations at a glance
- Field-origin coverage (how well can users trace fields?)
- Recommendations (identity imports that could be generator-backed)

**Click into a constraint:**

```
┌─ Constraint: tls-required-in-prod ──────────────────────────────────────┐
│                                                                          │
│  Applies when: variant=prod                                              │
│  Rule: Ingress resources must have TLS termination                       │
│                                                                          │
│  Status across fleet:                                                    │
│    ✓ payments-api/prod       TLS enabled (Ingress + cert-manager)       │
│    ✓ orders-service/prod     TLS enabled (Ingress + cert-manager)       │
│    ✗ billing-api/prod        NO TLS — violation since 2026-02-20        │
│      └─ Owner: @billing-team                                             │
│      └─ [Notify] [Create issue] [Block next publish]                    │
│                                                                          │
│    ✓ 11 other apps passing                                               │
└──────────────────────────────────────────────────────────────────────────┘
```

### Repo View (Platform Engineer Perspective)

```
┌─ ConfigHub > Repos > acme/platform-config ──────────────────────────────┐
│                                                                          │
│  charts/                                                                 │
│    traefik/          helm    33.2.1    used by: 4 apps                  │
│    redis/            helm    7.1.0     used by: 6 apps                  │
│    postgres/         helm    12.5.3    used by: 3 apps                  │
│                                                                          │
│  values/                                                                 │
│    payments-api/                                                         │
│      values-dev.yaml         variant=dev      rendered 2h ago  ✓        │
│      values-staging.yaml     variant=staging   rendered 2h ago  ✓        │
│      values-prod.yaml        variant=prod      rendered 5d ago  ⚠ stale │
│        └─ Changed 3d ago: replicas 2→5, not re-rendered                 │
│        └─ [Re-render now] [View diff] [View generated output]           │
│                                                                          │
│  constraints/                                                            │
│    acme-production.yaml      12 rules    0 violations                   │
│    acme-staging.yaml          8 rules    0 violations                   │
│                                                                          │
│  Linked generators: helm (3 charts)                                      │
│  Connected apps: 13                                                      │
└──────────────────────────────────────────────────────────────────────────┘
```

Their repo, but enriched. Every values file shows which variant it maps to,
when it was last rendered, and whether it is stale. Charts show how many apps
depend on them. Stale renders have one-click re-render.

### App View with Environment Cards (App Developer Perspective)

```
┌─ ConfigHub > Apps > payments-api ───────────────────────────────────────┐
│                                                                          │
│  ┌─ Environment Overview ────────────────────────────────────────────┐  │
│  │                                                                    │  │
│  │         dev          staging         prod                         │  │
│  │  ┌──────────┐   ┌──────────┐   ┌──────────┐                     │  │
│  │  │ 1.2.3    │   │ 1.2.3    │   │ 1.2.1 ⚠  │  ← image behind   │  │
│  │  │ 1 replica│   │ 2 replica│   │ 5 replica │                     │  │
│  │  │ no TLS   │   │ TLS ✓    │   │ TLS ✓     │                     │  │
│  │  │ ✓ healthy│   │ ✓ healthy│   │ ✓ healthy │                     │  │
│  │  └──────────┘   └──────────┘   └──────────┘                     │  │
│  │                                                                    │  │
│  │  [Compare envs ▾]  [Promote staging → prod]                      │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌─ App Graph ───────────────────────────────────────────────────────┐  │
│  │                                                                    │  │
│  │  score.yaml ──→ score-gen ──→ ┬─ Deployment/payments-api         │  │
│  │  (DRY)          (generator)   ├─ Service/payments-api             │  │
│  │                               ├─ Ingress/payments-api             │  │
│  │  platform-context.yaml ──────→├─ HPA/payments-api                 │  │
│  │  (constraints)                └─ NetworkPolicy/payments-api       │  │
│  │                                                                    │  │
│  │  Click any resource to see field origins and edit                  │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌─ Recent Changes ──────────────────────────────────────────────────┐  │
│  │  10:30 today  spec.replicas: 2 → 5 (prod)    by: alexis   ✓     │  │
│  │  09:15 today  image: 1.2.1 → 1.2.3 (staging) by: CI       ✓     │  │
│  │  3 days ago   generator: score-gen 0.9 → 1.0  by: platform ✓     │  │
│  │  [Full audit trail →]                                              │  │
│  └────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

This is the app developer's home screen. Three sections:

1. **Environment overview** — cards for each variant, key values at a glance,
   visual flags for drift or version skew. One-click promote.

2. **App graph** — the DRY to generator to WET to resources lineage as a visual graph.
   Click any node to see detail. Click a resource to see field origins. Click a
   field to edit it (routes to DRY source via field-origin map).

3. **Recent changes** — timeline of mutations with attribution. Expandable to
   full ChangeInteractionCards.

### App Graph (App Developer Perspective)

The app graph shows the component dependency graph — how DRY inputs flow through
a generator to produce the Kubernetes resources that make up the app.

```
score.yaml ──→ score-gen ──→ ┬─ Deployment/payments-api
(DRY)          (generator)   ├─ Service/payments-api
                             ├─ Ingress/payments-api
platform-context.yaml ──────→├─ HPA/payments-api
(constraints)                └─ NetworkPolicy/payments-api
```

Click any resource to see field origins and edit. Click the generator node
to see version, schema, and which teams use it. Click the DRY source to
see the file contents and edit directly.

### Field-Origin Edit View (Both Personas)

**Click on Deployment/payments-api in the app graph:**

```
┌─ Deployment/payments-api (variant=prod) ────────────────────────────────┐
│                                                                          │
│  Field                    Value      Source                    Edit      │
│  ─────────────────────────────────────────────────────────────────────── │
│  spec.replicas            5          score.yaml:8              [✏️]      │
│  spec.containers[0].image 1.2.1      score.yaml:6              [✏️]      │
│  resources.limits.memory  1Gi        platform-context.yaml     [🔒]      │
│  resources.limits.cpu     500m       platform-context.yaml     [🔒]      │
│  ingress.host             pay.ex...  score.yaml:11             [✏️]      │
│  ingress.tls              true       platform constraint       [🔒]      │
│  sidecar.inject           true       platform constraint       [🔒]      │
│  replicas.min (HPA)       2          platform constraint       [🔒]      │
│                                                                          │
│  [✏️] = editable by you (app-team)   [🔒] = owned by platform-team      │
│                                                                          │
│  Click [✏️] to edit → ConfigHub commits to your DRY source              │
│  Click [🔒] to see why → shows constraint rule and owner                │
└──────────────────────────────────────────────────────────────────────────┘
```

This is the money screen. Every field shows its current value, where it comes
from (which file, which line), and whether the current user can edit it.

- **Pencil icon** = editable by this user. Clicking it routes through the
  field-origin map to the DRY source. The user enters a new value, previews
  the diff, and commits — all without knowing which repo, file, or line to
  find. ConfigHub resolves it.

- **Lock icon** = platform-controlled. Not editable by the app team. Clicking
  it explains why: shows the constraint rule, the reason string, and the
  owning team. The app developer knows who to contact if they need an
  exception.

**The edit flow:**

1. Click the pencil icon on a field
2. See the DRY source (file, line, current value)
3. Enter the new value
4. Preview the diff (re-rendered output showing exactly what changes)
5. Constraint check runs automatically
6. Commit to the DRY source repo

---

## 4. CLI Experience

The CLI is for automation, scripting, CI pipelines, and power users who live
in the terminal. It mirrors the GUI capabilities but is optimized for
composability and non-interactive use.

### Platform Engineer CLI

```bash
# Fleet-wide generator status (CI dashboard, weekly report)
confighub query --generator helm --format json | jq '.[] | select(.stale)'

# Find all constraint violations across prod
confighub validate --variant prod --format table

# Check if a chart upgrade will break anything before rolling it out
confighub render --chart traefik@33.2.1 --dry-run --all-apps --diff

# Re-render all stale Units
confighub render --stale --all --confirm

# Fleet-wide generator view as a table
confighub query --generator helm --format table

# Stale render detection across the fleet
confighub check-freshness

# Constraint checking before publish
confighub validate --pre-publish
```

### App Developer CLI

```bash
# Quick status — app across all environments
confighub query --app payments-api --format table

# Cross-environment diff
confighub diff payments-api --from dev --to prod

# Trace field origins (provenance)
confighub provenance payments-api --variant prod

# Edit a field via field-origin map (routes to DRY, commits, re-renders)
cub edit payments-api --field spec.replicas --variant prod --value 5

# Trace a specific field to its source
confighub provenance payments-api --variant prod --field spec.replicas
```

---

## 5. Import from Git

### Who Is This User?

They already have YAML in Git. Helm charts, Kustomize overlays, raw manifests,
or a mix. They use Flux or Argo today. They are not going to rewrite anything.
They want to see what they have, compare across environments, and maybe get
governance later.

### The Experience

**Step 1: Connect a repo**

```
$ confighub import connect --repo github.com/acme/platform-config

Scanning repository...
Found:
  12 Helm releases (charts/ + values files)
   4 Kustomize overlays (overlays/dev, staging, prod, canary)
   3 raw manifests (jobs/, one-off CRDs)

Import as Units? [y/n]
```

No migration. No restructuring. ConfigHub reads the repo, classifies what it
finds, and offers to import. The repo structure stays exactly as it is. Git
remains the source of truth.

**Step 2: See everything in one place**

```
$ confighub query --format table

APP              VARIANT   TYPE         SOURCE                        UPDATED
payments-api     dev       helm         charts/payments + values-dev   2h ago
payments-api     staging   helm         charts/payments + values-stg   2h ago
payments-api     prod      helm         charts/payments + values-prod  3d ago
orders-service   dev       kustomize    overlays/dev/orders            1d ago
orders-service   prod      kustomize    overlays/prod/orders           5d ago
cert-rotation    —         raw          jobs/cert-rotation.yaml        30d ago
```

This is the first "aha" moment. They have never seen all their config in one
table before. They have had to navigate repo directories, read kustomization.yaml
files, or run `helm list` per cluster.

**Step 3: Compare across environments**

```
$ confighub diff payments-api --from dev --to prod

  replicas:                1 → 5
  image.tag:               1.2.3 → 1.2.1        ← prod is behind
  resources.limits.memory: 256Mi → 1Gi
  tls:                     false → true
  values file:             values-dev → values-prod
  chart version:           same (33.2.1)
```

Second "aha." They can see why environments differ — not just that the YAML
is different, but which values-file knobs are turned differently. For Helm
imports, this is a diff of the values files, not the rendered output.

**Step 4: Detect staleness (passive, automatic)**

```
$ confighub check-freshness

⚠ payments-api (variant=prod)
  values-prod.yaml changed 3 days ago
  Last render: 5 days ago
  → WET output is stale. Re-render to pick up changes.

✓ orders-service (all variants fresh)
✓ cert-rotation (raw manifest, no generator)
```

They did not have this before. They had "did Flux sync?" but not "did someone
change the values file and forget to re-render?"

### What Import Does NOT Do

This is critical:

- Does not copy YAML into ConfigHub and delete it from Git
- Does not require changing repo structure
- Does not require changing Flux/Argo configuration
- Does not require adopting generators

Import is observation + queryability. Git is still the source. ConfigHub is a
read layer that makes the source queryable and comparable.

### What Import Enables Next

- **Provenance tracking (Level 2):** ConfigHub knows which generator and inputs
  produced each Unit, so it can detect staleness and trace field origins
- **Editing through ConfigHub:** once field-origin maps exist, users can edit
  values through ConfigHub instead of hunting for files in Git
- **Governance:** policy evaluation on the imported config before it reaches clusters

### The Pitch

"You keep your repos exactly as they are. ConfigHub reads them and gives you a
single query surface across all your environments. You can see what's deployed,
compare environments, and find stale renders — without changing your workflow."

---

## 6. Using Generators (The Platform Engineer Experience)

### Who Is This User?

The Platform Engineer builds the generator. They define constraints (TLS required
in prod, min 2 replicas, images from approved registry). They want app teams to
self-serve without breaking things.

### Step 1: Register an Existing Generator

```
$ confighub generators register \
    --name helm \
    --version "3.14.0 / traefik@33.2.1" \
    --input-schema values-schema.json

Registered: helm
Input schema: 14 fields (replicas, image.tag, resources.*, tls, ...)
Field-origin map: auto-generated from schema
```

They are not building a new tool. They are naming what they already have (Helm,
Kustomize, Score) and telling ConfigHub about it. The field-origin map can be
auto-generated from the values schema for common cases — or hand-authored for
complex charts.

### Step 2: Define Constraints

```yaml
# platform-constraints.yaml
apiVersion: confighub.io/v1
kind: PlatformConstraints
metadata:
  name: acme-production
spec:
  when:
    variant: prod
  rules:
    - field: spec.replicas
      min: 2
      reason: "Production requires at least 2 replicas for availability"
    - field: spec.template.spec.containers[*].image
      pattern: "registry.acme.io/*"
      reason: "Production images must come from the approved registry"
    - field: spec.template.spec.containers[*].resources.limits
      required: true
      reason: "All production containers must have resource limits"
```

These are declarative rules, not scripts. They evaluate against the rendered
WET output before publish. The app developer never sees them unless they
violate one.

### Step 3: See What the Generator Produces Across All Teams

```
$ confighub query --generator helm --format table

APP              VARIANT   CHART VERSION   VALUES DIGEST   RENDERED    STATUS
payments-api     prod      33.2.1          sha256:a1b2     3d ago      ⚠ stale
payments-api     dev       33.2.1          sha256:c3d4     2h ago      ✓ fresh
orders-service   prod      32.1.0          sha256:e5f6     1d ago      ✓ fresh
orders-service   staging   33.2.1          sha256:g7h8     1d ago      ✓ fresh
```

The Platform Engineer can see every team's generator output in one view. They
can spot version skew (orders-prod is still on chart 32.1.0), stale renders,
and constraint violations across the fleet.

---

## 7. Using Generators (The App Developer Experience)

### Who Is This User?

The App Developer writes DRY intent (Score workload, Helm values, Spring Boot
config). They do not want to think about Kubernetes primitives. They want to
change a value and have it deploy correctly.

### Step 1: Write DRY Intent (They Already Do This)

```yaml
# score.yaml (or values.yaml, or application.yaml — whatever they already use)
apiVersion: score.dev/v1b1
kind: Workload
metadata:
  name: payments-api
spec:
  image: ghcr.io/acme/payments-api:1.2.3
  ports:
    - port: 8080
  service:
    expose: true
    host: pay.example.com
```

Nothing new here. They write what they already write.

### Step 2: See What Gets Generated

```
$ confighub render --input score.yaml --variant prod --preview

Generator: score-gen@1.0.0
Context: variant=prod, platform=acme-production

Generated resources:
  Deployment/payments-api
    replicas: 2              (from: platform constraint min-replicas-2)
    image: ghcr.io/acme/payments-api:1.2.3  (from: score.yaml:6)
    resources.limits.memory: 1Gi             (from: platform default)
  Service/payments-api
    port: 8080               (from: score.yaml:8)
  Ingress/payments-api
    host: pay.example.com    (from: score.yaml:11)
    tls: true                (from: platform constraint tls-required-in-prod)

Constraint check:
  ✓ tls-required-in-prod
  ✓ min-replicas-2
  ✓ images-from-approved-registry

Field origins saved. You can trace any field back to its source.
```

This is the key moment. The developer sees exactly what gets produced, why
each field has its value (from their input vs. from platform constraint), and
whether it passes validation — all before anything is deployed.

### Step 3: Change a Value Through ConfigHub

Later, they need to scale to 5 replicas in prod:

```
$ confighub edit --app payments-api --variant prod --field spec.replicas

Current value: 2
Source: score.yaml:8 → platform constraint min-replicas-2 (floor)
  Note: platform sets minimum 2. You can go higher.

New value: 5

→ Updating score.yaml in acme/payments-api repo...
→ Re-rendering with updated input...
→ Diff:
    spec.replicas: 2 → 5
    (no other fields affected)
→ Constraint check: ✓ all pass

→ Commit to acme/payments-api? [y/n]
```

They edited through ConfigHub. ConfigHub resolved the field-origin map, found
the DRY source (score.yaml), wrote the change there, re-rendered, validated,
and showed the diff. The developer did not need to know that `replicas` in the
Deployment comes from their Score workload filtered through a platform
constraint. They said "change replicas to 5" and the system did the right
thing.

### Step 4: See What Is Deployed (Ongoing)

```
$ confighub query --app payments-api --format table

VARIANT   REPLICAS   IMAGE TAG   TLS    GENERATOR          RENDERED    STATUS
dev       1          1.2.3       false  score-gen@1.0.0    2h ago      ✓
staging   2          1.2.3       true   score-gen@1.0.0    2h ago      ✓
prod      5          1.2.3       true   score-gen@1.0.0    just now    ✓
```

This becomes the daily view. Open ConfigHub, see your app across all
environments. Click any field to trace it or change it.

### Step 5: When Something Breaks

```
$ confighub provenance payments-api --variant prod

generator:     score-gen@1.0.0
inputs.digest: sha256:a1b2c3...  (✓ fresh)
rendered.at:   2026-02-26T10:30:00Z

field_origins:
  spec.replicas             → score.yaml:8           (editable_by: app-team)
  spec.resources.limits     → platform-context.yaml  (editable_by: platform-team)
  spec.containers[0].image  → score.yaml:6           (editable_by: app-team)
  ingress.tls               → platform constraint    (editable_by: platform-team)

Last 3 changes:
  10:30  spec.replicas: 2 → 5      (by: alexis, decision: ALLOW)
  09:15  image.tag: 1.2.1 → 1.2.3  (by: ci/github-actions, decision: ALLOW)
  3d ago chart: score-gen 0.9 → 1.0 (by: platform-team, decision: ALLOW)
```

Full provenance. Every field traceable. Every change attributed. No archaeology.

---

## 8. How A Leads to B (Import to Generators)

Import (path A) is how most users start. They have YAML in Git, they connect
it, they see the queryable view. That builds trust.

Generators (path B) come when they are ready. Maybe they register their existing
Helm charts as generators to get provenance. Maybe they adopt Score for new
services. Maybe the platform team builds a Spring Boot generator. Each step is
incremental.

The adoption path:

```
1. Import from Git           → "I can see everything in one place"
2. Add provenance metadata   → "I can trace what produced this"
3. Add field-origin maps     → "I can trace any field to its source"
4. Edit through ConfigHub    → "I can change values without Git archaeology"
5. Add constraints           → "I can validate before deploy"
6. Add governance            → "I can prove who authorized what"
```

At no point does the user need to abandon their existing workflow. Git stays.
Flux/Argo stay. Helm stays. ConfigHub adds a layer on top — first for
visibility, then for traceability, then for editing, then for governance.

---

## 9. AI Tooling and the Claude Skill

### Why a Skill

If Claude already knows the cub-scout/cub-track/ConfigHub toolchain, it
naturally routes people to the right command instead of letting them do kubectl
archaeology or hand-edit generated YAML. The skill becomes the adoption
mechanism — people learn the tools by being guided to them in context, not by
reading docs.

### Trigger Moments

| User is doing... | Claude should suggest... |
|---|---|
| `kubectl get deploy -o yaml` to understand what is running | `confighub query --app X` or `cub-scout trace` |
| Grepping across repos to find which values file controls a field | `confighub provenance X --variant prod` (field-origin map) |
| Hand-editing a generated manifest | "This field has a DRY source — edit through `confighub edit` or `cub edit`" |
| `git log` across 3 repos to find what changed | `cub-track explain --commit` or `confighub audit` |
| Writing a config change commit without governance context | `cub-track` trailers and ChangeInteractionCard |
| Diffing YAML files manually across environments | `confighub diff --from staging --to prod` |
| Wondering if a change is safe to deploy | `confighub validate --pre-publish` |

### What the Skill Needs to Know

Three things:

1. **The commands** — what each tool can do, with real syntax
2. **The operating boundary** — which tool to reach for (not cub-scout for
   staleness, not ConfigHub for reconciliation)
3. **The editing model** — when to redirect to DRY, when WET is legitimate,
   how field-origin maps work

### Skill Flavors (Not Mutually Exclusive)

**1. Tool routing skill (primary)**

The core skill. Loaded when people are working with Kubernetes config. Makes
Claude fluent in cub-scout/cub-track/ConfigHub commands. Proactively suggests
the right tool when it sees someone doing things the hard way.

**2. Architecture oracle skill**

Loaded when working in any of the related repos. Knows: the two-loop model,
operating boundary (who owns what), the DRY/WET split, field-origin maps,
generator maturity levels, the invariants. When you ask "should cub-scout detect
stale renders?" it can answer "no, that's ConfigHub's publishing pipeline — see
[06 — Governed Execution](../02-design/40-governed-execution.md#2-evidence-and-the-drift-loop)"
without you having to remember.

**3. Doc consistency skill**

When editing planning docs (GitOps\*.md, roadmap, product guides), checks that
new text is consistent with the invariants, the operating boundary, and the
terminology. Catches things like: calling cub-scout a "controller", saying
ConfigHub "reconciles", using "overlay" when you mean "variant", or describing a
feature that violates "nothing implicit ever deploys."

**4. Issue/PR drafting skill**

When filing issues or writing PRs in cub-scout or cub-track, pre-fills the
structure from CLAUDE.md (deterministic tests, graceful degradation, definition
of done) and frames the work in terms of this strategy. "This PR adds
`cub-track suggest` — the DRY-source redirection capability from
[03 — Field-Origin Maps](../02-design/20-field-origin-maps-and-editing.md#7-cub-track-as-the-redirection-layer)."

**5. Cross-repo context skill**

The hardest one. When you are in the cub-scout repo working on evidence bundles,
it knows how that connects to cub-track's ChangeInteractionCards and ConfigHub's
Units. When you are in cub-track adding the `suggest` command, it knows the
field-origin map schema from [03 — Field-Origin Maps](../02-design/20-field-origin-maps-and-editing.md#1-what-is-a-field-origin-map).

### AI Dialog Examples

**Example 1: Claude suggests confighub query instead of kubectl**

```
Platform Engineer: "Are any prod apps running old chart versions?"

Claude: Let me check.

  $ confighub query --variant prod --format table

  3 apps are on traefik 32.1.0 (current: 33.2.1):
    - orders-service/prod (pinned — see constraint override)
    - billing-api/prod (not pinned — likely missed the upgrade)
    - legacy-gateway/prod (identity import, no generator)

  For billing-api, I can:
  1. Show the diff between chart 32.1.0 and 33.2.1 output
  2. Run a pre-publish validation with the new chart
  3. Create a PR to bump the chart version

  For legacy-gateway, this is a raw import — no generator. You'd need to
  register one or update the manifest manually.
```

```
Platform Engineer: "Run the pre-publish check for billing-api with the new chart"

Claude: Running validation...

  $ confighub validate --unit billing-api --variant prod \
      --override chart=traefik@33.2.1 --pre-publish

  ✓ tls-required-in-prod         (unchanged)
  ✓ min-replicas-2               (unchanged)
  ✗ images-from-approved-registry
    → traefik 33.2.1 uses image docker.io/traefik:v3.1
    → your allow-list requires registry.acme.io/*
    → You'll need to mirror the image before upgrading.

  The chart upgrade would fail the image registry constraint. Should I
  create an issue for the billing team to mirror the image first?
```

**Example 2: Claude explains field origins and suggests cub edit**

```
App Developer: "Why is my payments-api different in prod vs staging?"

Claude: Let me compare.

  $ confighub diff payments-api --from staging --to prod

  Three differences:
  1. image.tag: 1.2.3 (staging) → 1.2.1 (prod)
     → Prod is one patch behind. Staging was updated by CI 2 hours ago,
       prod hasn't been promoted yet.

  2. replicas: 2 (staging) → 5 (prod)
     → You changed prod replicas to 5 earlier today.
       Staging has min-replicas-2 from platform constraint.

  3. resources.limits.memory: 512Mi (staging) → 1Gi (prod)
     → This comes from platform-context.yaml, not your config.
       Platform team sets different resource limits per environment.
       You can't change this directly — contact @platform-team.

  Want me to promote the image tag from staging to prod?
```

```
App Developer: "Yes, promote the image"

Claude: I'll update the image tag in your DRY source for the prod variant.

  $ cub edit payments-api --field spec.containers[0].image \
      --variant prod --value "ghcr.io/acme/payments-api:1.2.3"

  → Source: score.yaml:6 (acme/payments-api)
  → Re-rendering...
  → Diff: image: 1.2.1 → 1.2.3 (no other fields affected)
  → Constraint check: ✓ all pass

  → Commit to acme/payments-api? [y/n]
```

```
App Developer: "Actually, what would break if I set replicas to 1 in prod?"

Claude: Let me check the constraints.

  The platform constraint min-replicas-2 applies when variant=prod.
  Setting replicas to 1 would be blocked at pre-publish:

  $ confighub validate --unit payments-api --variant prod \
      --override spec.replicas=1 --pre-publish

  ✗ min-replicas-2 (replicas: 1 < minimum: 2)
    → "Production requires at least 2 replicas for availability"
    → Owner: @platform-team

  You can't go below 2 in prod. You could:
  - Set it to 1 in dev or staging (no constraint there)
  - Ask @platform-team for an exception
```

The AI experience is the most natural for app developers. They do not need to
learn commands or navigate a GUI. They ask what they want in plain language.
The AI resolves the right tool, the right surface, and the right action. It
knows which fields they can edit (field-origin map + editable_by), which are
constrained (platform policy), and what would happen if they tried something
that violates constraints.

### AI as cub-track Governance Layer

When AI agents propose changes, cub-track evaluates them against governance
policy and escalates when necessary:

```
AI Agent (automated): "Scale payments-api to 10 replicas in prod based on
  traffic forecast"

cub-track: Recording mutation intent...
  Intent: scale spec.replicas 5 → 10 (variant=prod)
  Agent: traffic-forecaster
  Trust tier: 2 (medium-risk, requires human approval)

  → Escalating to human approval.
  → Created PR: acme/payments-api#47 "Scale payments-api to 10 replicas"
  → Notified: @alexis (on-call), #payments-ops (Slack)

  Waiting for approval...

Alexis (in ConfigHub GUI or Slack): [Approve]

cub-track: Decision: ALLOW (approved by alexis)
  → Committing to score.yaml
  → Re-rendering...
  → Publishing...
  → Flux reconciling...
  → cub-scout observing...
  → ✓ 10/10 replicas healthy

  Attestation recorded:
    actor: traffic-forecaster
    approved_by: alexis
    intent_revision: sha256:...
    observed_result: applied, healthy
```

### Common Redirects

| Instead of: | Suggest: |
|---|---|
| `kubectl get deploy -o yaml` | `cub-scout trace deploy/X -n Y` |
| `grep -r "replicas"` across repos | `confighub provenance X --variant prod` |
| `git log` across 3 repos | `cub-track explain --commit <sha>` |
| Manually diffing YAML files | `confighub diff X --from staging --to prod` |
| Editing rendered manifests | `confighub edit --field Y` (routes to DRY) |
| "Is this safe to deploy?" | `confighub validate --pre-publish` |
| "What changed in the last day?" | `confighub audit X --last 24h` |

### Consolidated Skill Definition

The skill definition below is a reference artifact — extractable for use in Claude
skill configuration, CLAUDE.md files, or agent system prompts. It encodes the
commands, boundaries, editing model, and invariants in one place.

```markdown
# Skill: Agentic GitOps Toolchain

## When to activate

User is working with Kubernetes configuration, GitOps workflows, Helm/Kustomize
output, or asking about what's deployed, what changed, or how to change config.

## Core tools

Three tools, distinct responsibilities. Never confuse them.

### cub-scout — Observe the cluster

Read-only. Compares cluster reality against the reconciler's intended state.
Does NOT know about DRY sources, generators, or the publishing pipeline.

cub-scout map                    # TUI: what's running, who owns it
cub-scout map list               # Plain text ownership map
cub-scout trace deploy/X -n Y    # Trace ownership lineage for a resource
cub-scout scan                   # Scan for drift and anomalies
cub-scout gitops status          # GitOps pipeline health
cub-scout graph export           # Resource dependency graph
cub-scout patterns detect        # Detect configuration patterns

Use when: "What's running?", "Who owns this?", "Is there drift?", "What does
the cluster look like?"

Do NOT use for: "What inputs produced this?", "Is the render stale?", "Who
approved this change?"

### cub-track — Record and redirect mutations

Git-native mutation ledger. Records governed mutation history. Redirects WET
edits back to DRY sources when field-origin maps are available.

cub-track enable                          # Install hooks, init metadata branch
cub-track explain --commit <sha>          # Why was this change made?
cub-track search --text "replicas"        # Search mutation history
cub-track search --agent codex            # Find changes by a specific agent
cub-track search --decision ESCALATE      # Find escalated changes

# Planned (post-MVP):
cub-track explain --commit <sha> --fields # + show DRY origins for changed fields
cub-track suggest                         # Before commit: check if fields have DRY origins

Use when: "Why was this changed?", "What did the agent do?", "Am I editing the
right layer?"

### confighub — Store, query, edit, govern

System of record for intended state (WET). Resolves field-origin maps. Routes
editing to DRY sources. Detects stale renders. Does NOT reconcile.

confighub query --app X --format table              # What's deployed across envs
confighub get unit X --variant prod                  # Full Unit contents
confighub provenance X --variant prod                # Generator, inputs, field origins
confighub diff X --from staging --to prod            # Cross-environment diff
confighub validate --unit X --pre-publish            # Check constraints before deploy
confighub audit X --variant prod --last 24h          # Who changed what, with proof
confighub check-freshness --app X                    # Are renders stale?
confighub edit --unit X --variant prod --field Y     # Edit via field-origin map

Use when: "What's deployed where?", "Why is prod different from staging?",
"Where does this value come from?", "Change this field", "Is this safe to
deploy?", "Who approved this?"

## The editing model

When a user wants to change a config value, route them correctly:

**If the field has a DRY source** (generator-backed: Helm values, Score workload,
Spring Boot config): edit the DRY source, not the WET output. ConfigHub resolves
which file and line via the field-origin map. Use `confighub edit` or `cub edit`.

**If the field is platform-native** (network policy, security constraints, resource
quotas, deployment topology, governance state): edit directly in ConfigHub. There
is no DRY upstream.

**If the user is hand-editing generated YAML**: warn them. Suggest the field-origin
path instead. If they proceed, the change is an overlay — transitional, not
permanent. cub-track will classify it as overlay drift.

**Emergency overrides**: legitimate in WET with a TTL. Record via cub-track. Promote
to DRY or let expire.

## Operating boundary (never cross these)

- cub-scout observes the CLUSTER, not the generator pipeline
- cub-track records MUTATIONS, not runtime state
- ConfigHub STORES and GOVERNS, it does not reconcile
- Flux/Argo RECONCILE, they are not replaced
- Staleness (DRY changed, WET not re-rendered) is ConfigHub's concern, not cub-scout's
- Drift (cluster differs from intended state) is cub-scout's concern, not ConfigHub's

## Invariants (never violate)

1. Nothing implicit ever deploys
2. Nothing observed silently overwrites intent
3. Configuration is data, not code
```

### Key Concepts the Skill Must Preserve

These ideas should be embedded in any skill or persistent context:

- **Field-origin maps**: the concept, the schema, the editing UX it enables.
  Generators produce a mapping from WET output fields to DRY input sources
  (file, path, line, editable_by). Not all generators can produce complete maps
  — start with the fields people actually edit.

- **cub-scout's scope boundary**: observes cluster state vs. reconciler intended
  state. Does NOT compare stored WET against hypothetical generator output. "Is
  the WET stale relative to DRY inputs?" is ConfigHub's publishing pipeline
  concern, detected via `inputs.digest` comparison.

- **cub-track's redirection role**: when someone edits WET directly for a field
  that has a DRY origin, cub-track detects this and guides the user back.
  `suggest` (pre-commit check) and `explain --fields` (post-commit enrichment).

- **The WET-native edit taxonomy**: what legitimately lives in ConfigHub without
  a DRY upstream — platform-injected fields, emergency overrides with TTL,
  deployment topology, variant lifecycle, governance metadata, legacy YAML
  (identity generator).

- **The adoption ladder**: View → Compare → Trace → Edit → Govern. Start
  with app config — highest-frequency, lowest-stakes edits. Build trust
  incrementally.

- **Overlay drift as a distinct category**: WET field changed without
  `inputs.digest` change. Different from runtime drift (cluster vs. intended
  state). Overlays are transitional escape hatches, not steady-state.

- **ConfigHub as the editing surface that writes back to DRY**: users don't
  need to know which repo, file, or line. ConfigHub resolves it via the
  field-origin map, commits upstream, triggers re-render.

---

## 10. Surface Connection Diagram

```
                        App Developer                Platform Engineer
                             │                              │
                    ┌────────┴────────┐            ┌───────┴────────┐
                    │                  │            │                 │
                    ▼                  ▼            ▼                 ▼
              AI Tooling         ConfigHub GUI    ConfigHub GUI    AI Tooling
             "scale my app"      App View         Catalog View    "find violations"
              │                  │  │  │           │  │  │          │
              │    ┌─────────────┘  │  └──────┐    │  │  │          │
              │    │                │          │    │  │  │          │
              ▼    ▼                ▼          ▼    ▼  ▼  ▼          ▼
           cub edit            Field-origin   App    Repo  Generator  confighub
           (DRY commit)        edit view      Graph  View  Catalog   validate
              │                    │           │
              │                    │           │
              ▼                    ▼           ▼
         cub-scout TUI ◄──── "explore in TUI" ───── deep link from graph
         (cluster view)
```

**Key handoffs:**

- **AI to CLI:** AI executes `cub edit`, `confighub validate` under the hood
- **GUI to TUI:** "Open in cub-scout" link from app graph to interactive cluster view
- **TUI to GUI:** "Open in ConfigHub" link from resource detail to editing/governance
- **CLI to GUI:** commands output URLs to ConfigHub for visual inspection

All four surfaces share the same data model through the ConfigHub API. The TUI
reads cluster state directly but links to ConfigHub for anything involving
provenance, editing, or governance. The CLI is the scripting surface — every
GUI action has a CLI equivalent. The AI surface wraps the CLI with natural
language understanding and contextual routing.

---

## 11. Day 2 Scenarios

These six scenarios describe what changes in practice — the questions you can
answer on Day 2 that you could not answer on Day 1.

### Scenario 1: 3am Incident — "What changed?"

A Deployment is crashing in production. The on-call engineer opens ConfigHub
and asks: "What changed in `payments-api` in the last 24 hours?"

**Without ConfigHub:** Check Git log across 3 repos. Run `kubectl describe` on
the Deployment. Cross-reference Helm release history with Flux events. Grep
Slack for who merged what. Reconstruct the timeline manually. This takes 30-60
minutes — during an outage.

**With ConfigHub:** One query returns the Unit's revision history: the chart
version bumped from 32.1.0 to 33.2.1, the values file changed `replicas` from
3 to 2, and the generator version shows a Helm CLI upgrade from 3.13.2 to
3.14.0. The evidence bundle shows the specific field-level diff between
intended and observed state. The cub-track mutation record shows who merged it,
what policy evaluated it, and what the decision was. Time to root cause: under
5 minutes.

### Scenario 2: Fleet Visibility — "What's deployed where?"

The VP of Engineering asks: "Which version of the payments API is running in
each environment? Are any clusters running a version more than a week old?"

**Without ConfigHub:** Someone writes a script to iterate over 40 repos and
kubectl contexts, parsing labels and annotations. The script breaks when a
team uses a different naming convention. The answer arrives three days later,
already stale.

**With ConfigHub:** One API call: `GET /units?app=payments-api` returns every
Unit across every variant and target, with the generator version, render
timestamp, and current image tag. Filter by `rendered.at < 7d` to find stale
renders. The answer is instant, live, and queryable by anyone with dashboard
access.

### Scenario 3: AI Agent Audit — "What did the agent do?"

An AI agent made 47 configuration changes overnight — scaling, image updates,
feature flag toggles. The compliance team asks: "Can you prove each change was
authorized and validated?"

**Without ConfigHub:** The evidence is scattered: Git commits (some with useful
messages, some not), Flux sync events, Argo health checks, and a chat
transcript in Claude Code. Reconstructing which change was authorized by what
policy requires manual archaeology across systems.

**With ConfigHub + cub-track:** Each of the 47 changes has a
ChangeInteractionCard with structured fields: intent (what the agent proposed),
decision (which policy evaluated it, ALLOW/ESCALATE/BLOCK), execution (what
actually ran), and outcome (what cub-scout observed afterward). The mutation
ledger is the compliance export — every change has a who, what, why, and proof
chain. The audit takes minutes, not days.

### Scenario 4: Config Change — "I need to change replicas in prod"

A developer needs to scale `payments-api` from 2 to 5 replicas in production.

**Without ConfigHub:** The developer knows the app runs on Helm, but is not sure
which repo has the values file, what it is called, or whether there are
per-environment overrides. They find a `values.yaml` in the platform repo, change
`replicas: 5`, open a PR — and learn from code review that production uses
`values-prod.yaml` in a different directory, and their change only affected
staging. They fix it, wait for CI, merge, wait for Flux to sync. Elapsed time:
45 minutes, two PR cycles.

**With ConfigHub:** The developer opens ConfigHub, finds `payments-api` variant
`prod`, clicks `spec.replicas`. ConfigHub shows: "This field is generated from
`values-prod.yaml:14` in `acme/platform-config`. Current value: 2." The developer
changes it to 5. ConfigHub commits the change to the correct file in the correct
repo, the generator re-renders, and the diff shows exactly one field changed.
Elapsed time: 2 minutes. The developer never needed to know which repo, which
file, or which values overlay — ConfigHub resolved it via the field-origin map.

### Scenario 5: CI Hub — "Our pipeline should call a platform API, not edit files"

The platform team has a standard CI job that applies rollout metadata and image
pin updates across many services.

**Without ConfigHub:** The CI job is full of repo/file/path logic. It clones repos,
patches YAML, handles naming exceptions, opens ad hoc PRs, and fails when teams
use different layouts. Every new repo pattern means more scripting.

**With ConfigHub:** CI calls a semantic API:

1. `POST /v1/changes/upsert` with intent (`set image tag`, `set replicas`, labels)
2. `POST /v1/changes/{change_id}/evaluate`
3. `POST /v1/changes/{change_id}/decision`
4. ConfigHub performs deterministic write-back PR/MR and links execution evidence

The pipeline stays the same shape. The brittle file-edit logic disappears.

### Scenario 6: Fleet Change Wave — "Patch this CVE everywhere, safely"

A base image CVE requires updating all internet-facing services in `prod` and
`staging` across dozens of repos.

**Without ConfigHub:** Teams open many independent PRs, with inconsistent
labels/approvals and no single change identity. Progress tracking is manual and
audit evidence is fragmented.

**With ConfigHub:** Platform opens one governed wave keyed by one `change_id`:

1. Query affected Units by labels and provenance.
2. Generate per-service change proposals.
3. Apply trust-tier decisions per target (`ALLOW | ESCALATE | BLOCK`).
4. Execute allowed targets, escalate protected ones.
5. Collect one verification/attestation chain per target under one wave identity.

Result: one fleet campaign with explicit governance and complete audit closure.

---

## 12. Cross-References

| Document | Relationship |
|----------|-------------|
| [01-introducing-agentic-gitops.md](../01-vision/01-introducing-agentic-gitops.md) | The "why": invariants, classical GitOps gaps, agentic changes |
| [02-generators-prd.md](../02-design/10-generators-prd.md) | Generators PRD: DRY-to-WET pipeline, generator lifecycle, maturity levels |
| [03-field-origin-maps-and-editing.md](../02-design/20-field-origin-maps-and-editing.md) | Field-origin maps, editing model, authoring landscape |
| [04-app-model-and-contracts.md](../02-design/30-app-model-and-contracts.md) | Entity definitions, operating boundary, constraints, operations |
| [05-cub-track.md](../05-rollout/10-cub-track.md) | cub-track: mutation ledger, ChangeInteractionCards, governance |
| [06-governed-execution.md](../02-design/40-governed-execution.md) | Two-loop model, evidence, trust tiers, attestation |
| [08-adoption-and-reference.md](../05-rollout/40-adoption-and-reference.md) | Adoption path, value analysis, Day 2 scenarios, FAQ |
