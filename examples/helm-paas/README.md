# Helm PaaS (Platform + App)

**Pattern: platform team owns the Helm chart contract and runtime guardrails; app team owns values overlays.**

## 1. What is this?

A payments API team maintains a Kubernetes service deployed via Helm. The platform team owns the chart structure (templates, resource policies, network policies), and the app team controls what they can safely change: image tags, feature flags, replica counts. Production overrides live in a separate values file with explicit platform guardrails.

This is the most common cub-gen pattern: a Helm chart where the DRY/WET boundary aligns with the platform/app ownership boundary.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **App team** | `values.yaml` — image tags, feature flags, app-level config | `image.tag`, `featureFlags.*`, `replicaCount` |
| **Ops team** | `values-prod.yaml` — production overrides | Production replicas, resource requests/limits |
| **Platform team** | `Chart.yaml`, `templates/*`, `platform/*` | Chart version, templates, runtime policies, network policies |
| **GitOps reconciler** | Flux HelmRelease / ArgoCD Application | Reconciles rendered WET manifests to cluster LIVE state |

## 3. What does cub-gen add?

- **Generator detection**: recognizes `Chart.yaml` + `values.yaml` as `helm-paas` profile (capabilities: `render-manifests`, `values-overrides`, `inverse-values-patch`)
- **DRY/WET mapping**: values files (DRY) → HelmRelease + Deployments + Services (WET)
- **Field-origin tracing**: `image.tag` traces to `values.yaml` (base) or `values-prod.yaml` (prod override)
- **Inverse-edit guidance**: "to change the image tag, edit values file and keep chart template unchanged"

```bash
./cub-gen gitops discover --space platform --json ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
```

## 4. How do I run it?

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > /tmp/helm-bundle.json
./cub-gen verify --in /tmp/helm-bundle.json
./cub-gen attest --in /tmp/helm-bundle.json --verifier ci-bot > /tmp/helm-attestation.json
./cub-gen verify-attestation --in /tmp/helm-attestation.json --bundle /tmp/helm-bundle.json
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

## 5. Real-world example using ConfigHub

A payments team at a fintech company deploys their API via Helm. They need to release a new feature behind a feature flag.

**Day 1: App team makes the change**

```yaml
# values.yaml
image:
  tag: v2.4.1       # bumped from v2.4.0
featureFlags:
  newCheckout: true  # toggled on
```

**Day 2: Governed pipeline**

```bash
# cub-gen detects the change, produces provenance
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas
# Field-origin: image.tag and featureFlags.newCheckout changed in values.yaml (app-team owned)

# Produce evidence chain
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# ConfigHub ingests
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example

# Decision engine: image tag change + feature flag toggle → ALLOW (app-team scope)
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by app-lead --reason "v2.4.1 release with newCheckout flag"
```

**Day 3: Production rollout**

Ops applies `values-prod.yaml` with increased replicas for the release:

```yaml
replicaCount: 5   # scaled up for release traffic
```

Same governed pipeline, but now the decision engine also checks the platform's resource baseline policy. Bridge worker syncs to Flux, which reconciles the HelmRelease.

**Day 4: Promotion**

After successful rollout, the promotion flow opens a platform DRY PR to update the base `replicaCount` default for future releases.

## Narrative turns

1. **App feature change** — App team bumps `image.tag` and toggles a flag in `values.yaml`. Import shows this as app-editable DRY with inverse pointers.
2. **Environment rollout** — Ops applies override in `values-prod.yaml`. Import keeps the env override lineage explicit.
3. **Governed decision** — Bundle + attestation move through decision state. Platform approver issues explicit `ALLOW | ESCALATE | BLOCK`.
4. **Promotion upstream** — After successful rollout, promotion path opens a platform DRY PR for reusable defaults.

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `Chart.yaml` | Platform team | Chart contract — name, version, dependencies |
| `values.yaml` | App team | App defaults — image tags, feature flags, replicas |
| `values-prod.yaml` | Ops team | Production overrides — prod image, scaled replicas, resources |
| `platform/base/runtime-policy.yaml` | Platform team | Runtime policy — required probes, resource limits |
| `platform/base/network-policy.yaml` | Platform team | Network policy — egress rules |
| `platform/overlays/prod/resource-baseline.yaml` | Platform team | Prod resource baselines |
| `gitops/flux/helmrelease.yaml` | Platform team | Flux v2 HelmRelease transport |
| `gitops/argo/application.yaml` | Platform team | ArgoCD Application transport |
| `docs/user-stories.md` | — | Narrative user stories |
