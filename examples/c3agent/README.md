# C3 Agent Fleet (Standalone)

**Pattern: AI agent fleet configuration — a team authors 30 lines of YAML, the platform enforces model approval, budget caps, and credential hygiene.**

> For the full platform story with registry and constraints, see [`ai-ops-paas`](../ai-ops-paas/).

## 1. What is this?

A team runs a fleet of Claude Code agents for automated code review. They declare the fleet configuration in `c3agent.yaml`: how many tasks to run concurrently, which AI model to use, budget limits, storage, and credential references. The platform ensures approved models, budget ceilings, and HA requirements are met.

This is the standalone version of the c3agent pattern. It shows the minimal config a team authors to get a governed agent fleet. The [`ai-ops-paas`](../ai-ops-paas/) example shows the full platform with registry and constraints.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **App team** | `c3agent.yaml` — fleet config, agent runtime settings | Concurrency, model, budget, credential refs |
| **App team** | `c3agent-prod.yaml` — production overlay | HA replicas, higher budget, prod credentials |
| **Platform team** | `platform/` — fleet policies (future) | Approved models, budget ceilings, credential rotation |
| **GitOps reconciler** | Flux/ArgoCD syncs rendered Kubernetes manifests | Reconciles WET to LIVE |

## 3. What does cub-gen add?

The c3agent generator maps DRY fleet config to governed Kubernetes resources:

- **Generator detection**: recognizes `c3agent.yaml` with `service: c3agent` + `apiVersion: c3agent/v1` (capabilities: `fleet-config`, `agent-orchestration`, `inverse-fleet-config-patch`)
- **DRY/WET mapping**: 30 lines of DRY config → 11 WET Kubernetes targets (Deployments, Services, ConfigMaps, Secret, PVC, RBAC)
- **Field-origin tracing**: `fleet.agent_model` traces to `c3agent.yaml`, overridden by `c3agent-prod.yaml` in production
- **Inverse-edit guidance**: "to change the budget in production, edit `c3agent-prod.yaml` agent_runtime.max_budget_usd"

```bash
# Discover — detects c3agent with high confidence
./cub-gen gitops discover --space platform --json ./examples/c3agent

# Import — produces DRY/WET classification with provenance
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

## 4. How do I run it?

```bash
# Build
go build -o ./cub-gen ./cmd/cub-gen

# Discover
./cub-gen gitops discover --space platform ./examples/c3agent

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent

# Full bridge flow
./cub-gen publish --space platform ./examples/c3agent ./examples/c3agent > /tmp/c3-bundle.json
./cub-gen verify --in /tmp/c3-bundle.json
./cub-gen attest --in /tmp/c3-bundle.json --verifier ci-bot > /tmp/c3-attestation.json
./cub-gen verify-attestation --in /tmp/c3-attestation.json --bundle /tmp/c3-bundle.json

# Cleanup
./cub-gen gitops cleanup --space platform ./examples/c3agent
```

## 5. Real-world example using ConfigHub

A fintech company runs 5 C3 agent fleets across different teams: code review, test generation, documentation, security scanning, and incident response.

**Scenario: Upgrading the AI model across all fleets**

Anthropic releases a new model version. The platform team needs to approve the upgrade and roll it out fleet by fleet.

```bash
# 1. Team updates their fleet config
# c3agent.yaml: fleet.agent_model: claude-sonnet-4-20250514 → claude-sonnet-4.5-20260101

# 2. cub-gen detects the model change
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent
# Field-origin: fleet.agent_model changed in c3agent.yaml

# 3. Produce evidence chain
./cub-gen publish --space platform ./examples/c3agent ./examples/c3agent > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 4. ConfigHub ingests and evaluates
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example
# Platform policy checks: is claude-sonnet-4.5-20260101 in the approved models list?
# If approved → ALLOW. If not yet approved → ESCALATE to platform-owner.
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-owner --reason "model upgrade approved after eval"
```

**What ConfigHub provides:**
- **Model governance**: "which fleets are running which models?" — cross-repo query
- **Budget visibility**: aggregate budget across all fleets
- **Credential audit**: "which fleets reference prod credentials?" — provenance index
- **Rollout tracking**: decision history shows which fleets have upgraded vs. pending

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `c3agent.yaml` | App team | DRY fleet config — identity, model, concurrency, components, storage, credentials |
| `c3agent-prod.yaml` | App team | Production overlay — HA replicas, higher budget, prod credential refs |

## Related examples

- [`ai-ops-paas`](../ai-ops-paas/) — Full platform version with FrameworkRegistry and platform constraints. Shows the IDP portal perspective.
