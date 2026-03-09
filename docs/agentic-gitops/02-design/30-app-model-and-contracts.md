# ConfigHub App Model and Contracts

> Teams think in apps, not in repos or namespaces. The app model makes that
> the organizing primitive for configuration, governance, and operations.

**Part of:** [AI and GitOps v7 Document Set](../00-index/00-gitops7-index.md)
**Status:** Planning doc (v7)
**Date:** 2026-02-28
**Audience:** Implementors, API consumers, platform teams
**Purpose:** Entity definitions, operating boundary, constraints, staging model, and operations

---

## Table of Contents

1. [Why App-Centric Language](#1-why-app-centric-language)
2. [Operating Boundary](#2-operating-boundary)
3. [The App Model](#3-the-app-model)
4. [Cardinality Invariants](#4-cardinality-invariants)
5. [Staging Model](#5-staging-model)
6. [Constraints and Governance](#6-constraints-and-governance)
7. [Operations (Intent as Code)](#7-operations-intent-as-code)
8. [Cross-References](#8-cross-references)

---

## 1. Why App-Centric Language

Platform engineers work with namespaces, clusters, and Git repositories. App
teams work with "the payments service" or "the checkout app." When tooling
forces app teams to think in infrastructure terms, the result is
miscommunication and configuration that nobody can reason about end to end.

The ConfigHub app model makes the **App** the primary organizing concept.
Everything else -- Deployments, Targets, Units, Variants -- exists in relation
to an App:

- **Discovery is intuitive.** "Show me everything about `payment-service`"
  returns all components, all environments, all constraints, across all clusters.

- **Governance is expressible.** Platform teams write constraints against Apps,
  not against file paths across twelve repositories.

- **Promotion is traceable.** Moving from staging to production is an explicit
  operation on an App's Variant, with full provenance.

- **Ownership is unambiguous.** Every Unit belongs to exactly one App. Every
  App has a team. The ownership chain is inspectable without parsing directory
  structures or CODEOWNERS files.

The app model does not replace Kubernetes resources. It provides a vocabulary
layer above them that matches how teams think about their software.

---

## 2. Operating Boundary

The following table is the authoritative division of labor across ConfigHub,
cub-scout, the reconcilers, and the satellite tools.

| Responsibility | Owned By |
|----------------|----------|
| Store and publish explicit intended state + provenance | ConfigHub |
| Resolve field-origin maps, route editing to DRY sources | ConfigHub |
| Detect stale renders (inputs changed, output not re-rendered) | ConfigHub (publishing pipeline) |
| Reconcile runtime from published artifacts | Flux / Argo (inner loop) |
| Observe cluster reality, capture evidence, detect drift from intended state | cub-scout |
| Record governed mutation history in Git; redirect WET edits to DRY sources | cub-track |
| Evaluate risk and policy signals | confighub-scan |
| Evaluate semantic assertions + issue verification attestation | verified |
| Issue decisions and attestations | confighub (decision authority) |
| Execute token-scoped runtime actions | confighub-actions |

### Reading the boundary

The boundary separates three concerns that are often conflated:

**Intended state vs. runtime state.** ConfigHub owns what *should* exist. It
holds both DRY inputs and WET outputs, linked by provenance. Flux or Argo owns
making that state *actually* exist. ConfigHub publishes to Git or OCI; the
reconciler picks it up. ConfigHub never applies manifests directly.

**Staleness vs. drift.** "Did DRY inputs change since the last render?" is
staleness -- answered by comparing `inputs.digest` in ConfigHub's pipeline.
"Does the cluster match what the reconciler was told to apply?" is drift --
answered by cub-scout. Different questions, different tools.

**Evidence vs. decisions.** cub-scout and confighub-scan produce evidence.
verified produces attestations. ConfigHub consumes both and issues decisions:
proceed, hold, or rollback. confighub-actions executes the decision with a
token scoped to that specific action.

**Editing routes through field-origin maps.** When a user changes a rendered
field, ConfigHub traces it to its DRY source. cub-track intercepts direct WET
edits and performs the same redirection. See [03 -- Field-Origin Maps](../02-design/20-field-origin-maps-and-editing.md).

### Key clarifications

**ConfigHub is not a controller.** It does not reconcile. It stores, governs,
and publishes. Flux and Argo remain the runtime reconcilers.

**cub-scout observes the cluster, not the generator pipeline.** "Is the WET in
ConfigHub stale relative to DRY inputs?" is a ConfigHub publishing pipeline
concern -- detected via `inputs.digest` comparison -- not a cub-scout concern.

### Qualification rule: Agentic GitOps vs governed config automation

1. A system is **Agentic GitOps** only if an active GitOps reconciler loop
   (`WET -> LIVE`) is continuously running via Flux/Argo (or equivalent).
2. If governance and mutation controls exist without that reconciler loop,
   classify it as **governed config automation**.
3. This naming rule is mandatory for demos, docs, and architecture claims.

---

## 3. The App Model

The core entities below are current implementation mappings -- concepts are
stable, but the API surface may evolve. The **cardinality invariants** (Section
4) and **staging model** (Section 5) are durable; the API shape is not.

### App

A named collection of components owned by one team. It emerges from querying
Units by label:

```
App: payment-service
  Team: payments-squad
  Components: api, worker, redis
  Deployments: dev, staging, prod
```

An App is a label value (`app=payment-service`), not a separate API object
today. It comes into existence when the first Unit carries that label.

### Deployment

The junction of an App and a Target -- deploying an App to a specific
environment.

```
Deployment: payment-service-prod
  App: payment-service
  Target: prod-cluster-eu
  Reconciler: flux
  Variant: prod
  Units: [api, worker, redis]
```

A Deployment answers "where does this App run?" Same App in three clusters =
three Deployments.

### Target

A Kubernetes cluster (or other managed system) connected via a Worker. Targets
exist independently of which Apps are deployed to them.

```
Target: prod-cluster-eu  |  Provider: EKS  |  Region: eu-west-1  |  Worker: worker-eu-01
```

### Unit

The atomic element: a single deployable workload with labels, source mapping,
and provenance. Units remain the implementation primitive. Apps and Deployments
are queries over Units.

```
Unit: payment-service/api
  App: payment-service  |  Variant: prod  |  Kind: Deployment (K8s)
  Source: generators/spring-boot/payment-api.py
  Generator: spring-boot-generator v2.1.0  |  Inputs digest: sha256:a1b2c3...
```

A Unit carries its own provenance, making the DRY-to-WET chain inspectable per
workload.

### Variant

A label indicating environment or configuration flavor. Not a folder -- Git
paths like `overlays/prod` map to `variant=prod` on the Unit.

```
payment-service/api (variant=dev)       payment-service/api (variant=prod)
  Replicas: 1                             Replicas: 3
  Image: payment-api:latest               Image: payment-api:v2.4.1
  Resource limits: none                   Resource limits: cpu=500m, mem=512Mi
```

The Unit identity is the same; the Variant label selects the configuration
snapshot.

---

## 4. Cardinality Invariants

These rules hold regardless of API evolution. Platform teams, generators, and
tooling can rely on them as durable contracts.

| # | Relationship | Rule |
|---|-------------|------|
| C1 | App -> Units | An App is a **grouping key** (label query), not a mutable container. Adding a Unit to an App means labeling the Unit, not modifying the App. |
| C2 | Deployment -> Target | A Deployment binds **one** App to **one** Target. The same App in two clusters is two Deployments. |
| C3 | Deployment -> Reconciler | A Deployment has **exactly one** reconciler (Flux *or* Argo, not both). |
| C4 | Unit -> App | A Unit belongs to **exactly one** App. |
| C5 | Variant | A Variant is a **label** on a Unit, not a separate object. Staging is variant-driven, not folder-fragmented. |

**C1** prevents the App from becoming a bottleneck -- the App is always exactly
what its Units say it is. **C2** eliminates ambiguity about which configuration
applies to a given cluster. **C3** prevents split-brain reconciliation. **C4**
ensures every Unit has exactly one owning team; governance and audit follow this
single chain. **C5** keeps the staging model from fragmenting into directories.

---

## 5. Staging Model

Promotion across environments is explicit, variant-driven, and policy-gated.

### S1: Environments are Variant labels, not folders

```
Classical GitOps (folder-per-env):       ConfigHub (variant-per-env):
  overlays/                                Unit: payment-service/api
    dev/deployment.yaml   # copy 1           variant=dev      -> snapshot A
    staging/deployment.yaml # copy 2         variant=staging  -> snapshot B
    prod/deployment.yaml  # copy 3           variant=prod     -> snapshot C
```

Each snapshot is a full, self-contained configuration -- not a Kustomize patch
on top of a base. This eliminates "overlay drift" where dev and prod silently
diverge.

### S2: Promotion is an explicit operation

Promotion copies Unit configuration from one Variant to another, subject to
approval policy.

```
promote payment-service from staging to prod:
  source: variant=staging (snapshot B)
  target: variant=prod   (new snapshot C')
  policy: requires human approval (prod)
  result: C' = copy of B + prod constraints applied
```

`variant=prod` may require human approval. `variant=dev` may auto-promote on
successful CI. The policy is per-Variant, declared in constraints.

### S3: Each Variant is a full snapshot

Each Variant is a complete, auditable snapshot -- not a diff against another
Variant. You can inspect exactly what is deployed without resolving patch chains.

### S4: Constraints can vary per Variant

```
Constraint: min-replicas                 Constraint: image-policy
  variant=dev:     not applied             variant=dev:  any registry
  variant=staging: >= 1                    variant=prod: approved-registry.example.com only
  variant=prod:    >= 2
```

Platform teams enforce production-grade requirements without burdening dev.

---

## 6. Constraints and Governance

The platform team defines **constraints** that apply to Apps, Deployments, and
Targets. App teams make choices within those rules.

### The constraint tree

```
Platform constraints (acme-platform):
|
+-- MUST: All Units have resource limits
+-- MUST: Images from approved registry
+-- CAN:  May use Flux or ArgoCD
|
Applied to:
|
+-- App: payments-service
|   +-- Deployment: payments-dev   -> dev-cluster,  reconciler: flux
|   +-- Deployment: payments-prod  -> prod-cluster, reconciler: flux
|
+-- App: orders-service
    +-- Deployment: orders-dev     -> dev-cluster,  reconciler: argo
    +-- Deployment: orders-prod    -> prod-cluster, reconciler: argo
```

Both Apps inherit the same constraints. The payments team chose Flux; the orders
team chose Argo. Both valid: the platform says `CAN: Flux or ArgoCD`.

### Composition rules

When multiple platform packages contribute constraints, they compose by three
rules:

**G1: Constraints are additive.** MUST rules accumulate. Duplicates apply once;
new rules join the set.

```
acme-platform:       security-baseline:       Effective:
  MUST: res limits     MUST: res limits         MUST: resource limits  (once)
  MUST: approved reg   MUST: no privileged      MUST: approved registry
                                                MUST: no privileged containers
```

**G2: Capabilities are intersected.** CAN rules narrow to the intersection. A
new platform package can only restrict, never widen.

```
acme-platform:       security-baseline:       Effective:
  CAN: Flux or Argo    CAN: Flux only           CAN: Flux only
```

A team using ArgoCD under `acme-platform` alone receives a constraint violation
when `security-baseline` is added -- not a silent behavior change.

**G3: Conflicts surface as errors.** When two MUST rules cannot both be
satisfied, the system reports a conflict. It does not pick a winner.

```
package-a: MUST max-replicas <= 5   +   package-b: MUST min-replicas >= 10
  -> CONFLICT: "Cannot satisfy both"
  -> Deployment blocked until platform team resolves
```

Conflicts are always surfaced, never auto-resolved.

---

## 7. Operations (Intent as Code)

An operation is a method in a framework SDK that produces a configuration diff.
Operations are intent, not execution. They exist in "DRY space" -- describing
desired changes in the framework's vocabulary. When a generator renders them,
the result crosses into "WET space": explicit, deployable manifests in
ConfigHub. The operation itself is never deployed; only its rendered output is.

```
DRY space (intent)                         WET space (manifests)

app.add_router("pay.example.com")      --> Ingress YAML
app.scale(min=2, max=10)               --> HPA YAML
app.set_env("LOG_LEVEL", "info")       --> Deployment YAML (env section)
```

### Worked example

A platform team publishes a `spring-boot-generator`. An app team uses it:

```python
from acme_platform import AcmeApp

app = AcmeApp("payments-api", framework="spring-boot")
app.add_router("pay.example.com", tls=True)          # operation (DRY)
app.scale(min=2, max=10)                              # operation (DRY)
app.set_resource_limits(cpu="500m", memory="512Mi")   # satisfies platform MUST

# Generator renders -> Deployment, Service, Ingress, HPA manifests (WET)
# Each carries provenance: generator version, source file, inputs digest, timestamp
```

The app team never writes Kubernetes YAML. The generator translates intent into
manifests. ConfigHub stores both DRY and WET, linked by provenance.

### Four artifacts every operation produces

| Artifact | Purpose | Example |
|----------|---------|---------|
| **Plan** | What will change (preview) | "Will create Ingress for pay.example.com with TLS" |
| **Patch** | Explicit config delta | Ingress manifest diff |
| **Explanation** | Why, risks, assumptions | "Adding public endpoint; requires TLS cert" |
| **Provenance** | Who/what, when, from what | "payments-api.py at commit abc123 via spring-boot-gen v2.1.0" |

Before a change is applied, the Plan and Explanation allow review. After the
change, the Patch and Provenance provide audit evidence.

### Ownership and registration

Operations live in framework SDKs, not in ConfigHub. Frameworks own the
operations; ConfigHub stores the receipts. The SDK registers operations with
ConfigHub for discovery, validation, and audit.

```
Operation Registry Entry:
  name: add_router
  sdk: acme-platform/spring-boot-generator
  inputs: { hostname: string, tls: boolean }
  produces: [Ingress, Service]
  validates-against: [image-policy, ingress-allowlist]
```

This makes the DRY-to-WET boundary inspectable, enabling tooling (including AI
agents) to discover available operations without executing them.

---

## 8. Cross-References

| Document | Relationship |
|----------|-------------|
| [01 -- Introducing Agentic GitOps](../01-vision/01-introducing-agentic-gitops.md) | The "why": classical gaps that the app model and operating boundary address |
| [02 -- Generators PRD](../02-design/10-generators-prd.md) | Generator model, maturity levels, and provenance requirements |
| [03 -- Field-Origin Maps and Editing](../02-design/20-field-origin-maps-and-editing.md) | How field-origin maps trace WET fields back to DRY sources within the app model |
| [05 -- cub-track](../05-rollout/10-cub-track.md) | Governed mutation history: how cub-track records changes to Units and Deployments |
| [06 -- Governed Execution](../02-design/40-governed-execution.md) | The two-loop model, evidence, and decision authority referenced in the operating boundary |
| [07 -- User Experience](../05-rollout/30-user-experience.md) | How app-centric language appears in CLI, TUI, and AI skill interfaces |
| [08 -- Adoption and Reference](../05-rollout/40-adoption-and-reference.md) | Adoption path showing how teams progressively adopt the app model |
