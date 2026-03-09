# Live Reconciliation Fixtures

These fixtures are used by the Flux live e2e demo:

- `flux/manifests-v1`: initial desired state
- `flux/manifests-v2`: updated desired state

The demo script:

- `examples/demo/e2e-live-reconcile-flux.sh`

runs both versions against a real `kind` cluster with Flux controllers and
proves:

1. create reconciliation (`v1` applied to LIVE),
2. update reconciliation (`v2` rolled out to LIVE),
3. drift correction (manual LIVE drift reconciled back to desired state).

## Scope caveats

1. This is Flux-only today; there is no Argo live e2e entrypoint in this repo yet.
2. These are static fixtures, not manifests rendered per example by `cub-gen` during the e2e run.
