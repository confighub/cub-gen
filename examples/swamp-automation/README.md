# Swamp Automation — Governed AI Workflow Orchestration

Your AI-agent-driven workflows — validate, deploy, health-check, rotate
credentials — run through [Swamp](https://github.com/systeminit/swamp), a
Git-native workflow engine with typed models of external systems. ConfigHub
adds the governance layer: every workflow definition change gets provenance,
every model binding gets policy enforcement, and every execution gets an
audit trail.

Together they form a complete agentic app platform: ConfigHub declares *what
should exist* (governance), Swamp executes *how to make it so* (automation),
and cub-gen connects the two with traceable change bundles.

> For deploying the Swamp runtime itself on Kubernetes, see
> [`swamp-project`](../swamp-project/).

## What you get

- **Workflow-as-config governance**: DAG steps, model bindings, and vault
  config are traced with full provenance
- **Model binding policy**: platform controls which models (app-validator,
  app-deployer, app-healthcheck) are approved
- **Execution windows**: production workflows restricted to business hours
- **Vault policy**: encryption key rotation and local-encryption enforcement

## How Swamp workflows map to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              SWAMP (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ .swamp.yaml         │          │ Workflow manifest    │         │ DAG execution    │
│ workflow-deploy.yaml │──import─▶│ Model bindings       │──exec──▶│ Model calls      │
│ platform/swamp-     │          │ Vault config         │         │ Infrastructure   │
│   constraints.yaml  │          │ with provenance      │         │   mutations      │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  App team: workflow definitions.  Governed workflow manifest        What Swamp
  Platform: model + vault policy.  with field-origin tracing.       actually runs.
```

**DRY** is what teams author: `workflow-deploy.yaml` defines the DAG — job
steps, model method bindings, execution order. `.swamp.yaml` configures the
Swamp runtime (vault, logging, version). Platform constraints define approved
models and execution windows.

**WET** is what cub-gen traces: a structured workflow manifest with every step,
model binding, and configuration traced back to its DRY source.

**LIVE** is what Swamp executes against external systems — cloud APIs, clusters,
managed services.

| File | Owner | What it controls |
|------|-------|-----------------|
| `.swamp.yaml` | Platform | Swamp repo config — vault type, encryption, logging |
| `workflow-deploy.yaml` | App team | Deployment workflow — validate → deploy steps |
| `platform/swamp-constraints.yaml` | Platform | Approved models, execution windows, vault policy |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect Swamp workflow
./cub-gen gitops discover --space platform --json ./examples/swamp-automation

# Import with field-origin tracing
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

cub-gen detects `.swamp.yaml` as a Swamp source and collects `workflow-*.yaml`
files as workflow definitions. The import traces every job step and model
binding back to its DRY source.

## Real-world scenario: adding a pre-deploy health check

**Who**: An SRE team at a fintech company using Swamp for infrastructure
automation. They have workflows for deploying services, rotating credentials,
and scaling compute.

### The change — new health check step

```yaml
# workflow-deploy.yaml — add healthcheck before validate
jobs:
  - name: deploy-flow
    steps:
      - name: healthcheck           # new step
        task:
          type: model_method
          modelIdOrName: app-healthcheck
          methodName: verify
      - name: validate
        task:
          type: model_method
          modelIdOrName: app-validator
          methodName: check
      - name: apply
        task:
          type: model_method
          modelIdOrName: app-deployer
          methodName: apply
```

### Governed pipeline

```bash
# cub-gen detects the workflow change
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation

# Evidence chain
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Platform checks: is app-healthcheck in approved models? → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by sre-lead --reason "added pre-deploy health check"
```

The platform's `swamp-constraints.yaml` checks: is `app-healthcheck` in the
approved models list? Is the required `validate` step still present? Both
pass → **ALLOW**.

If someone tried to add an unapproved model or remove the required validate
step, the decision engine would **BLOCK**.

## How it works

cub-gen's `swamp` generator detects `.swamp.yaml` and collects sibling files
matching the `workflow-*` prefix. On import:

1. **Classifies inputs** — `.swamp.yaml` (role: runtime-config),
   `workflow-deploy.yaml` (role: workflow-definition)
2. **Maps field origins** — job names, step tasks, and model bindings trace
   to their workflow definition file
3. **Validates constraints** — approved models, execution windows, required
   steps, vault policy
4. **Emits inverse guidance** — "to change the deployer model, edit
   `workflow-deploy.yaml` steps section"

## The ConfigHub + Swamp stack

```
ConfigHub (governance)     ←→    cub-gen (bridge)    ←→    Swamp (execution)
  Decision state                   DRY/WET tracing            AI-agent workflows
  Provenance index                 Change bundles              Typed model calls
  Policy enforcement               Verification               DAG orchestration
  Audit + attestation                                          Git-native state
```

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `.swamp.yaml` | Platform | Swamp config — vault, logging, version |
| `workflow-deploy.yaml` | App team | Deployment workflow — validate → deploy |
| `platform/swamp-constraints.yaml` | Platform | Approved models, windows, vault policy |

## The complete Swamp + ConfigHub picture

| Layer | Example | Generator | What it governs |
|-------|---------|-----------|-----------------|
| **Workflow** | `swamp-automation` (this) | `swamp` | Workflow definitions, model bindings |
| **Runtime** | [`swamp-project`](../swamp-project/) | `helm-paas` | Swamp engine deployment on K8s |

## Next steps

- **Swamp runtime deployment**: [`swamp-project`](../swamp-project/) — Helm
  chart deploying the Swamp engine
- **Operations workflows**: [`ops-workflow`](../ops-workflow/) — same governed
  operations pattern, different engine
- **AI agent fleets**: [`c3agent`](../c3agent/) — standalone fleet config
- **E2E demo**: `../demo/ai-work-platform/scenario-2-swamp.sh`

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline"
./examples/swamp-automation/demo-local.sh

echo "connected (requires ConfigHub auth)"
cub auth login
./examples/swamp-automation/demo-connected.sh
```
