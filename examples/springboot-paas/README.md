# Spring Boot PaaS: The Release-Day Challenge

A major product launch is in 24 hours. Three requests land at the same time:

1. **Flip a feature flag in prod** — safe, urgent, do it now
2. **Add Redis caching** — valuable, but it requires a code change
3. **Point staging at a different database** — dangerous, must be refused

These map to the three mutation routes every platform team needs:

| Request | Route | Why |
|---------|-------|-----|
| Enable optimistic reservation mode | **Apply here** | App-owned feature flag — mutate directly in ConfigHub |
| Add Redis-backed caching | **Lift upstream** | Requires new Maven dependency and Spring config — route back to source |
| Change the staging datasource | **Block/escalate** | Platform-owned boundary — the managed datasource must not diverge |

The app is `inventory-api`, a Spring Boot 3.3.2 service (Java 21) deployed across `dev`, `stage`, and `prod`.

## Quick start

```bash
# Structural proof (no cluster, no ConfigHub needed)
./verify.sh                           # all fixtures consistent
./lift-upstream-verify.sh             # Redis bundle consistent
./block-escalate-verify.sh           # datasource boundary consistent

# cub-gen source-side path
go build -o ./cub-gen ./cmd/cub-gen
./examples/springboot-paas/demo-local.sh

# Connected ConfigHub path
cub auth login
./examples/springboot-paas/demo-connected.sh
```

## Understand the generator

Before changing anything, see how inputs become outputs:

```bash
./generator/render.sh --explain                               # what the generator does
./generator/render.sh --trace                                 # field-by-field: input → output
./generator/render.sh --explain-field feature.inventory.reservationMode  # MUTABLE
./generator/render.sh --explain-field spring.datasource.url              # BLOCKED
```

## Handle request #1: flip the feature flag

This is the **apply-here** path. The field `feature.inventory.reservationMode` is app-owned — mutate it directly in ConfigHub.

```bash
./confighub-setup.sh                  # create dev/stage/prod spaces + units

cub function do --space inventory-api-prod --unit inventory-api \
  --change-desc "release-day: reservation mode strict → optimistic" \
  set-env inventory-api "FEATURE_INVENTORY_RESERVATIONMODE=optimistic"
```

Verify it worked:

```bash
./confighub-compare.sh                    # see the * on prod's reservationMode
./confighub-refresh-preview.sh prod       # PRESERVE: your change survives refresh
cub mutation list --space inventory-api-prod --json inventory-api | \
  jq '[.[-1] | {mutationNum, description, author: .Author.Email, createdAt: .CreatedAt}]'
```

## Handle request #2: add Redis caching

This is the **lift-upstream** path. Caching requires a new Maven dependency (`spring-boot-starter-data-redis`) and new Spring config entries. That's a code change, not just a config mutation.

```bash
./lift-upstream.sh --explain          # why this routes upstream
./lift-upstream.sh --render-diff      # the exact patch bundle
```

You'll see diffs for `pom.xml` (add redis starter) and `application.yaml` (add `spring.cache.type: redis`), plus refreshed ConfigHub YAMLs. Automated PR creation is not implemented yet.

## Handle request #3: reject the datasource override

This is the **block/escalate** path. The field `spring.datasource.*` is platform-owned per `platform/base/runtime-policy.yaml`.

```bash
./block-escalate.sh --explain         # why this is blocked
./block-escalate.sh --render-attempt  # the dry-run: what would happen
```

Server-side enforcement is not yet implemented — today this is documented and previewed, not enforced.

## Real Kubernetes deployment

Deploy the real app to a Kind cluster and verify via HTTP:

```bash
./bin/create-cluster                  # Kind cluster + namespace
./bin/build-image                     # mvn package + docker build + kind load
./bin/install-worker                  # ConfigHub worker with real K8s target

./confighub-setup.sh                  # create spaces + units
cub unit apply --space inventory-api-prod inventory-api
./verify-e2e.sh                       # port-forward + curl /api/inventory/summary
```

The e2e verification checks: cluster reachable, deployment ready, pods running, `/api/inventory/summary` returns items, `/actuator/health` returns UP.

Cleanup:

```bash
./confighub-cleanup.sh
./bin/teardown
```

## The app

A real Spring Boot service with three API endpoints:

| Endpoint | What it returns |
|----------|-----------------|
| `GET /api/inventory/items` | 3 inventory items (SKU-100, SKU-200, SKU-300) |
| `GET /api/inventory/summary` | Service name, environment, reservationMode, cacheBackend, items |
| `GET /actuator/health` | Spring Boot health status |

HTTP-level tests verify both dev profile (optimistic mode, no cache) and prod profile (strict mode, redis cache):
- `src/test/java/.../InventoryControllerHttpTest.java`
- `src/test/java/.../InventoryControllerProdHttpTest.java`

## Field ownership

| Field pattern | Owner | Route | Governed by |
|---------------|-------|-------|-------------|
| `feature.inventory.*` | App team | Apply here | `operational/field-routes.yaml` |
| `spring.cache.*` | App team | Lift upstream | `operational/field-routes.yaml` |
| `spring.datasource.*` | Platform | Block/escalate | `platform/base/runtime-policy.yaml` |
| `securityContext.*` | Platform | Block/escalate | `platform/base/runtime-policy.yaml` |

## What's implemented

| Capability | Status |
|------------|--------|
| Real Spring Boot app with HTTP tests | Real |
| Generator detection (cub-gen springboot profile) | Real |
| Source-chain verification (discover/import/bridge) | Real |
| ConfigHub mutation + audit history | Real |
| Real Kubernetes delivery (Kind) | Real |
| Structural proof (verify.sh) | Real |
| Lift-upstream Redis bundle | Real (bundle only, no automated PR) |
| Block/escalate boundary | Real (documented, not server-enforced) |
| Refresh-survival preview | Real (client-side simulation) |
| Generator visibility (explain-field) | Real |
| Cross-environment comparison | Real (with fixture fallback) |

## File layout

```
springboot-paas/
  pom.xml                              # App team — Maven build
  Dockerfile                           # App team — container image
  src/main/java/.../                   # App team — Spring Boot app
  src/main/resources/application*.yaml # App team — Spring config + profiles
  src/test/java/.../                   # App team — HTTP tests
  platform/base/runtime-policy.yaml    # Platform — security + datasource policy
  platform/overlays/prod/slo-policy.yaml # Platform — SLO targets
  platform/registry.yaml              # Platform — framework operations
  gitops/flux/kustomization.yaml      # Platform — Flux transport
  gitops/argo/application.yaml        # Platform — Argo transport
  confighub/inventory-api-{dev,stage,prod}.yaml  # ConfigHub unit YAMLs
  operational/                         # Rendered K8s manifests + field routes
  bin/                                 # Infrastructure scripts (Kind, image, worker)
  lift-upstream/redis-cache/           # Redis adoption bundle
  changes/                             # Three change scenario docs
```

## If you already ship Spring Boot services

You know `application.yaml`, profiles, and the gap between "app config" and "platform config." This example makes that boundary explicit and governable:

- `feature.inventory.*` is yours — mutate it directly
- `spring.datasource.*` is the platform's — you'll be blocked
- `spring.cache.*` is yours, but adding Redis requires a source change — the tooling produces the diff bundle

The field routes in `operational/field-routes.yaml` are the machine-readable version of the ownership rules your team already argues about in code review.

## Why this maps to the cub-gen model

| Spring concept | cub-gen concept | What it enables |
|----------------|-----------------|-----------------|
| `application*.yaml` + profiles | DRY inputs | Spring remains the authoring surface for app teams |
| Platform-rendered manifests | WET outputs with provenance | Every runtime field traces back to a Spring property |
| Datasource and security policy | Ownership + governance gates | Platform-owned fields are blocked before deploy |
| Flux/Argo reconciliation | LIVE state | Existing deployment pipeline unchanged |

## Local and Connected Entrypoints

```bash
# Local/offline — source-side proof, no ConfigHub needed
./examples/springboot-paas/demo-local.sh

# Connected — requires ConfigHub auth
cub auth login
./examples/springboot-paas/demo-connected.sh
```

## Next steps

- [`helm-paas`](../helm-paas/) — same governance model for Helm charts
- [`live-reconcile`](../live-reconcile/) — prove governed output survives Flux/Argo
- [`scoredev-paas`](../scoredev-paas/) — workload-spec-first alternative
- [spring-platform](https://github.com/confighub/examples/tree/main/spring-platform) — teaching examples with visibility scripts and scaffold tooling
- [FROM-DEMO-TO-PRODUCT.md](https://github.com/confighub/examples/tree/main/spring-platform/FROM-DEMO-TO-PRODUCT.md) — concept mapping between teaching examples and this product path
