# ConfigHub + Swamp: Agentic App Platform

**Pattern: AI-native workflow automation with governed configuration — ConfigHub provides the governance plane, [Swamp](https://github.com/systeminit/swamp) provides the execution plane.**

> See also: [`swamp-project`](../swamp-project/) — the Helm chart that deploys the Swamp runtime itself (uses `helm-paas` generator).

## 1. What is this?

An infrastructure team uses Swamp to automate cloud provisioning workflows. Swamp provides AI-agent-driven, Git-native workflow orchestration — typed models of external systems, DAG-based job execution, and structured secrets management. ConfigHub provides the governance layer: every workflow definition change gets field-origin tracing, every deployment gets a governed decision, and every credential reference gets an audit trail.

Together they form a complete agentic app platform:
- **ConfigHub** declares *what should exist* (the governance plane)
- **Swamp** executes *how to make it so* (the execution plane)
- **cub-gen** connects the two with provenance and change bundles

This example shows a deployment workflow with two steps: validate the app, then deploy it. The Swamp generator detects `.swamp.yaml` + workflow definitions and maps them to governed configuration units.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **App team** | `workflow-deploy.yaml` — workflow steps, model references | Job names, step ordering, model method bindings |
| **Platform team** | `.swamp.yaml` — vault config, log level, version | Vault type, encryption key refs, Swamp version |
| **Platform team** | `platform/` — workflow policies (future) | Approved models, execution windows, budget limits |
| **GitOps reconciler** | N/A — Swamp executes locally against external systems | N/A |

The key insight: Swamp workflows are *configuration*, not code. A workflow YAML file declaring "validate then deploy" is a governed artifact — it should have provenance, inverse-edit guidance, and a change bundle, just like a Helm values file.

## 3. What does cub-gen add?

cub-gen treats Swamp workflow definitions the same as any other generator source:

- **Generator detection**: recognizes `.swamp.yaml` as a swamp source (capabilities: `workflow-automation`, `model-orchestration`, `inverse-workflow-patch`)
- **DRY/WET classification**: `.swamp.yaml` is platform DRY (vault config), `workflow-deploy.yaml` is app DRY (workflow intent)
- **Field-origin tracing**: job names, step tasks, model references all trace back to the DRY workflow file
- **Inverse-edit guidance**: "to change the deployer model in production, edit `workflow-deploy.yaml` steps section"
- **Change bundles**: same `publish → verify → attest` pipeline as every other generator

```bash
# Discover — detects swamp generator
./cub-gen gitops discover --space platform --json ./examples/swamp-automation

# Import — produces DRY/WET classification with provenance
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, provenance: .provenance[0].field_origin_map}'
```

## 4. How do I run it?

```bash
# Build
go build -o ./cub-gen ./cmd/cub-gen

# Discover
./cub-gen gitops discover --space platform ./examples/swamp-automation

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation

# Full bridge flow
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation > /tmp/swamp-bundle.json
./cub-gen verify --in /tmp/swamp-bundle.json
./cub-gen attest --in /tmp/swamp-bundle.json --verifier ci-bot > /tmp/swamp-attestation.json
./cub-gen verify-attestation --in /tmp/swamp-attestation.json --bundle /tmp/swamp-bundle.json

# Cleanup
./cub-gen gitops cleanup --space platform ./examples/swamp-automation
```

## 5. Real-world example using ConfigHub

An SRE team at a fintech company uses Swamp to automate infrastructure provisioning. They have workflows for deploying services, rotating credentials, and scaling compute.

**Scenario: Adding a pre-deploy health check**

The team wants to add a health check step before deploying. They edit `workflow-deploy.yaml`:

```yaml
jobs:
  - name: deploy-flow
    steps:
      - name: healthcheck        # new step
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

**Governed pipeline:**

```bash
# 1. cub-gen detects the workflow change
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation

# 2. Produce evidence chain
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 3. ConfigHub ingests and evaluates
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by sre-lead --reason "added pre-deploy health check"
```

**What ConfigHub provides:**
- **Audit trail**: every workflow definition change is recorded with provenance
- **Decision authority**: workflow changes to production require explicit ALLOW
- **Cross-repo visibility**: "which workflows reference the `app-deployer` model?" — answerable from ConfigHub's provenance index
- **Drift detection**: if someone edits a workflow outside the governed pipeline, ConfigHub flags the drift

## The ConfigHub + Swamp stack

```
┌─────────────────────────────────────────┐
│  ConfigHub (governance plane)            │
│  - Governed decision state               │
│  - Provenance index                      │
│  - Policy enforcement                    │
│  - Audit trail + attestation             │
├─────────────────────────────────────────┤
│  cub-gen (bridge)                        │
│  - DRY/WET classification                │
│  - Field-origin tracing                  │
│  - Change bundles + verification         │
├─────────────────────────────────────────┤
│  Swamp (execution plane)                 │
│  - AI-agent-driven workflows             │
│  - Typed models of external systems      │
│  - DAG-based job orchestration           │
│  - Git-native state (.swamp/ directory)  │
├─────────────────────────────────────────┤
│  Infrastructure                          │
│  - Cloud APIs (AWS, GCP, Azure)          │
│  - Kubernetes clusters                   │
│  - Managed services                      │
└─────────────────────────────────────────┘
```

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `.swamp.yaml` | Platform team | Swamp repo config — vault, logging, version |
| `workflow-deploy.yaml` | App team | Deployment workflow — validate → deploy steps |

## Related examples

- [`swamp-project`](../swamp-project/) — Helm chart deploying the Swamp runtime (uses `helm-paas` generator). Shows the infrastructure side: how you deploy Swamp itself on Kubernetes with governed configuration.
- [`ops-workflow`](../ops-workflow/) — Operations workflow using the `ops-workflow` generator. Shows the same governed-operations pattern with a different workflow engine.
