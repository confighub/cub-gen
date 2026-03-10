# Demo Scripts

Runnable demo scripts for every `cub-gen` example.

Each script demonstrates part of the flow:

```
detect -> import -> publish -> verify -> attest -> (optional) bridge ingest/query
```

## Local mode (no ConfigHub login required)

## Core platform/app track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `module-1-helm-import.sh` | [`helm-paas`](../helm-paas/) | Helm detection, values ownership, field-origin tracing |
| `module-2-score-field-map.sh` | [`scoredev-paas`](../scoredev-paas/) | Score field-origin and inverse edit mapping |
| `module-3-spring-ownership.sh` | [`springboot-paas`](../springboot-paas/) | Spring ownership boundaries (app vs platform) |
| `module-4-bridge-governance.sh` | Multiple | Local bridge contract simulation |
| `module-5-ably-platform.sh` | [`just-apps-no-platform-config`](../just-apps-no-platform-config/) | Provider config governance without platform layer |
| `run-all-modules.sh` | All above | Run all core modules |

## AI work platform track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `ai-work-platform/scenario-1-c3agent.sh` | [`c3agent`](../c3agent/) + [`ai-ops-paas`](../ai-ops-paas/) | c3agent 11-target coverage |
| `ai-work-platform/scenario-2-swamp.sh` | [`swamp-automation`](../swamp-automation/) | Swamp workflow/model governance |
| `ai-work-platform/scenario-3-confighub-actions.sh` | [`confighub-actions`](../confighub-actions/) | Recursive governance |
| `ai-work-platform/scenario-4-operations.sh` | [`ops-workflow`](../ops-workflow/) | Operations workflow governance |
| `ai-work-platform/run-all.sh` | All above | Run all AI platform scenarios |

## Lifecycle simulation scripts

| Script | What it demonstrates |
|--------|---------------------|
| `simulate-confighub-lifecycle.sh <repo> <target> [slug]` | Full local lifecycle simulation |
| `run-all-confighub-lifecycles.sh` | Lifecycle simulation across current fixtures |
| `simulate-confighub-lifecycle-connected.sh <repo> <target> [slug]` | Connected lifecycle: real ConfigHub ingest + decision query |
| `run-all-connected-lifecycles.sh` | Connected lifecycle run across current fixtures with pass/fail summary |
| `simulate-repo-wizard.sh <repo> <target> [hint]` | GUI wizard simulation path |

## Phase 3 connected story scripts

| Script | User story | What it demonstrates |
|--------|------------|---------------------|
| `story-1-existing-repo-connected.sh` | 1 | Existing repo import + connected change query by `change_id` |
| `story-7-ci-api-flow-connected.sh` | 7 | Non-interactive CI flow using `CONFIGHUB_TOKEN` |
| `story-9-multi-repo-wave-connected.sh` | 9 | Multi-repo wave with per-target ALLOW/ESCALATE/BLOCK outcomes |
| `story-12-unified-actor-evidence.sh` | 12 | Unified human/CI/AI attestation chain under one `change_id` |
| `run-phase-3-connected-stories.sh` | 1,7,9,12 | Runs all four connected Phase 3 stories |

## Connected mode (ConfigHub)

Start with authentication:

```bash
cub auth login
TOKEN="$(cub auth get-token)"
cub context get --json | jq -r '.coordinate.user'
```

Then run bridge calls with `--token "$TOKEN"` and your `--base-url`.

Shared connected preflight helper:

- `examples/demo/lib/connected-preflight.sh`

Connected flow shape:

```
publish -> verify -> attest -> bridge ingest -> decision query
```

Connected ingest/query are real ConfigHub API calls. Some decision/promotion steps in current demo scripts are still local contract simulation.

If ingest returns `404 Not Found`, point `CONFIGHUB_BASE_URL` to a backend with the governed-wet bridge endpoints enabled.

Connected runner:

```bash
./examples/demo/run-all-connected-lifecycles.sh
./examples/demo/run-phase-3-connected-stories.sh
```

## Live reconciler e2e (Flux + Argo + kind)

| Script | What it demonstrates |
|--------|---------------------|
| `e2e-live-reconcile-flux.sh` | Real WET->LIVE reconciliation with Flux on local kind cluster |
| `e2e-live-reconcile-argo.sh` | Real WET->LIVE reconciliation with Argo CD on local kind cluster |

These scripts prove:

1. Create reconciliation (v1 to LIVE).
2. Update reconciliation (v2 rollout).
3. Drift correction (manual drift reverted).

Uses fixtures from [`live-reconcile`](../live-reconcile/).

## Quick start

```bash
go build -o ./cub-gen ./cmd/cub-gen
./examples/demo/run-all-modules.sh
./examples/demo/ai-work-platform/run-all.sh
```

## Qualification caveat

Without a live `WET -> LIVE` reconciler loop shown end-to-end, classify the flow as `governed config automation`, not full `Agentic GitOps`.

References:

- `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md`
- `docs/agentic-gitops/02-design/10-generators-prd.md`

## PRD user-story coverage snapshot

| Status | User stories |
|---|---|
| Met/strong in current demos | 2, 3, 4, 5, 6, 13 |
| Partial (simulated/local-first, not full backend/runtime integration) | 1, 7, 9, 12 |
| Deferred | 8, 10, 11 |
