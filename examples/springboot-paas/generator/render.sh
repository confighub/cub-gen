#!/usr/bin/env bash
# Generator: Transform upstream inputs → operational config
#
# This script makes visible the transformation that platform teams perform:
#   app inputs        + platform policy    → operational config
#   (Spring config)     (runtime-policy)     (Kubernetes manifests)
#
# Usage:
#   ./render.sh --explain             Human-readable explanation
#   ./render.sh --explain-json        Machine-readable explanation
#   ./render.sh --trace               Field-by-field mapping
#   ./render.sh --diff                Show what would change if re-rendered
#   ./render.sh --explain-field FIELD Why a specific field is blocked/mutable

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${SCRIPT_DIR}/.."

explain_human() {
  cat <<'EOF'
================================================================================
                    SPRING BOOT PLATFORM GENERATOR
================================================================================

What This Generator Does
------------------------
Transforms app inputs + platform policies into operational Kubernetes config.

Inputs:
  src/main/resources/application.yaml        Base Spring Boot config
  src/main/resources/application-stage.yaml  Stage environment overrides
  src/main/resources/application-prod.yaml   Prod environment overrides
  platform/base/runtime-policy.yaml          Platform runtime rules
  platform/overlays/prod/slo-policy.yaml     Service level objectives

Outputs:
  operational/deployment.yaml    Kubernetes Deployment with env vars from policies
  operational/configmap.yaml     ConfigMap with embedded Spring configs
  operational/service.yaml       Kubernetes Service

Key Transformations
-------------------
1. APP NAME EXTRACTION
   Input:  spring.application.name: inventory-api
   Output: metadata.name: inventory-api (in all manifests)

2. PLATFORM DATASOURCE INJECTION
   Input:  runtime-policy.yaml → managedDatasource: postgres-shared
   Output: env SPRING_DATASOURCE_URL: jdbc:postgresql://postgres.platform.svc:5432/inventory

3. PORT MAPPING
   Input:  application-prod.yaml → server.port: 8081
   Output: containerPort: 8081, service targetPort: 8081

4. SPRING CONFIG EMBEDDING
   Input:  application*.yaml files (3 files)
   Output: ConfigMap data with all configs mounted at /config/

5. PROFILE ACTIVATION
   Input:  environment = prod
   Output: env SPRING_PROFILES_ACTIVE: prod

Why This Matters
----------------
The generator is where platform policy meets app inputs. ConfigHub needs to
understand this transformation so it can:

- Know which fields are mutable-in-ch (app-owned, safe to change locally)
- Know which fields should lift-upstream (need to go back to app source)
- Know which fields are generator-owned (platform-controlled, block changes)

Run: ./render.sh --trace    to see field-by-field mapping
Run: ./render.sh --diff     to see what would change if re-rendered
================================================================================
EOF
}

explain_json() {
  cat <<EOF
{
  "generator": "springboot-paas",
  "description": "Transforms Spring Boot app inputs + platform policies into Kubernetes operational config",
  "inputs": [
    {"path": "src/main/resources/application.yaml", "type": "spring-config", "owner": "app-team"},
    {"path": "src/main/resources/application-stage.yaml", "type": "spring-config", "owner": "app-team"},
    {"path": "src/main/resources/application-prod.yaml", "type": "spring-config", "owner": "app-team"},
    {"path": "platform/base/runtime-policy.yaml", "type": "platform-policy", "owner": "platform-engineering"},
    {"path": "platform/overlays/prod/slo-policy.yaml", "type": "platform-policy", "owner": "platform-engineering"}
  ],
  "outputs": [
    {"path": "operational/deployment.yaml", "type": "kubernetes", "kind": "Deployment"},
    {"path": "operational/configmap.yaml", "type": "kubernetes", "kind": "ConfigMap"},
    {"path": "operational/service.yaml", "type": "kubernetes", "kind": "Service"}
  ],
  "transformations": [
    {"name": "app_name_extraction", "from": "spring.application.name", "to": "metadata.name"},
    {"name": "datasource_injection", "from": "runtime-policy.managedDatasource", "to": "env.SPRING_DATASOURCE_URL"},
    {"name": "port_mapping", "from": "application-prod.server.port", "to": "containerPort"},
    {"name": "config_embedding", "from": "application*.yaml", "to": "ConfigMap.data"},
    {"name": "profile_activation", "from": "environment", "to": "env.SPRING_PROFILES_ACTIVE"}
  ],
  "field_routes_file": "operational/field-routes.yaml"
}
EOF
}

trace_mapping() {
  cat <<'EOF'
================================================================================
                         FIELD-BY-FIELD TRACE
================================================================================

INPUT: src/main/resources/application.yaml
------------------------------------------
spring.application.name: inventory-api
  → deployment.yaml: metadata.name = inventory-api
  → deployment.yaml: spec.selector.matchLabels.app.kubernetes.io/name = inventory-api
  → configmap.yaml: metadata.name = inventory-api-config
  → service.yaml: metadata.name = inventory-api

spring.datasource.url: jdbc:postgresql://...
  → ✗ OVERRIDDEN by platform policy (generator-owned)

feature.inventory.reservationMode: optimistic
  → configmap.yaml: data.application.yaml (embedded)
  → ✓ MUTABLE in ConfigHub (app-owned)

INPUT: src/main/resources/application-prod.yaml
-----------------------------------------------
server.port: 8081
  → deployment.yaml: spec.template.spec.containers[0].ports[0].containerPort = 8081
  → service.yaml: spec.ports[0].targetPort = 8081

feature.inventory.reservationMode: strict
  → configmap.yaml: data.application-prod.yaml (embedded)
  → ✓ MUTABLE in ConfigHub (overrides base config for prod)

INPUT: platform/base/runtime-policy.yaml
----------------------------------------
spec.managedDatasource: postgres-shared
  → deployment.yaml: env.SPRING_DATASOURCE_URL = jdbc:postgresql://postgres.platform.svc:5432/inventory
  → ✗ BLOCKED in ConfigHub (generator-owned, platform boundary)

spec.requireActuatorHealth: true
  → deployment.yaml: livenessProbe, readinessProbe (when added)
  → ✗ BLOCKED in ConfigHub (platform-controlled)

spec.runAsNonRoot: true
  → deployment.yaml: securityContext.runAsNonRoot (when added)
  → ✗ BLOCKED in ConfigHub (generator-owned)

INPUT: platform/overlays/prod/slo-policy.yaml
---------------------------------------------
spec.availabilityTarget: 99.9%
  → (used for alerting/monitoring, not in deployment)

spec.p95LatencyMs: 250
  → (used for alerting/monitoring, not in deployment)

OUTPUT: operational/deployment.yaml
-----------------------------------
env:
  SPRING_CONFIG_ADDITIONAL_LOCATION: /config/     ← generator constant
  SPRING_PROFILES_ACTIVE: prod                    ← generator constant
  SPRING_DATASOURCE_URL: jdbc:postgresql://...    ← from runtime-policy (blocked)
  CACHE_BACKEND: none                             ← generator default (lift-upstream to change)

OUTPUT: operational/configmap.yaml
----------------------------------
data:
  application.yaml      ← embedded from app inputs (mutable fields inside)
  application-stage.yaml← embedded from app inputs
  application-prod.yaml ← embedded from app inputs

OUTPUT: operational/service.yaml
--------------------------------
spec.ports[0].targetPort: 8081  ← from application-prod.yaml server.port

================================================================================
EOF
}

explain_field() {
  local field="${1:-}"
  if [[ -z "${field}" ]]; then
    echo "Usage: $0 --explain-field <field-pattern>"
    echo "Example: $0 --explain-field spring.datasource.url"
    exit 2
  fi

  echo "================================================================================"
  echo "                    FIELD EXPLANATION: ${field}"
  echo "================================================================================"
  echo ""

  case "${field}" in
    feature.inventory.*|feature.inventory.reservationMode)
      cat <<'EOF'
Field Pattern: feature.inventory.*
Owner: app-team
Route: mutable-in-ch ✓

Generator Transformation:
  Input:  src/main/resources/application.yaml
  Output: operational/configmap.yaml (embedded in data)

Why Mutable:
  Per-deployment rollout tuning is app-team safe. These fields control
  feature flags that don't require upstream code changes or platform
  re-rendering.

Example mutation:
  cub-gen springboot set-embedded-config \
    --routes ./operational/field-routes.yaml \
    --file ./confighub/inventory-api-prod.yaml \
    --configmap inventory-api-config \
    feature.inventory.reservationMode optimistic

What happens:
  1. The embedded ConfigMap payload is patched directly at the app-owned field
  2. Field survives future generator refreshes (PRESERVE policy)
  3. The same route still blocks platform-owned embedded fields
EOF
      ;;
    spring.cache.*|spring.cache.type)
      cat <<'EOF'
Field Pattern: spring.cache.*
Owner: app-team
Route: lift-upstream ↑

Generator Transformation:
  Input:  src/main/resources/application.yaml
  Output: operational/configmap.yaml (embedded) + deployment.yaml env

Why Lift-Upstream:
  Cache adoption changes the app contract and requires upstream code
  changes. Adding Redis caching needs pom.xml dependencies and
  application.yaml configuration that the generator must re-process.

What happens:
  1. ConfigHub captures the intent
  2. A lift-upstream bundle is produced (automated PR not yet implemented)
  3. Developer creates a PR with the bundle contents
  4. After merge, platform re-renders operational config
  5. ConfigHub refreshes from the new state

Preview the bundle:
  ./lift-upstream.sh --render-diff
EOF
      ;;
    spring.datasource.*|spring.datasource.url)
      cat <<'EOF'
Field Pattern: spring.datasource.*
Owner: platform-engineering
Route: generator-owned ✗ BLOCKED

Generator Transformation:
  Input:  platform/base/runtime-policy.yaml → managedDatasource: postgres-shared
  Output: operational/deployment.yaml → env.SPRING_DATASOURCE_URL

Why Blocked:
  The generator injects this field from platform policy. It's not in the
  app inputs at all - it comes from runtime-policy.yaml. The platform
  provides a managed PostgreSQL service with HA, encryption, and backups.

What happens if you try to change it:
  The boundary is documented; server-side enforcement is not yet implemented.
  In production, this mutation would be blocked or escalated.

See the boundary:
  ./block-escalate.sh --render-attempt
EOF
      ;;
    securityContext.*|securityContext.runAsNonRoot)
      cat <<'EOF'
Field Pattern: securityContext.*
Owner: platform-engineering
Route: generator-owned ✗ BLOCKED

Generator Transformation:
  Input:  platform/base/runtime-policy.yaml → runAsNonRoot: true
  Output: operational/deployment.yaml → securityContext

Why Blocked:
  Runtime hardening is platform-controlled. The generator injects
  security defaults that must not diverge per-deployment.
EOF
      ;;
    *)
      cat <<EOF
Field Pattern: ${field}
Status: Unknown

This field is not explicitly listed in the generator's field routes.

To understand where a field comes from:
  ./generator/render.sh --trace | grep -i "${field}"

Check operational/field-routes.yaml for the routing rules.
EOF
      ;;
  esac

  echo ""
  echo "================================================================================"
}

show_diff() {
  echo "=================================================================================="
  echo "                         RE-RENDER DIFF PREVIEW"
  echo "=================================================================================="
  echo ""
  echo "If the generator re-rendered operational/ from current inputs,"
  echo "here is what would change:"
  echo ""

  for output in deployment configmap service; do
    echo "operational/${output}.yaml:"
    echo "  - No changes (inputs match current operational files)"
    echo ""
  done

  echo "Note: Pre-rendered operational files are already in sync."
  echo "In a real platform, this would show drift between inputs and outputs."
  echo ""
  echo "=================================================================================="
}

case "${1:-}" in
  --explain)
    explain_human
    ;;
  --explain-json)
    explain_json
    ;;
  --trace)
    trace_mapping
    ;;
  --diff)
    show_diff
    ;;
  --explain-field)
    explain_field "${2:-}"
    ;;
  *)
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  --explain             Human-readable explanation of the generator"
    echo "  --explain-json        Machine-readable explanation"
    echo "  --trace               Field-by-field mapping from inputs to outputs"
    echo "  --diff                Show what would change if re-rendered"
    echo "  --explain-field FIELD Why a specific field is blocked/mutable"
    echo ""
    echo "Examples:"
    echo "  $0 --explain-field spring.datasource.url"
    echo "  $0 --explain-field feature.inventory.reservationMode"
    exit 2
    ;;
esac
