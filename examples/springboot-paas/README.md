# Spring Boot PaaS (Java Service)

**Pattern: Java service with business config and platform-owned runtime policy — app teams edit application.yaml, platform teams control datasource and SLO boundaries.**

## 1. What is this?

An inventory API team maintains a Spring Boot service. They control business configuration — server ports, feature flags, logging levels — in `application.yaml` and `application-prod.yaml`. The platform team owns runtime policy (resource limits, required probes) and SLO targets. The boundary between app and platform config is explicit and governed.

This is the Spring Boot pattern: the `application.yaml` is the DRY source, but not all keys are app-owned. Datasource configuration, for example, is platform-controlled even though it lives in the same file format.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **App team** | `src/main/resources/application*.yaml` — business config | `server.port`, `feature.*`, logging levels |
| **App team** | `src/main/java/*` — service implementation | Java source code |
| **Platform team** | `platform/base/runtime-policy.yaml` | Required probes, resource limits |
| **Platform team** | `platform/overlays/prod/slo-policy.yaml` | Production SLO requirements |
| **GitOps reconciler** | Flux Kustomization / ArgoCD Application | Reconciles WET to LIVE |

## 3. What does cub-gen add?

- **Generator detection**: recognizes `pom.xml` + Spring Boot structure as `springboot-paas` profile (capabilities: `render-app-config`, `profile-overrides`, `inverse-app-config-patch`)
- **DRY/WET mapping**: application config (DRY) → rendered ConfigMaps, Deployments (WET)
- **Field-origin tracing**: `server.port` traces to `application.yaml`, `spring.datasource.*` traces to `application-prod.yaml` overlay
- **Inverse-edit guidance**: "to change the feature flag in production, edit `application-prod.yaml`"

```bash
./cub-gen gitops discover --space platform --json ./examples/springboot-paas
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas \
  | jq '{profile: .discovered[0].generator_profile, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
```

## 4. How do I run it?

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/springboot-paas
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas > /tmp/spring-bundle.json
./cub-gen verify --in /tmp/spring-bundle.json
./cub-gen attest --in /tmp/spring-bundle.json --verifier ci-bot > /tmp/spring-attestation.json
./cub-gen verify-attestation --in /tmp/spring-attestation.json --bundle /tmp/spring-bundle.json
./cub-gen gitops cleanup --space platform ./examples/springboot-paas
```

## 5. Real-world example using ConfigHub

A logistics company has 40 Spring Boot microservices. Each service has `application.yaml` for base config and `application-prod.yaml` for production overrides.

**Scenario: Database connection pool change**

The DBA team determines the inventory service needs a larger connection pool. This is platform-owned config (datasource settings), not app-owned. They update `application-prod.yaml`:

```yaml
spring:
  datasource:
    hikari:
      maximum-pool-size: 30   # was 20
```

**Governed pipeline:**

```bash
# 1. cub-gen detects the datasource change
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas
# Field-origin: spring.datasource.hikari.maximum-pool-size changed in application-prod.yaml

# 2. Produce evidence chain
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 3. ConfigHub ingests and evaluates
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example
# Decision engine checks: datasource changes require platform-owner approval
# (app-team edits to server.port or feature.* would auto-ALLOW)
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-dba --reason "connection pool increase for Q3 traffic"
```

**What ConfigHub provides:**
- **Ownership-aware decisions**: datasource changes escalate to platform-owner, feature flag changes auto-allow for app-team
- **Cross-service audit**: "which services have `maximum-pool-size > 20`?" — cross-repo query
- **SLO linkage**: the SLO policy in `platform/overlays/prod/slo-policy.yaml` can reference datasource settings, linking performance targets to configuration
- **Drift detection**: if someone edits the connection pool on a running cluster without going through the governed pipeline, ConfigHub flags the drift

## Narrative turns

1. **Feature rollout** — App team changes `server.port` and feature flags in `application-prod.yaml`. Import shows these as app-team DRY with inverse pointers.
2. **Platform safety check** — Platform verifies datasource/runtime boundaries and policy evidence. Governance decision is explicit before apply.
3. **Runtime reconciliation** — Flux/Argo reconciles rendered WET manifests from Git/OCI.
4. **Upstream promotion** — Reusable app-level defaults can be promoted to platform base after successful rollout.

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `pom.xml` | App team | Maven config — Spring Boot 3.3.2, Java 21 |
| `src/main/resources/application.yaml` | App team | App defaults — server port, logging, feature flags |
| `src/main/resources/application-prod.yaml` | App/Platform | Prod overrides — datasource (platform), features (app) |
| `src/main/java/com/example/inventory/` | App team | Java service implementation |
| `platform/base/runtime-policy.yaml` | Platform team | Runtime policy — probes, resource limits |
| `platform/overlays/prod/slo-policy.yaml` | Platform team | Production SLO targets |
| `gitops/flux/kustomization.yaml` | Platform team | Flux Kustomization transport |
| `gitops/argo/application.yaml` | Platform team | ArgoCD Application transport |
| `docs/user-stories.md` | — | Narrative user stories |
