# Demo entrypoints

## Core platform/app track

- `module-1-helm-import.sh`
- `module-2-score-field-map.sh`
- `module-3-spring-ownership.sh`
- `module-4-bridge-governance.sh`
- `module-5-ably-platform.sh`
- `run-all-modules.sh`

## GUI-wizard simulation

- `simulate-repo-wizard.sh <repo-path> <render-target-path> [profile-hint]`

## ConfigHub backend-connected GitOps (real server path)

- `confighub-connected-gitops.sh` (uses `cub gitops discover/import/cleanup` against a real ConfigHub backend)

Run with:

```bash
SPACE=<space-slug> \
DISCOVERY_TARGET=<discovery-target-slug> \
RENDER_TARGET=<renderer-target-slug> \
./examples/demo/confighub-connected-gitops.sh
```

Guide:
- `docs/workflows/confighub-backend-connected-loop.md`

## ConfigHub lifecycle simulation

- `simulate-confighub-lifecycle.sh <repo-path> <render-target-path> [example-slug]`
- `run-all-confighub-lifecycles.sh`

This flow runs for each example:

1. Create/import path (`discover -> import -> publish -> verify -> attest`)
2. Decision + promote path (`bridge decision` + `bridge promote`)
3. Update source config and re-run governance chain
4. Surface summaries for:
   - OCI bundle output URIs
   - Flux fixture files (if present)
   - Argo fixture files (if present)
   - cub-scout watchlist (from wet targets)

Qualification caveat:
Without a live `WET -> LIVE` reconciler loop shown end-to-end, this is `governed config automation`, not full `Agentic GitOps` (see `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md` and `docs/agentic-gitops/02-design/10-generators-prd.md`).

PRD user-story coverage snapshot:

| Status | User stories |
|---|---|
| Met/strong in current demos | 2, 3, 4, 5, 6, 13 |
| Partial (simulated/local-first, not full backend/runtime integration) | 1, 7, 9, 12 |
| Deferred | 8, 10, 11 |

Note:
- This table is for `cub-gen` local-first demo coverage.
- Backend-connected GitOps import exists via `cub` CLI and is documented in `docs/workflows/confighub-backend-connected-loop.md`.

## Live reconciler e2e (Flux + kind)

- `e2e-live-reconcile-flux.sh`

This optional script creates a local `kind` cluster, installs Flux, and proves
real `WET -> LIVE` reconciliation:

1. apply v1 desired state and wait for deployment availability,
2. update to v2 desired state and wait for rollout,
3. inject live drift (replicas), then verify Flux corrects it.

## AI work platform track

- `ai-work-platform/scenario-1-c3agent.sh` (11-target c3agent metadata coverage)
- `ai-work-platform/scenario-2-swamp.sh`
- `ai-work-platform/scenario-3-confighub-actions.sh`
- `ai-work-platform/scenario-4-operations.sh`
- `ai-work-platform/run-all.sh`
