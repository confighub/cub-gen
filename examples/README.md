# Example Repos For Demo Narratives

These fixtures are intentionally small but realistic enough to demonstrate platform + app collaboration and governance turns.

## Lifecycle matrix fixtures (current: 10)

The lifecycle aggregator (`examples/demo/run-all-confighub-lifecycles.sh`) runs these 10 examples:

| Example | What it demonstrates |
|---|---|
| `helm-paas` | Helm import/govern path with values ownership and inverse guidance |
| `scoredev-paas` | score.dev abstraction with field-origin and inverse mapping |
| `springboot-paas` | Framework config ownership split (app vs platform) |
| `backstage-idp` | Backstage-style IDP config in the same contract model |
| `ably-config` | App-config style governance path |
| `ops-workflow` | Governed operational workflow config |
| `c3agent` | AI fleet config to governed multi-resource runtime model |
| `ai-ops-paas` | AI Ops PaaS narrative path with platform constraints |
| `swamp-automation` | AI-native workflow automation with governed config |
| `confighub-actions` | Tokened execution/decision path illustration |

Run all lifecycle fixtures:

```bash
./examples/demo/run-all-confighub-lifecycles.sh
```

## Other entry paths

Core module walkthrough:

```bash
./examples/demo/run-all-modules.sh
```

AI work platform track:

```bash
./examples/demo/ai-work-platform/run-all.sh
```

## Live reconciliation caveat

The live end-to-end reconciler proof today is Flux-only via:

```bash
./examples/demo/e2e-live-reconcile-flux.sh
```

This live proof uses `examples/live-reconcile/flux/manifests-v1|v2` fixtures.

## Docs and illustrations

1. `docs/agentic-gitops/00-index/02-illustrated-cheat-sheet.md`
2. `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md`
3. `docs/agentic-gitops/05-rollout/94-demo-illustration-pack.md`
