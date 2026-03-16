# AI Ops PaaS — Self-Service AI Fleet Platform

An internal Heroku for AI agent fleets — governed, auditable, self-service.

Your platform team wants to offer AI agent fleet provisioning as a service.
Teams author 30 lines of YAML describing their fleet (model, concurrency,
budget, credentials). The platform transforms this into 11 governed Kubernetes
resources with full provenance, constraint enforcement, and inverse-edit
guidance.

This is the full platform version of the c3agent story. For the standalone
fleet config (without registry and constraints), see [`c3agent`](../c3agent/).

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding AI fleet platform | Jump to [Run from ConfigHub](#run-from-confighub-connected-mode) |
| **Platform team building AI-ops PaaS** | Jump to [Try it](#try-it) — full registry + constraints |

Both paths lead to the same outcome: self-service AI fleet provisioning with full governance.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real app** | C3 agent fleet (controlplane + gateway) |
| **Real cluster objects** | 11 targets: 2 Deployments, 2 Services, 2 ConfigMaps, Secret, PVC, 3 RBAC |
| **Real inspection target** | `kubectl get deployment c3agent-controlplane -o yaml` |
| **Platform layer** | FrameworkRegistry (7 operations) + ConstraintSet (guardrails) |
| **GitOps transport** | Flux Kustomization or ArgoCD Application |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "How do I provision an AI fleet?" | 30 lines → 11 governed targets | Self-service → IDP portal |
| "Which models are approved?" | Platform registry + constraints | Unapproved model → BLOCK |
| "What budget limits apply?" | Constraint enforcement with tiers | Budget violation → ESCALATE |

## Domain POV (platform teams launching AI-ops PaaS)

Use this when you are building a "Heroku for AI operations" internally:

- app/AI teams should author short fleet intent, not Kubernetes objects,
- platform teams must enforce model, budget, credential, and RBAC guardrails,
- runtime should still flow through existing Flux/Argo + OCI operations.

The first value is platform clarity: one contract for teams, one governance
surface for platform owners.

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

## If you already run AI/ops platforms on Kubernetes

This example is for platform teams supporting autonomous agent workloads:

- App and AI teams want a short, high-level fleet interface.
- Platform teams still need strict policy on model, budget, credentials, and RBAC.
- Runtime fan-out (1 DRY spec -> many Kubernetes resources) must remain auditable.

cub-gen lets app teams stay in high-level fleet config while platform teams keep
deterministic control of generated runtime and governance decisions.

## Why this maps cleanly to the cub-gen framework

| Existing AI platform model | cub-gen concept | Why it matters |
|------|------|------|
| Fleet config (`c3agent*.yaml`) | DRY intent | Teams declare desired behavior, not low-level Kubernetes objects. |
| Fleet runtime resources (11 targets) | WET targets with provenance | Every generated object is linked to the source field and owner. |
| Registry + constraints | Verification/governance loop | Policy checks and attestable decisions can gate autonomous changes. |
| Flux/Argo reconciliation | LIVE state | Existing GitOps runtime still executes deploys and drift correction. |

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

### Scenario A — Fleet passes all constraints (ALLOW)

Team authors fleet config that meets all requirements:

```yaml
# c3agent.yaml — 30 lines of DRY intent
service: c3agent
apiVersion: c3agent/v1
fleet:
  name: ml-review-fleet
  max_concurrent_tasks: 3
  agent_model: claude-sonnet-4-20250514   # in approved list
agent_runtime:
  image: ghcr.io/acme-corp/ai-ops-agent:0.3.1  # approved registry
  max_budget_usd: 8.0                          # within ceiling
```

```bash
# Import with constraint validation
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas

# Evidence chain
./cub-gen publish --space ai-ops ./examples/ai-ops-paas ./examples/ai-ops-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-owner --reason "ML review fleet approved"
```

The platform's constraints check:
- **approved-models-only**: is `claude-sonnet-4-20250514` in the approved list? → yes
- **approved-registries-only**: is `ghcr.io/*` the only image source? → yes
- **budget-ceiling-per-tier**: is $8.00 within the budget ceiling? → yes
- **no-plaintext-secrets**: are all credentials reference-only? → yes

All constraints pass → **ALLOW**.

### Scenario B — Constraint violation (BLOCK/ESCALATE)

Team tries to use unapproved model or exceed budget:

```yaml
# c3agent.yaml — violates constraints
service: c3agent
apiVersion: c3agent/v1
fleet:
  name: ml-review-fleet
  agent_model: untested-model-v0.1        # NOT in approved list
agent_runtime:
  image: docker.io/random/image:latest    # NOT approved registry
  max_budget_usd: 500.0                   # exceeds tier ceiling
```

```bash
# Import with constraint validation
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas

# Evidence chain
./cub-gen publish --space ai-ops ./examples/ai-ops-paas ./examples/ai-ops-paas > bundle.json
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Constraint violations → BLOCK
./cub-gen bridge decision apply --decision decision.json --state BLOCK \
  --approved-by governance-bot \
  --reason "Model 'untested-model-v0.1' not approved. Registry 'docker.io' not approved. Budget $500 exceeds tier ceiling."
```

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

## Run from ConfigHub (connected mode)

If you already have ConfigHub, start here:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"

# Publish and ingest
./cub-gen publish --space ai-ops ./examples/ai-ops-paas ./examples/ai-ops-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map (fleet fields → K8s targets)
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas \
  | jq '.provenance[0].field_origin_map'

# Fleet analysis (11 WET targets from 30 DRY lines)
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas \
  | jq '.provenance[0].c3agent_fleet_analysis'

# Registry operations (7 typed operations)
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas \
  | jq '.provenance[0].registry_operations'

# Evidence bundle
./cub-gen publish --space ai-ops ./examples/ai-ops-paas ./examples/ai-ops-paas \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

## 7. Try one governed change

**ALLOW path**: Team uses approved model and stays within budget:

```yaml
# c3agent.yaml change
fleet:
  agent_model: claude-sonnet-4-20250514  # approved
agent_runtime:
  max_budget_usd: 8.0                    # within ceiling
```

Result: All constraints pass → **ALLOW**

**BLOCK path**: Team uses unapproved model or exceeds budget:

```yaml
# c3agent.yaml change
fleet:
  agent_model: untested-model-v0.1       # NOT approved
agent_runtime:
  max_budget_usd: 500.0                  # exceeds ceiling
```

Result: Constraint violations → **BLOCK**

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/ai-ops-paas/demo-local.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/ai-ops-paas/demo-connected.sh
```
