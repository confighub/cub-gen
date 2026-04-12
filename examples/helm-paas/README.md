# Helm PaaS — Governed Helm for Platform Teams

Your Helm charts already define the contract between platform and app teams.
The platform owns the chart structure; app teams own their values overlays.
ConfigHub makes that contract explicit, traceable, and auditable — without
changing how you use Helm.

This is the platform-first flagship path for `cub-gen`.

The first question it should answer is:

"Which values file or chart layer actually controls the field I am looking at?"

This example now owns four flagship proof paths:

1. source-side provenance,
2. governed ownership proof for allowed vs blocked edits,
3. layered ApplicationSet plus cluster-selector plus overlay proof,
4. connected + live Helm proof through an example-owned runtime wrapper.

## What this proves today

| Slice | Status | How to prove it now |
|-------|--------|---------------------|
| Source-side Helm provenance | Real | `./examples/helm-paas/demo-local.sh` |
| Governed ALLOW + BLOCK ownership proof | Real | `./examples/helm-paas/demo-governed-change.sh` |
| Layered ApplicationSet + overlay + security proof | Real | `./examples/helm-paas/demo-layered-trace.sh` |
| Deep connected bridge path | Real | `./examples/helm-paas/demo-connected.sh` |
| Connected + live Helm proof from this example | Real | `RECONCILER=both ./examples/helm-paas/demo-runtime.sh` |
| Runtime WET->LIVE harness beneath the wrapper | Paired | `RECONCILER=both ./examples/live-reconcile/demo-local.sh` |

Known gaps still open:

- umbrella/subchart/alias/conditional chart support is still open work ([#238](https://github.com/confighub/cub-gen/issues/238))
- one-command multi-variant fan-out is not exposed yet ([#237](https://github.com/confighub/cub-gen/issues/237))
- helper chains, `lookup`, and external value-source provenance honesty are still partial ([#239](https://github.com/confighub/cub-gen/issues/239))

## AI-first bundle

Use these together if the first operator is an AI assistant or if you want a
safe, step-by-step handoff:

- [AI_START_HERE.md](./AI_START_HERE.md)
- [prompts.md](./prompts.md)
- [contracts.md](./contracts.md)

## Start here by audience

| If you are... | First command | Then do this | What you can inspect |
|---|---|---|---|
| Existing Helm plus Argo/Flux user | `./examples/helm-paas/demo-local.sh` | `cub auth login && RECONCILER=argo ./examples/helm-paas/demo-runtime.sh` | values ownership, rendered targets, connected evidence, and a live Argo deployment |
| Existing ConfigHub user adding Helm governance | `cub auth login && ./examples/helm-paas/demo-connected.sh` | `RECONCILER=both ./examples/helm-paas/demo-runtime.sh` | change/bundle/decision output, then Argo+Flux runtime proof |
| Existing Argo-first operator starting from a running app | ConfigHub GitOps import + [`cub-scout`](https://github.com/confighub/cub-scout) first | come back to `./examples/helm-paas/demo-local.sh` for source provenance | live app first, source trace second |

That sequence gives you:

1. source-side provenance and ownership,
2. connected ConfigHub evidence and governed decision state,
3. WET to LIVE reconciliation proof across Argo and Flux,
4. cluster-side inspection of what actually ran.

## What you can inspect after each step

| Step | Command | Inspectable result |
|---|---|---|
| Local source-side proof | `./examples/helm-paas/demo-local.sh` | field origin, inverse-edit guidance, dry inputs, and rendered targets |
| Local governed ownership proof | `./examples/helm-paas/demo-governed-change.sh` | one app-team change that passes and one platform-contract change that fails the DRY ownership gate |
| Local layered Kubara-like proof | `./examples/helm-paas/demo-layered-trace.sh` | cluster labels, ApplicationSet selector, values overlay selection, and one blocked customer security weakening |
| Deep connected bridge proof | `./examples/helm-paas/demo-connected.sh` | change ID, bundle digest, attestation, and backend decision/query output |
| Connected + live proof | `RECONCILER=both ./examples/helm-paas/demo-runtime.sh` | live Deployment rollout, pods, services, and reconciler status for Flux and Argo |

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding Helm governance | Jump to [Run from ConfigHub](#run-from-configHub-connected-mode) |
| **Existing Helm/Argo/Flux user** adding ConfigHub | Jump to [Fastest path](#fastest-path-to-believe-it) then connect later |

Both paths lead to the same outcome: governed Helm with field-origin tracing.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real app** | `payments-api` Helm chart (Deployment + Service + ConfigMap) |
| **Real cluster objects** | Kubernetes Deployment, Service, ConfigMap |
| **Real inspection target** | `kubectl get deployment payments-api -o yaml` |
| **GitOps transport** | Argo CD Application or Flux HelmRelease |

## Argo/Flux note

Argo is not an afterthought in this example. The repo includes both:

- `gitops/argo/application.yaml`
- `gitops/flux/helmrelease.yaml`

The current runtime proof is exposed through `./examples/helm-paas/demo-runtime.sh`,
which uses the shared [`live-reconcile`](../live-reconcile/) harness under the
hood so you can inspect both reconcilers from the same flagship Helm story.

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "Who owns this deployed field?" | Field-origin tracing to `values.yaml` or `templates/` | App team changes → ALLOW |
| "What file do I edit to fix this?" | Inverse-edit guidance with confidence scores | Platform changes → ESCALATE/BLOCK |
| "How do I prove what changed?" | Evidence bundle with attestation chain | Audit trail for compliance |

## Domain POV (Helm platform teams)

This example is written for teams like IITS/Kubara-style platform environments:

- umbrella charts + subcharts + environment overlays,
- Flux/Argo reconciliation from Git/OCI,
- repeated incidents caused by unclear value precedence and ownership.

The first win is visibility, not policy complexity: who owns each value, which
source won, and what to edit next.

## What you get

- **Field-origin tracing**: every deployed field maps back to `values.yaml`,
  `values-prod.yaml`, a Helm CLI override, a Helm built-in/default, or a chart
  template — with owner and confidence score
- **Inverse-edit guidance**: "to change the image tag in production, edit
  `values-prod.yaml`, not the chart template"
- **Governance decisions**: ALLOW, ESCALATE, or BLOCK changes based on who
  changed what and which policy applies
- **Zero migration**: your `Chart.yaml`, templates, and values files stay
  exactly where they are

## How Helm maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              FLUX/ARGO (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ Chart.yaml          │          │ Deployment           │         │ Running pods     │
│ values.yaml         │──import─▶│ Service              │──sync──▶│ Live services    │
│ values-prod.yaml    │          │ ConfigMap            │         │ Cluster state    │
│ templates/*.yaml    │          │ HelmRelease (Flux)   │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  App team edits values.           Rendered manifests with           What's actually
  Platform team edits chart.       full field provenance.            running.
```

**DRY** is what humans edit: `values.yaml` (app team), `values-prod.yaml` (ops),
`Chart.yaml` + `templates/` (platform team). These are the source of truth for
intent.

**WET** is what generators produce: rendered Kubernetes manifests with every
field traced back to its DRY source. cub-gen classifies chart structure,
values-file precedence, and invocation-time Helm overrides so you can see which
layer actually won for a rendered field.

**LIVE** is what's running in your cluster. Flux or ArgoCD reconciles WET to LIVE.
cub-gen doesn't touch this layer — your existing reconciler stays in control.

| File | Owner | What it controls |
|------|-------|-----------------|
| `Chart.yaml` | Platform team | Chart contract — name, version, dependencies |
| `values.yaml` | App team | App defaults — image tags, feature flags, replicas |
| `values-prod.yaml` | Ops team | Production overrides — scaled replicas, resources |
| `templates/deployment.yaml` | Platform team | Deployment structure and resource layout |
| `platform/base/runtime-policy.yaml` | Platform team | Required probes and resource limits |
| `platform/base/network-policy.yaml` | Platform team | Egress rules and namespace boundaries |
| `platform/overlays/prod/resource-baseline.yaml` | Platform team | Production resource floors |
| `platform/registry.yaml` | Platform team | FrameworkRegistry typed operations + constraints for Helm platform APIs |
| `gitops/flux/helmrelease.yaml` | Platform team | Flux HelmRelease transport |
| `gitops/argo/application.yaml` | Platform team | ArgoCD Application transport |

## If you already run Helm heavily

This example is written for teams that already depend on Helm conventions:

- You keep app settings in `values*.yaml` and chart structure in `templates/`.
- You use env overlays and still get disputes about "who should edit what".
- You have drift incidents where people patch rendered manifests instead of DRY inputs.

cub-gen is additive: it does not replace Helm templating or your reconciler. It
adds ownership-aware tracing so Helm users can answer "which values key controls
this deployed field?" without manual chart archaeology.

## Why this maps cleanly to the cub-gen framework

| Existing Helm concept | cub-gen concept | Why it matters |
|------|------|------|
| `values.yaml` / `values-prod.yaml` | DRY app intent | Keep app-team edits in values files, not rendered manifests. |
| `templates/*.yaml` | DRY platform contract | Platform structure stays explicit and reviewable. |
| Rendered Kubernetes objects | WET targets with provenance | Every WET field is traced back to values or templates with confidence. |
| Flux/Argo applying Helm output | LIVE state | Existing runtime path stays unchanged; only governance visibility is added. |

## Advanced reality check: umbrella charts, overlays, and GitOps transports

If you run Helm at enterprise scale, the pain is usually in merge precedence,
not in writing templates:

- umbrella chart values overriding subchart defaults,
- environment overlays overriding umbrella defaults,
- Flux/Argo transport layers adding extra value sources (`valuesFrom`, inline overrides),
- OCI chart/version drift across many repos and clusters.

This example is intentionally a single-chart baseline so new users can see the
mapping quickly. In real environments, apply the same cub-gen flow at each
layer where DRY intent exists:

1. Chart + subchart value defaults (platform-owned contract layer).
2. Environment overlays (app/ops-owned intent layer).
3. Reconciler transport config (Flux `HelmRelease`, Argo `Application`).

The key outcome does not change: every WET field should have one clear edit
path and owner. That is what prevents "edit rendered manifests and hope" during
incidents.

## Fastest path to believe it

Start with the documented entrypoints:

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Local source-side path
./examples/helm-paas/demo-local.sh

# Local governed change path
./examples/helm-paas/demo-governed-change.sh

# Local layered Kubara-like trace path
./examples/helm-paas/demo-layered-trace.sh

# Connected ConfigHub path
cub auth login
./examples/helm-paas/demo-connected.sh

# Connected + live proof from the example itself
RECONCILER=argo ./examples/helm-paas/demo-runtime.sh
RECONCILER=both ./examples/helm-paas/demo-runtime.sh
```

If you want the raw commands underneath the wrappers:

```bash
# Detect the generator and classify all files
./cub-gen gitops discover --space platform --json ./examples/helm-paas

# Import with full provenance and field-origin tracing
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
```

You'll see cub-gen classify `Chart.yaml` + `values.yaml` as `helm-paas`, map
fields like `image.tag` to their source files, and emit ownership metadata for
every editable field.

## Real-world scenario: feature flag release with governance

**Who**: A payments team at a fintech company. They deploy via Helm + Flux.
The platform team owns the chart; the app team owns `values.yaml`.

### Day 1 — App team enables a new feature

The app team bumps the image and toggles a feature flag. These are app-owned
fields in `values.yaml`:

```yaml
# values.yaml (app team edits)
image:
  tag: v2.4.1       # was v2.4.0
featureFlags:
  newCheckout: true  # was false
```

### Day 2 — Governed pipeline (ALLOW path)

```bash
# cub-gen detects the change and traces field origins
./cub-gen gitops import --space platform --json ./examples/helm-paas

# Produce evidence bundle
./cub-gen publish --space platform ./examples/helm-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Decision engine: image tag + feature flag are app-team owned → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by app-lead --reason "v2.4.1 release with newCheckout flag"
```

The decision engine sees that both changed fields (`image.tag` and
`featureFlags.newCheckout`) are owned by the app team. No platform-owned
fields were touched → **ALLOW**.

### Day 3 — Blocked change (BLOCK path)

The same app team tries to edit the Deployment template directly — changing
resource limits in `templates/deployment.yaml`, which is platform-owned:

```yaml
# templates/deployment.yaml — platform-owned file
resources:
  limits:
    cpu: 2000m     # was controlled by values, now hardcoded
```

```bash
# cub-gen detects the template edit
./cub-gen gitops import --space platform --json ./examples/helm-paas

# Produce evidence
./cub-gen publish --space platform ./examples/helm-paas > bundle.json
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Decision engine: template change is platform-owned → BLOCK
./cub-gen bridge decision apply --decision decision.json --state BLOCK \
  --approved-by governance-bot \
  --reason "App team edited platform-owned template. Route to platform-team for review."
```

The field-origin trace shows `resources.limits.cpu` originates from
`templates/deployment.yaml` (platform-owned, confidence 0.86). The app team
doesn't own this file → **BLOCK**. The change must be routed to the platform
team for review.

### Day 4 — Production rollout

Ops applies `values-prod.yaml` with increased replicas. Same governed pipeline,
same evidence chain. The decision engine checks the platform's resource baseline
policy. Flux reconciles the HelmRelease to the cluster.

## How it works

cub-gen's `helm-paas` generator recognizes any directory containing `Chart.yaml`
alongside `values*.yaml` files. On import, it:

1. **Classifies inputs** — `Chart.yaml` (role: chart), `values.yaml` (role:
   values-base), `values-prod.yaml` (role: values-overlay)
2. **Maps field origins** — traces `image.tag` through the Helm template
   structure to the Deployment spec, keeping both the base values source
   (`values.yaml`, confidence: 0.86) and the winning prod overlay
   (`values-prod.yaml`, confidence: 0.90)
3. **Computes ownership** — values files are app-team editable; chart templates
   and platform policies are platform-owned
4. **Emits inverse-edit guidance** — "to change the prod `image.tag`, edit
   `values-prod.yaml`; use `values.yaml` for defaults; to change resource
   structure, edit `templates/deployment.yaml` (platform review required)"

A concrete field trace:

```
DRY:  values.yaml      → image.tag = "v1.0.0"
      values-prod.yaml → image.tag = "v1.0.3"
      ↓ helm-template + helm-values-overlay
WET:  Deployment/spec/template/spec/containers[0]/image = "ghcr.io/example/payments-api:v1.0.3"
```

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `Chart.yaml` | Platform | Chart contract — name, version, dependencies |
| `values.yaml` | App team | App defaults — image, flags, replicas |
| `values-prod.yaml` | Ops | Prod overrides — scaled replicas, resources |
| `templates/deployment.yaml` | Platform | K8s Deployment structure |
| `platform/base/runtime-policy.yaml` | Platform | Required probes, resource limits |
| `platform/base/network-policy.yaml` | Platform | Egress rules |
| `platform/overlays/prod/resource-baseline.yaml` | Platform | Prod resource floors |
| `platform/registry.yaml` | Platform | FrameworkRegistry for Helm operations/constraints |
| `gitops/flux/helmrelease.yaml` | Platform | Flux v2 HelmRelease transport |
| `gitops/argo/application.yaml` | Platform | ArgoCD Application transport |
| `docs/user-stories.md` | — | Narrative user stories |

## Next steps

- **Runtime proof companion**: [`live-reconcile`](../live-reconcile/) — prove
  the same governed path survives real Flux/Argo reconciliation under the hood
- **Cluster-side inspection companion**: [`cub-scout`](https://github.com/confighub/cub-scout)
  — inspect the reconciled runtime side after delivery
- **Spring Boot version**: [`springboot-paas`](../springboot-paas/) — same
  governance model for Java services with `application.yaml`
- **Score.dev version**: [`scoredev-paas`](../scoredev-paas/) — platform-agnostic
  workload specs with full field mapping
- **AI runtime deployment**: [`swamp-project`](../swamp-project/) — Helm chart
  deploying an AI model orchestration runtime
- **E2E demo script**: `../demo/module-1-helm-import.sh`
- **Bridge governance demo**: `../demo/module-4-bridge-governance.sh`

### PR-MR pairing and promotion flows

- **Flow A (Git PR → ConfigHub MR)**: `../demo/flow-a-git-pr-to-mr-connected.sh`
  — developer opens PR, ConfigHub creates MR with evidence
- **Flow B (ConfigHub MR → Git PR)**: `../demo/flow-b-mr-to-git-pr-connected.sh`
  — ConfigHub initiates change, generates Git PR after approval
- **FR8 promotion**: `../demo/fr8-promotion-upstream-dry-connected.sh`
  — promote successful app change to upstream platform DRY

## Run from ConfigHub (connected mode)

If you already have ConfigHub, start with the wrapper entrypoint:

```bash
cub auth login
./examples/helm-paas/demo-connected.sh
```

Start with `./examples/demo/run-connected-smoke.sh` first to confirm the
standard ConfigHub-connected environment. This wrapper is the deeper
bridge-backed walkthrough for this example.

If you want the connected path that also deploys and proves the live result:

```bash
cub auth login
RECONCILER=argo ./examples/helm-paas/demo-runtime.sh
RECONCILER=both ./examples/helm-paas/demo-runtime.sh
```

That example-owned wrapper runs the connected governed lifecycle first, then
shows live Deployment, Pod, Service, and reconciler status for Argo or Flux.

If you want the local governed edit proof before the connected path, run:

```bash
./examples/helm-paas/demo-governed-change.sh
```

That wrapper clones the repo into `.tmp`, proves an app-team `values.yaml`
change passes the ownership gate, then proves an app-team edit to
`templates/deployment.yaml` is rejected as a platform-owned runtime contract
change.

If you specifically need the deeper bridge API path, use:

```bash
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"
./cub-gen publish --space platform ./examples/helm-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | jq '.provenance[0].field_origin_map'

# Inverse-edit guidance
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | jq '.provenance[0].inverse_edit_pointers'

# Evidence bundle digest
./cub-gen publish --space platform ./examples/helm-paas \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

After the deep bridge API path, query ConfigHub for decision state.

## 8. Generation chain (Kubara-like platforms)

For umbrella charts, overlays, and ApplicationSets, trace the full chain:

```
Cluster labels          →  ApplicationSet selector  →  HelmRelease values  →  Deployed resources
  env: prod                 match: env=prod             replicas: 3            Deployment.spec.replicas: 3
  region: eu                                            image.tag: v2.4.1      container.image: ...v2.4.1
```

The key question: "Why does this cluster have this addon enabled?"

Answer: Trace from cluster labels through overlay selection to the deployed field.

Today that proof is runnable from this example:

```bash
./examples/helm-paas/demo-layered-trace.sh

# Or inspect the raw layered analysis directly
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | jq '.provenance[0].helm_layered_analysis'
```

If this is the section where you start thinking "where is my framework or
generator?", use [Custom Generator Onboarding](../../docs/workflows/custom-generator-onboarding.md)
for the user-facing extension path and Kubara-like layered requirements.

```bash
# Show rendered lineage
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | jq '.provenance[0].rendered_object_lineage'
```

## 9. Ownership boundary

Platform-owned fields cannot be weakened by downstream edits unless escalated:

| Layer | Owner | What's enforced |
|-------|-------|-----------------|
| `Chart.yaml` | Platform | Chart contract, dependencies |
| `templates/*.yaml` | Platform | Resource structure, security context |
| `platform/base/*.yaml` | Platform | Probes, limits, network policies |
| `values.yaml` | App team | Can edit within platform constraints |
| `values-prod.yaml` | Ops | Production overrides (escalation may apply) |

If an app team edits `templates/deployment.yaml` directly → **BLOCK**.
If an app team edits `values.yaml` → **ALLOW** (within constraints).

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/helm-paas/demo-local.sh
./examples/helm-paas/demo-layered-trace.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/helm-paas/demo-connected.sh

# Connected + live proof
cub auth login
RECONCILER=both ./examples/helm-paas/demo-runtime.sh
```
