# Swamp Runtime (Helm Deployment)

**Pattern: platform-provided Helm chart deploying the Swamp AI automation runtime on Kubernetes.**

> This is the infrastructure side of the Swamp story. For the workflow governance side, see [`swamp-automation`](../swamp-automation/).

## 1. What is this?

A platform team provides a Helm chart that deploys the Swamp runtime — the execution engine that runs AI-agent-driven workflows. Project teams customize the runtime for their use case (model gateway selection, replica count, image version) via values overlays. The platform controls the chart contract and runtime policy.

This uses the standard `helm-paas` generator. The Swamp-specific story is that this chart *deploys the workflow engine itself* — the same engine whose workflow definitions are governed in the [`swamp-automation`](../swamp-automation/) example.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **Project team** | `values.yaml` — replica count, image tag, model gateway | `replicaCount`, `image.tag`, `runtime.modelGateway` |
| **Platform team** | `Chart.yaml` — chart contract, version | Chart version, appVersion |
| **Platform team** | `platform/` — runtime policy (future) | Resource baselines, model gateway allowlist |
| **GitOps reconciler** | Flux/ArgoCD syncs the rendered HelmRelease | Reconciles WET to LIVE |

## 3. What does cub-gen add?

Standard Helm generator features:

- **Generator detection**: recognizes `Chart.yaml` + `values.yaml` as a helm-paas source
- **DRY/WET classification**: chart contract is platform DRY, values overlays are project DRY
- **Field-origin tracing**: `runtime.modelGateway` traces to `values.yaml` (base) or `values-prod.yaml` (prod override)
- **Inverse-edit guidance**: "to change the model gateway in production, edit `values-prod.yaml` runtime section"

```bash
./cub-gen gitops discover --space platform --json ./examples/swamp-project
./cub-gen gitops import --space platform --json ./examples/swamp-project ./examples/swamp-project
```

## 4. How do I run it?

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/swamp-project
./cub-gen gitops import --space platform --json ./examples/swamp-project ./examples/swamp-project
./cub-gen publish --space platform ./examples/swamp-project ./examples/swamp-project > /tmp/swamp-proj-bundle.json
./cub-gen verify --in /tmp/swamp-proj-bundle.json
./cub-gen attest --in /tmp/swamp-proj-bundle.json --verifier ci-bot
./cub-gen gitops cleanup --space platform ./examples/swamp-project
```

## 5. Real-world example using ConfigHub

The ML platform team maintains the Swamp runtime chart. When a project team needs to switch from `llama-gateway` to `mistral-gateway`:

1. Project team edits `values-prod.yaml`: `runtime.modelGateway: mistral-gateway`
2. `cub-gen publish` produces a change bundle showing the gateway change
3. ConfigHub's decision engine checks the platform's model gateway allowlist
4. If `mistral-gateway` is approved → ALLOW; if not → ESCALATE to platform-owner
5. After ALLOW, Flux reconciles the updated HelmRelease to the cluster

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `Chart.yaml` | Platform team | Helm chart contract — name, version |
| `values.yaml` | Project team | Base values — replicas, image, model gateway |
| `values-prod.yaml` | Project team | Production overlay — higher replicas, pinned image, prod gateway |

## The complete Swamp + ConfigHub picture

| Layer | Example | Generator |
|-------|---------|-----------|
| **Workflow governance** | [`swamp-automation`](../swamp-automation/) | `swamp` — governs workflow definitions |
| **Runtime deployment** | `swamp-project` (this) | `helm-paas` — deploys the Swamp engine |

Both use the same canonical DRY+WET+LIVE pattern. Both produce ConfigHub-ready change bundles. Together they show how an AI automation platform gets full governance — from the workflows it runs to the infrastructure it runs on.
