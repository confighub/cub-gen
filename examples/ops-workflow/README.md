# Operations Workflow — Governed Execution for SRE Teams

Your maintenance workflows run on schedules: deploy a new image at 2 AM,
restart a service, scale replicas for a traffic event. Today these live in
CI pipelines, cron jobs, or Slack runbooks — ungoverned, unaudited, and
invisible to the platform.

ConfigHub treats operations workflows as *configuration*, not code. Every
workflow definition gets the same field-origin tracing, ownership mapping,
and decision pipeline as a Helm chart. Nothing executes without an explicit
ALLOW decision.

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding ops workflow governance | Jump to [Run from ConfigHub](#run-from-confighub-connected-mode) |
| **Existing ops/SRE team** adding ConfigHub | Jump to [Try it](#try-it) then connect later |

Both paths lead to the same outcome: governed operational workflows with
field-origin tracing and execution policy enforcement.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real workflow** | Operations workflow with deploy, restart, and scale actions |
| **Real policy** | Execution policy with allowed/blocked actions and scheduling windows |
| **Real inspection target** | `operations.yaml` and `operations-prod.yaml` with governance decisions |
| **Execution transport** | Scheduled jobs, CI pipelines, or manual triggers |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "Who changed the maintenance schedule?" | Field-origin tracing to `operations-prod.yaml` | Schedule changes → ALLOW within window |
| "Is this action allowed in production?" | Execution policy enforcement | Blocked actions (destroy) → BLOCK |
| "What happened during the incident?" | Attestable evidence chain | Full audit trail for post-incident review |

## Domain POV (SRE and operations workflow teams)

This example fits teams that already run scheduled/triggered operational
actions (deploy, restart, scale) through scripts, CI jobs, or runbooks:

- changes to schedules/actions are high impact,
- approvals and safe windows are often tribal knowledge,
- post-incident review needs traceable "who changed what and why."

The first value is operational clarity: workflows are governed config artifacts,
not opaque automation scripts.

## AI prompt-as-DRY for operations workflows

Operations workflows are a natural fit for the AI prompt-as-DRY pattern:

| Concept | How it maps to ops workflows |
|---------|------------------------------|
| **Prompt + context** | "Deploy checkout-service at 3 AM, restart after backup completes" |
| **LLM/agent layer** | Agent compiles prompt into `operations.yaml` structure |
| **Verification + attestation** | cub-gen verifies structure matches execution policy |
| **Mutation ledger** | ConfigHub records who proposed, who approved, what evidence |

Example agent-assisted workflow change:

```text
Human prompt: "Move the nightly deploy from 2 AM to 3 AM to avoid backup overlap"
Agent output: operations-prod.yaml with triggers.schedule: "0 3 * * *"
Governance: cub-gen imports, verifies window compliance, ConfigHub records decision
```

See also: [Prompt as DRY](../../docs/workflows/prompt-as-dry.md)

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
│ platform/registry   │          │ with provenance      │         │ Service restarts │
│ platform/execution- │          │                      │         │                 │
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
| `platform/registry.yaml` | Platform | Typed operation contracts for workflow APIs and portals |
| `platform/execution-policy.yaml` | Platform | Allowed/blocked actions, scheduling windows, approval gates |

## If you already run operational workflows at scale

This example is aimed at SRE and operations teams that already manage scheduled
deploy/restart/maintenance workflows:

- Workflow YAML defines intent, but ownership and safety boundaries are implicit.
- Schedule or action changes can have broad operational impact.
- Post-incident review needs a clear source-of-truth for "who changed what".

cub-gen keeps the operations workflow interface and adds explicit provenance,
decision-state gating, and attestable evidence for each operational mutation.

## Why this maps cleanly to the cub-gen framework

| Existing ops workflow model | cub-gen concept | Why it matters |
|------|------|------|
| `operations*.yaml` | DRY operational intent | Ops teams keep authoring high-level workflow steps. |
| `platform/registry.yaml` | Operation registry contract | Portals/agents can discover typed operations instead of guessing fields. |
| Execution plan/action manifest | WET governed output | Schedules and actions become traceable and reviewable. |
| Execution policy | Governance layer | Risky schedule/action changes can be blocked or escalated. |
| Actual job execution | LIVE state | Runtime execution remains separate while governance becomes explicit. |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect operations workflow
./cub-gen gitops discover --space platform --json ./examples/ops-workflow

# Import with field-origin tracing
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'

# Inspect structural ops workflow analysis
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '.provenance[0].ops_workflow_analysis'
```

cub-gen detects `operations.yaml` with `actions:` structure and classifies it
as `ops-workflow`. The import traces every action parameter, schedule trigger,
and service target back to its DRY source.

## Real-world scenario: changing the maintenance schedule

**Who**: An e-commerce ops team managing release workflows for 12 services.
Nightly deploys run at 2 AM but overlap with database backups.

### Scenario A — Schedule change within allowed window (ALLOW)

```yaml
# operations-prod.yaml — move maintenance window
triggers:
  schedule: "0 3 * * *"   # was "0 2 * * *"
```

```bash
# cub-gen detects the schedule change
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow

# Evidence chain
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# New schedule is within allowed window → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by ops-lead --reason "avoid database backup overlap"
```

The execution policy checks: is the new schedule within the allowed window?
Are all actions (deploy, restart) in the allowed list? Does production require
approval? All pass → **ALLOW**.

### Scenario B — Blocked action (BLOCK)

Someone tries to add a `destroy` action to the workflow:

```yaml
# operations-prod.yaml — dangerous action
actions:
  deploy:
    image_tag: v1.2.4
  destroy:           # blocked by execution policy!
    service: checkout-api
```

```bash
# cub-gen detects the action change
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow

# Evidence chain
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow > bundle.json
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# destroy action is in blocked list → BLOCK
./cub-gen bridge decision apply --decision decision.json --state BLOCK \
  --approved-by governance-bot \
  --reason "Action 'destroy' is in the blocked actions list. Requires platform-ops escalation."
```

The execution policy sees `destroy` in the blocked actions list → **BLOCK**.
The workflow cannot execute until the action is removed or escalated.

## How it works

cub-gen's `ops-workflow` generator detects `operations.yaml` containing
`actions:` or `workflow:` structure. On import:

1. **Classifies inputs** — `operations.yaml` (role: workflow-definition),
   `operations-prod.yaml` (role: workflow-overlay)
2. **Loads operation contracts** — `platform/registry.yaml` defines typed
   operations like `scheduleWorkflow`, `deployServiceAction`, and
   `rollbackServiceAction`
3. **Maps field origins** — action types, schedule triggers, and image tags
   trace to their source file with ownership metadata
4. **Validates execution policy** — allowed/blocked actions, scheduling windows,
   and approval thresholds
5. **Emits inverse guidance** — "to change the deploy image tag in production,
   edit `operations-prod.yaml` actions.deploy section"

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `operations.yaml` | Ops team | Workflow definition — actions, schedule, targets |
| `operations-prod.yaml` | Ops team | Prod overlay — schedule timing, prod image tags |
| `platform/registry.yaml` | Platform | FrameworkRegistry v1 operation contracts for ops workflows |
| `platform/execution-policy.yaml` | Platform | Allowed/blocked actions, windows, approvals |

## Next steps

- **ConfigHub lifecycle**: [`confighub-actions`](../confighub-actions/) — same
  operations model for ConfigHub's own release lifecycle
- **AI workflow governance**: [`swamp-automation`](../swamp-automation/) —
  AI-agent-driven workflows with model binding governance
- **E2E demo**: `../demo/ai-work-platform/scenario-4-operations.sh`
- **Prompt-as-DRY demo**: `../demo/prompt-as-dry-local.sh`

### PR-MR pairing and promotion flows

- **Flow A (Git PR → ConfigHub MR)**: `../demo/flow-a-git-pr-to-mr-connected.sh`
  — ops team opens PR, ConfigHub creates MR with evidence
- **Flow B (ConfigHub MR → Git PR)**: `../demo/flow-b-mr-to-git-pr-connected.sh`
  — ConfigHub initiates change, generates Git PR after approval
- **FR8 promotion**: `../demo/fr8-promotion-upstream-dry-connected.sh`
  — promote successful workflow change to upstream platform base

## Run from ConfigHub (connected mode)

If you already have ConfigHub, start here:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"

# Publish and ingest
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '.provenance[0].field_origin_map'

# Ops workflow analysis
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '.provenance[0].ops_workflow_analysis'

# Evidence bundle
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

## 7. Try one governed change

**ALLOW path**: Ops team changes schedule within allowed window:

```yaml
# operations-prod.yaml change
triggers:
  schedule: "0 3 * * *"  # was "0 2 * * *"
```

Result: Schedule is within allowed window → **ALLOW**

**BLOCK path**: Ops team adds blocked action:

```yaml
# operations-prod.yaml change
actions:
  deploy:
    image_tag: v1.2.4
  destroy:          # blocked action
    service: checkout-api
```

Result: `destroy` is in blocked actions list → **BLOCK**

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/ops-workflow/demo-local.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/ops-workflow/demo-connected.sh
```
