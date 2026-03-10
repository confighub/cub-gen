# Operations Workflow (Governed Execution)

**Pattern: operations become configuration — scheduled maintenance workflows with approval gates, governed the same way as app deployments.**

## 1. What is this?

An operations team runs a nightly release maintenance workflow: deploy a new image version and restart the checkout service. Instead of encoding this in a CI pipeline or a cron script, they declare it as configuration in `operations.yaml`. The platform enforces scheduling policies, approval gates, and action constraints.

This is the operations model: workflows are not code, they are *governed configuration*. Every change to a workflow definition gets field-origin tracing, every execution window gets policy enforcement, and every action gets an attestation chain.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **Ops team** | `operations.yaml` — workflow name, actions, image tags | Action parameters, service targets |
| **Ops team** | `operations-prod.yaml` — production overrides | Schedule timing, prod-specific image tags |
| **Platform team** | `platform/` — execution policies | Allowed actions, scheduling windows, approval requirements |
| **GitOps reconciler** | N/A — execution is governed by ConfigHub, not cluster sync | N/A |

The key distinction: this is not GitOps in the Flux/ArgoCD sense. There are no Kubernetes manifests to reconcile. Instead, the workflow *definition* is the governed artifact, and ConfigHub's decision engine controls when and how it executes.

## 3. What does cub-gen add?

cub-gen treats operations workflows as a first-class generator:

- **Generator detection**: recognizes `operations.yaml` with `workflow:` + `actions:` structure as `ops-workflow` profile (capabilities: `workflow-plan`, `governed-execution-intent`, `inverse-workflow-patch`)
- **DRY/WET classification**: workflow definition is DRY intent, rendered execution plan is WET
- **Field-origin tracing**: action parameters, schedule triggers, and service targets all trace back to the DRY operations file
- **Inverse-edit guidance**: "to change the deploy image tag in production, edit `operations-prod.yaml` actions.deploy section"

```bash
# Discover — detects ops-workflow generator
./cub-gen gitops discover --space platform --json ./examples/ops-workflow

# Import — produces DRY/WET classification with provenance
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, provenance: .provenance[0].field_origin_map}'
```

## 4. How do I run it?

```bash
# Build
go build -o ./cub-gen ./cmd/cub-gen

# Discover
./cub-gen gitops discover --space platform ./examples/ops-workflow

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow

# Full bridge flow
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow > /tmp/ops-bundle.json
./cub-gen verify --in /tmp/ops-bundle.json
./cub-gen attest --in /tmp/ops-bundle.json --verifier ci-bot > /tmp/ops-attestation.json
./cub-gen verify-attestation --in /tmp/ops-attestation.json --bundle /tmp/ops-bundle.json

# Cleanup
./cub-gen gitops cleanup --space platform ./examples/ops-workflow
```

## 5. Real-world example using ConfigHub

An e-commerce platform runs weekly maintenance windows. The operations team manages release workflows for 12 services.

**Scenario: Changing the maintenance schedule**

The team needs to move the nightly deploy from 2 AM to 3 AM to avoid overlap with database backups. They edit `operations-prod.yaml`:

```yaml
triggers:
  schedule: "0 3 * * *"   # was "0 2 * * *"
```

**Governed pipeline:**

```bash
# 1. cub-gen detects the schedule change
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow
# Field-origin shows: triggers.schedule changed in operations-prod.yaml (ops-team owned)

# 2. Produce evidence chain
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 3. ConfigHub ingests and evaluates
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example
# Decision engine checks: new schedule is within allowed execution window → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by ops-lead --reason "avoid database backup overlap"
```

**What this prevents:**
- Someone silently changes a maintenance schedule without review
- A workflow targets a service outside the team's ownership
- An image tag gets bumped in production without attestation
- Schedule changes violate the platform's execution window policy

**What ConfigHub provides:**
- Cross-service view of all maintenance schedules
- Audit trail of every schedule and action change
- Policy enforcement at write time (not at execution time)
- Evidence chain linking every workflow change to who approved it and why

## The operations model

The operations model is the same canonical pattern used by every generator:

```
operations.yaml (DRY intent)
    |
    v
ops-workflow generator (detect → import)
    |
    v
Governed execution plan (WET)
    |
    v
publish → verify → attest → ConfigHub decision
```

What changes is the *kind* of output — instead of Kubernetes manifests, the WET artifact is an execution plan. But the governance is identical: nothing executes without an explicit ALLOW decision with attestation linkage.

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `operations.yaml` | Ops team | DRY intent — workflow name, actions, service targets |
| `operations-prod.yaml` | Ops team | Production overlay — schedule timing, prod image tags |

## Related examples

- [`confighub-actions`](../confighub-actions/) — ConfigHub lifecycle actions (plan → verify → deploy). Same operations model, different workflow shape.
- [`swamp-automation`](../swamp-automation/) — Swamp workflow automation. AI-agent-driven execution with the same governance pattern.
