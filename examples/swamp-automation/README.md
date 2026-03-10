# Swamp Automation — Governed Changes for Agent-Written Workflows

Swamp teams use AI agents to compose workflows from typed models, then run those workflows against real systems.

For this pattern, `cub-gen` is most useful as a **workflow change classifier and policy gate**:

- what changed in the workflow graph,
- which models/methods were introduced,
- whether required steps were removed,
- whether the change is allowed.

> For deploying the Swamp runtime itself on Kubernetes, see [`swamp-project`](../swamp-project/).

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
│ constraints policy  │         │ ALLOW/ESCALATE/BLOCK    │       │ real infrastructure  │
└─────────────────────┘         └─────────────────────────┘       └──────────────────────┘
```

Important nuance for Swamp:

- Workflow YAML is already executable intent; it is not a Helm-style template render.
- Field-origin is usually straightforward.
- The high-value governance question is **"what changed structurally, and is it allowed?"**

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

## Real-world scenario: agent adds a new model method step

A team asks an agent to add a pre-deploy health verification step.

Workflow change:

```yaml
jobs:
  - name: deploy-flow
    steps:
      - name: healthcheck
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

Governance outcome:

- `app-healthcheck.verify` is in approved model-method policy.
- Required `validate` step is still present.
- Decision can be `ALLOW`.

If the agent removed `validate` or introduced an unapproved model/method, decision should be `BLOCK` (or `ESCALATE`).

## How it works

For Swamp, use `cub-gen` mainly for **structural classification** of workflow changes:

1. Parse workflow and Swamp repo config.
2. Capture model/method references and step structure.
3. Compare change against policy constraints.
4. Produce evidence bundle and attestation.
5. Optionally ingest/query in ConfigHub for org-wide reporting.

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

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline"
./examples/swamp-automation/demo-local.sh

echo "connected"
cub auth login
./examples/swamp-automation/demo-connected.sh
```
