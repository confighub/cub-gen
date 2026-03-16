# Live Reconciliation — WET→LIVE Proof Harness

This example proves LIVE reconciliation loops with both Flux and Argo CD using a real `kind` cluster. It's the final piece of the DRY→WET→LIVE chain: cub-gen handles DRY→WET with governance, and this harness proves WET→LIVE reconciliation actually works.

Without a real WET→LIVE reconciler loop shown end-to-end, the flow is "governed config automation" not full "Agentic GitOps." This harness proves the full loop.

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** proving reconciler integration | Jump to [Run from ConfigHub](#run-from-confighub-connected-mode) |
| **Flux/Argo operator** validating reconciler behavior | Jump to [Try it](#try-it) — local Flux/Argo proof |

Both paths lead to the same outcome: proven WET→LIVE reconciliation with create/update/drift-correction.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real cluster** | `kind` cluster with Flux or Argo CD installed |
| **Real app** | Deployment + Service for a test workload |
| **Real reconciliation** | Create → Update → Drift correction loop |
| **Real inspection target** | `kubectl get deployment -o yaml` showing actual LIVE state |
| **GitOps transport** | Flux Kustomization or ArgoCD Application |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "Does our reconciler actually work?" | Create/update/drift proof | Repeatable validation |
| "Flux vs Argo — same behavior?" | Side-by-side comparison | Controller confidence |
| "Does governance break reconciliation?" | Connected + LIVE proof | Full loop validation |

## Domain POV (reconciler reliability owners)

Use this harness if your team already trusts GitOps reconcilers and wants
repeatable proof of behavior:

- create/update/drift-correction in one script,
- side-by-side Flux vs Argo verification,
- connected governed path from `helm-paas` artifacts to LIVE correction.

The first value is confidence that governance tooling remains additive to real
reconciler behavior.

## What You Get

- **Flux fixtures**: `flux/manifests-v1` (replicas=1) and `flux/manifests-v2` (replicas=2)
- **Helm-derived fixtures**: `helm-paas/manifests-v1` and `helm-paas/manifests-v2`
- **E2E proof scripts**:
  - `e2e-live-reconcile-flux.sh`: Flux create → update → drift-correction
  - `e2e-live-reconcile-argo.sh`: Argo create → update → drift-correction
  - `e2e-connected-governed-reconcile-helm.sh`: full ConfigHub → Flux/Argo loop

Both scripts verify:

1. **Create reconciliation** (desired v1 reaches LIVE)
2. **Update reconciliation** (desired v2 reaches LIVE)
3. **Drift correction** (manual LIVE change gets corrected back to desired)

## How Live Reconcile maps to DRY / WET / LIVE

```
  cub-gen TRACES (WET)              RECONCILER (LIVE)             CLUSTER STATE
┌──────────────────────┐         ┌─────────────────────┐        ┌─────────────────┐
│ Deployment manifest  │         │ Flux Kustomization  │        │ Running pods     │
│ Service manifest     │──apply─▶│ or ArgoCD App       │──sync─▶│ Live services    │
│ ConfigMap            │         │                     │        │ Cluster state    │
│                      │         │ (drift correction)  │        │                 │
└──────────────────────┘         └─────────────────────┘        └─────────────────┘
  Governed manifests with          Real reconciler loop            What's actually
  field provenance.                proving WET→LIVE.               running.
```

This example focuses on the **WET→LIVE** part of the chain. cub-gen handles
DRY→WET with governance (see other examples). This harness proves the
reconciler actually works.

| Fixture | Version | What it demonstrates |
|---------|---------|---------------------|
| `flux/manifests-v1` | v1 | Initial state (replicas=1) |
| `flux/manifests-v2` | v2 | Updated state (replicas=2) |
| `helm-paas/manifests-v1` | v1 | Helm-derived v1.0.3 (replicas=4) |
| `helm-paas/manifests-v2` | v2 | Helm-derived v1.0.9-canary (replicas=3) |

## If you already operate Flux/Argo at scale

This fixture is for teams that already trust GitOps reconciliation and want a
fast proof harness:

- Validate create/update/drift behavior on demand.
- Compare controller behavior (Flux vs Argo) with the same manifests.
- Reproduce reconciliation incidents quickly in an isolated `kind` cluster.

## Why this maps to the cub-gen framework

| Existing reconciler concern | cub-gen DRY/WET/LIVE framing | Why it matters |
|------|------|------|
| Desired manifests in Git | WET input to reconciler | Matches where `cub-gen` hands off after governance. |
| Controller reconciliation | WET → LIVE loop | Proves the active loop required for Agentic GitOps claims. |
| Live drift correction | LIVE feedback loop | Shows why provenance + inverse-edit guidance are useful upstream. |

## Try it

```bash
# Build cub-gen (if not already built)
go build -o ./cub-gen ./cmd/cub-gen

# Run Flux live reconcile proof (requires kind + flux CLI)
./examples/demo/e2e-live-reconcile-flux.sh

# Run Argo live reconcile proof (requires kind + network access)
./examples/demo/e2e-live-reconcile-argo.sh

# Run both side-by-side
./examples/demo/e2e-live-reconcile-flux.sh && ./examples/demo/e2e-live-reconcile-argo.sh
```

## Real-world scenario: proving drift correction

**Who**: A platform team validating their GitOps reconciliation before go-live.

### Scenario A — Normal reconciliation (v1 → v2 update)

```bash
# Create cluster and install Flux
kind create cluster --name live-test
flux install

# Apply v1 manifests (replicas=1)
kubectl apply -f ./examples/live-reconcile/flux/manifests-v1/

# Verify v1 is LIVE
kubectl get deployment -o jsonpath='{.items[0].spec.replicas}'
# Output: 1

# Apply v2 manifests (replicas=2)
kubectl apply -f ./examples/live-reconcile/flux/manifests-v2/

# Verify v2 is LIVE
kubectl get deployment -o jsonpath='{.items[0].spec.replicas}'
# Output: 2
```

Result: Reconciler correctly applies v1 → v2 update → **PASS**

### Scenario B — Drift correction (manual change reverted)

```bash
# Manually introduce drift
kubectl scale deployment test-app --replicas=5

# Wait for reconciler to detect and correct drift
sleep 30

# Verify drift is corrected back to desired (v2 = 2 replicas)
kubectl get deployment -o jsonpath='{.items[0].spec.replicas}'
# Output: 2
```

Result: Reconciler corrects manual drift → **PASS**

## Key files

| File | Purpose |
|------|---------|
| `flux/manifests-v1/` | Flux v1 fixtures (replicas=1) |
| `flux/manifests-v2/` | Flux v2 fixtures (replicas=2) |
| `helm-paas/manifests-v1/` | Helm-derived v1 fixtures |
| `helm-paas/manifests-v2/` | Helm-derived v2 fixtures |
| `demo-local.sh` | Local harness entry point |
| `demo-connected.sh` | Connected harness with ConfigHub |

## Next steps

- **Helm + governance**: [`helm-paas`](../helm-paas/) — DRY→WET governance before reconciliation
- **Full connected loop**: `e2e-connected-governed-reconcile-helm.sh` — ConfigHub → Flux/Argo

## Run from ConfigHub (connected mode)

If you already have ConfigHub, run the full governed reconciliation loop:

```bash
cub auth login

# Connected governed full loop (helm-paas → ConfigHub → Flux/Argo LIVE)
RECONCILER=both ./examples/demo/e2e-connected-governed-reconcile-helm.sh
```

This proves:
1. DRY→WET governance (cub-gen publish/verify/attest)
2. ConfigHub ingest/decision
3. WET→LIVE reconciliation (Flux or Argo)
4. Drift correction after manual interference

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline (default: flux)
./examples/live-reconcile/demo-local.sh

# Local/offline with argo
RECONCILER=argo ./examples/live-reconcile/demo-local.sh

# Connected (requires ConfigHub auth, default: flux)
cub auth login
./examples/live-reconcile/demo-connected.sh

# Connected with both reconciler proofs
cub auth login
RECONCILER=both ./examples/live-reconcile/demo-connected.sh

# Connected governed full loop (helm-paas → ConfigHub → Flux/Argo LIVE)
cub auth login
RECONCILER=both ./examples/demo/e2e-connected-governed-reconcile-helm.sh
```

## Notes

- Argo mode installs Argo CD into the cluster if missing.
- Flux mode requires the `flux` CLI on your machine.
- Argo mode requires network access to install Argo manifests from the official upstream URL.
- The `kind` cluster is created fresh for each test run.
