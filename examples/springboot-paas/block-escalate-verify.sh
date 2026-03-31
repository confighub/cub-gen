#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

fail() { echo "FAIL: $*" >&2; exit 1; }

grep -q 'defaultAction: generator-owned' "$ROOT_DIR/operational/field-routes.yaml" \
  || fail "field-routes.yaml missing generator-owned action for datasource"

grep -q 'spring.datasource' "$ROOT_DIR/operational/field-routes.yaml" \
  || fail "field-routes.yaml missing datasource route"

grep -q 'managedDatasource' "$ROOT_DIR/platform/base/runtime-policy.yaml" \
  || fail "runtime-policy.yaml missing managedDatasource"

ATTEMPT_OUTPUT=$("$ROOT_DIR/block-escalate.sh" --render-attempt 2>&1)
echo "$ATTEMPT_OUTPUT" | grep -q 'SPRING_DATASOURCE_URL' \
  || fail "render-attempt missing datasource env var"
echo "$ATTEMPT_OUTPUT" | grep -q 'BLOCK or ESCALATE' \
  || fail "render-attempt missing block/escalate decision"

echo "ok: block-escalate boundary is consistent"
