# Swamp Runtime — Governed Helm for AI Model Orchestration

Your platform provides Swamp — an AI model orchestration runtime — as a
managed service. Project teams deploy it via a Helm chart, customizing the
model gateway, replica count, and image version through values overlays.
The platform controls the chart contract, resource baselines, and model
gateway allowlist.

This is the infrastructure side of the Swamp story: deploying the engine
itself. For governance over the *workflows* that run on Swamp, see
[`swamp-automation`](../swamp-automation/).

## What you get

- **Standard Helm governance**: same field-origin tracing, ownership mapping,
  and decision pipeline as any `helm-paas` example
- **Model gateway policy**: platform controls which model gateways (llama,
  mistral, claude) are approved for each environment
- **Runtime policy enforcement**: resource limits, minimum replicas, and
  required probes enforced by platform policy
- **Two-layer Swamp story**: this example (infrastructure) pairs with
  `swamp-automation` (workflow governance) for complete coverage

## How Swamp deployment maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              RECONCILER (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ Chart.yaml          │          │ Deployment           │         │ Swamp runtime    │
│ values.yaml         │──import─▶│ Service              │──sync──▶│ Model gateway    │
│ values-prod.yaml    │          │ HelmRelease          │         │ Running agents   │
│ platform/runtime-   │          │                      │         │                 │
│   policy.yaml       │          │                      │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  Project team: values overlays.   Rendered manifests with           What's actually
  Platform: chart + runtime policy. field provenance.                serving models.
```

**DRY** is what teams edit: `values.yaml` sets the model gateway, replicas, and
image version. `values-prod.yaml` overrides for production. The platform team
owns `Chart.yaml`, templates, and runtime policy.

**WET** is what cub-gen traces: rendered Kubernetes manifests with every field
mapped back to its DRY source — values file, chart template, or platform policy.

**LIVE** is the running Swamp runtime. Flux or ArgoCD reconciles the HelmRelease.

| File | Owner | What it controls |
|------|-------|-----------------|
| `Chart.yaml` | Platform | Chart contract — name, version |
| `values.yaml` | Project team | Base values — replicas, image, model gateway |
| `values-prod.yaml` | Project team | Prod overlay — higher replicas, pinned image, prod gateway |
| `templates/deployment.yaml` | Platform | K8s Deployment structure |
| `platform/runtime-policy.yaml` | Platform | Resource limits, model gateway allowlist, min replicas, required probes |

## If you already operate Helm-based AI runtimes

This example targets teams already managing AI runtime delivery with Helm:

- Project teams tune runtime values (gateway, replicas, image).
- Platform teams own chart structure and runtime safety constraints.
- Production changes need clear ownership and policy checks before rollout.

cub-gen keeps Helm workflows familiar while adding policy-aware tracing for
runtime-specific knobs like model gateway selection.

## Why this maps cleanly to the cub-gen framework

| Existing Helm runtime model | cub-gen concept | Why it matters |
|------|------|------|
| `values*.yaml` for runtime tuning | DRY app/project intent | Teams edit only high-level runtime knobs. |
| Rendered Deployment/Service/HelmRelease | WET targets with provenance | Runtime rollouts can be traced to exact values keys. |
| Runtime policy constraints | Governance layer | Unsafe gateway or capacity changes can be blocked/escalated. |
| Flux/Argo reconcile path | LIVE state | Existing release and reconciliation tooling remains intact. |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect Helm chart for Swamp runtime
./cub-gen gitops discover --space platform --json ./examples/swamp-project

# Import with full provenance
./cub-gen gitops import --space platform --json ./examples/swamp-project ./examples/swamp-project
```

cub-gen detects `Chart.yaml` + `values.yaml` as a `helm-paas` project. The
import traces field origins through the Helm template structure, including
Swamp-specific fields like `runtime.modelGateway`.

## Real-world scenario: switching model gateways

**Who**: An ML platform team maintaining the Swamp runtime chart. A project
team wants to switch from `llama-gateway` to `mistral-gateway`.

### The change

```yaml
# values-prod.yaml — project team edits
runtime:
  modelGateway: mistral-gateway  # was llama-gateway
```

### Governed pipeline

```bash
# cub-gen detects the gateway change
./cub-gen gitops import --space platform --json ./examples/swamp-project ./examples/swamp-project

# Evidence chain
./cub-gen publish --space platform ./examples/swamp-project ./examples/swamp-project > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Platform checks: is mistral-gateway in the allowlist?
# Runtime policy allows llama, mistral, claude → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-owner --reason "mistral gateway approved for team workload"
```

The runtime policy's `model_gateway_allowlist` includes `mistral` → **ALLOW**.
If the team requested an unapproved gateway, the decision engine would
**BLOCK** and route to the platform owner.

## How it works

This example uses the standard `helm-paas` generator — same as
[`helm-paas`](../helm-paas/). The Swamp-specific aspect is the runtime policy
which governs model gateway selection and AI-specific resource requirements:

- **Model gateway allowlist**: only approved gateways (llama, mistral, claude)
  can be deployed
- **Minimum replicas by environment**: production requires at least 2 replicas
- **Resource limits**: CPU 4, memory 8Gi — higher than typical web services
  because model serving is resource-intensive
- **Required probes**: readiness and liveness probes are mandatory

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `Chart.yaml` | Platform | Helm chart contract |
| `values.yaml` | Project team | Base values — gateway, replicas, image |
| `values-prod.yaml` | Project team | Prod overlay |
| `templates/deployment.yaml` | Platform | K8s Deployment structure |
| `platform/runtime-policy.yaml` | Platform | Gateway allowlist, resource limits, HA |

## The complete Swamp + ConfigHub picture

| Layer | Example | Generator | What it governs |
|-------|---------|-----------|-----------------|
| **Workflow** | [`swamp-automation`](../swamp-automation/) | `swamp` | Workflow definitions, model bindings, vault policy |
| **Runtime** | `swamp-project` (this) | `helm-paas` | Swamp engine deployment, model gateway, resources |

Both use the same DRY→WET→LIVE pattern and produce ConfigHub-ready change
bundles. Together they give an AI automation platform full governance — from
the workflows it runs to the infrastructure it runs on.

## Next steps

- **Workflow governance**: [`swamp-automation`](../swamp-automation/) — govern
  the workflows that run on this Swamp instance
- **AI agent fleets**: [`c3agent`](../c3agent/) — standalone agent fleet config
- **E2E Helm demo**: `../demo/module-1-helm-import.sh`

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline"
./examples/swamp-project/demo-local.sh

echo "connected (requires ConfigHub auth)"
cub auth login
./examples/swamp-project/demo-connected.sh
```
