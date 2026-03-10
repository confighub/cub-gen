# Operations Workflow — Governed Execution for SRE Teams

Your maintenance workflows run on schedules: deploy a new image at 2 AM,
restart a service, scale replicas for a traffic event. Today these live in
CI pipelines, cron jobs, or Slack runbooks — ungoverned, unaudited, and
invisible to the platform.

ConfigHub treats operations workflows as *configuration*, not code. Every
workflow definition gets the same field-origin tracing, ownership mapping,
and decision pipeline as a Helm chart. Nothing executes without an explicit
ALLOW decision.

## What you get

- **Workflow-as-config governance**: schedule changes, action parameters, and
  service targets are traced with full provenance
- **Execution policy enforcement**: allowed actions, scheduling windows,
  approval gates — enforced at write time, not execution time
- **Production safety**: blocked actions (destroy, force-restart) are rejected
  by policy before they can run
- **Audit trail**: every workflow change links to who proposed it, who approved
  it, and what evidence was evaluated

## How operations workflows map to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              EXECUTION (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ operations.yaml     │          │ Execution plan       │         │ Scheduled jobs   │
│ operations-prod     │──import─▶│ Action manifest      │──exec──▶│ Running deploys  │
│ platform/execution- │          │ with provenance      │         │ Service restarts │
│   policy.yaml       │          │                      │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  Ops team: workflow definition.   Governed execution plan           What actually
  Platform: execution policy.      with field-origin tracing.        ran.
```

**DRY** is what the ops team writes: `operations.yaml` declares the workflow —
name, trigger schedule, actions (deploy, restart, scale). `operations-prod.yaml`
overrides for production.

**WET** is what cub-gen produces: a structured execution plan with every action,
parameter, and schedule traced back to its DRY source.

**LIVE** is what actually executes. This is *not* Kubernetes reconciliation —
there are no manifests to sync. Instead, the execution plan is the governed
artifact, and ConfigHub's decision engine controls when and how it runs.

| File | Owner | What it controls |
|------|-------|-----------------|
| `operations.yaml` | Ops team | Workflow name, actions (deploy, restart), schedule trigger |
| `operations-prod.yaml` | Ops team | Prod overlay — schedule timing, prod image tags |
| `platform/execution-policy.yaml` | Platform | Allowed/blocked actions, scheduling windows, approval gates |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect operations workflow
./cub-gen gitops discover --space platform --json ./examples/ops-workflow

# Import with field-origin tracing
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

cub-gen detects `operations.yaml` with `actions:` structure and classifies it
as `ops-workflow`. The import traces every action parameter, schedule trigger,
and service target back to its DRY source.

## Real-world scenario: changing the maintenance schedule

**Who**: An e-commerce ops team managing release workflows for 12 services.
Nightly deploys run at 2 AM but overlap with database backups.

### The change

```yaml
# operations-prod.yaml — move maintenance window
triggers:
  schedule: "0 3 * * *"   # was "0 2 * * *"
```

### Governed pipeline

```bash
# cub-gen detects the schedule change
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow

# Evidence chain
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# New schedule is within allowed window → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by ops-lead --reason "avoid database backup overlap"
```

The execution policy checks: is the new schedule within the allowed window?
Are all actions (deploy, restart) in the allowed list? Does production require
approval? All pass → **ALLOW**.

If someone tried to add a `destroy` action, the execution policy would
**BLOCK** it — `destroy` is in the blocked actions list.

## How it works

cub-gen's `ops-workflow` generator detects `operations.yaml` containing
`actions:` or `workflow:` structure. On import:

1. **Classifies inputs** — `operations.yaml` (role: workflow-definition),
   `operations-prod.yaml` (role: workflow-overlay)
2. **Maps field origins** — action types, schedule triggers, and image tags
   trace to their source file with ownership metadata
3. **Validates execution policy** — allowed/blocked actions, scheduling windows,
   and approval thresholds
4. **Emits inverse guidance** — "to change the deploy image tag in production,
   edit `operations-prod.yaml` actions.deploy section"

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `operations.yaml` | Ops team | Workflow definition — actions, schedule, targets |
| `operations-prod.yaml` | Ops team | Prod overlay — schedule timing, prod image tags |
| `platform/execution-policy.yaml` | Platform | Allowed/blocked actions, windows, approvals |

## Next steps

- **ConfigHub lifecycle**: [`confighub-actions`](../confighub-actions/) — same
  operations model for ConfigHub's own release lifecycle
- **AI workflow governance**: [`swamp-automation`](../swamp-automation/) —
  AI-agent-driven workflows with model binding governance
- **E2E demo**: `../demo/ai-work-platform/scenario-4-operations.sh`

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline"
./examples/ops-workflow/demo-local.sh

echo "connected (requires ConfigHub auth)"
cub auth login
./examples/ops-workflow/demo-connected.sh
```
