# C3 Agent Fleet — Governed AI Agent Orchestration

Your AI agent fleet runs Claude, GPT, or other LLMs for automated code review,
test generation, security scanning — whatever your teams need. The fleet
configuration lives in `c3agent.yaml`: concurrency limits, model selection,
budget caps, storage, and credential references.

The challenge: AI fleets mutate configuration at high velocity. Model upgrades,
budget adjustments, credential rotations — each change needs governance. Which
model versions are approved? What's the budget ceiling? Are prod credentials
properly isolated? ConfigHub makes AI fleet governance explicit and traceable.

> For the full platform story with registry and constraint enforcement, see
> [`ai-ops-paas`](../ai-ops-paas/).

## What you get

- **Fleet config governance**: 30 lines of DRY config → 11 governed WET
  Kubernetes targets (Deployments, Services, ConfigMaps, Secret, PVC, RBAC)
- **Model approval tracking**: "which fleets run which models?" — answerable
  from the provenance index
- **Budget visibility**: aggregate budget tracking across all agent fleets
- **Credential audit**: "which fleets reference prod credentials?" — cross-repo
  query

## How C3 Agent maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              RECONCILER (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ c3agent.yaml        │          │ Deployment (control)  │         │ Agent fleet      │
│ c3agent-prod.yaml   │──import─▶│ Deployment (gateway)  │──sync──▶│ Running agents   │
│ platform/           │          │ Services (gRPC+HTTP)  │         │ Task execution   │
│   fleet-policy.yaml │          │ ConfigMap, Secret     │         │                 │
└─────────────────────┘          │ PVC, RBAC             │         └─────────────────┘
  App team: fleet config.         └──────────────────────┘           What's actually
  Platform: model + budget policy. 11 WET targets from 30 DRY lines. orchestrating.
```

**DRY** is what the team authors: `c3agent.yaml` defines the fleet — agent model,
concurrency, components (controlplane + gateway), storage, and credential
references. `c3agent-prod.yaml` overrides for HA and higher budgets.

**WET** is what cub-gen traces: 11 Kubernetes resources generated from the fleet
config — Deployments for controlplane and gateway, Services for gRPC and HTTP,
ConfigMap, Secret, PVC, and RBAC resources. Each field maps back to its DRY source.

**LIVE** is the running agent fleet. Flux or ArgoCD reconciles WET to LIVE.

| File | Owner | What it controls |
|------|-------|-----------------|
| `c3agent.yaml` | App team | Fleet config — model, concurrency, components, storage, credentials |
| `c3agent-prod.yaml` | App team | Prod overlay — HA replicas, higher budget, prod credentials |
| `platform/fleet-policy.yaml` | Platform | Approved models, budget ceilings, HA requirements, credential hygiene |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect c3agent fleet config
./cub-gen gitops discover --space platform --json ./examples/c3agent

# Import with full provenance — see 11 WET targets from 30 lines
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

cub-gen detects `c3agent.yaml` with `service: c3agent` and `apiVersion: c3agent/v1`.
The import maps every fleet setting to its rendered Kubernetes targets.

## Real-world scenario: model upgrade across fleets

**Who**: A fintech company running 5 C3 agent fleets — code review, testing,
docs, security scanning, and incident response.

### The change — upgrading the AI model

```yaml
# c3agent.yaml — model upgrade
fleet:
  agent_model: claude-sonnet-4-20250514  # was claude-sonnet-4-20250514
  # changing to: claude-sonnet-4.5-20260101
```

### Governed pipeline

```bash
# cub-gen detects the model change
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent

# Evidence chain
./cub-gen publish --space platform ./examples/c3agent ./examples/c3agent > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Platform checks: is the new model in the approved list?
# If approved → ALLOW. If not yet evaluated → ESCALATE to platform-owner.
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-owner --reason "model upgrade approved after eval"
```

The fleet policy checks the approved models list. If the new model isn't
approved yet, the decision engine **ESCALATE**s to the platform owner.
After evaluation and approval, the fleet upgrade proceeds through the
governed pipeline.

## How it works

cub-gen's `c3agent` generator detects `c3agent.yaml` containing `service: c3agent`
with `apiVersion: c3agent/v1`. On import:

1. **Classifies inputs** — `c3agent.yaml` (role: fleet-config-base),
   `c3agent-prod.yaml` (role: fleet-config-overlay)
2. **Maps 30 DRY lines to 11 WET targets** — controlplane Deployment, gateway
   Deployment, gRPC Service, HTTP Service, ConfigMap, Secret, PVC, and RBAC
   resources
3. **Traces field origins** — `fleet.agent_model` traces from DRY to the
   controlplane Deployment image spec; budget settings trace to ConfigMap
4. **Computes ownership** — fleet config is app-team editable; fleet policy
   constraints are platform-owned

A concrete field trace:

```
DRY:  c3agent.yaml → fleet.agent_model = "claude-sonnet-4-20250514"
      ↓ c3agent transform
WET:  Deployment(controlplane)/spec/template/spec/containers[0]/env[AGENT_MODEL]
```

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `c3agent.yaml` | App team | Fleet config — model, concurrency, components |
| `c3agent-prod.yaml` | App team | Prod overlay — HA, budget, prod credentials |
| `platform/fleet-policy.yaml` | Platform | Approved models, budgets, HA, credential rules |

## Next steps

- **Full platform version**: [`ai-ops-paas`](../ai-ops-paas/) — FrameworkRegistry
  and ConstraintSet for enterprise AI fleet governance
- **AI workflow governance**: [`swamp-automation`](../swamp-automation/) — DAG
  workflows with model binding governance
- **E2E demo**: `../demo/ai-work-platform/scenario-1-c3agent.sh`
