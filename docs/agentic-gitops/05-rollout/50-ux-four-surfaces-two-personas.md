# User Experience: Four Surfaces, Two Personas

**For:** Brian, Jesper
**From:** Alexis (via conversation with Claude, 2026-02-26)
**Companion to:** ux-import-and-generators-for-brian-jesper.md

---

## The Four Surfaces

| Surface | Strength | When |
|---------|----------|------|
| **cub-scout TUI** | Explore one cluster interactively | "What's running? Who owns this?" |
| **ConfigHub GUI** | Fleet view, comparison, editing, governance | "What's deployed where? Change this. Who approved that?" |
| **CLI** | Scripting, CI, power users, Git hooks | Pipelines, automation, terminal workflows |
| **AI tooling** | Natural language, contextual routing, agentic execution | "Scale payments to 5 in prod", "Why is staging different?", "What broke?" |

These are not competing interfaces. Each has a natural entry point and they
hand off to each other.

---

## The Two Personas

### Platform Engineer

**Owns:** generators, constraints, platform context, the "how" of deployment
**Thinks in:** charts, schemas, constraint rules, fleet-wide consistency
**Worries about:** version skew, constraint violations, teams doing unsafe things,
generator adoption

### App Developer

**Owns:** the app — code, DRY config (score.yaml, values.yaml, application.yaml)
**Thinks in:** "my app", environments, replicas, image tags, feature flags
**Worries about:** "is my thing working?", "how do I change this value?", "why is
prod different from staging?"

---

## Surface 1: cub-scout TUI

### What the Platform Engineer sees

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

This is the bridge. They see provenance in the TUI. They can drill into
ConfigHub GUI for editing and governance. The TUI is the exploration layer;
the GUI is the action layer.

### What the App Developer sees

Same TUI, but they filter to their app:

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
press [d] to diff.

---

## Surface 2: ConfigHub GUI

### What the Platform Engineer sees: Generator Catalog

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

### What the Platform Engineer sees: Repo View

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
when it was last rendered, and whether it's stale. Charts show how many apps
depend on them. Stale renders have one-click re-render.

### What the App Developer sees: App View

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

This is their home screen. Three things:

1. **Environment overview** — cards for each variant, key values at a glance,
   visual flags for drift or version skew. One-click promote.

2. **App graph** — the DRY→generator→WET→resources lineage as a visual graph.
   Click any node to see detail. Click a resource to see field origins. Click a
   field to edit it (routes to DRY source via field-origin map).

3. **Recent changes** — timeline of mutations with attribution. Expandable to
   full ChangeInteractionCards.

**Click on Deployment/payments-api in the graph:**

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

This is the money screen. Every field, its value, where it comes from, and
whether the app developer can edit it. The [✏️]/[🔒] distinction answers
"what can I change?" instantly. Clicking edit routes through the field-origin
map to the DRY source. Clicking a locked field explains the constraint.

---

## Surface 3: CLI

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
```

### App Developer CLI

```bash
# Quick status
confighub query --app payments-api --format table

# Diff environments
confighub diff payments-api --from staging --to prod

# Edit a field (routes to DRY, commits, re-renders)
cub edit payments-api --field spec.replicas --variant prod --value 5

# Trace a field to its source
confighub provenance payments-api --variant prod --field spec.replicas
```

---

## Surface 4: AI Tooling

The AI surface is where Claude (or any AI agent) acts as a contextual
assistant that knows the toolchain and routes users to the right place.

### What the Platform Engineer gets from AI

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

The AI knows the operating boundary. It uses `confighub validate`, not
`kubectl`. It knows constraints exist and checks them. It suggests the
right next step.

### What the App Developer gets from AI

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

The AI experience is the most natural for app developers. They don't need to
learn commands or navigate a GUI. They ask what they want in plain language.
The AI resolves the right tool, the right surface, and the right action. It
knows which fields they can edit (field-origin map + editable_by), which are
constrained (platform policy), and what would happen if they tried something
that violates constraints.

### AI as cub-track governance layer

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

---

## How the Surfaces Connect

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

Key handoffs:
- AI → CLI: AI executes `cub edit`, `confighub validate` under the hood
- GUI → TUI: "Open in cub-scout" link from app graph to interactive cluster view
- TUI → GUI: "Open in ConfigHub" link from resource detail to editing/governance
- CLI → GUI: commands output URLs to ConfigHub for visual inspection

---

## Summary: Who Sees What

| View | Platform Engineer | App Developer |
|------|-------------------|---------------|
| **cub-scout TUI** | Full cluster map, all namespaces, ownership overview | Filtered to their app, cross-env status, link to edit |
| **ConfigHub GUI** | Generator catalog, repo view with annotations, fleet-wide constraints, stale render dashboard | App view with env cards, app graph (DRY→WET→resources), field-origin edit view, change timeline |
| **CLI** | Fleet queries, bulk re-render, constraint validation, CI integration | App queries, env diff, `cub edit`, provenance trace |
| **AI tooling** | "Find violations", "What's on old chart versions?", "Pre-check this upgrade" | "Why is prod different?", "Change replicas to 5", "What would break if...?" |
