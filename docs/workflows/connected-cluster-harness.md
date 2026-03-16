# Connected Cluster Harness

This document describes the shared real-cluster harness for connected tests.

## Purpose

The connected cluster harness provides a unified way to run `cub-gen` connected
tests against real Kubernetes clusters with real GitOps reconciliation.

It bridges two flows:

1. **Connected lifecycle**: `publish → verify → attest → bridge ingest → decision`
2. **Live reconciliation**: WET manifests → GitOps controller → LIVE cluster state

## When to use

| Scenario | Harness needed? |
|----------|-----------------|
| Local mode testing (no ConfigHub) | No |
| Connected mode testing (ConfigHub API only) | No |
| Live reconciliation proof (WET→LIVE) | Yes |
| Full end-to-end proof (ConfigHub → GitOps → LIVE) | Yes |

## Prerequisites

```bash
# Required tools
docker --version       # Docker daemon must be running
kind --version         # Kubernetes in Docker
kubectl version        # Kubernetes CLI
flux --version         # Flux CLI (for Flux tests)
# argocd version       # Argo CLI (optional, for Argo tests)
```

## Quick start

```bash
# Source the harness library
source examples/demo/lib/cluster-harness.sh

# Set up a cluster with Flux
setup_connected_cluster flux

# Run your connected tests
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json

# Clean up (optional)
cleanup_connected_cluster force
```

## API reference

### setup_connected_cluster

```bash
setup_connected_cluster [reconciler]
```

Creates or reuses a kind cluster and installs the specified reconciler.

| Argument | Options | Default |
|----------|---------|---------|
| reconciler | `flux`, `argo`, `both` | `flux` |

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `CLUSTER_NAME` | Kind cluster name | `cub-gen-connected` |
| `CLEANUP_CLUSTER` | Delete cluster on exit if `1` | `0` |
| `RECONCILE_TIMEOUT` | Timeout in seconds | `300` |
| `TARGET_NS` | Namespace for test deployments | `demo-live` |

### cleanup_connected_cluster

```bash
cleanup_connected_cluster [force]
```

Deletes the kind cluster if `CLEANUP_CLUSTER=1` or if `force` is passed.

### wait_for_deployment

```bash
wait_for_deployment <deployment-name> [namespace]
```

Waits for a deployment to become available.

### print_cluster_state

```bash
print_cluster_state
```

Prints cluster state for debugging (nodes, namespaces, deployments, GitOps resources).

## CI integration

In CI, set `CLEANUP_CLUSTER=1` to ensure clusters are cleaned up:

```yaml
- name: Run connected cluster tests
  env:
    CLEANUP_CLUSTER: "1"
    CONFIGHUB_TOKEN: ${{ secrets.CONFIGHUB_TOKEN }}
    CONFIGHUB_BASE_URL: ${{ secrets.CONFIGHUB_BASE_URL }}
  run: |
    source examples/demo/lib/cluster-harness.sh
    setup_connected_cluster flux
    ./examples/demo/run-connected-cluster-proof.sh
```

## End-to-end proof flow

The full end-to-end proof demonstrates:

```
DRY source
    ↓ (cub-gen discover/import)
WET manifests + provenance
    ↓ (cub-gen publish)
Change bundle with digest
    ↓ (cub-gen verify + attest)
Evidence chain
    ↓ (bridge ingest)
ConfigHub changeset + decision
    ↓ (GitOps controller)
LIVE cluster state
    ↓ (verify)
Mutation ledger proof
```

This proves the full governance loop:

1. Human or AI edits DRY source
2. `cub-gen` produces governed artifacts
3. ConfigHub evaluates policy
4. GitOps reconciles to cluster
5. Mutation ledger records evidence

## Troubleshooting

### Cluster creation fails

```bash
# Check Docker is running
docker info

# Check kind can create clusters
kind create cluster --name test-cluster
kind delete cluster --name test-cluster
```

### Flux installation times out

```bash
# Check flux prerequisites
flux check --pre

# Increase timeout
export RECONCILE_TIMEOUT=600
setup_connected_cluster flux
```

### Reconciliation never completes

```bash
# Check GitOps controller status
print_cluster_state

# Check controller logs
kubectl --context kind-cub-gen-connected logs -n flux-system deployment/source-controller
kubectl --context kind-cub-gen-connected logs -n flux-system deployment/kustomize-controller
```

## Related documentation

- [Connected preflight](../examples/demo/lib/connected-preflight.sh) — ConfigHub auth validation
- [Live reconcile Flux](../examples/demo/e2e-live-reconcile-flux.sh) — Flux e2e proof
- [Live reconcile Argo](../examples/demo/e2e-live-reconcile-argo.sh) — Argo e2e proof
