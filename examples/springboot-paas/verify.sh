#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROUTES_YAML="$ROOT_DIR/operational/field-routes.yaml"

required_files=(
  "$ROOT_DIR/README.md"
  "$ROUTES_YAML"
  "$ROOT_DIR/pom.xml"
  "$ROOT_DIR/Dockerfile"
  "$ROOT_DIR/src/main/java/com/example/inventory/InventoryApplication.java"
  "$ROOT_DIR/src/main/java/com/example/inventory/api/InventoryController.java"
  "$ROOT_DIR/src/main/java/com/example/inventory/api/InventoryFeatureProperties.java"
  "$ROOT_DIR/src/main/java/com/example/inventory/api/InventoryRuntimeProperties.java"
  "$ROOT_DIR/src/main/java/com/example/inventory/api/InventoryItem.java"
  "$ROOT_DIR/src/main/java/com/example/inventory/api/InventorySummary.java"
  "$ROOT_DIR/src/main/java/com/example/inventory/api/InventoryService.java"
  "$ROOT_DIR/src/main/resources/application.yaml"
  "$ROOT_DIR/src/main/resources/application-stage.yaml"
  "$ROOT_DIR/src/main/resources/application-prod.yaml"
  "$ROOT_DIR/src/test/java/com/example/inventory/api/InventoryControllerHttpTest.java"
  "$ROOT_DIR/src/test/java/com/example/inventory/api/InventoryControllerProdHttpTest.java"
  "$ROOT_DIR/platform/base/runtime-policy.yaml"
  "$ROOT_DIR/platform/overlays/prod/slo-policy.yaml"
  "$ROOT_DIR/operational/configmap.yaml"
  "$ROOT_DIR/operational/deployment.yaml"
  "$ROOT_DIR/operational/service.yaml"
  "$ROOT_DIR/changes/01-mutable-in-ch.md"
  "$ROOT_DIR/changes/02-lift-upstream.md"
  "$ROOT_DIR/changes/03-generator-owned.md"
  "$ROOT_DIR/block-escalate.sh"
  "$ROOT_DIR/block-escalate-verify.sh"
  "$ROOT_DIR/lift-upstream.sh"
  "$ROOT_DIR/lift-upstream-verify.sh"
  "$ROOT_DIR/lift-upstream/redis-cache/upstream-app/pom.xml"
  "$ROOT_DIR/lift-upstream/redis-cache/upstream-app/src/main/resources/application.yaml"
  "$ROOT_DIR/lift-upstream/redis-cache/confighub/inventory-api-dev.yaml"
  "$ROOT_DIR/lift-upstream/redis-cache/confighub/inventory-api-stage.yaml"
  "$ROOT_DIR/lift-upstream/redis-cache/confighub/inventory-api-prod.yaml"
  "$ROOT_DIR/confighub-setup.sh"
  "$ROOT_DIR/confighub-cleanup.sh"
  "$ROOT_DIR/confighub-verify.sh"
  "$ROOT_DIR/verify-e2e.sh"
  "$ROOT_DIR/bin/create-cluster"
  "$ROOT_DIR/bin/build-image"
  "$ROOT_DIR/bin/install-worker"
  "$ROOT_DIR/bin/teardown"
  "$ROOT_DIR/var/.gitignore"
  "$ROOT_DIR/confighub/inventory-api-dev.yaml"
  "$ROOT_DIR/confighub/inventory-api-stage.yaml"
  "$ROOT_DIR/confighub/inventory-api-prod.yaml"
)

for file in "${required_files[@]}"; do
  [[ -f "$file" ]] || {
    echo "missing required file: $file" >&2
    exit 1
  }
done

grep -q 'defaultAction: mutable-in-ch' "$ROUTES_YAML"
grep -q 'defaultAction: lift-upstream' "$ROUTES_YAML"
grep -q 'defaultAction: generator-owned' "$ROUTES_YAML"

echo "ok: springboot-paas fixtures are consistent"
