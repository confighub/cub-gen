# Demo Scripts

Runnable demo scripts for every `cub-gen` example.

Each script demonstrates part of the flow:

```
detect -> import -> publish -> verify -> attest -> (optional) bridge ingest/query
```

## Workflow-first start (Ops + Swamp)

If your platform is workflow-heavy, start here before app-manifest demos:

```bash
# Swamp workflow graph governance (models/methods/required steps)
./examples/demo/ai-work-platform/scenario-2-swamp.sh
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '.provenance[0].swamp_workflow_analysis'

# Ops workflow governance (actions/schedules/approval gates)
./examples/demo/ai-work-platform/scenario-4-operations.sh
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '.provenance[0].ops_workflow_analysis'
```

## Local mode (no ConfigHub login required)

## Core platform/app track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `module-1-helm-import.sh` | [`helm-paas`](../helm-paas/) | Helm detection, values ownership, field-origin tracing |
| `module-2-score-field-map.sh` | [`scoredev-paas`](../scoredev-paas/) | Score field-origin and inverse edit mapping |
| `module-3-spring-ownership.sh` | [`springboot-paas`](../springboot-paas/) | Spring ownership boundaries (app vs platform) |
| `module-4-bridge-governance.sh` | Multiple | Local bridge contract simulation |
| `module-5-no-config-platform.sh` | [`just-apps-no-platform-config`](../just-apps-no-platform-config/) | No-config-platform provider governance without a platform layer |
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
| `simulate-confighub-lifecycle-connected.sh <repo> <target> [slug]` | Connected lifecycle: real ConfigHub ingest/query when bridge endpoints are available, with backend `changeset` fallback when they are not |
| `run-all-connected-lifecycles.sh` | Connected lifecycle run across current fixtures with pass/fail summary |
| `run-all-connected-entrypoints.sh` | Runs every `examples/*/demo-connected.sh` entrypoint (all examples, optional `live-reconcile`) |
| `simulate-repo-wizard.sh <repo> <target> [hint]` | GUI wizard simulation path |

## Phase 3 connected story scripts

| Script | User story | What it demonstrates |
|--------|------------|---------------------|
| `story-1-existing-repo-connected.sh` | 1 | Existing repo import + connected change query by `change_id` |
| `story-7-ci-api-flow-connected.sh` | 7 | Non-interactive CI flow using `CONFIGHUB_TOKEN` |
| `story-9-multi-repo-wave-connected.sh` | 9 | Multi-repo wave with per-target ALLOW/ESCALATE/BLOCK outcomes |
| `story-12-unified-actor-evidence.sh` | 12 | Unified human/CI/AI attestation chain under one `change_id` |
| `run-phase-3-connected-stories.sh` | 1,7,9,12 | Runs all four connected Phase 3 stories |

## Phase 4 connected story scripts

| Script | User story | What it demonstrates |
|--------|------------|---------------------|
| `story-8-label-evolution-connected.sh` | 8 | Backend-persisted label/taxonomy migration anchor with compatibility queries (no repo surgery) |
| `story-10-signed-writeback-proof-connected.sh` | 10 | Real GitHub PR/commit/branch-protection evidence for signed write-back proof |
| `story-11-live-breakglass-proposal-connected.sh` | 11 | Persist accept/revert break-glass proposals as backend changesets with queryable evidence |
| `run-phase-4-connected-stories.sh` | 8,10,11 | Runs all three connected Phase 4 stories |

Story 10 required inputs (real GitHub evidence):

```bash
export APP_PR_REPO=owner/app-repo
export APP_PR_NUMBER=123
export PROMOTION_PR_REPO=owner/promotion-repo
export PROMOTION_PR_NUMBER=456
# optional if not already authenticated with gh:
export GH_TOKEN=...
```

If you run `run-phase-4-connected-stories.sh` without these inputs, Story 10 is skipped by default.
Set `REQUIRE_STORY_10=1` to fail fast instead of skipping.

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

Connected ingest/query and decision-state authority are real ConfigHub API calls in connected lifecycle scripts when governed-wet bridge endpoints are available.
Local contract simulation is limited to explicit local-only demos (`simulate-confighub-lifecycle.sh`, `module-4-bridge-governance.sh`).

Bridge endpoint behavior:

- Default (`CONNECTED_FALLBACK_MODE=auto`): if ingest returns `404 Not Found`, the scripts switch to backend `changeset` fallback and still persist evidence in ConfigHub.
- Strict (`CONNECTED_FALLBACK_MODE=off`): fail fast unless bridge ingest/query endpoints are reachable.
- Forced fallback (`CONNECTED_FALLBACK_MODE=changeset`): always use backend `changeset` fallback.

Fallback mode records `ingest.json`/`decision-final.json` in the standard contract shape with `source=confighub-backend-changeset-fallback`, so story scripts and CI gates remain deterministic.

If your backend exposes non-default paths, set `BRIDGE_INGEST_ENDPOINT` and `BRIDGE_DECISION_ENDPOINT`.

Connected runner:

```bash
./examples/demo/run-all-connected-lifecycles.sh
./examples/demo/run-all-connected-entrypoints.sh
./examples/demo/run-phase-3-connected-stories.sh
./examples/demo/run-phase-4-connected-stories.sh
```

## Live reconciler e2e (Flux + Argo + kind)

| Script | What it demonstrates |
|--------|---------------------|
| `e2e-live-reconcile-flux.sh` | Real WET->LIVE reconciliation with Flux on local kind cluster |
| `e2e-live-reconcile-argo.sh` | Real WET->LIVE reconciliation with Argo CD on local kind cluster |
| `e2e-connected-governed-reconcile-helm.sh` | Real connected ConfigHub governance round-trip for `helm-paas` + Flux/Argo create/update/drift-correction |

These scripts prove:

1. Create reconciliation (v1 to LIVE).
2. Update reconciliation (v2 rollout).
3. Drift correction (manual drift reverted).

Uses fixtures from [`live-reconcile`](../live-reconcile/).

Connected full-loop proof:

```bash
cub auth login
RECONCILER=both ./examples/demo/e2e-connected-governed-reconcile-helm.sh
```

If your backend does not expose the default ingest/query routes, set
`BRIDGE_INGEST_ENDPOINT` and `BRIDGE_DECISION_ENDPOINT` before running.

## Quick start

```bash
go build -o ./cub-gen ./cmd/cub-gen
./examples/demo/run-all-modules.sh
./examples/demo/ai-work-platform/run-all.sh
```

Persona-first quick starts:

- [Persona 5-minute runbooks](/Users/alexis/Public/github-repos/cub-gen/docs/workflows/persona-5-minute-runbooks.md)

## Qualification caveat

Without a live `WET -> LIVE` reconciler loop shown end-to-end, classify the flow as `governed config automation`, not full `Agentic GitOps`.

References:

- `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md`
- `docs/agentic-gitops/02-design/10-generators-prd.md`

## PRD user-story coverage snapshot

| Status | User stories |
|---|---|
| Met/strong in current demos | 1, 2, 3, 4, 5, 6, 7, 9, 10, 12, 13 |
| Partial (simulated/local-first, not full backend/runtime integration) | 8, 11 |
| Deferred | None |
