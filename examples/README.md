# Examples

Every example follows the same canonical pattern regardless of generator, app type, or workload:

```
DRY intent → cub-gen detect/import → WET with provenance → publish/verify/attest → ConfigHub governed decision
```

The generator changes. The app type changes. The pattern stays the same.

See [Example Checklist](../docs/workflows/example-checklist.md) for the verification criteria.

## App generators (Kubernetes workloads)

| Example | Generator | DRY Source | Story |
|---------|-----------|------------|-------|
| [`helm-paas`](helm-paas/) | `helm-paas` | `Chart.yaml` + `values.yaml` | Platform owns chart contract; app team owns values overlays |
| [`scoredev-paas`](scoredev-paas/) | `scoredev-paas` | `score.yaml` | App-first authoring in Score format with platform contracts |
| [`springboot-paas`](springboot-paas/) | `springboot-paas` | `application.yaml` | Java service with app config + platform runtime policy |

## Operations generators (governed execution)

| Example | Generator | DRY Source | Story |
|---------|-----------|------------|-------|
| [`ops-workflow`](ops-workflow/) | `ops-workflow` | `operations.yaml` | Scheduled maintenance workflows with approval gates |
| [`confighub-actions`](confighub-actions/) | `ops-workflow` | `operations.yaml` | ConfigHub lifecycle: plan → verify → deploy |

## Integration generators (external services)

| Example | Generator | DRY Source | Story |
|---------|-----------|------------|-------|
| [`just-apps-no-platform-config`](just-apps-no-platform-config/) | `ably-config` | `ably.yaml` | Just apps, no platform config — provider config only |
| [`backstage-idp`](backstage-idp/) | `backstage-idp` | `catalog-info.yaml` | Developer portal catalog with governed ownership |

## Automation generators (AI-native platforms)

| Example | Generator | DRY Source | Story |
|---------|-----------|------------|-------|
| [`c3agent`](c3agent/) | `c3agent` | `c3agent.yaml` | Standalone AI agent fleet config |
| [`ai-ops-paas`](ai-ops-paas/) | `c3agent` | `c3agent.yaml` | Full platform with registry + constraints |
| [`swamp-automation`](swamp-automation/) | `swamp` | `.swamp.yaml` | ConfigHub + Swamp agentic app platform |
| [`swamp-project`](swamp-project/) | `helm-paas` | `Chart.yaml` | Helm chart deploying the Swamp runtime |

## Special purpose

| Example | Purpose |
|---------|---------|
| [`live-reconcile`](live-reconcile/) | Flux e2e test fixture — proves WET→LIVE reconciliation |

## Quick start

```bash
# Build once
go build -o ./cub-gen ./cmd/cub-gen

# Try any example
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas

# Full bridge flow (works with any example)
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen verify-attestation --in /tmp/attestation.json --bundle /tmp/bundle.json
```

## The canonical pattern

Every example answers five questions:

1. **What is this?** — Real-world scenario in plain English
2. **Who does what?** — Explicit ownership map (app / platform / reconciler)
3. **What does cub-gen add?** — DRY→WET mapping with runnable commands
4. **How do I run it?** — Copy-paste commands from repo root
5. **Show me a real-world example using ConfigHub** — Governed pipeline walkthrough
