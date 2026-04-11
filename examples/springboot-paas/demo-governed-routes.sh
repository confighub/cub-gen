#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/springboot-governed-routes}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
ROUTES="${ROUTES:-./examples/springboot-paas/operational/field-routes.yaml}"
ALLOW_FIELD="${ALLOW_FIELD:-feature.inventory.reservationMode}"
BLOCK_FIELD="${BLOCK_FIELD:-spring.datasource.url}"

mkdir -p "$OUT_DIR"

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[spring-routes] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

echo "[spring-routes] proof 1/2: app-owned field stays mutable"
./cub-gen springboot validate-mutation --routes "$ROUTES" "$ALLOW_FIELD" \
  | tee "$OUT_DIR/allow.txt"

echo "[spring-routes] proof 2/2: platform-owned field is blocked"
set +e
./cub-gen springboot validate-mutation --routes "$ROUTES" "$BLOCK_FIELD" \
  >"$OUT_DIR/block.txt" 2>&1
block_exit="$?"
set -e

if [ "$block_exit" -ne 1 ]; then
  echo "error: expected blocked mutation exit code 1 for $BLOCK_FIELD, got $block_exit" >&2
  exit 1
fi

cat "$OUT_DIR/block.txt"

cat <<EOF

[spring-routes] success
  allowed field: $ALLOW_FIELD
  blocked field: $BLOCK_FIELD
  artifacts: $OUT_DIR
EOF
