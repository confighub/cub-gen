#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUNDLE_DIR="$ROOT_DIR/lift-upstream/redis-cache"

case "${1:-}" in
  --explain)
    cat <<'EOF'
lift-upstream: Redis cache adoption

This bundle shows the "after" state when the inventory-api service
adopts Redis-backed caching. The changes span both upstream app inputs
and the platform-rendered ConfigHub YAMLs.

Upstream changes:
  - pom.xml gains spring-boot-starter-data-redis
  - application.yaml gains spring.cache.type: redis

ConfigHub changes:
  - All three env YAMLs update CACHE_BACKEND from none to redis
  - ConfigMap gains spring.cache.type: redis

Use --render-diff to see the full patch.
EOF
    ;;
  --render-diff)
    echo "=== Upstream app diff ==="
    diff -u "$ROOT_DIR/pom.xml" "$BUNDLE_DIR/upstream-app/pom.xml" || true
    echo ""
    echo "=== Application config diff ==="
    diff -u "$ROOT_DIR/src/main/resources/application.yaml" "$BUNDLE_DIR/upstream-app/src/main/resources/application.yaml" || true
    echo ""
    echo "=== ConfigHub dev diff ==="
    diff -u "$ROOT_DIR/confighub/inventory-api-dev.yaml" "$BUNDLE_DIR/confighub/inventory-api-dev.yaml" || true
    echo ""
    echo "=== ConfigHub stage diff ==="
    diff -u "$ROOT_DIR/confighub/inventory-api-stage.yaml" "$BUNDLE_DIR/confighub/inventory-api-stage.yaml" || true
    echo ""
    echo "=== ConfigHub prod diff ==="
    diff -u "$ROOT_DIR/confighub/inventory-api-prod.yaml" "$BUNDLE_DIR/confighub/inventory-api-prod.yaml" || true
    ;;
  *)
    echo "Usage: $0 [--explain | --render-diff]" >&2
    exit 2
    ;;
esac
