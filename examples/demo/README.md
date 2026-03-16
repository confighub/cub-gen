# Demo Scripts — Your Starting Point for cub-gen

Runnable demo scripts for every `cub-gen` example. Each script demonstrates
part of the governed change flow:

```
detect → import → publish → verify → attest → (optional) bridge ingest/query
```

## 1. Pick your demo by persona

| Persona | Start here | What you'll prove |
|---------|------------|-------------------|
| **Helm platform engineer** | `module-1-helm-import.sh` | DRY source mapping for chart/value changes |
| **Score platform team** | `module-2-score-field-map.sh` | `score.yaml` → runtime field trace |
| **Spring app/platform team** | `module-3-spring-ownership.sh` | App-vs-platform ownership boundaries |
| **Backstage catalog admin** | [`backstage-idp`](../backstage-idp/) demo | Owner/lifecycle governance |
| **AI fleet operator** | `ai-work-platform/scenario-1-c3agent.sh` | 30 DRY lines → 11 governed WET targets |
| **Swamp/workflow maintainer** | `ai-work-platform/scenario-2-swamp.sh` | Structural workflow-change classification |
| **Ops/SRE workflow owner** | `ai-work-platform/scenario-4-operations.sh` | Governed workflow mutation path |
| **Platform control-plane operator** | `ai-work-platform/scenario-3-confighub-actions.sh` | Recursive governance |
| **Reconciler reliability owner** | `e2e-live-reconcile-flux.sh` | Real create/update/drift correction |
| **App team (no platform layer)** | `module-5-no-config-platform.sh` | Provider config governance |

See also: [Domain POV Matrix](../../docs/workflows/domain-pov-matrix.md) | [Persona 5-minute runbooks](../../docs/workflows/persona-5-minute-runbooks.md)

## 2. Quick start

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Run your first demo (Score example)
./examples/demo/app-ai-change-run.sh ./examples/scoredev-paas

# Run all core modules
./examples/demo/run-all-modules.sh

# Run all AI platform scenarios
./examples/demo/ai-work-platform/run-all.sh
```

## 3. Local mode (no ConfigHub login required)

### Core platform/app track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `module-1-helm-import.sh` | [`helm-paas`](../helm-paas/) | Helm detection, values ownership, field-origin tracing |
| `module-2-score-field-map.sh` | [`scoredev-paas`](../scoredev-paas/) | Score field-origin and inverse edit mapping |
| `module-3-spring-ownership.sh` | [`springboot-paas`](../springboot-paas/) | Spring ownership boundaries (app vs platform) |
| `module-4-bridge-governance.sh` | Multiple | Local bridge contract simulation |
| `module-5-no-config-platform.sh` | [`just-apps-no-platform-config`](../just-apps-no-platform-config/) | No-platform provider governance |
| `run-all-modules.sh` | All above | Run all core modules |

### AI work platform track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `ai-work-platform/scenario-1-c3agent.sh` | [`c3agent`](../c3agent/) + [`ai-ops-paas`](../ai-ops-paas/) | c3agent 11-target coverage |
| `ai-work-platform/scenario-2-swamp.sh` | [`swamp-automation`](../swamp-automation/) | Swamp workflow/model governance |
| `ai-work-platform/scenario-3-confighub-actions.sh` | [`confighub-actions`](../confighub-actions/) | Recursive governance |
| `ai-work-platform/scenario-4-operations.sh` | [`ops-workflow`](../ops-workflow/) | Operations workflow governance |
| `ai-work-platform/run-all.sh` | All above | Run all AI platform scenarios |

### Workflow-first start (Ops + Swamp)

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

## 4. Connected mode (ConfigHub)

Start with authentication:

```bash
cub auth login
TOKEN="$(cub auth get-token)"
cub context get --json | jq -r '.coordinate.user'
```

Connected flow shape:

```
publish → verify → attest → bridge ingest → decision query
```

### Connected runners

```bash
./examples/demo/run-all-connected-lifecycles.sh
./examples/demo/run-all-connected-entrypoints.sh
./examples/demo/run-phase-3-connected-stories.sh
./examples/demo/run-phase-4-connected-stories.sh
```

### Bridge endpoint behavior

| Mode | Behavior |
|------|----------|
| Default (`CONNECTED_FALLBACK_MODE=off`) | Fail fast unless bridge endpoints are reachable |
| Auto fallback (`CONNECTED_FALLBACK_MODE=auto`) | Fall back to backend `changeset` on 404 |
| Forced fallback (`CONNECTED_FALLBACK_MODE=changeset`) | Always use backend fallback (troubleshooting) |

CI behavior:
- `make ci-connected` enforces strict mode (`CONNECTED_FALLBACK_MODE=off`)
- `make ci-connected-troubleshoot` is the only fallback-enabled lane

See also: [connected-ci-bootstrap.md](../../docs/workflows/connected-ci-bootstrap.md)

## 5. Live reconciler e2e (Flux + Argo + kind)

| Script | What it demonstrates |
|--------|---------------------|
| `e2e-live-reconcile-flux.sh` | Real WET→LIVE reconciliation with Flux on local kind cluster |
| `e2e-live-reconcile-argo.sh` | Real WET→LIVE reconciliation with Argo CD on local kind cluster |
| `e2e-connected-governed-reconcile-helm.sh` | Connected ConfigHub governance + Flux/Argo create/update/drift-correction |

These scripts prove:
1. Create reconciliation (v1 to LIVE)
2. Update reconciliation (v2 rollout)
3. Drift correction (manual drift reverted)

Uses fixtures from [`live-reconcile`](../live-reconcile/).

Connected full-loop proof:

```bash
cub auth login
RECONCILER=both ./examples/demo/e2e-connected-governed-reconcile-helm.sh
```

## 6. Lifecycle simulation scripts

| Script | What it demonstrates |
|--------|---------------------|
| `app-ai-change-run.sh <repo> [target]` | One-command app/AI path: import + publish + verify + attest + mutation card |
| `prompt-as-dry-local.sh [repo]` | Prompt-as-DRY local path with AI-only scope guardrails |
| `prompt-as-dry-connected.sh [repo] [target] [slug]` | Prompt-as-DRY connected path with backend ingest/query |
| `simulate-confighub-lifecycle.sh <repo> <target> [slug]` | Full local lifecycle simulation |
| `run-all-confighub-lifecycles.sh` | Lifecycle simulation across all fixtures |
| `run-confighub-lifecycle-connected.sh <repo> <target> [slug]` | Connected lifecycle with ConfigHub ingest/query |
| `simulate-repo-wizard.sh <repo> <target> [hint]` | GUI wizard simulation path |

### Change API adapters

| Script | What it demonstrates |
|--------|---------------------|
| `change-api-adapter.sh --request <json> [--out <json>]` | API-style JSON adapter for `change preview\|run\|explain` |
| `change-api-http-e2e.sh [repo] [target]` | Native HTTP compatibility flow using `/v1/changes` endpoints |

## 7. CI policy gates (PR path)

Use these when you want merge-blocking enforcement, not just local guidance:

- `test/checks/pr-dry-ownership-gate.sh <repo-path> <base-ref> <head-ref> [actor-role] --report-json <path>`
  - Blocks direct WET edits by requiring recognized DRY input files
  - Emits JSON with failures plus inverse-edit suggestions
- `.github/workflows/pr-dry-ownership-gate.yml`
  - Runs the gate for Helm + Spring examples
  - Posts a PR comment with actionable DRY edit guidance

## 8. PR-MR pairing and promotion flows

| Script | What it demonstrates |
|--------|---------------------|
| `flow-a-git-pr-to-mr-connected.sh` | Flow A: Git PR → ConfigHub MR with evidence |
| `flow-b-mr-to-git-pr-connected.sh` | Flow B: ConfigHub MR → Git PR proposal |
| `fr8-promotion-upstream-dry-connected.sh` | FR8: live→WET→DRY upstream promotion |

## 9. Phase 3 connected story scripts

| Script | User story | What it demonstrates |
|--------|------------|---------------------|
| `story-1-existing-repo-connected.sh` | 1 | Existing repo import + connected change query by `change_id` |
| `story-7-ci-api-flow-connected.sh` | 7 | Non-interactive CI flow using `CONFIGHUB_TOKEN` |
| `story-7-agent-tool-call-connected.sh` | 7 | Agent/tool-call adapter flow with shared `change_id` |
| `story-9-multi-repo-wave-connected.sh` | 9 | Multi-repo wave with per-target ALLOW/ESCALATE/BLOCK |
| `story-12-unified-actor-evidence.sh` | 12 | Unified human/CI/AI attestation chain |
| `run-phase-3-connected-stories.sh` | 1,7,9,12 | Run all Phase 3 stories |

## 10. Phase 4 connected story scripts

| Script | User story | What it demonstrates |
|--------|------------|---------------------|
| `story-8-label-evolution-connected.sh` | 8 | Backend-persisted label/taxonomy migration |
| `story-10-signed-writeback-proof-connected.sh` | 10 | Real GitHub PR/commit/branch-protection evidence |
| `story-11-live-breakglass-proposal-connected.sh` | 11 | Break-glass proposals as backend changesets |
| `run-phase-4-connected-stories.sh` | 8,10,11 | Run all Phase 4 stories |

Story 10 required inputs (real GitHub evidence):

```bash
export APP_PR_REPO=owner/app-repo
export APP_PR_NUMBER=123
export PROMOTION_PR_REPO=owner/promotion-repo
export PROMOTION_PR_NUMBER=456
# optional if not already authenticated with gh:
export GH_TOKEN=...
```

`run-phase-4-connected-stories.sh` enforces Story 10 by default. Set `ALLOW_STORY_10_SKIP=1` only for local troubleshooting.

## 11. Example directory quick reference

| Example | Generator | Key demo |
|---------|-----------|----------|
| [`helm-paas`](../helm-paas/) | `helm-paas` | `module-1-helm-import.sh` |
| [`scoredev-paas`](../scoredev-paas/) | `scoredev-paas` | `module-2-score-field-map.sh` |
| [`springboot-paas`](../springboot-paas/) | `springboot-paas` | `module-3-spring-ownership.sh` |
| [`backstage-idp`](../backstage-idp/) | `backstage-idp` | `demo-local.sh` / `demo-connected.sh` |
| [`just-apps-no-platform-config`](../just-apps-no-platform-config/) | `no-config-platform` | `module-5-no-config-platform.sh` |
| [`c3agent`](../c3agent/) | `c3agent` | `ai-work-platform/scenario-1-c3agent.sh` |
| [`ai-ops-paas`](../ai-ops-paas/) | `c3agent` | `ai-work-platform/scenario-1-c3agent.sh` |
| [`swamp-automation`](../swamp-automation/) | `swamp` | `ai-work-platform/scenario-2-swamp.sh` |
| [`confighub-actions`](../confighub-actions/) | `ops-workflow` | `ai-work-platform/scenario-3-confighub-actions.sh` |
| [`ops-workflow`](../ops-workflow/) | `ops-workflow` | `ai-work-platform/scenario-4-operations.sh` |
| [`live-reconcile`](../live-reconcile/) | — | `e2e-live-reconcile-flux.sh` / `e2e-live-reconcile-argo.sh` |

## Qualification caveat

Without a live `WET → LIVE` reconciler loop shown end-to-end, classify the flow as `governed config automation`, not full `Agentic GitOps`.

See: `e2e-live-reconcile-*.sh` and `e2e-connected-governed-reconcile-helm.sh` for full loop proofs.

## PRD user-story coverage

| Status | User stories |
|--------|--------------|
| Met/strong in current demos | 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13 |
| Partial | None |
| Deferred | None |

References:
- `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md`
- `docs/agentic-gitops/02-design/10-generators-prd.md`
