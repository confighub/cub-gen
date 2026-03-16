# Swamp Automation — Governed Changes for Agent-Written Workflows

AI agents compose workflows from typed models, then run those workflows against
real systems. This is powerful — and risky. What changed in the workflow graph?
Which models/methods were introduced? Were required safety steps removed?

ConfigHub + cub-gen adds governed oversight to agent-written workflows without
slowing down the agent iteration loop.

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding agent workflow governance | Jump to [Run from ConfigHub](#run-from-confighub-connected-mode) |
| **Existing Swamp/workflow team** adding ConfigHub | Jump to [Try it](#try-it) then connect later |

Both paths lead to the same outcome: governed agent workflows with structural
change classification and policy enforcement.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real workflow** | Swamp workflow with model-method tasks (validate, apply, verify) |
| **Real constraints** | Platform constraints for approved models/methods and required steps |
| **Real inspection target** | `workflow-deploy.yaml` with governed decisions |
| **Execution transport** | Swamp runtime executes workflow steps against infrastructure |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "What did the agent change in the workflow?" | Structural workflow diffing | New model/method → ESCALATE |
| "Was a required safety step removed?" | Required-step policy checks | Missing `validate` step → BLOCK |
| "Is this model/method approved?" | Approved model-method policy | Unapproved model → BLOCK |

## Domain POV (Swamp/workflow maintainers)

This example is tuned for local-first workflow teams:

- AI agents compose workflow steps from typed models,
- teams review workflow diffs in Git,
- safety hinges on structural checks (models/methods/required steps), not
  Helm-style template rendering.

The first value is structural governance: classify workflow mutations quickly
and gate risky changes before execution.

## AI prompt-as-DRY for agent workflows

Swamp is the canonical example of the AI prompt-as-DRY pattern. The LLM/agent
behaves like a non-deterministic generator:

| Concept | How it maps to Swamp |
|---------|----------------------|
| **Prompt + context** | "Deploy checkout-service to staging. Validate first. Use approved models only." |
| **LLM/agent layer** | Agent compiles prompt into `workflow-deploy.yaml` with model-method steps |
| **Verification + attestation** | cub-gen verifies structure, checks approved models, validates required steps |
| **Mutation ledger** | ConfigHub records who proposed, who approved, what evidence |

Example agent-assisted workflow composition:

```text
Human prompt: "Add a pre-deploy health check using the app-healthcheck model"
Agent output: workflow-deploy.yaml with new step using app-healthcheck.verify
Governance: cub-gen imports, checks model is approved, ConfigHub records decision
```

The key insight: verification, attestation, and governance make non-deterministic
AI output safe for production. The agent iterates fast; governance catches mistakes.

See also: [Prompt as DRY](../../docs/workflows/prompt-as-dry.md)

## What you get

- **Structural workflow diffing**: step graph and model-method changes are visible in one governed bundle.
- **Policy-ready metadata**: approved model/method checks, required-step checks, vault safety checks.
- **Fast local loop**: run `import -> publish -> verify -> attest` locally before any backend call.
- **Optional connected reporting**: send bundles to ConfigHub for centralized audit/search.

## How Swamp maps to DRY / WET / LIVE

```
  AUTHORING (DRY)                  GOVERNED BUNDLE (WET)            EXECUTION (LIVE)
┌─────────────────────┐         ┌─────────────────────────┐       ┌──────────────────────┐
│ workflow-deploy.yaml│         │ change_id + provenance  │       │ Swamp runs steps     │
│ .swamp.yaml         │──import▶│ structural change view  │──run─▶│ model methods mutate │
│ platform/registry   │         │ ALLOW/ESCALATE/BLOCK    │       │ real infrastructure  │
│ constraints policy  │         │                         │       │                      │
└─────────────────────┘         └─────────────────────────┘       └──────────────────────┘
```

Important nuance for Swamp:

- Workflow YAML is already executable intent; it is not a Helm-style template render.
- Field-origin is usually straightforward.
- The high-value governance question is **"what changed structurally, and is it allowed?"**

| File | Owner | What it controls |
|------|-------|-----------------|
| `workflow-deploy.yaml` | Workflow team | Workflow graph — jobs, steps, model-method tasks |
| `workflow-rotate-creds.yaml` | Workflow team | Credential rotation workflow |
| `.swamp.yaml` | Workflow team | Local Swamp runtime config (vault/logging/version) |
| `platform/registry.yaml` | Platform | Typed operation contracts for workflow APIs |
| `platform/swamp-constraints.yaml` | Platform | Approved models/methods, required steps |

## If you already build workflow automation systems

This example matches teams that already run local-first, model-driven automation:

- Agents compose workflows from typed models.
- Teams review workflow changes in Git.
- Safety comes from model availability + policy checks on workflow changes.

`cub-gen` adds a reproducible governance loop around that workflow change process.

## Why this maps cleanly to the cub-gen framework

| Existing Swamp concern | cub-gen concept | Why it matters |
|------|------|------|
| Agent modifies workflow steps | DRY change import | Captures exact workflow mutation in a governed record. |
| `platform/registry.yaml` operations | Typed contract surface | Portals and agents can discover allowed operation schemas. |
| New model/method references | Policy evaluation input | Enables ALLOW/BLOCK on risky capability expansion. |
| Required validation steps | Structural constraint checks | Prevents unsafe workflow edits from merging unnoticed. |
| Team wants fast iteration | Local verify/attest loop | Keeps agent/human loop fast without backend latency. |
| Org wants central audit | Connected ingest/query | Enables cross-repo compliance reporting. |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# 1) Detect/import workflow config
./cub-gen gitops discover --space platform --json ./examples/swamp-automation
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation > /tmp/swamp-import.json

# 2) Build local governed evidence (fast loop)
./cub-gen publish --in /tmp/swamp-import.json > /tmp/swamp-bundle.json
./cub-gen verify --json --in /tmp/swamp-bundle.json
./cub-gen attest --in /tmp/swamp-bundle.json --verifier ci-bot > /tmp/swamp-attestation.json
```

## Real-world scenario: agent adds a new model-method step

**Who**: A team using AI agents to compose deployment workflows. The agent
has been asked to add a pre-deploy health verification step.

### Scenario A — Approved model/method (ALLOW)

```yaml
# workflow-deploy.yaml — agent adds health check
jobs:
  - name: deploy-flow
    steps:
      - name: healthcheck       # new step
        task:
          type: model_method
          modelIdOrName: app-healthcheck  # approved model
          methodName: verify              # approved method
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

```bash
# cub-gen detects the workflow change
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation

# Evidence chain
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# app-healthcheck.verify is in approved model-method policy → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by workflow-lead --reason "healthcheck step uses approved model"
```

Governance outcome: `app-healthcheck.verify` is in the approved model-method
policy, and the required `validate` step is still present → **ALLOW**.

### Scenario B — Unapproved model or missing required step (BLOCK)

The agent introduces an unapproved model or removes the required `validate` step:

```yaml
# workflow-deploy.yaml — risky change
jobs:
  - name: deploy-flow
    steps:
      - name: risky-step
        task:
          type: model_method
          modelIdOrName: untrusted-model  # NOT in approved list
          methodName: do_something
      # validate step removed!
      - name: apply
        task:
          type: model_method
          modelIdOrName: app-deployer
          methodName: apply
```

```bash
# cub-gen detects the structural change
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation

# Evidence chain
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation > bundle.json
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# untrusted-model is not approved, validate step is missing → BLOCK
./cub-gen bridge decision apply --decision decision.json --state BLOCK \
  --approved-by governance-bot \
  --reason "Model 'untrusted-model' is not approved. Required 'validate' step is missing."
```

Governance outcome: `untrusted-model` is not in the approved list, AND the
required `validate` step was removed → **BLOCK**. The workflow cannot execute.

## How it works

For Swamp, use `cub-gen` mainly for **structural classification** of workflow changes:

1. Parse workflow and Swamp repo config.
2. Load `platform/registry.yaml` operation contracts (for typed operation discovery).
3. Capture model/method references and step structure.
4. Compare change against policy constraints.
5. Produce evidence bundle and attestation.
6. Optionally ingest/query in ConfigHub for org-wide reporting.

## What this is not

- Not a Swamp runtime replacement.
- Not a DAG executor.
- Not a mandatory backend round-trip for every agent iteration.

Recommended pattern:

- Use local loop for fast agent iteration.
- Use connected mode when you need centralized governance history and cross-repo queries.

## Key files

| File | Typical owner in Swamp teams | Purpose |
|------|-------------------------------|---------|
| `.swamp.yaml` | Team owning the workflow repo | Local Swamp runtime config (vault/logging/version). |
| `workflow-deploy.yaml` | Team owning the workflow repo | Workflow graph and model-method tasks. |
| `workflow-rotate-creds.yaml` | Team owning the workflow repo | Credential rotation workflow. |
| `platform/registry.yaml` | Org/platform/security | FrameworkRegistry v1 operation contracts for workflow updates. |
| `platform/swamp-constraints.yaml` | Org/platform/security | Organizational guardrails for workflow changes. |

## The complete Swamp + ConfigHub picture

| Layer | Example | Generator | What it governs |
|-------|---------|-----------|-----------------|
| Workflow changes | `swamp-automation` (this) | `swamp` | Workflow graph/method change governance |
| Runtime deployment | [`swamp-project`](../swamp-project/) | `helm-paas` | Swamp runtime on Kubernetes |

## Next steps

- Runtime side: [`swamp-project`](../swamp-project/)
- Ops policy analog: [`ops-workflow`](../ops-workflow/)
- Connected demo script index: [`../demo/README.md`](../demo/README.md)
- Prompt-as-DRY demo: `../demo/prompt-as-dry-local.sh`

### PR-MR pairing and promotion flows

- **Flow A (Git PR → ConfigHub MR)**: `../demo/flow-a-git-pr-to-mr-connected.sh`
  — workflow team opens PR, ConfigHub creates MR with evidence
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
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '.provenance[0].field_origin_map'

# Swamp workflow analysis
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '.provenance[0].swamp_workflow_analysis'

# Evidence bundle
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

## 7. Try one governed change

**ALLOW path**: Agent adds step using approved model:

```yaml
# workflow-deploy.yaml change
steps:
  - name: healthcheck
    task:
      type: model_method
      modelIdOrName: app-healthcheck  # approved
      methodName: verify
```

Result: Model is approved, required steps present → **ALLOW**

**BLOCK path**: Agent introduces unapproved model:

```yaml
# workflow-deploy.yaml change
steps:
  - name: risky-step
    task:
      type: model_method
      modelIdOrName: untrusted-model  # NOT approved
      methodName: do_something
```

Result: Model not in approved list → **BLOCK**

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/swamp-automation/demo-local.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/swamp-automation/demo-connected.sh
```
