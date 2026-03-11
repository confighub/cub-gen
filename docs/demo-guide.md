# Demo Guide

Runnable demo scripts and scenarios for every cub-gen generator.

## Quick demo modules

Run any module independently:

```bash
./examples/demo/module-1-helm-import.sh
./examples/demo/module-2-score-field-map.sh
./examples/demo/module-3-spring-ownership.sh
./examples/demo/module-4-bridge-governance.sh
./examples/demo/module-5-no-config-platform.sh
```

Or run all modules in one pass:

```bash
./examples/demo/run-all-modules.sh
```

## Wizard simulation (repo-first)

Simulate a future GUI discover/import wizard against any repo fixture:

```bash
./examples/demo/simulate-repo-wizard.sh ./examples/helm-paas ./examples/helm-paas auto
./examples/demo/simulate-repo-wizard.sh ./examples/springboot-paas ./examples/springboot-paas springboot-paas
```

This script walks the same step sequence planned for the GUI:

1. Source selection
2. Discover preview
3. Import graph preview (DRY &rarr; GEN &rarr; WET)
4. Provenance/inverse hint preview
5. Import confirmation + bundle/verify/attest summary

## ConfigHub lifecycle demo (create &rarr; deploy &rarr; update)

Run one example through the full governance lifecycle with surface views:

```bash
./examples/demo/simulate-confighub-lifecycle.sh ./examples/c3agent ./examples/c3agent c3agent
```

Run all current platform examples (10 fixtures):

```bash
./examples/demo/run-all-confighub-lifecycles.sh
```

Each run shows:

1. Create path (`discover` &rarr; `import` &rarr; `publish` &rarr; `verify` &rarr; `attest`)
2. Decision/promotion path (`bridge decision` + `bridge promote`)
3. Update path (source config mutation + re-run chain)
4. Visibility surfaces:
   - OCI (bundle digest + output URIs)
   - Flux fixtures (`gitops/flux/*` when present)
   - Argo fixtures (`gitops/argo/*` when present)
   - cub-scout watchlist (derived from wet targets)

## AI work platform demo track

Second demo track focused on AI-work platform scenarios:

```bash
./examples/demo/ai-work-platform/run-all.sh
```

Or run scenarios individually:

```bash
./examples/demo/ai-work-platform/scenario-1-c3agent.sh
./examples/demo/ai-work-platform/scenario-2-swamp.sh
./examples/demo/ai-work-platform/scenario-3-confighub-actions.sh
./examples/demo/ai-work-platform/scenario-4-operations.sh
```

AI Ops PaaS narrative demo (self-service fleet provisioning):

```bash
./examples/ai-ops-paas/demo.sh
```

## Live Flux reconciliation proof

Optional full live reconciliation proof (requires `kind` and Flux):

```bash
./examples/demo/e2e-live-reconcile-flux.sh
```

This script creates a local cluster, installs Flux controllers, and proves:
create reconciliation, update reconciliation, and drift correction against LIVE resources.

## Complete inventory

### App demos

| Example | Demo script |
|---------|-------------|
| helm-paas | `./examples/demo/module-1-helm-import.sh` |
| scoredev-paas | `./examples/demo/module-2-score-field-map.sh` |
| springboot-paas | `./examples/demo/module-3-spring-ownership.sh` |
| just-apps-no-platform-config | `./examples/demo/module-5-no-config-platform.sh` |
| backstage-idp | Included in `./examples/demo/run-all-confighub-lifecycles.sh` |

### Platform demos

| Scenario | Demo script |
|----------|-------------|
| Governance bridge path | `./examples/demo/module-4-bridge-governance.sh` |
| AI work platform track | `./examples/demo/ai-work-platform/run-all.sh` |
| AI Ops PaaS narrative | `./examples/ai-ops-paas/demo.sh` |
| Full lifecycle matrix | `./examples/demo/run-all-confighub-lifecycles.sh` |

### Additional resources

- Repo-first wizard simulation: `./examples/demo/simulate-repo-wizard.sh`
- Core module aggregator: `./examples/demo/run-all-modules.sh`
- Live reconciler E2E: `./examples/demo/e2e-live-reconcile-flux.sh`
- Demo index and track entrypoints: [`examples/demo/README.md`](../examples/demo/README.md)
- Story-card matrix: [Eight Example Story Cards](agentic-gitops/03-worked-examples/04-eight-example-story-cards.md)

!!! note "Qualification"
    Without live WET &rarr; LIVE reconciler evidence, classify the flow as "governed config automation", not full "Agentic GitOps". Running `./examples/demo/e2e-live-reconcile-flux.sh` provides that evidence for the Flux path.
