# Score.dev PaaS (App-First DRY)

**Pattern: app-first authoring — developers declare workload intent in Score format, platform enforces contracts at publish time.**

## 1. What is this?

A Node.js team describes their service using Score.dev — a platform-agnostic workload specification. They declare the container image, environment variables, and resource dependencies (like a Postgres database) in `score.yaml`. The platform team provides workload class contracts and network policies that the Score output must satisfy. Flux/ArgoCD reconciles the rendered manifests.

Score.dev lets app teams describe *what they need* without knowing the Kubernetes details. cub-gen maps Score intent to governed WET manifests with full provenance.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **App team** | `score.yaml` — workload spec, env vars, resource deps | Container image, environment, resource requirements |
| **App team** | `app/` — application source code | Node.js server, package.json |
| **Platform team** | `platform/contracts/workload-class.yaml` | Required probes, resource minimums |
| **Platform team** | `platform/policies/network-egress.yaml` | Network policy enforcement |
| **GitOps reconciler** | Flux Kustomization / ArgoCD Application | Reconciles WET to LIVE |

## 3. What does cub-gen add?

- **Generator detection**: recognizes `score.yaml` with `Score.dev/v1b1` apiVersion as `scoredev-paas` profile (capabilities: `render-manifests`, `workload-spec`, `inverse-score-patch`)
- **DRY/WET mapping**: Score workload spec (DRY) → Kubernetes Deployments, Services, ConfigMaps (WET)
- **Field-origin tracing**: `containers.main.image` traces to `score.yaml`, resource dependencies trace to `resources` section
- **Inverse-edit guidance**: "to change the container image, edit `score.yaml` containers section"

```bash
./cub-gen gitops discover --space platform --json ./examples/scoredev-paas
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '{profile: .discovered[0].generator_profile, field_origin_map: .provenance[0].field_origin_map}'
```

## 4. How do I run it?

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/scoredev-paas
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas > /tmp/score-bundle.json
./cub-gen verify --in /tmp/score-bundle.json
./cub-gen attest --in /tmp/score-bundle.json --verifier ci-bot > /tmp/score-attestation.json
./cub-gen verify-attestation --in /tmp/score-attestation.json --bundle /tmp/score-bundle.json
./cub-gen gitops cleanup --space platform ./examples/scoredev-paas
```

## 5. Real-world example using ConfigHub

A product team at a SaaS company uses Score.dev for all their microservices. They have 15 services, each with a `score.yaml` in its repo.

**Scenario: Adding a Redis dependency**

The team needs to add Redis caching to their checkout service. They edit `score.yaml`:

```yaml
resources:
  db:
    type: postgres
  cache:           # new resource dependency
    type: redis
```

**Governed pipeline:**

```bash
# 1. cub-gen detects the new resource dependency
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas
# Field-origin: resources.cache added in score.yaml (app-team owned)

# 2. Produce evidence chain
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 3. ConfigHub ingests and evaluates
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example
# Decision engine checks: workload class contract allows Redis? → ALLOW
# Network policy allows egress to Redis? → Check platform/policies/network-egress.yaml
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-lead --reason "Redis approved for caching use case"
```

**What ConfigHub provides:**
- **Resource dependency audit**: "which services depend on Redis?" — cross-repo query
- **Contract enforcement**: workload class contract validates that the service meets probe and resource requirements *before* the Redis dependency is provisioned
- **Provenance chain**: the Redis addition is linked to the CI verification, the platform approval, and the Flux reconciliation

## Narrative turns

1. **Prompt-first app change** — Developer updates workload image and env vars in `score.yaml`. Import shows this as app-team DRY with field-origin map.
2. **Platform guardrail check** — Platform policy checks required probes/resources from contract files. Decision path remains explicit.
3. **Runtime rollout** — Flux/Argo syncs WET manifests. ConfigHub keeps attestation + provenance continuity.
4. **Promotion** — Reusable app conventions can be promoted to platform defaults.

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `score.yaml` | App team | Score workload spec — image, env vars, resource deps |
| `app/server.js` | App team | Node.js application source |
| `app/package.json` | App team | Node.js dependencies |
| `platform/contracts/workload-class.yaml` | Platform team | Required probes, resource minimums |
| `platform/policies/network-egress.yaml` | Platform team | Network egress policy |
| `gitops/flux/kustomization.yaml` | Platform team | Flux Kustomization transport |
| `gitops/argo/application.yaml` | Platform team | ArgoCD Application transport |
| `docs/user-stories.md` | — | Narrative user stories |
