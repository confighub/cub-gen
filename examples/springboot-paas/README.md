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

## What this proves today

| Slice | Status | How to prove it now |
|-------|--------|---------------------|
| Source-side provenance and ownership | Real | `./examples/springboot-paas/demo-local.sh` |
| Deep connected bridge path | Real | `./examples/springboot-paas/demo-connected.sh` |
| Standalone live-cluster app proof | Real | `./bin/create-cluster && ./bin/build-image && ./bin/install-worker && ./verify-e2e.sh` |
| Governed route proof (`ALLOW` + `BLOCKED`) | Real but client-side | `./examples/springboot-paas/demo-governed-routes.sh` |
| Direct embedded ConfigHub payload mutation | Real but client-side | `./examples/springboot-paas/demo-embedded-config-mutation.sh` |

The strongest caveat is enforcement depth, not demo truth. The ownership
boundary is real and documented, but server-side rejection in ConfigHub is not
implemented yet.

## What this example is (and isn't)

This is a minimal but real Spring Boot application. You can build it with
Maven, run it with `spring-boot:run` (modulo the Postgres dependency), and
point `cub-gen` at it to get governance output. It exists to demonstrate
config-level ownership and field tracing, not application complexity.

cub-gen only inspects config files (`application*.yaml`, `pom.xml`,
`platform/*.yaml`). It does not parse or compile Java source. A 3-file
inventory service and a 200-class payment gateway produce the same governance
output for the same config surface. If your real app has `application.yaml`
and a Spring Boot build file, cub-gen works on it identically.

What this example **is**:
- A runnable Spring Boot project with real Java source, Maven build, and
  Spring config files
- A working target for `cub-gen gitops discover` and `cub-gen gitops import`
- A self-contained demo (`demo-local.sh`) that runs the full lifecycle locally
  with no external dependencies

What this example **is not**:
- A production application template -- the Java code is intentionally minimal
- A Kubernetes deployment -- it produces manifests but does not apply them
- A substitute for reading the conceptual model (see the relationship section
  below)

## What's in the box

**Java source** (3 files):

| File | Purpose |
|------|---------|
| `src/main/java/.../InventoryApplication.java` | Standard `@SpringBootApplication` entry point |
| `src/main/java/.../api/InventoryController.java` | `/healthz` REST endpoint |
| `src/main/java/.../service/InventoryService.java` | Reservation mode logic |

**Spring config** (3 files):

| File | What it provides |
|------|-----------------|
| `src/main/resources/application.yaml` | Base config: app name, server port, datasource, actuator |
| `src/main/resources/application-dev.yaml` | Dev profile overrides |
| `src/main/resources/application-prod.yaml` | Prod profile: port override, feature flags |

**Platform governance** (3 files):

| File | What it controls |
|------|-----------------|
| `platform/registry.yaml` | FrameworkRegistry: typed operations, constraints, validation rules |
| `platform/base/runtime-policy.yaml` | Required actuator health, managed datasource policy |
| `platform/overlays/prod/slo-policy.yaml` | Production SLO targets (99.9% availability, p95 250ms) |

**GitOps transport** (2 files):

| File | Reconciler |
|------|-----------|
| `gitops/flux/kustomization.yaml` | Flux Kustomization |
| `gitops/argo/application.yaml` | ArgoCD Application |

## Relationship to spring-platform

The [`spring-platform`](https://github.com/confighub/examples/tree/main/spring-platform)
repo teaches the conceptual model: three mutation routes (apply-here,
lift-upstream, block/escalate), field provenance, and ownership boundaries.
It uses fixed inputs and `--explain` scripts to walk through each concept.

This example is where those concepts become commands. If you have done the
spring-platform walkthrough, you already understand *why* `spring.datasource.*`
is platform-owned. Here, you run `cub-gen gitops import` and see cub-gen
*compute* that classification from the registry and policy files.

| spring-platform | springboot-paas |
|-----------------|-----------------|
| Fixed inventory-api fixture | Real (minimal) Spring Boot app |
| `render.sh --explain-field` | `cub-gen gitops import --json` |
| Hardcoded field explanations | Computed from source files |
| Three conceptual lenses (vanilla/ADT/ADTP) | One runnable target |
| Teaches the model | Runs the tooling |

You do not need spring-platform to use this example. You do not need this
example to learn from spring-platform. They complement each other.

## Quick start

```bash
# cub-gen source-side path
go build -o ./cub-gen ./cmd/cub-gen
./examples/springboot-paas/demo-local.sh

# Governed route proof
./examples/springboot-paas/demo-governed-routes.sh

# Direct embedded payload mutation proof
./examples/springboot-paas/demo-embedded-config-mutation.sh

# Connected ConfigHub path
cub auth login
./examples/springboot-paas/demo-connected.sh

# Standalone live-cluster proof
./bin/create-cluster
./bin/build-image
./bin/install-worker
./confighub-setup.sh
./verify-e2e.sh
```

Start with `./examples/demo/run-connected-smoke.sh` first to confirm the
standard ConfigHub-connected environment. Use
`./examples/springboot-paas/demo-connected.sh` when you specifically want the
deeper bridge-backed walkthrough for this example.

Fixture consistency checks and bundle-only proofs:

```bash
./verify.sh                  # all fixtures consistent
./lift-upstream-verify.sh    # Redis bundle consistent
./block-escalate-verify.sh   # datasource boundary consistent
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

This is the **apply-here** path. The field `feature.inventory.reservationMode`
is app-owned, and this example now mutates it directly inside the embedded
`application.yaml` payload instead of routing through an env-var workaround.

```bash
# Example-owned direct payload proof
./examples/springboot-paas/demo-embedded-config-mutation.sh

# Raw command underneath the wrapper
./cub-gen springboot set-embedded-config \
  --routes ./operational/field-routes.yaml \
  --file ./confighub/inventory-api-prod.yaml \
  --configmap inventory-api-config \
  feature.inventory.reservationMode optimistic
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

### Enforcement via validate-mutation

Use `cub-gen springboot validate-mutation` to enforce field routes directly, or
run the example-owned wrapper first:

```bash
./examples/springboot-paas/demo-governed-routes.sh
```

That wrapper proves both sides of the route boundary:

- `feature.inventory.reservationMode` returns `ALLOWED`
- `spring.datasource.url` returns `BLOCKED`

For the stronger direct payload path, use:

```bash
./examples/springboot-paas/demo-embedded-config-mutation.sh
```

That wrapper proves:

- `feature.inventory.reservationMode` is patched directly inside
  `ConfigMap.data["application.yaml"]`
- `confighub-compare.sh` shows the prod value diverging as expected
- `confighub-refresh-preview.sh prod` returns `PRESERVE`
- `spring.datasource.url` is still blocked when you try the same direct path

If you want the raw commands underneath it:

```bash
# Allowed: app-owned field
cub-gen springboot validate-mutation --routes ./operational/field-routes.yaml \
  feature.inventory.reservationMode
# ALLOWED (exit 0)

# Blocked: platform-owned field
cub-gen springboot validate-mutation --routes ./operational/field-routes.yaml \
  spring.datasource.url
# BLOCKED (exit 1)
```

This command reads `field-routes.yaml` and rejects mutations to fields with `defaultAction: generator-owned` (platform-owned) or `defaultAction: lift-upstream` (requires source change).

Direct embedded payload edits use the companion command:

```bash
cub-gen springboot set-embedded-config \
  --routes ./operational/field-routes.yaml \
  --file ./confighub/inventory-api-prod.yaml \
  --configmap inventory-api-config \
  feature.inventory.reservationMode optimistic
```

It edits `ConfigMap.data["application.yaml"]` in-place, after validating the
field route when `--routes` is provided.

**What is now enforced:**
- Field routes are read from `operational/field-routes.yaml`
- Mutations to `spring.datasource.*` and `securityContext.*` are blocked (exit 1)
- Mutations to `spring.cache.*` are blocked (lift-upstream: requires source change)
- Mutations to `feature.{app}.*` are allowed (exit 0)

**What is NOT yet enforced:**
- Server-side rejection in ConfigHub (this is client-side validation)
- Automatic rejection during `cub unit apply`
- Worker-side enforcement at deploy time

This is a client-side gate. Integrate it into CI/CD to enforce the boundary before mutations reach ConfigHub.

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

## Onboard your own Spring Boot app

Use `cub-gen springboot init` to generate starter cub-gen material for your app:

```bash
# Dry run: see what would be generated
cub-gen springboot init --dry-run ./path/to/your-spring-app

# Generate starter files
cub-gen springboot init --app my-service ./path/to/your-spring-app

# Or generate to a different output directory
cub-gen springboot init --app my-service --output ./my-service-cub-gen ./path/to/your-spring-app
```

This generates:
- `platform/base/runtime-policy.yaml` — platform policy skeleton
- `platform/overlays/prod/slo-policy.yaml` — SLO skeleton
- `operational/field-routes.yaml` — field ownership rules (with sensible defaults)
- `confighub/{app}-{dev,stage,prod}.yaml` — ConfigHub unit starters
- `.cub-gen/config.yaml` — generator config

### What this does NOT do

The init command generates starter skeletons, not production-ready manifests:

- It does NOT parse your actual Spring config values
- It does NOT generate actual Kubernetes manifests (deployment.yaml, etc.)
- It does NOT infer ownership beyond the default patterns
- It does NOT support every Spring Boot project shape

After init, you still need to:
1. Review and customize the generated files
2. Add actual Kubernetes manifests to `operational/`
3. Update ConfigHub unit YAMLs with correct images, ports, and config
4. Run `cub-gen gitops discover` and `cub-gen gitops import`

### Default field ownership

The generated `field-routes.yaml` uses these defaults (matching the inventory-api example):

| Pattern | Owner | Route |
|---------|-------|-------|
| `feature.{app}.*` | app-team | mutable-in-ch |
| `spring.cache.*` | app-team | lift-upstream |
| `spring.datasource.*` | platform-engineering | generator-owned (blocked) |
| `securityContext.*` | platform-engineering | generator-owned (blocked) |

Customize these based on your actual ownership boundaries.

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

Start with `./examples/demo/run-connected-smoke.sh` first to confirm the
standard ConfigHub-connected environment. This wrapper is the deeper
bridge-backed walkthrough for this example.

## Next steps

- [`helm-paas`](../helm-paas/) — same governance model for Helm charts
- [`live-reconcile`](../live-reconcile/) — prove governed output survives Flux/Argo
- [`scoredev-paas`](../scoredev-paas/) — workload-spec-first alternative
- [spring-platform](https://github.com/confighub/examples/tree/main/spring-platform) — teaching examples with visibility scripts and scaffold tooling
- [FROM-DEMO-TO-PRODUCT.md](https://github.com/confighub/examples/tree/main/spring-platform/FROM-DEMO-TO-PRODUCT.md) — concept mapping between teaching examples and this product path
