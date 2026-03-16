# Score.dev PaaS — Platform-Agnostic Workloads with Governance

Your developers describe what their service needs in `score.yaml` — container
image, environment variables, resource dependencies — without touching Kubernetes
YAML. The platform team defines the contracts (required probes, resource minimums,
network policies) separately. Score bridges the gap between developer intent and
platform enforcement.

ConfigHub adds the missing piece: traceable provenance from Score workload spec
through to governed Kubernetes manifests, with field-level ownership and
inverse-edit guidance.

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding Score governance | Jump to [Run from ConfigHub](#run-from-configHub-connected-mode) |
| **Existing Score.dev user** adding ConfigHub | Jump to [Try it](#try-it) then connect later |

Both paths lead to the same outcome: governed Score workloads with field-origin tracing.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real app** | Node.js checkout API defined in `score.yaml` |
| **Real cluster objects** | Kubernetes Deployment, Service, ConfigMap |
| **Real inspection target** | `kubectl get deployment checkout-api -o yaml` |
| **GitOps transport** | Flux Kustomization or ArgoCD Application |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "What K8s fields came from my score.yaml?" | Field-origin tracing with 0.94 confidence | Container image changes → ALLOW |
| "Why did my deployment fail validation?" | Platform contract enforcement at publish | Missing probes → ESCALATE |
| "Who owns this rendered field?" | DRY/WET ownership boundary | Resource dependency adds → governed review |

## Domain POV (Score platform teams)

This example is for organizations already using Score at scale:

- developers author `score.yaml` and do not touch Kubernetes YAML,
- platform teams own provisioners, workload classes, and policy defaults,
- incidents still require tracing "what happened between Score and runtime."

The first value is post-render clarity: which runtime fields came from Score
intent vs platform defaults, and who should edit each one.

## Which import path should a Score team use?

For Score users, start with `cub-gen`, not ConfigHub's GitOps Import wizard.

- ConfigHub GitOps Import is for importing ArgoCD/Flux application resources that already exist in a cluster.
- This example is about source-side Score intent in `score.yaml`.

That distinction matters because a Score team thinks in workload intent, workload
classes, and provisioners, not in Argo `Application` or Flux `HelmRelease`
objects.

So the recommended path is:

1. keep `score.yaml` as the app-team contract,
2. add `cub-gen` for field-origin tracing and inverse-edit guidance,
3. connect to ConfigHub when you want governed decisions, evidence, and
   cross-repo visibility.

## What you get

- **Full field-origin mapping**: every rendered Kubernetes field traces back to
  a specific line in `score.yaml` — with 0.94 confidence for container images
- **Platform contract enforcement**: workload class contracts validate probe
  requirements and resource minimums at publish time
- **Developer-friendly authoring**: app teams write Score, not Kubernetes YAML.
  cub-gen handles the DRY→WET mapping automatically
- **Resource dependency tracking**: "which services depend on Postgres?" is
  answerable from the provenance index

## How Score maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              RECONCILER (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ score.yaml          │          │ Deployment           │         │ Running pods     │
│ app/src/server.js   │──import─▶│ Service              │──sync──▶│ Live service     │
│ platform/contracts/ │          │ ConfigMap            │         │ Cluster state    │
│ platform/policies/  │          │ Kustomization (Flux) │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  App team: score.yaml.            Rendered K8s manifests            What's actually
  Platform: contracts + policies.  with field provenance.            running.
```

**DRY** is what your developers write: `score.yaml` declares the workload —
containers, env vars, ports, and resource dependencies (Postgres, Redis). This
is platform-agnostic intent.

**WET** is what cub-gen produces: Kubernetes Deployments, Services, and ConfigMaps
with every field traced back to Score. The platform's workload class contract
validates that the rendered output meets requirements.

**LIVE** is what's running. Flux or ArgoCD reconciles WET to LIVE. Your
reconciler stays in control.

| File | Owner | What it controls |
|------|-------|-----------------|
| `score.yaml` | App team | Workload spec — containers, env vars, ports, resource deps |
| `app/server.js` | App team | Node.js application source |
| `app/package.json` | App team | Node.js dependencies |
| `platform/contracts/workload-class.yaml` | Platform | Required probes, resource minimums |
| `platform/policies/network-egress.yaml` | Platform | Network egress policy |
| `gitops/flux/kustomization.yaml` | Platform | Flux Kustomization transport |
| `gitops/argo/application.yaml` | Platform | ArgoCD Application transport |

## If you already use Score.dev in production

This example is for teams already committed to Score-style app specs:

- Developers define workload intent in `score.yaml`.
- Platform teams map that intent into runtime policy and infrastructure.
- Incidents still require tracing rendered behavior back to Score fields.

cub-gen keeps Score as the app-team interface and adds deterministic provenance
and ownership routing so debugging and governance use the same contract.

If you are already a ConfigHub user, the framing is slightly different:

- use ConfigHub's cluster-side GitOps import for existing Argo/Flux-managed apps,
- use this Score example when you want to govern the source-side `score.yaml`
  contract directly.

## Why this maps cleanly to the cub-gen framework

| Existing Score concept | cub-gen concept | Why it matters |
|------|------|------|
| `score.yaml` | DRY app intent | Teams keep writing one high-level workload spec. |
| Score-to-K8s expansion | WET targets with lineage | Rendered manifests stop being opaque. |
| Platform contracts/policies | Governance layer | Rules run before deploy, not as after-the-fact review. |
| Flux/Argo deployment loop | LIVE state | Runtime remains unchanged while visibility improves. |

## Advanced reality check: provisioner fan-out and environment context

In mature Score platforms, one DRY resource declaration often fans out into
multiple runtime objects. Example:

`resources.db.type: postgres` may imply a credential secret, network policy,
runtime env wiring, and database provisioning metadata.

The governance requirement is the same: each generated field needs a traceable
source and owner, even when one Score field expands to many WET objects.

The same applies to environment context. The same `score.yaml` can produce
different WET output in dev vs prod due to platform contracts and provisioner
rules. This is why provenance must include both:

1. the app-authored DRY source (`score.yaml`), and
2. the platform-authored contract/policy layer that influenced rendering.

That combined trace is what makes Score practical at scale without forcing app
teams to learn Kubernetes internals.

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect Score workload spec
./cub-gen gitops discover --space platform --json ./examples/scoredev-paas

# Import with full field-origin mapping
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '{profile: .discovered[0].generator_profile, field_origin_map: .provenance[0].field_origin_map}'
```

cub-gen detects `score.yaml` with `Score.dev/v1b1` apiVersion and classifies
it as `scoredev-paas`. The field-origin map shows every container image, env
var, and port traced back to Score with confidence scores.

## Real-world scenario: adding a Redis cache dependency

**Who**: A product team at a SaaS company with 15 Score-based microservices.
Each repo has a `score.yaml` describing the workload.

### The change — app team adds Redis

```yaml
# score.yaml — adding a cache dependency
resources:
  db:
    type: postgres
  cache:            # new resource dependency
    type: redis
```

### Governed pipeline

```bash
# cub-gen detects the new resource dependency
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas

# Produce evidence chain
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Decision: workload class allows Redis, network policy permits egress → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-lead --reason "Redis approved for caching use case"
```

ConfigHub checks two things: the workload class contract (does the service meet
probe and resource requirements?) and the network egress policy (is the service
allowed to reach Redis?). Both pass → **ALLOW**.

If Redis weren't in the approved resource types, the decision engine would
**ESCALATE** to the platform owner for review.

## How it works

cub-gen's `scoredev-paas` generator detects `score.yaml` containing a
`Score.dev/v1b1` apiVersion. On import:

1. **Classifies inputs** — `score.yaml` (role: score-spec)
2. **Maps field origins** — `containers.main.image` traces to the Score
   container definition (confidence: 0.94); env vars at 0.90; ports at 0.91
3. **Applies transform** — `score-to-k8s` transform maps Score workload
   abstractions to concrete Kubernetes resources
4. **Validates contracts** — workload class contract checks probe requirements
   and resource minimums

A concrete field trace:

```
DRY:  score.yaml → containers.main.image = "ghcr.io/example/checkout-api:latest"
      ↓ score-to-k8s transform (confidence: 0.94)
WET:  Deployment/spec/template/spec/containers[name=main]/image
```

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `score.yaml` | App team | Workload spec — containers, env, ports, resources |
| `app/server.js` | App team | Node.js application source |
| `platform/contracts/workload-class.yaml` | Platform | Probe + resource requirements |
| `platform/policies/network-egress.yaml` | Platform | Network egress rules |
| `gitops/flux/kustomization.yaml` | Platform | Flux Kustomization transport |
| `gitops/argo/application.yaml` | Platform | ArgoCD Application transport |

## Next steps

- **Helm version**: [`helm-paas`](../helm-paas/) — same governance for
  chart-based deployments
- **Spring Boot version**: [`springboot-paas`](../springboot-paas/) — framework
  config with ownership boundaries
- **E2E demo**: `../demo/module-2-score-field-map.sh`
- **Worked example**: `../../docs/agentic-gitops/03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md`

## Run from ConfigHub (connected mode)

If you already have ConfigHub, start here:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"

# Publish and ingest
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map (Score → Kubernetes)
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '.provenance[0].field_origin_map'

# Score-specific analysis
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '.provenance[0].score_workload_analysis'

# Evidence bundle
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

## 7. Try one governed change

**ALLOW path**: App team updates container image in `score.yaml`:

```yaml
# score.yaml change
containers:
  main:
    image: ghcr.io/example/checkout-api:v2.0.0  # was :latest
```

Result: Image field is app-team owned → **ALLOW**

**ESCALATE path**: App team adds unapproved resource dependency:

```yaml
# score.yaml change
resources:
  db:
    type: postgres
  ml-gpu:          # new resource type
    type: gpu-pool # not in approved list
```

Result: GPU pool not in workload class approved types → **ESCALATE** to platform team

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/scoredev-paas/demo-local.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/scoredev-paas/demo-connected.sh
```
