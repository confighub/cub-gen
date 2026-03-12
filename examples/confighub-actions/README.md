# ConfigHub Actions — Recursive Governance

ConfigHub's own release lifecycle — plan → verify → deploy — is expressed as
governed operations configuration. When a commit lands, three actions execute
in sequence: a dry-run plan, a policy check, and a governed apply. In production,
the verify step requires two approvals and deploys are restricted to business
hours.

This is governance governing itself. The lifecycle definition is configuration,
and configuration gets governed. The same provenance, ownership, and decision
pipeline that governs your app changes also governs the governance rules.

## Domain POV (platform control-plane operators)

This example is for teams operating policy-driven platform lifecycles:

- release/verification/apply stages are already encoded as workflow config,
- production controls (approvals, windows, break-glass) evolve frequently,
- governance changes themselves need auditable governance.

The first value is recursive safety: lifecycle policy edits are treated with the
same rigor as app/runtime changes.

## What you get

- **Recursive governance**: changes to lifecycle rules go through the same
  ALLOW/ESCALATE/BLOCK pipeline as app changes
- **Action sequencing**: plan → verify → deploy is enforced by policy, not
  convention
- **Production safety**: deploy windows, approval thresholds, and trigger
  policies are platform-owned configuration
- **Break-glass override**: emergency changes bypass approval thresholds and
  deploy windows, with mandatory post-incident retrospective

## How ConfigHub Actions map to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              EXECUTION (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ operations.yaml     │          │ Lifecycle plan       │         │ plan (dry-run)   │
│ operations-prod     │──import─▶│ Action manifest      │──exec──▶│ verify (check)   │
│ platform/lifecycle- │          │ with provenance      │         │ deploy (apply)   │
│   policy.yaml       │          │                      │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  Platform: lifecycle actions.     Governed execution plan           What actually
  Platform: lifecycle policy.      with field-origin tracing.        ran.
```

**DRY** is what the platform team authors: `operations.yaml` defines the action
types and trigger (commit-driven). `operations-prod.yaml` adds production
approval requirements and deploy windows.

**WET** is what cub-gen traces: a structured lifecycle plan with every action,
approval threshold, and deploy window traced back to its DRY source.

**LIVE** is what actually executes within ConfigHub's decision engine.

| File | Owner | What it controls |
|------|-------|-----------------|
| `operations.yaml` | Platform | Action types (dry-run, policy-check, apply), commit trigger |
| `operations-prod.yaml` | Platform | Prod overlay — approval count, deploy window |
| `platform/lifecycle-policy.yaml` | Platform | Required actions, sequence, approvals, windows, break-glass |

## If you already automate platform lifecycles

This example is for teams running policy-driven operational pipelines already:

- You model lifecycle stages (verify, policy-check, apply) as explicit steps.
- You frequently adjust approval thresholds and deployment windows.
- You need auditable proof that governance changes were themselves governed.

cub-gen keeps your operations model intact and adds deterministic provenance for
every lifecycle field so recursive governance is practical, not theoretical.

## Why this maps cleanly to the cub-gen framework

| Existing lifecycle model | cub-gen concept | Why it matters |
|------|------|------|
| `operations*.yaml` workflow definitions | DRY operational intent | Teams keep editing explicit workflow config. |
| Execution plan/action manifest | WET governed plan | Each action/approval field is traceable by source and owner. |
| Lifecycle policy constraints | Governance layer | Policy changes can be reviewed with explicit ALLOW/BLOCK outcomes. |
| ConfigHub action execution | LIVE workflow run | Runtime execution remains in ConfigHub decision engine. |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect ConfigHub Actions lifecycle
./cub-gen gitops discover --space platform --json ./examples/confighub-actions

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/confighub-actions ./examples/confighub-actions \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

cub-gen detects `operations.yaml` and classifies it as `ops-workflow` — the
same generator used for [`ops-workflow`](../ops-workflow/). The ConfigHub
Actions lifecycle is just a specific workflow shape.

## Real-world scenario: tightening production approvals

**Who**: A platform team managing 30 microservices through ConfigHub. The
security team requires three approvals for production deploys (up from two).

### The change

```yaml
# operations-prod.yaml — tighten approvals
actions:
  verify:
    approvals:
      required: 3   # was 2
```

### Governed pipeline — governance governs itself

```bash
# cub-gen detects the approval threshold change
./cub-gen gitops import --space platform --json ./examples/confighub-actions ./examples/confighub-actions

# Evidence chain
./cub-gen publish --space platform ./examples/confighub-actions ./examples/confighub-actions > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub — the lifecycle change is itself governed
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by security-lead --reason "SOC2 compliance: 3 approvals for prod"
```

The recursive loop: ConfigHub Actions define the lifecycle (plan → verify →
deploy). Changes to the lifecycle are *themselves* governed through ConfigHub.
The decision engine evaluates the lifecycle change using the current rules.

### Break-glass: emergency override

During an incident, the platform owner can bypass approval thresholds and
deploy windows:

```yaml
# platform/lifecycle-policy.yaml — break-glass section
emergency_override:
  enabled: true
  requires: [platform-owner, justification, incident-ticket]
  bypasses: [approval_thresholds, deploy_windows]
  post_actions: [notify-security-lead, create-incident, require-retrospective]
```

The override requires a platform-owner, a justification, and an incident ticket.
After the emergency, mandatory post-actions ensure a security notification,
incident record, and retrospective.

## How it works

This example uses the `ops-workflow` generator — the same generator used by
[`ops-workflow`](../ops-workflow/). What makes it distinctive:

1. **Lifecycle-specific policy** — the `LifecyclePolicy` kind defines required
   actions (dry-run, policy-check), action sequencing, approval thresholds,
   deploy windows, trigger policies, and break-glass overrides
2. **Recursive governance** — changes to governance rules are governed by the
   same rules
3. **Break-glass path** — emergency overrides are explicit, audited, and require
   post-incident actions

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `operations.yaml` | Platform | Action types — dry-run, policy-check, apply |
| `operations-prod.yaml` | Platform | Prod overlay — approval count, deploy window |
| `platform/lifecycle-policy.yaml` | Platform | Required actions, sequence, approvals, break-glass |

## Next steps

- **Operations workflows**: [`ops-workflow`](../ops-workflow/) — scheduled
  maintenance workflows with the same governance model
- **AI workflow governance**: [`swamp-automation`](../swamp-automation/) —
  AI-agent-driven workflows
- **E2E demo**: `../demo/ai-work-platform/scenario-3-confighub-actions.sh`

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline"
./examples/confighub-actions/demo-local.sh

echo "connected (requires ConfigHub auth)"
cub auth login
./examples/confighub-actions/demo-connected.sh
```
