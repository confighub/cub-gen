# Build Your Own Heroku in a Weekend

You can build one internal platform that supports web apps, ops workflows, and AI agent fleets without building a custom controller stack from scratch.

This project is for teams with an existing platform pattern and for teams rolling out a new one quickly. It builds on ConfigHub's control plane and config database by adding an agentic application layer in Git.
In this model, a generator is a function that turns app config into platform config, plus governance and attestation artifacts.

## The model in one line

Team writes a short config -> platform renders Kubernetes resources with guardrails -> Flux/Argo deploys.

## What this is (and what it is not)

- It is one generic platform model.
- It is not a separate PaaS per workload type.
- Workload-specific behavior lives in adapters (Spring Boot, Helm, score.dev, c3agent, etc.).

Think "one Heroku, many buildpacks."

## What an app looks like on this platform

An app is three things:

1. A workload image (the code that runs).
2. A small team-owned config file (what they want).
3. An optional production override file (what changes by environment).

For the `c3agent` workload, a team-owned config can look like this:

```yaml
service: c3agent
fleet:
  name: ml-review-fleet
  max_concurrent_tasks: 3
  agent_model: claude-sonnet-4-20250514
agent_runtime:
  image: ghcr.io/acme-corp/ai-ops-agent:latest
  max_budget_usd: 8.0
components:
  controlplane:
    replicas: 1
  gateway:
    replicas: 1
storage:
  task_pvc_size: 50Gi
credentials:
  anthropic_key_ref: ANTHROPIC_API_KEY
  github_token_ref: GH_TOKEN
```

The team does not author Kubernetes manifests directly.

## Why this becomes 11 Kubernetes resources (and why these 11)

For `c3agent`, the platform renders 11 resources because each one covers a real runtime need:

1. `Deployment` (control plane): runs the scheduler/orchestrator process.
2. `Service` (control plane): stable network endpoint for the control plane.
3. `Deployment` (gateway): runs request ingress/dispatch process.
4. `Service` (gateway): stable endpoint for incoming requests.
5. `ConfigMap` (fleet settings): model, concurrency, and fleet behavior settings.
6. `ConfigMap` (agent runtime template): agent image and runtime defaults.
7. `Secret`: API keys/tokens and sensitive runtime references.
8. `PersistentVolumeClaim`: task data that must survive pod restarts.
9. `ServiceAccount`: execution identity for agent jobs.
10. `ClusterRole`: least-privilege permissions needed for job orchestration.
11. `ClusterRoleBinding`: binds the role to the service account.

The short config expresses intent. The platform fills in the infrastructure details consistently.

## Same platform, different app types

The core platform behavior stays the same across workloads. Only the adapter changes.

| Workload type | What the app team writes | What the adapter knows |
|---|---|---|
| Spring Boot | `application.yaml` + Java code | ports, health endpoints, config patterns |
| Helm | `Chart.yaml` + `values.yaml` | chart render outputs and value ownership |
| score.dev | `score.yaml` | workload shape and rendered resources |
| c3agent | `c3agent.yaml` | control plane/gateway/storage/credential/RBAC needs |

This is why the platform is "generic": one governance and deployment flow, many adapters.

## Platform team vs app team

These roles are different.

Platform team responsibilities:

1. Define supported workload adapters.
2. Set defaults and guardrails (approved images/models, budget caps, HA rules).
3. Maintain platform-level policies and ownership boundaries.

App team responsibilities:

1. Write workload code and team-owned config.
2. Choose values within allowed guardrails.
3. Ship changes through normal Git flow.

The platform team does the heavy lifting once. App teams get a simple self-service path.

## Which comes first: platform design or app onboarding?

Both paths are valid, but most organizations start with import-first.

1. Path 1 (common): import existing repos first to get immediate visibility and governance.
2. Path 2 (next): standardize a cleaner self-service contract after teams can see and trust the model.

This is usually easier than trying to design a perfect platform contract before seeing real repos.

## Weekend build plan

Saturday:

1. Pick one workload type (for example, `c3agent` or Spring Boot).
2. Define adapter expectations and guardrails.
3. Run discover/import against a real repo and validate ownership + edit guidance.

Sunday:

1. Add production override rules.
2. Validate key policy failures (budget, image registry, HA).
3. Run through Flux/Argo reconciliation path end-to-end.

Monday:

1. First team ships via short config + Git push.

## Try it in this repo

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/ai-ops-paas
./cub-gen gitops import --space platform --json ./examples/ai-ops-paas ./examples/ai-ops-paas
```

Related examples:

- `examples/ai-ops-paas`
- `examples/springboot-paas`
- `examples/helm-paas`
- `examples/scoredev-paas`
