# Operation Registry in Real Apps

`FrameworkRegistry` is not limited to the AI Ops example.
This repo includes three runnable platform patterns using registry-driven operations.

## Where it is used

| Example | Registry file | What it models |
|---|---|---|
| AI Ops PaaS | `examples/ai-ops-paas/platform/registry.yaml` | Agent fleet platform operations (fleet config, runtime, credentials, RBAC). |
| Ops Workflow Platform | `examples/ops-workflow/platform/registry.yaml` | Operational workflows (deploy/restart/scale/rollback) with schedule and policy checks. |
| Swamp Workflow Platform | `examples/swamp-automation/platform/registry.yaml` | Agent-authored workflow graph changes with model/method and required-step constraints. |

## Why this helps

`FrameworkRegistry` gives one typed surface for:

1. operation discovery (`what can this platform do?`)
2. input validation (`is this change allowed for this environment?`)
3. governance/audit (`which operation and policy produced this outcome?`)

This works for app workloads and ops/workflow workloads.

## Quick run paths

```bash
# AI Ops registry-backed platform
./examples/ai-ops-paas/demo-local.sh

# Ops workflow registry-backed platform
./examples/ops-workflow/demo-local.sh

# Swamp workflow registry-backed platform
./examples/swamp-automation/demo-local.sh
```

Connected mode for any of the above:

```bash
cub auth login
./examples/<example>/demo-connected.sh
```

## Practical "real app" shape

For a typical Spring/Helm app platform, registry operations usually look like:

- route/expose app
- scale app
- bind database
- upgrade image

For ops/workflow platforms, operations usually look like:

- define workflow
- schedule workflow
- add action/model-method step
- enforce required validation/approval step

The same cub-gen flow applies to both:
`discover -> import -> publish -> verify -> attest -> (connected) decision/query`.
