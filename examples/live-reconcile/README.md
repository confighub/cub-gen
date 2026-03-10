# Live Reconciliation Fixtures

This example proves LIVE reconciliation loops with both Flux and Argo CD using a real `kind` cluster.

## What You Get

- `flux/manifests-v1`: initial desired state (`replicas=1`, revision `v1`)
- `flux/manifests-v2`: updated desired state (`replicas=2`, revision `v2`)
- `examples/demo/e2e-live-reconcile-flux.sh`: Flux create -> update -> drift-correction
- `examples/demo/e2e-live-reconcile-argo.sh`: Argo create -> update -> drift-correction

Both scripts verify:

1. Create reconciliation (desired v1 reaches LIVE)
2. Update reconciliation (desired v2 reaches LIVE)
3. Drift correction (manual LIVE change gets corrected back to desired)

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
```

## Notes

- Argo mode installs Argo CD into the cluster if missing.
- Flux mode requires the `flux` CLI on your machine.
- Argo mode requires network access to install Argo manifests from the official upstream URL.
