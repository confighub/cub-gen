#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/springboot-governed-routes}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
ROUTES="${ROUTES:-./examples/springboot-paas/operational/field-routes.yaml}"
ALLOW_FIELD="${ALLOW_FIELD:-feature.inventory.reservationMode}"
LIFT_FIELD="${LIFT_FIELD:-spring.cache.type}"
BLOCK_FIELD="${BLOCK_FIELD:-spring.datasource.url}"

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required for spring route proof" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[spring-routes] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

echo "[spring-routes] proof 1/3: app-owned field stays mutable"
./cub-gen gate mutation --routes "$ROUTES" --json "$ALLOW_FIELD" \
  | tee "$OUT_DIR/allow.json"

jq -e '.route.kind == "apply-here" and .decision.state == "ALLOW"' "$OUT_DIR/allow.json" >/dev/null

echo "[spring-routes] proof 2/3: source change is lifted upstream"
./cub-gen gate mutation --routes "$ROUTES" --json "$LIFT_FIELD" \
  | tee "$OUT_DIR/lift.json"

jq -e '.route.kind == "lift-upstream" and .decision.state == "ESCALATE"' "$OUT_DIR/lift.json" >/dev/null

echo "[spring-routes] proof 3/3: platform-owned field is blocked"
set +e
./cub-gen gate mutation --routes "$ROUTES" --json --enforce "$BLOCK_FIELD" \
  >"$OUT_DIR/block.json" 2>"$OUT_DIR/block.err"
block_exit="$?"
set -e

if [ "$block_exit" -eq 0 ]; then
  echo "error: expected blocked mutation exit code 1 for $BLOCK_FIELD, got $block_exit" >&2
  exit 1
fi

jq -e '.route.kind == "block/escalate" and .decision.state == "BLOCK"' "$OUT_DIR/block.json" >/dev/null
cat "$OUT_DIR/block.json"

cat <<EOF

[spring-routes] success
  allowed field: $ALLOW_FIELD
  lift-upstream field: $LIFT_FIELD
  blocked field: $BLOCK_FIELD
  artifacts: $OUT_DIR
EOF
