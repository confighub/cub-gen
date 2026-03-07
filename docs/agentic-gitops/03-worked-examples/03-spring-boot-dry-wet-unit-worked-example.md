# Spring Boot Worked Example: Dry/Wet Units with FluxCD and ArgoCD

Status: reference walkthrough (product semantics)  
Last updated: 2026-03-05

This is the Spring Boot equivalent of the dry/wet unit model used in the FluxCD
and ArgoCD rendered-pipeline examples.

It is designed for the primary user in this plan: a platform engineer working
with AI tools, who needs fast adoption and explicit governance without changing
controller topology.

---

## 1. What This Example Proves

For a Spring Boot service, this walkthrough shows:

1. how framework config + minimal deploy intent become DRY inputs,
2. how a deterministic generator produces explicit WET manifests,
3. how `cub-gen` imports and explains the resulting dry/wet lineage,
4. how Flux/Argo reconciliation remains unchanged,
5. how provenance + field-origin maps enable fast edits without repo archaeology.

No reconciler replacement is required.

---

## 2. Equivalent Mapping (Spring Boot vs Dry/Wet Model)

| Concept | Dry/Wet model | Spring Boot equivalent |
|---|---|---|
| DRY authoring input | team intent | `application.yaml` + `deploy/intent-prod.yaml` |
| Render operation | deterministic generator | `spring-gen render` |
| WET deployment contract | explicit manifests | rendered `Deployment/Service/HPA/Ingress` |
| Transport | OCI preferred | digest-pinned OCI artifact (or Git path) |
| Reconciler | Flux/Argo | Flux/Argo (unchanged) |
| Control-plane model | dry/wet units + lineage | imported by `cub-gen` with provenance + origin map |

---

## 3. Repository Layout (Example)

```text
acme-payments-service/
  src/main/resources/application.yaml
  deploy/
    intent-prod.yaml
    intent-staging.yaml

acme-platform-gitops/
  apps/payments/prod/rendered.yaml
  apps/payments/staging/rendered.yaml
  flux/ or argo/ wiring
```

Two repos are common in practice:

1. app intent repo (developer-facing DRY),
2. GitOps deploy repo (controller-facing WET contract).

---

## 4. Step-by-Step Walkthrough

### Step 1: Author DRY inputs

`src/main/resources/application.yaml` (framework-owned semantics):

```yaml
server:
  port: 8080

management:
  server:
    port: 9090
  endpoint:
    health:
      probes:
        enabled: true

spring:
  lifecycle:
    timeout-per-shutdown-phase: 30s
```

`deploy/intent-prod.yaml` (platform/app choices):

```yaml
unit: payments-api
namespace: payments-prod
image: ghcr.io/acme/payments-api:1.2.3

route:
  host: pay.example.com
  tls: true

autoscale:
  min: 2
  max: 10
  metric: cpu
  targetUtilization: 70

resources:
  requests:
    cpu: "250m"
    memory: "512Mi"
  limits:
    cpu: "1"
    memory: "1Gi"
```

### Step 2: Render DRY -> WET deterministically

```bash
spring-gen render \
  --app src/main/resources/application.yaml \
  --intent deploy/intent-staging.yaml \
  --out /tmp/payments-staging-rendered.yaml

spring-gen render \
  --app src/main/resources/application.yaml \
  --intent deploy/intent-prod.yaml \
  --out /tmp/payments-prod-rendered.yaml
```

Promote rendered artifacts into GitOps repo:

```bash
cp /tmp/payments-staging-rendered.yaml acme-platform-gitops/apps/payments/staging/rendered.yaml
cp /tmp/payments-prod-rendered.yaml acme-platform-gitops/apps/payments/prod/rendered.yaml
```

### Step 3: Keep Flux/Argo wiring unchanged

Flux-style source (illustrative):

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: payments-prod
  namespace: flux-system
spec:
  interval: 1m
  url: oci://ghcr.io/acme/platform-wet/payments
  ref:
    digest: sha256:ab44...771e
```

Argo-style source (illustrative):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: payments-prod
  namespace: argocd
spec:
  source:
    repoURL: oci://ghcr.io/acme/platform-wet/payments
    targetRevision: sha256:ab44...771e
  destination:
    server: https://kubernetes.default.svc
    namespace: payments-prod
```

Controllers continue reconciling WET exactly as before.

### Step 4: Import and explain with `cub-gen`

```bash
cub-gen detect --repo https://github.com/acme/acme-platform-gitops --ref main --json
cub-gen import --repo https://github.com/acme/acme-platform-gitops --ref main --space platform-prod --json
```

Then resolve ownership for any deployed field:

```bash
cub-gen origin \
  --change-id chg_01HXSPRINGABC123 \
  --wet-path 'Deployment/spec/template/spec/containers[name=app]/resources/requests/cpu' \
  --json
```

This gives a deterministic answer to: "which DRY input owns this WET field?"

### Step 5: AI-assisted change with governed evaluation

Example: an AI agent proposes higher concurrency and max replicas for prod.

1. Agent updates `deploy/intent-prod.yaml` (`autoscale.max: 15`).
2. Generator re-renders WET.
3. Reviewer opens/updates CH MR.
4. `cub-gen evaluate --change-id ... --json` returns policy result.
5. Decision path is explicit: `ALLOW | ESCALATE | BLOCK`.

### Step 6: App-team change -> platform approval -> upstream promotion

Use the same promotion path agreed in this plan:

1. App team ships feature + config change in app DRY overlay.
2. ConfigHub renders/evaluates candidate WET and opens/updates CH MR.
3. Platform engineer merge-approves in ConfigHub.
4. Governed decision gate enforces `ALLOW|ESCALATE|BLOCK` for deploy.
5. After successful rollout, ConfigHub opens promotion PR/MR to upstream platform DRY/base when reusable, if not already done.
6. Platform team merges upstream PR in Git.
7. App overlay is reduced to avoid long-lived drift.

Guardrail:

1. Never auto-write platform main DRY without separate upstream review.

---

## 5. Provenance and Field-Origin Map (Extra Value)

### Provenance snapshot (illustrative)

```yaml
unit: payments-prod-rendered
kind: wet
generator:
  name: spring-boot-generator
  version: spring-gen@1.0.0
inputs:
  digest: sha256:91d0...4bb2
source_artifacts:
  - role: app_config
    repo: https://github.com/acme/acme-payments-service
    path: src/main/resources/application.yaml
    revision: 3f2a9c1
  - role: deploy_intent
    repo: https://github.com/acme/acme-payments-service
    path: deploy/intent-prod.yaml
    revision: 3f2a9c1
rendered:
  artifact: oci://ghcr.io/acme/platform-wet/payments@sha256:ab44...771e
controller:
  type: flux
  object: OCIRepository/payments-prod
```

### Field-origin map snapshot (illustrative)

```yaml
field_origin_map:
  - wet_path: Deployment.spec.template.spec.containers[name=app].ports[0].containerPort
    dry_source:
      file: src/main/resources/application.yaml
      path: server.port
      editable_by: app-team

  - wet_path: Deployment.spec.template.spec.terminationGracePeriodSeconds
    dry_source:
      file: src/main/resources/application.yaml
      path: spring.lifecycle.timeout-per-shutdown-phase
      editable_by: app-team

  - wet_path: HorizontalPodAutoscaler.spec.maxReplicas
    dry_source:
      file: deploy/intent-prod.yaml
      path: autoscale.max
      editable_by: platform-team
```

Operational value:

1. no guessing which repo/file to edit,
2. faster reviews because ownership is explicit,
3. stale-render and overlay-drift signals become machine-checkable.

---

## 6. Adoption and Cognitive Simplicity (Priority)

This example is intentionally built around a fast first session:

1. Keep current repos and Flux/Argo topology.
2. Run `cub-gen detect` and `cub-gen import`.
3. Run one `cub-gen origin` query on a known production field.
4. Show one governed evaluation result for a candidate change.

Target outcome:

1. team understands value in <= 10 minutes,
2. no migration demanded,
3. immediate trust gain from explainability.

Why this matters:

1. generators are the adoption bridge,
2. cognitive simplicity drives user growth,
3. governance value is only credible after first-use simplicity is proven.

---

## 7. What Belongs in Git vs ConfigHub in This Example

Git stores DRY + compact linkage:

1. `application.yaml` + intent files,
2. rendered artifact pointers/commits,
3. compact receipts and stable IDs.

ConfigHub stores WET governance state:

1. dry/wet units and merge lineage,
2. provenance graph and field-origin maps,
3. decisions, execution telemetry, verification, attestation.

---

## 8. Equivalence Checklist

If these are true, the Spring flow is equivalent to the Flux/Argo dry/wet model:

1. DRY intent and framework config are explicit and versioned.
2. Generator output is explicit WET and deterministic for same inputs.
3. Flux/Argo reconciliation loop is unchanged.
4. `cub-gen` can detect/import/explain lineage deterministically.
5. Critical WET fields map back to DRY source paths.
6. Governed decisions are explicit (`ALLOW|ESCALATE|BLOCK`).
7. Reusable app changes can be promoted upstream via separate review.

---

## 9. Mandatory Enforcement Proof (Agentic GitOps Qualification)

This example should only be presented as Agentic GitOps when all controls pass:

1. Active GitOps reconciler proof exists (`WET -> LIVE` via Flux/Argo or equivalent).
2. Signed `GeneratorContract` is present with deterministic output hash.
3. `ProvenanceRecord` contains immutable `input_hash`, `toolchain_version`, `policy_version`, `run_id`, and artifact digests.
4. Inverse plan references `OwnershipMap`.
5. Out-of-scope inverse write evaluates to `BLOCK`.
6. Replay mismatch evaluates to `ESCALATE`.
7. Decision gate is explicit: `ALLOW|ESCALATE|BLOCK`.
8. `ALLOW` path includes attestation linkage.
9. Protected DRY updates are PR/MR-only.
10. Verification failure downgrades to read-only evidence mode.
11. Mutation ledger append is recorded for the change.

Naming rule:

1. If item 1 fails, this flow is `governed config automation`, not Agentic GitOps.

---

## 10. Related Docs

1. `../02-design/10-generators-prd.md`
2. `01-scoredev-dry-wet-unit-worked-example.md`
3. `02-traefik-helm-dry-wet-unit-worked-example.md`
4. `../02-design/20-field-origin-maps-and-editing.md`
5. `../02-design/50-dual-approval-gitops-gh-pr-and-ch-mr.md`
