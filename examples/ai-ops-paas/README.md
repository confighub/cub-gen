# AI Ops PaaS — Self-Service AI Fleet Platform

An internal Heroku for AI agent fleets — governed, auditable, self-service.

Your platform team wants to offer AI agent fleet provisioning as a service.
Teams author 30 lines of YAML describing their fleet (model, concurrency,
budget, credentials). The platform transforms this into 11 governed Kubernetes
resources with full provenance, constraint enforcement, and inverse-edit
guidance.

This is the full platform version of the c3agent story. For the standalone
fleet config (without registry and constraints), see [`c3agent`](../c3agent/).

## What you get

- **Self-service provisioning**: teams author `c3agent.yaml`, platform provides
  everything else — Deployments, Services, ConfigMaps, Secrets, PVC, RBAC
- **Constraint enforcement**: approved models only, budget ceilings per tier,
  minimum replicas for production, no plaintext secrets
- **Framework registry**: 7 typed operations that an IDP portal can use to
  render self-service forms
- **30 lines → 11 targets**: minimal DRY input produces a complete governed
  Kubernetes footprint

## How AI Ops maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              RECONCILER (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ c3agent.yaml        │          │ Deployment x2        │         │ Agent fleet      │
│ c3agent-prod.yaml   │──import─▶│ Service x2           │──sync──▶│ Control plane    │
│ platform/           │          │ ConfigMap x2         │         │ Gateway          │
│   registry.yaml     │          │ Secret, PVC          │         │ Task execution   │
│   constraints.yaml  │          │ RBAC (SA, CR, CRB)   │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  Team: fleet config.              11 governed K8s resources          What's actually
  Platform: registry + constraints. with field provenance.            orchestrating.
```

**DRY** is what the team authors: `c3agent.yaml` defines the fleet config —
model, concurrency, budget, components, storage, credentials. The platform
provides `registry.yaml` (typed operations) and `constraints.yaml` (guardrails).

**WET** is what cub-gen traces: 11 Kubernetes resources, each with field-origin
confidence, inverse-edit hints, and provenance digest.

**LIVE** is the running agent fleet on Kubernetes.

| DRY Section | WET Targets | What it governs |
|-------------|-------------|-----------------|
| `fleet.*` | ConfigMap (fleet-config) | Model, concurrency, schedule |
| `components.controlplane.*` | Deployment, Service | Control plane pods + ports |
| `components.gateway.*` | Deployment, Service | Gateway pods + ports |
| `agent_runtime.*` | ConfigMap (agent-template) | Budget, image, paths |
| `credentials.*` | Secret | API keys, tokens, DB URL |
| `storage.*` | PersistentVolumeClaim | Task data volume |
| (implicit) | ServiceAccount, ClusterRole, CRB | RBAC for agent workloads |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Discover — finds c3agent generator
./cub-gen gitops discover --space ai-ops --json ./examples/ai-ops-paas

# Import — traces 30 DRY lines to 11 WET targets
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas

# See the full c3agent generator triple
./cub-gen generators --json --details | jq '.families[] | select(.kind == "c3agent")'
```

## Real-world scenario: onboarding a new ML review fleet

**Who**: An ML team wants an agent fleet for automated code review. The platform
team provides self-service provisioning with guardrails.

### Step 1 — Team authors fleet config

```yaml
# c3agent.yaml — 30 lines of DRY intent
service: c3agent
apiVersion: c3agent/v1
fleet:
  name: ml-review-fleet
  max_concurrent_tasks: 3
  agent_model: claude-sonnet-4-20250514
agent_runtime:
  image: ghcr.io/acme-corp/ai-ops-agent:0.3.1
  max_budget_usd: 8.0
```

### Step 2 — Platform validates against constraints

```bash
# Import with constraint validation
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas

# Evidence chain
./cub-gen publish --space ai-ops ./examples/ai-ops-paas ./examples/ai-ops-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-owner --reason "ML review fleet approved"
```

The platform's constraints check:
- **approved-models-only**: is `claude-sonnet-4-20250514` in the approved list? → yes
- **approved-registries-only**: is `ghcr.io/*` the only image source? → yes
- **budget-ceiling-per-tier**: is $8.00 within the budget ceiling? → yes (ESCALATE if over)
- **no-plaintext-secrets**: are all credentials reference-only? → yes

All constraints pass → **ALLOW**.

## How it works — the platform layer

What distinguishes `ai-ops-paas` from the standalone [`c3agent`](../c3agent/)
is the platform layer:

### FrameworkRegistry (`platform/registry.yaml`)

Defines 7 typed operations the platform offers as self-service:
`configureFleet`, `deployControlPlane`, `deployGateway`, `configureAgentRuntime`,
`bindCredentials`, `provisionStorage`, `grantJobPermissions`.

Each operation has typed inputs (JSON Schema), resource outputs, constraints,
and validation rules. An IDP portal can render self-service forms from this.

### ConstraintSet (`platform/constraints.yaml`)

Platform guardrails with enforcement levels:
- **BLOCK**: approved models only, approved registries, no plaintext secrets,
  min 2 replicas in prod, least-privilege RBAC
- **ESCALATE**: budget ceiling per tier, prod credentials separate from dev

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `c3agent.yaml` | Team | Fleet config — model, concurrency, budget |
| `c3agent-prod.yaml` | Team | Prod overlay — HA, budget, credentials |
| `platform/registry.yaml` | Platform | Typed operations for IDP self-service |
| `platform/constraints.yaml` | Platform | Guardrails — models, budget, HA, secrets |
| `agent/run.sh` | Team | Agent workload entry point |

## Next steps

- **Standalone fleet config**: [`c3agent`](../c3agent/) — minimal version
  without registry and constraints
- **AI workflow governance**: [`swamp-automation`](../swamp-automation/) —
  DAG workflows with model binding governance
- **E2E demo**: `../demo/ai-work-platform/scenario-1-c3agent.sh`
