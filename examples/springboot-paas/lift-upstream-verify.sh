#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUNDLE="$ROOT_DIR/lift-upstream/redis-cache"

fail() { echo "FAIL: $*" >&2; exit 1; }

[[ -f "$BUNDLE/upstream-app/pom.xml" ]] || fail "missing Redis bundle pom.xml"
[[ -f "$BUNDLE/upstream-app/src/main/resources/application.yaml" ]] || fail "missing Redis bundle application.yaml"

grep -q 'spring-boot-starter-data-redis' "$BUNDLE/upstream-app/pom.xml" \
  || fail "Redis bundle pom.xml missing redis starter dependency"

grep -q 'type: redis' "$BUNDLE/upstream-app/src/main/resources/application.yaml" \
  || fail "Redis bundle application.yaml missing cache type"

DIFF_OUTPUT=$("$ROOT_DIR/lift-upstream.sh" --render-diff 2>&1)
echo "$DIFF_OUTPUT" | grep -q 'spring-boot-starter-data-redis' \
  || fail "render-diff missing redis starter"
echo "$DIFF_OUTPUT" | grep -q 'cache' \
  || fail "render-diff missing cache changes"

echo "ok: lift-upstream Redis bundle is consistent"
