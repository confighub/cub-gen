# Live Reconciliation Fixtures

This example proves LIVE reconciliation loops with both Flux and Argo CD using a real `kind` cluster.

## What You Get

- `flux/manifests-v1`: initial desired state (`replicas=1`, revision `v1`)
- `flux/manifests-v2`: updated desired state (`replicas=2`, revision `v2`)
- `helm-paas/manifests-v1`: Helm-derived desired state v1 (`payments-api`, `ghcr.io/example/payments-api:v1.0.3`, `replicas=4`)
- `helm-paas/manifests-v2`: Helm-derived desired state v2 (`payments-api`, `ghcr.io/example/payments-api:v1.0.9-canary`, `replicas=3`)
- `examples/demo/e2e-live-reconcile-flux.sh`: Flux create -> update -> drift-correction
- `examples/demo/e2e-live-reconcile-argo.sh`: Argo create -> update -> drift-correction
- `examples/demo/e2e-connected-governed-reconcile-helm.sh`: connected ConfigHub governance + Flux/Argo live reconcile using `helm-paas`-derived manifests

Both scripts verify:

1. Create reconciliation (desired v1 reaches LIVE)
2. Update reconciliation (desired v2 reaches LIVE)
3. Drift correction (manual LIVE change gets corrected back to desired)

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
| Controller reconciliation | WET -> LIVE loop | Proves the active loop required for Agentic GitOps claims. |
| Live drift correction | LIVE feedback loop | Shows why provenance + inverse-edit guidance are useful upstream. |

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline (default: flux)"
./examples/live-reconcile/demo-local.sh

echo "local/offline with argo"
RECONCILER=argo ./examples/live-reconcile/demo-local.sh

echo "connected (requires ConfigHub auth, default: flux)"
cub auth login
./examples/live-reconcile/demo-connected.sh

echo "connected with both reconciler proofs"
cub auth login
RECONCILER=both ./examples/live-reconcile/demo-connected.sh

echo "connected governed full loop (helm-paas -> ConfigHub -> Flux/Argo LIVE)"
cub auth login
RECONCILER=both ./examples/demo/e2e-connected-governed-reconcile-helm.sh
```

## Notes

- Argo mode installs Argo CD into the cluster if missing.
- Flux mode requires the `flux` CLI on your machine.
- Argo mode requires network access to install Argo manifests from the official upstream URL.
