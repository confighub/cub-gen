# Demo Scripts — Your Starting Point for cub-gen

Runnable demo scripts for every `cub-gen` example. Each script demonstrates
part of the governed change flow:

```
detect → import → publish → verify → attest
                                          → connected smoke
                                          → (optional, deep) bridge ingest/query
```

If you are new, do not start with "run everything." Start with one concrete
adoption path that matches what you already run.

## Proof ladder

Use the demo surface in this order:

1. **Local source-side proof**: `detect -> import -> publish -> verify -> attest`
2. **Connected smoke proof**: `cub auth login` plus `./examples/demo/run-connected-smoke.sh`
3. **Deep connected proof**: example wrappers, standard changeset-based flows, and bridge-only demos when needed
4. **Runtime proof**: real WET->LIVE reconciliation or a real deployed app

That ordering matters. The first run should answer "do I trust the source-side
trace?" before you spend time on backend or cluster setup.

For exact per-example proof tiers, use the generated
[Example Truth Matrix](../../docs/testing/example-truth-matrix.md).
For workflow and AI example quality, use the
[AI Example Hygiene Checklist](../../docs/workflows/ai-example-hygiene-checklist.md).

The flagship examples also now publish an explicit AI-first bundle:

- [`helm-paas`](../helm-paas/AI_START_HERE.md) with companion [prompts](../helm-paas/prompts.md) and [contracts](../helm-paas/contracts.md)
- [`scoredev-paas`](../scoredev-paas/AI_START_HERE.md) with companion [prompts](../scoredev-paas/prompts.md) and [contracts](../scoredev-paas/contracts.md)
- [`springboot-paas`](../springboot-paas/AI_START_HERE.md) with companion [prompts](../springboot-paas/prompts.md) and [contracts](../springboot-paas/contracts.md)
- [`swamp-automation`](../swamp-automation/AI_START_HERE.md) with companion [prompts](../swamp-automation/prompts.md) and [contracts](../swamp-automation/contracts.md)

## 1. Start with one of these

| If you already run... | Start here | What actually runs | What you can inspect next |
|---|---|---|---|
| Helm plus Argo/Flux | `./examples/demo/start-platform-first.sh` | local lifecycle for `helm-paas`, then the example-owned governed-change proof, layered trace proof, and live wrapper | values ownership now, layered selector-to-overlay proof next, connected + live proof after |
| Spring Boot app repos | `./examples/demo/start-app-first.sh` | local lifecycle for `springboot-paas`, then the example-owned embedded payload mutation proof | app-vs-platform config ownership now, direct embedded apply-here proof next, standalone live app proof after |
| Score.dev workloads | `./examples/demo/start-score-first.sh` | Score field trace, inverse edit map, governed workload-contract proof, and standalone runtime proof | `score.yaml` to runtime-field mapping now, local ALLOW/ESCALATE proof next, live `checkout-api` proof after |
| Reconciler/runtime proof | `RECONCILER=both ./examples/live-reconcile/demo-local.sh` | real Flux and Argo reconciliation on kind | pods, rollout, and drift correction |

Cluster-side follow-on: pair the above with [`cub-scout`](https://github.com/confighub/cub-scout)
when you want to inspect what is actually running after reconciliation.

## 2. What ends with live proof today

| Path | Live thing you can inspect | Command | Truth today |
|---|---|---|---|
| Spring standalone app path | `inventory-api` on kind plus HTTP verification | `./examples/springboot-paas/verify-e2e.sh` | Strongest standalone app proof |
| Helm flagship live path | Argo and Flux reconciliation of Helm-derived manifests plus connected governance evidence | `RECONCILER=both ./examples/helm-paas/demo-runtime.sh` | Example-owned wrapper over the shared live-reconcile harness |
| Connected smoke | ConfigHub-authenticated flagship bundle/attestation chain | `./examples/demo/run-connected-smoke.sh` | Release-facing connected proof lane |
| Score standalone path | `checkout-api` on kind plus HTTP verification from merged Score inputs | `./examples/scoredev-paas/verify-e2e.sh` | Standalone Score live proof is now real |

## 3. Pick your demo by persona

| Persona | Start here | What you'll prove |
|---------|------------|-------------------|
| **Helm platform engineer** | `module-1-helm-import.sh` | DRY source mapping for chart/value changes |
| **Score platform team** | `start-score-first.sh` | `score.yaml` → runtime field trace with connected next steps |
| **Spring app/platform team** | `module-3-spring-ownership.sh` | App-vs-platform ownership boundaries |
| **Backstage catalog admin** | [`backstage-idp`](../backstage-idp/) demo | Owner/lifecycle governance |
| **AI fleet operator** | `ai-work-platform/scenario-1-c3agent.sh` | 30 DRY lines → 11 governed WET targets |
| **Swamp/workflow maintainer** | `ai-work-platform/scenario-2-swamp.sh` | Structural workflow-change classification with explicit ALLOW/BLOCK proof |
| **Ops/SRE workflow owner** | `ai-work-platform/scenario-4-operations.sh` | Governed workflow mutation path with explicit ALLOW/BLOCK proof |
| **Platform control-plane operator** | `ai-work-platform/scenario-3-confighub-actions.sh` | Recursive governance |
| **Reconciler reliability owner** | `e2e-live-reconcile-flux.sh` | Real create/update/drift correction |
| **App team (no platform layer)** | `module-5-no-config-platform.sh` | Provider config governance |

See also: [Domain POV Matrix](../../docs/workflows/domain-pov-matrix.md) | [Persona 5-minute runbooks](../../docs/workflows/persona-5-minute-runbooks.md)

## 4. Quick start

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Platform-first first run
./examples/demo/start-platform-first.sh

# App-first first run
./examples/demo/start-app-first.sh

# Score-first first run
./examples/demo/start-score-first.sh

# Runtime proof after source-side import
RECONCILER=both ./examples/live-reconcile/demo-local.sh
```

Once those are clear, then expand into the broader module and lifecycle surface.

```bash
./examples/demo/run-all-modules.sh
./examples/demo/ai-work-platform/run-all.sh
```

## 5. Local mode (no ConfigHub login required)

### Core platform/app track

| Script | Example | What it demonstrates |
|--------|---------|---------------------|
| `start-platform-first.sh` | [`helm-paas`](../helm-paas/) + [`live-reconcile`](../live-reconcile/) follow-on | Opinionated platform-first starter path |
| `../helm-paas/demo-governed-change.sh` | [`helm-paas`](../helm-paas/) | App-team ALLOW path vs platform-contract BLOCK path with the real PR ownership gate |
| `../helm-paas/demo-layered-trace.sh` | [`helm-paas`](../helm-paas/) | Cluster labels -> ApplicationSet selector -> values overlay proof plus blocked customer security weakening |
| `start-app-first.sh` | [`springboot-paas`](../springboot-paas/) + [`live-reconcile`](../live-reconcile/) follow-on | Opinionated app-first starter path |
| `../springboot-paas/demo-governed-routes.sh` | [`springboot-paas`](../springboot-paas/) | App-owned field ALLOW versus platform-owned field BLOCKED with `springboot validate-mutation` |
| `../springboot-paas/demo-embedded-config-mutation.sh` | [`springboot-paas`](../springboot-paas/) | Direct embedded `application.yaml` mutation in the ConfigHub payload plus blocked datasource proof |
| `start-score-first.sh` | [`scoredev-paas`](../scoredev-paas/) | Opinionated Score-first starter path with local governance and standalone runtime proof |
| `../scoredev-paas/demo-governed-workload.sh` | [`scoredev-paas`](../scoredev-paas/) | App-owned image change ALLOW versus unapproved resource type ESCALATE with `score validate-workload` |
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
| `ai-work-platform/scenario-2-swamp.sh` | [`swamp-automation`](../swamp-automation/) | Swamp workflow/model governance with example-owned local ALLOW/BLOCK proof |
| `ai-work-platform/scenario-3-confighub-actions.sh` | [`confighub-actions`](../confighub-actions/) | Recursive governance |
| `ai-work-platform/scenario-4-operations.sh` | [`ops-workflow`](../ops-workflow/) | Operations workflow governance with example-owned local ALLOW/BLOCK proof |
| `../swamp-automation/demo-governed-structure.sh` | [`swamp-automation`](../swamp-automation/) | Approved model-method ALLOW versus unapproved model plus missing `validate` BLOCK |
| `../ops-workflow/demo-governed-policy.sh` | [`ops-workflow`](../ops-workflow/) | Example-owned local ALLOW/BLOCK workflow policy proof |
| `ai-work-platform/run-all.sh` | All above | Run all AI platform scenarios |

### Workflow-first start (Ops + Swamp)

If your platform is workflow-heavy, start here before app-manifest demos:

```bash
# Swamp workflow graph governance (models/methods/required steps)
./examples/demo/ai-work-platform/scenario-2-swamp.sh
./examples/swamp-automation/demo-governed-structure.sh

# Ops workflow governance (actions/schedules/approval gates)
./examples/demo/ai-work-platform/scenario-4-operations.sh
./examples/ops-workflow/demo-governed-policy.sh
```

## 6. Connected mode (ConfigHub)

Start with authentication:

```bash
cub auth login
cub space list                                     # auth + API reachability check
cub context get --json | jq -r '.coordinate.user'
```

`--space` means the ConfigHub space where the connected scripts record their
bundle, decision, and evidence context. In the starter scripts, the current
ConfigHub context is the source of truth. In local-only runs, `platform` is the
default teaching value.

Connected smoke shape:

```bash
cub auth login
./examples/demo/run-connected-smoke.sh
```

Deep connected flow shape:

```
publish → verify → attest → standard connected wrapper or changeset flow
                         → (optional, deep) bridge ingest → decision query
```

### Connected smoke runner

```bash
./examples/demo/run-connected-smoke.sh
```

This is the repo's release-facing connected proof lane. It verifies ConfigHub
auth/context and runs the flagship example wrappers without depending on bridge
ingest/query endpoints.

### Deep connected runners

```bash
./examples/demo/run-all-connected-lifecycles.sh
./examples/demo/run-all-connected-entrypoints.sh
./examples/demo/run-phase-3-connected-stories.sh
./examples/demo/run-phase-4-connected-stories.sh
```

The extended deep-proof lane also covers:

- `flow-a-git-pr-to-mr-connected.sh`
- `flow-b-mr-to-git-pr-connected.sh`
- evidence validation via `test/checks/check-flow-evidence.sh`

These are advanced connected lanes. Some still depend on bridge-backed flows or
changeset fallbacks because they are demonstrating deeper optional behavior,
not the release-facing default connected path.

### Advanced bridge endpoint behavior

| Mode | Behavior |
|------|----------|
| Default (`CONNECTED_FALLBACK_MODE=off`) | Fail fast unless bridge endpoints are reachable |
| Auto fallback (`CONNECTED_FALLBACK_MODE=auto`) | Fall back to backend `changeset` on 404 |
| Forced fallback (`CONNECTED_FALLBACK_MODE=changeset`) | Always use backend fallback (troubleshooting) |

CI behavior:
- `make ci-connected` runs the smaller ConfigHub smoke lane and does not touch bridge endpoints
- `make ci-connected-deep` runs the broader connected stories, flows, and live reconciler proofs
- `make ci-connected-troubleshoot` is the only fallback-enabled deep lane

See also: [connected-ci-bootstrap.md](../../docs/workflows/connected-ci-bootstrap.md)

## 7. Live reconciler e2e (Flux + Argo + kind)

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

## 8. Lifecycle simulation scripts

| Script | What it demonstrates |
|--------|---------------------|
| `app-ai-change-run.sh <repo> [target]` | One-command app/AI path: import + publish + verify + attest + mutation card |
| `prompt-as-dry-local.sh [repo]` | Prompt-as-DRY local path with AI-only scope guardrails |
| `prompt-as-dry-connected.sh [repo] [target] [slug]` | Prompt-as-DRY deep connected path with backend ingest/query |
| `simulate-confighub-lifecycle.sh <repo> <target> [slug]` | Full local lifecycle simulation |
| `run-all-confighub-lifecycles.sh` | Lifecycle simulation across all fixtures |
| `run-confighub-lifecycle-connected.sh <repo> <target> [slug]` | Deep connected lifecycle with ConfigHub ingest/query |
| `simulate-repo-wizard.sh <repo> <target> [hint]` | GUI wizard simulation path |

### Change API helpers

| Script | What it demonstrates |
|--------|---------------------|
| `change-api-adapter.sh --request <json> [--out <json>]` | API-style JSON wrapper for `change preview\|run\|explain` |
| `change-api-http-e2e.sh [repo] [render-target]` | Native repo-first HTTP flow using `/v1/changes` endpoints |

## 9. CI policy gates (PR path)

Use these when you want merge-blocking enforcement, not just local guidance:

- `test/checks/pr-dry-ownership-gate.sh <repo-path> <base-ref> <head-ref> [actor-role] --report-json <path>`
  - Blocks direct WET edits by requiring recognized DRY input files
  - Emits JSON with failures plus inverse-edit suggestions
- `.github/workflows/pr-dry-ownership-gate.yml`
  - Runs the gate for Helm + Spring examples
  - Posts a PR comment with actionable DRY edit guidance

## 10. PR-MR pairing and promotion flows

| Script | What it demonstrates |
|--------|---------------------|
| `flow-a-git-pr-to-mr-connected.sh` | Flow A: Git PR → ConfigHub MR with evidence |
| `flow-b-mr-to-git-pr-connected.sh` | Flow B: ConfigHub MR → Git PR proposal |
| `fr8-promotion-upstream-dry-connected.sh` | FR8: live→WET→DRY upstream promotion |

## 11. Phase 3 connected story scripts

| Script | User story | What it demonstrates |
|--------|------------|---------------------|
| `story-1-existing-repo-connected.sh` | 1 | Existing repo import + connected change query by `change_id` |
| `story-7-ci-api-flow-connected.sh` | 7 | Non-interactive CI flow using `CONFIGHUB_TOKEN` |
| `story-7-agent-tool-call-connected.sh` | 7 | Agent/tool-call adapter flow with shared `change_id` |
| `story-9-multi-repo-wave-connected.sh` | 9 | Multi-repo wave with per-target ALLOW/ESCALATE/BLOCK |
| `story-12-unified-actor-evidence.sh` | 12 | Unified human/CI/AI attestation chain |
| `run-phase-3-connected-stories.sh` | 1,7,9,12 | Run all Phase 3 stories |

## 12. Phase 4 connected story scripts

| Script | User story | What it demonstrates |
|--------|------------|---------------------|
| `story-8-label-evolution-connected.sh` | 8 | Backend-persisted label/taxonomy migration |
| `story-10-signed-writeback-proof-connected.sh` | 10 | Real GitHub PR/commit/branch-protection evidence |
| `story-11-live-breakglass-proposal-connected.sh` | 11 | Break-glass proposals as backend changesets |
| `run-phase-4-connected-stories.sh` | 8,10,11 | Run all Phase 4 stories |

## PR-MR pairing and promotion flows (Wave 2)

These scripts demonstrate the bidirectional handoff between Git PR workflow and
ConfigHub MR workflow, plus upstream DRY promotion:

| Script | Flow | What it demonstrates |
|--------|------|---------------------|
| `flow-a-git-pr-to-mr-connected.sh` | A | Git PR → ConfigHub MR: developer opens PR, ConfigHub creates/updates MR with evidence |
| `flow-b-mr-to-git-pr-connected.sh` | B | ConfigHub MR → Git PR: ConfigHub initiates change, generates Git PR after approval |
| `fr8-promotion-upstream-dry-connected.sh` | FR8 | Promotion to upstream DRY: successful app change promoted to platform base |

### Flow A: Git PR → ConfigHub MR

The most common governed change flow:

```bash
export GIT_REPO=owner/repo
export PR_NUMBER=123
./examples/demo/flow-a-git-pr-to-mr-connected.sh ./examples/helm-paas
```

Steps:
1. Developer makes changes and opens Git PR
2. CI/webhook triggers cub-gen import
3. cub-gen creates/updates ConfigHub MR with evidence bundle
4. ConfigHub evaluates and returns governed decision (ALLOW/ESCALATE/BLOCK)
5. Decision is posted back to Git PR as status check

### Flow B: ConfigHub MR → Git PR

Reverse flow for ConfigHub-initiated changes:

```bash
./examples/demo/flow-b-mr-to-git-pr-connected.sh ./examples/helm-paas
```

Used for:
- Live-origin proposals (story 11 accept path)
- Platform-initiated changes
- Upstream DRY promotions

### FR8: Promotion to Upstream Platform DRY

Full promotion flow from live observation through WET to upstream DRY:

```bash
./examples/demo/fr8-promotion-upstream-dry-connected.sh ./examples/helm-paas
```

Phases:
1. **LIVE → WET**: Observe live state, capture delta
2. **WET → governed**: Evaluate against policies, reach ALLOW
3. **governed → promotion**: After successful rollout, propose promotion
4. **promotion → upstream DRY**: Platform team reviews and merges
5. **cleanup**: Reduce/remove app overlay to avoid drift

See also: [PR-MR Linkage Contract](../../docs/contracts/pr-mr-linkage-and-dry-promotion.md)

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

## PRD execution status

| Status | What is true today |
|--------|--------------------|
| Strong now | Story scripts exist for stories 1-13; Flux and Argo live reconcile proofs exist; connected lifecycle and PR/MR flow scripts are in the demo surface |
| In progress | Working `v0.4` roadmap: make cub-gen's role obvious as Component -> Deployable Variant -> Target/Connections/Change/Proof |
| Actively tracked | [#287](https://github.com/confighub/cub-gen/issues/287) plus milestone [v0.4: Component -> Deployable Variant -> Proof](https://github.com/confighub/cub-gen/milestone/4) |

For the per-example truth behind those claims, use the generated [Example Truth Matrix](../../docs/testing/example-truth-matrix.md). It is derived from the example catalog, connected runners, source-side tests, and live proof scripts.

That means the demo surface is broad, but we are still hardening the main examples
before we claim full PRD-complete product proof.

References:
- `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md`
- `docs/agentic-gitops/02-design/10-generators-prd.md`
