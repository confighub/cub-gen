# AI Ops PaaS

**An internal Heroku for AI agent fleets — governed, auditable, self-service.**

This example shows how a platform team can offer self-service AI agent fleet
provisioning using the c3agent generator with full DRY/WET governance.

## What this demonstrates

A team wants to run a fleet of Claude Code agents for automated ML code review.
Instead of hand-writing Kubernetes manifests, they author a short DRY config
(`c3agent.yaml`). The platform's c3agent generator transforms this into 11
governed Kubernetes resources with full provenance.

```
c3agent.yaml (DRY intent)
    |
    v
c3agent generator (detect -> import)
    |
    v
11 WET targets: Deployment x2, Service x2, ConfigMap x2, Secret,
                 PVC, ServiceAccount, ClusterRole, ClusterRoleBinding
    |
    v
Each target has: field-origin confidence, inverse edit hints,
                 rendered lineage template, provenance digest
```

## Pattern 5: Ops Apps

This follows the Ops Apps authoring pattern from the ConfigHub design:

- **DRY input:** Fleet config authored by the team (`c3agent.yaml`)
- **Generator:** c3agent (auto-detected from `service: c3agent`)
- **WET output:** Governed Kubernetes manifests with provenance
- **Value:** Operations become config — diffable, governed, attested

The key distinction from app scaffolding (Pattern 2): these operations manage
*operational infrastructure* (agent fleets, control planes, RBAC), not
application code. Governance is mandatory at this level because AI agents
authoring changes at velocity makes ungoverned execution an incident factory.

## Files

```
ai-ops-paas/
  c3agent.yaml            DRY base config (team-authored intent)
  c3agent-prod.yaml       Production overlay (platform-enforced)
  agent/
    run.sh                Agent workload entry point (the "app")
  platform/
    registry.yaml         FrameworkRegistry — operations the platform offers (illustrative)
    constraints.yaml      Platform policy guardrails (illustrative)
  demo.sh                 Run the full detect -> import -> explain demo
  README.md               This file
```

### DRY files (runtime)

- `c3agent.yaml` — Base fleet config. Detected as c3agent by structural matching
  (`service: c3agent` + `apiVersion: c3agent/v1`). This is what a team authors.
- `c3agent-prod.yaml` — Production overlay with HA replicas, higher concurrency,
  larger storage, separate credential refs.

### Platform specs (illustrative)

- `platform/registry.yaml` — A FrameworkRegistry defining 7 typed operations
  (configureFleet, deployControlPlane, deployGateway, configureAgentRuntime,
  bindCredentials, provisionStorage, grantJobPermissions). Each operation has
  typed inputs, resource outputs, constraints, and validation rules. This is
  what an IDP portal would use to render self-service forms.

- `platform/constraints.yaml` — Platform guardrails: approved models, approved
  registries, budget ceilings, HA requirements, secret hygiene, RBAC policy.

These specs are *illustrative* — they show what the platform offers as a
contract. In ConfigHub, they become governance objects. In `cub-gen` today,
the Go registry is the runtime source of truth.

### Workload

- `agent/run.sh` — The agent workload entry point. In production, this runs
  Claude Code with task parameters from the control plane.

## Running the demo

```bash
# From repo root
go build -o ./cub-gen ./cmd/cub-gen

# Option 1: Run the demo script
./examples/ai-ops-paas/demo.sh

# Option 2: Run commands individually

# Discover — finds c3agent generator with 0.92 confidence
./cub-gen gitops discover --space ai-ops --json ./examples/ai-ops-paas

# Import — produces DRY/WET with provenance
./cub-gen gitops import --space ai-ops --json ./examples/ai-ops-paas ./examples/ai-ops-paas

# See the full c3agent triple
./cub-gen generators --json --details | jq '.families[] | select(.kind == "c3agent")'
```

## What the generator produces

The c3agent generator maps DRY sections to WET targets:

| DRY Section | WET Targets | Inverse Key |
|---|---|---|
| `fleet.*` | ConfigMap (fleet-config) | fleet_config |
| `components.controlplane.*` | Deployment, Service | component_ports |
| `components.gateway.*` | Deployment, Service | component_ports |
| `agent_runtime.*` | ConfigMap (agent-template) | agent_runtime |
| `credentials.*` | Secret | credentials |
| `storage.*` | PersistentVolumeClaim | storage |
| (implicit) | ServiceAccount, ClusterRole, ClusterRoleBinding | rbac |

Each WET target includes:
- **Field-origin confidence** (0.85-0.95) — how certain we are about DRY→WET mapping
- **Inverse edit hints** — "to change X in prod, edit Y in fleet-prod.yaml"
- **Rendered lineage template** — provenance digest linking output to input

## The PaaS value proposition

Without this platform:
- Teams hand-write ~11 Kubernetes manifests per fleet
- No standard structure — every team reinvents RBAC, storage, secrets
- Prod config drifts from intent with no audit trail
- AI agent budget and model changes are invisible

With this platform:
- Teams author 30 lines of YAML (c3agent.yaml)
- Platform enforces constraints (approved models, budget caps, HA requirements)
- Every field traces back to DRY intent with provenance
- Inverse hints tell teams exactly what to edit and where

This is the "internal Heroku that doesn't hide the output" — teams get
self-service, platform gets governance, and everything is in Git.

## Pricing boundary

This demo shows both sides of the ConfigHub pricing boundary:

| Side | What | Free/Paid |
|---|---|---|
| **DRY** | c3agent.yaml, c3agent-prod.yaml, generator detection | Free (OSS, cub-gen) |
| **WET** | Units with provenance, cross-env queries, policy gates | Paid (ConfigHub) |

The generator boundary IS the pricing boundary. DRY authoring drives adoption.
WET governance drives revenue.
