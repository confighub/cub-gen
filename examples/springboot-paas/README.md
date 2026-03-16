# Spring Boot PaaS — Governed Config for Java Services

Your Spring Boot services already separate concerns: `application.yaml` for
business config, `application-prod.yaml` for production overrides, `pom.xml`
for build dependencies. Your developers know this layout.

The problem is that not everything in `application.yaml` has the same owner.
Feature flags are app-team territory. Datasource config is platform-controlled.
When someone changes `spring.datasource.hikari.maximum-pool-size`, is that an
app change or a platform change? Today, your PR reviewer has to know. With
ConfigHub, the ownership boundary is explicit and enforced.

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding Spring Boot governance | Jump to [Run from ConfigHub](#run-from-configHub-connected-mode) |
| **Existing Spring Boot user** adding ConfigHub | Jump to [Try it](#try-it) then connect later |

Both paths lead to the same outcome: governed Spring config with field-origin tracing.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real app** | Spring Boot 3.3.2 inventory service (Java 21) |
| **Real cluster objects** | Kubernetes Deployment, Service, ConfigMap |
| **Real inspection target** | `kubectl get deployment inventory-service -o yaml`, Spring Actuator `/health` |
| **GitOps transport** | Flux Kustomization or ArgoCD Application |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "Is this app config or platform config?" | Ownership by Spring property namespace | `server.port` → ALLOW, `spring.datasource.*` → BLOCK |
| "Which profile set this value?" | Profile overlay tracking with lineage | Trace `application-prod.yaml` override |
| "Can I change this in production?" | Governance decisions in Spring terms | "datasource is platform-owned" not "Deployment spec changed" |

## Domain POV (Spring Boot shops)

This example targets Spring-heavy teams where developers know Spring, not
Kubernetes:

- app teams edit `application*.yaml` and ship features,
- platform teams own datasource/SLO/secrets boundaries,
- reviews fail when ownership is implicit ("is this app config or platform config?").

The goal is invisible governance for app developers: they get a clear ALLOW or
BLOCK in PR terms they understand, without learning manifest internals.

## What you get

- **Ownership-aware field tracing**: `server.port` is app-team owned;
  `spring.datasource.*` is platform-owned — cub-gen knows the difference
- **Profile overlay tracking**: changes in `application-prod.yaml` are traced
  separately from base `application.yaml` with full lineage
- **Framework-native detection**: cub-gen recognizes `pom.xml` + Spring Boot
  structure automatically — no config file to write
- **Governance by field owner**: app-team changes auto-allow; platform-owned
  field edits require platform approval or get blocked

## How Spring Boot maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              RECONCILER (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ application.yaml    │          │ Deployment           │         │ Running JVM      │
│ application-prod    │──import─▶│ ConfigMap            │──sync──▶│ Actuator health  │
│ pom.xml             │          │ Service              │         │ Live datasource  │
│ platform/*.yaml     │          │ Kustomization (Flux) │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  Developers edit app config.      Rendered manifests with           What's actually
  Platform owns datasource +       field-origin provenance.          running.
  SLO policy.
```

**DRY** is what your team edits: Spring config files (`application*.yaml`), the
Maven build (`pom.xml`), and platform policies. These are the source of truth.

**WET** is what cub-gen produces: Kubernetes manifests with every field traced
back to its Spring config source — including which profile overlay contributed
each value.

**LIVE** is your running JVM. Flux Kustomization or ArgoCD reconciles WET
manifests to LIVE state. cub-gen doesn't touch your reconciler.

| File | Owner | What it controls |
|------|-------|-----------------|
| `pom.xml` | App team | Maven build — Spring Boot 3.3.2, Java 21 |
| `src/main/resources/application.yaml` | App team | Base config — server port, logging, app name |
| `src/main/resources/application-prod.yaml` | App + Platform | Prod overrides — port (app), datasource (platform) |
| `src/main/java/.../InventoryApplication.java` | App team | Service implementation |
| `platform/base/runtime-policy.yaml` | Platform | Required actuator health, managed datasource |
| `platform/overlays/prod/slo-policy.yaml` | Platform | Production SLO targets (99.9%, p95 250ms) |
| `platform/registry.yaml` | Platform | FrameworkRegistry typed operations + constraints for Spring platform APIs |
| `gitops/flux/kustomization.yaml` | Platform | Flux Kustomization transport |
| `gitops/argo/application.yaml` | Platform | ArgoCD Application transport |

## If you already ship Spring Boot services

This example targets teams that already standardize around Spring profiles and
application config:

- Developers own `application.yaml` behavior and feature toggles.
- Platform teams enforce datasource, SLO, and operational controls.
- Production issues still require brittle mapping from runtime fields back to
  Spring config keys.

cub-gen keeps Spring config as the source contract and makes the mapping to
runtime manifests explicit, including ownership boundaries by field.

## Why this maps cleanly to the cub-gen framework

| Existing Spring model | cub-gen concept | Why it matters |
|------|------|------|
| `application*.yaml` + profiles | DRY intent | Spring remains the authoring interface for app teams. |
| Spring-to-K8s transformation | WET targets with provenance | Each runtime field can be traced back to a Spring property. |
| Datasource and secret controls | Ownership + policy gates | Sensitive changes can be blocked/escalated before deploy. |
| Flux/Argo deployment path | LIVE state | Existing deployment runtime remains unchanged. |

## Advanced reality check: profile chains and developer workflow

Real Spring Boot shops rely on profile resolution, and that is where ownership
bugs hide. A practical trace should answer:

```
application.yaml        -> server.port = 8080   (base)
application-dev.yaml    -> server.port = 9090   (dev override)
application-prod.yaml   -> server.port = 8081   (prod override)
active profile: prod
effective value: 8081 (origin: application-prod.yaml)
```

For developer adoption, keep cub-gen in CI and return decisions in Spring terms
instead of Kubernetes terms. The useful message is:

`spring.datasource.hikari.maximum-pool-size` is platform-managed -> BLOCK.

Not:

`Deployment/spec/template/...` changed.

That is why this example treats Spring property namespaces as first-class
ownership boundaries.

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect Spring Boot project structure
./cub-gen gitops discover --space platform --json ./examples/springboot-paas

# Import with ownership-aware field tracing
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas \
  | jq '{profile: .discovered[0].generator_profile, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
```

cub-gen detects `pom.xml` + `src/main/resources/application.yaml` as a
`springboot-paas` project. The import traces field origins through Spring's
profile system and classifies each field by owner.

## Real-world scenario: database connection pool change

**Who**: An inventory team at a logistics company. 40 Spring Boot microservices.
Each has `application.yaml` for base config and `application-prod.yaml` for
production overrides.

### Scenario A — App team change (ALLOW)

The app team adds a new feature flag and changes the server port for a canary
test. These are app-owned fields:

```yaml
# application-prod.yaml (app-team fields)
server:
  port: 8082           # canary port
feature:
  inventory:
    reservationMode: optimistic  # was strict
```

```bash
# Import detects the changes
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas

# Evidence chain
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Decision engine: server.port + feature.* are app-owned → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by app-lead --reason "canary test with optimistic reservation"
```

### Scenario B — Platform-owned field edit (BLOCK)

The same app team edits the datasource configuration — a platform-owned field:

```yaml
# application-prod.yaml (platform-controlled field!)
spring:
  datasource:
    hikari:
      maximum-pool-size: 50   # was 20, platform-owned
```

```bash
# Import detects the datasource change
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas

# Evidence chain
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas > bundle.json
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Decision engine: spring.datasource.* is platform-owned → BLOCK
./cub-gen bridge decision apply --decision decision.json --state BLOCK \
  --approved-by governance-bot \
  --reason "Datasource config is platform-owned. Requires platform-dba approval."
```

The field-origin trace shows `spring.datasource.hikari.maximum-pool-size`
originates from `application-prod.yaml` but falls within the `spring.datasource.*`
namespace, which is platform-owned per the runtime policy. The app team cannot
change this without platform-dba review → **BLOCK**.

### The right way — platform-dba makes the change (ALLOW)

```bash
# Same change, but now submitted by the platform DBA team
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-dba --reason "connection pool increase for Q3 traffic"
```

## How it works

cub-gen's `springboot-paas` generator detects any directory containing `pom.xml`
(or `build.gradle`) with a Spring Boot `application.yaml`. On import:

1. **Classifies inputs** — `pom.xml` (role: build-config), `application.yaml`
   (role: app-config-base), `application-prod.yaml` (role: app-config-profile)
2. **Maps field origins** — `server.port` traces from `application.yaml` through
   Spring's profile merge to the Deployment spec (confidence: 0.92)
3. **Splits ownership** — `server.port` and `feature.*` are app-team editable;
   `spring.datasource.*` requires platform review (confidence: 0.78, review
   required)
4. **Emits inverse guidance** — "to change the feature flag in production,
   edit `application-prod.yaml`; to change the datasource pool, get
   platform-dba approval first"

A concrete field trace:

```
DRY:  application.yaml → server.port = 8080
      ↓ spring-config-to-manifest transform (confidence: 0.92)
WET:  Deployment/spec/template/spec/containers[0]/env[name=SERVER_PORT]/value = "8080"
```

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `pom.xml` | App team | Maven — Spring Boot 3.3.2, Java 21 |
| `application.yaml` | App team | Base config — port, logging, app name |
| `application-prod.yaml` | Shared | Prod overrides — port (app), datasource (platform) |
| `platform/base/runtime-policy.yaml` | Platform | Required actuator, managed datasource |
| `platform/overlays/prod/slo-policy.yaml` | Platform | SLO targets — 99.9%, p95 250ms |
| `platform/registry.yaml` | Platform | FrameworkRegistry for Spring operations/constraints |
| `gitops/flux/kustomization.yaml` | Platform | Flux Kustomization transport |
| `gitops/argo/application.yaml` | Platform | ArgoCD Application transport |

## Next steps

- **Helm version**: [`helm-paas`](../helm-paas/) — same governance for
  chart-based deployments
- **Score.dev version**: [`scoredev-paas`](../scoredev-paas/) — platform-agnostic
  workload specs
- **E2E demo**: `../demo/module-3-spring-ownership.sh`
- **Worked example**: `../../docs/agentic-gitops/03-worked-examples/03-spring-boot-dry-wet-unit-worked-example.md`

### PR-MR pairing and promotion flows

- **Flow A (Git PR → ConfigHub MR)**: `../demo/flow-a-git-pr-to-mr-connected.sh`
  — developer opens PR, ConfigHub creates MR with evidence
- **Flow B (ConfigHub MR → Git PR)**: `../demo/flow-b-mr-to-git-pr-connected.sh`
  — ConfigHub initiates change, generates Git PR after approval
- **FR8 promotion**: `../demo/fr8-promotion-upstream-dry-connected.sh`
  — promote successful app change to upstream platform DRY

## Run from ConfigHub (connected mode)

If you already have ConfigHub, start here:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"

# Publish and ingest
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map (Spring property → K8s field)
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas \
  | jq '.provenance[0].field_origin_map'

# Ownership-aware inverse pointers
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas \
  | jq '.provenance[0].inverse_edit_pointers'

# Evidence bundle
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/springboot-paas/demo-local.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/springboot-paas/demo-connected.sh
```
