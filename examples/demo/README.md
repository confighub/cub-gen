# Demo Scripts

Runnable demo scripts for every cub-gen example. Each script demonstrates the
full flow: detect → import → publish → verify → attest → bridge.

## Core platform/app track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `module-1-helm-import.sh` | [`helm-paas`](../helm-paas/) | Helm chart detection, values ownership, field-origin tracing |
| `module-2-score-field-map.sh` | [`scoredev-paas`](../scoredev-paas/) | Score workload spec with full field-origin mapping |
| `module-3-spring-ownership.sh` | [`springboot-paas`](../springboot-paas/) | Spring Boot ownership boundaries (app vs platform fields) |
| `module-4-bridge-governance.sh` | All examples | Bridge flow: ingest → decision → ALLOW/BLOCK |
| `module-5-ably-platform.sh` | [`just-apps-no-platform-config`](../just-apps-no-platform-config/) | Provider config governance without platform layer |
| `run-all-modules.sh` | All above | Run all core modules in sequence |

## AI work platform track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `ai-work-platform/scenario-1-c3agent.sh` | [`c3agent`](../c3agent/) + [`ai-ops-paas`](../ai-ops-paas/) | 11-target c3agent metadata coverage |
| `ai-work-platform/scenario-2-swamp.sh` | [`swamp-automation`](../swamp-automation/) | Swamp workflow governance with model bindings |
| `ai-work-platform/scenario-3-confighub-actions.sh` | [`confighub-actions`](../confighub-actions/) | Recursive governance — ConfigHub governing itself |
| `ai-work-platform/scenario-4-operations.sh` | [`ops-workflow`](../ops-workflow/) | Operations workflow with execution policy |
| `ai-work-platform/run-all.sh` | All above | Run all AI platform scenarios |

## ConfigHub lifecycle simulation

| Script | What it demonstrates |
|--------|---------------------|
| `simulate-confighub-lifecycle.sh <repo> <target> [slug]` | Full lifecycle: create → govern → update |
| `run-all-confighub-lifecycles.sh` | Run lifecycle sim for every example |
| `simulate-repo-wizard.sh <repo> <target> [hint]` | GUI-wizard simulation |

## Live reconciler e2e (Flux + kind)

| Script | What it demonstrates |
|--------|---------------------|
| `e2e-live-reconcile-flux.sh` | Real WET→LIVE reconciliation with Flux on a local kind cluster |

This script creates a local `kind` cluster, installs Flux, and proves:
1. Create reconciliation (v1 applied to LIVE)
2. Update reconciliation (v2 rolled out)
3. Drift correction (manual drift reverted by Flux)

Uses fixtures from [`live-reconcile`](../live-reconcile/).

## Quick start

```bash
# Build cub-gen first
go build -o ./cub-gen ./cmd/cub-gen

# Run all core demos
./examples/demo/run-all-modules.sh

# Run all AI platform demos
./examples/demo/ai-work-platform/run-all.sh

# Run a single module
./examples/demo/module-1-helm-import.sh
```

## Persona value bundles

`persona-value-bundles.sh` generates output organized by persona:

- **App team**: field-edit maps for Score and Spring Boot
- **GitOps team**: Helm bundles, attestations, verification
- **Platform engineer**: generator details, import summaries

Output goes to `output/persona-value/`.
