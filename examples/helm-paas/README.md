# Helm PaaS — Governed Helm for Platform Teams

Your Helm charts already define the contract between platform and app teams.
The platform owns the chart structure; app teams own their values overlays.
ConfigHub makes that contract explicit, traceable, and auditable — without
changing how you use Helm.

## What you get

- **Field-origin tracing**: every deployed field maps back to `values.yaml`,
  `values-prod.yaml`, or a chart template — with owner and confidence score
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

**WET** is what generators produce: rendered Kubernetes manifests with every field
traced back to its DRY source. cub-gen doesn't run `helm template` — it reads
your chart structure and classifies every field by origin and ownership.

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
| `gitops/flux/helmrelease.yaml` | Platform team | Flux HelmRelease transport |
| `gitops/argo/application.yaml` | Platform team | ArgoCD Application transport |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect the generator and classify all files
./cub-gen gitops discover --space platform --json ./examples/helm-paas

# Import with full provenance and field-origin tracing
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
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
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas

# Produce evidence bundle
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
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
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas

# Produce evidence
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > bundle.json
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
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
   structure to the Deployment spec (confidence: 0.86)
3. **Computes ownership** — values files are app-team editable; chart templates
   and platform policies are platform-owned
4. **Emits inverse-edit guidance** — "to change `image.tag`, edit `values.yaml`
   line 5; to change resource structure, edit `templates/deployment.yaml`
   (platform review required)"

A concrete field trace:

```
DRY:  values.yaml → image.tag = "v1.0.0"
      ↓ helm-template transform (confidence: 0.86)
WET:  Deployment/spec/template/spec/containers[0]/image = "ghcr.io/example/payments-api:v1.0.0"
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
| `gitops/flux/helmrelease.yaml` | Platform | Flux v2 HelmRelease transport |
| `gitops/argo/application.yaml` | Platform | ArgoCD Application transport |
| `docs/user-stories.md` | — | Narrative user stories |

## Next steps

- **Spring Boot version**: [`springboot-paas`](../springboot-paas/) — same
  governance model for Java services with `application.yaml`
- **Score.dev version**: [`scoredev-paas`](../scoredev-paas/) — platform-agnostic
  workload specs with full field mapping
- **AI runtime deployment**: [`swamp-project`](../swamp-project/) — Helm chart
  deploying an AI model orchestration runtime
- **E2E demo script**: `../demo/module-1-helm-import.sh`
- **Bridge governance demo**: `../demo/module-4-bridge-governance.sh`

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline"
./examples/helm-paas/demo-local.sh

echo "connected (requires ConfigHub auth)"
cub auth login
./examples/helm-paas/demo-connected.sh
```
