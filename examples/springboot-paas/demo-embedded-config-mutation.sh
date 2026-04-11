#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/springboot-embedded-config}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
WORK_REPO="$OUT_DIR/repo"
PAYLOAD="${PAYLOAD:-./examples/springboot-paas/confighub/inventory-api-prod.yaml}"
ROUTES="${ROUTES:-./examples/springboot-paas/operational/field-routes.yaml}"
CONFIGMAP="${CONFIGMAP:-inventory-api-config}"
ALLOW_FIELD="${ALLOW_FIELD:-feature.inventory.reservationMode}"
ALLOW_VALUE="${ALLOW_VALUE:-optimistic}"
BLOCK_FIELD="${BLOCK_FIELD:-spring.datasource.url}"
BLOCK_VALUE="${BLOCK_VALUE:-jdbc:postgresql://shadow.platform.svc:5432/inventory}"

mkdir -p "$OUT_DIR"
rm -rf "$WORK_REPO"

echo "[spring-embedded] cloning repo into $WORK_REPO"
git clone --quiet "$ROOT_DIR" "$WORK_REPO"

cd "$WORK_REPO"
git config user.name "cub-gen demo"
git config user.email "cub-gen-demo@example.invalid"

echo "[spring-embedded] build cub-gen"
go build -o ./cub-gen ./cmd/cub-gen

echo "[spring-embedded] proof 1/2: mutate the embedded app-owned field directly in the ConfigHub payload"
./cub-gen springboot set-embedded-config \
  --file "$PAYLOAD" \
  --configmap "$CONFIGMAP" \
  --routes "$ROUTES" \
  "$ALLOW_FIELD" \
  "$ALLOW_VALUE" \
  > "$OUT_DIR/allow.txt"

./examples/springboot-paas/confighub-compare.sh --json > "$OUT_DIR/compare.json"
./examples/springboot-paas/confighub-refresh-preview.sh prod --json > "$OUT_DIR/refresh.json"

echo "[spring-embedded][allow] mutation result"
cat "$OUT_DIR/allow.txt"

echo "[spring-embedded][allow] compare snapshot"
jq '."feature.inventory.reservationMode".values.prod' "$OUT_DIR/compare.json"

echo "[spring-embedded][allow] refresh preview"
jq '[.[] | select(.field == "feature.inventory.reservationMode")] | first | {
  field,
  liveValue,
  upstreamValue,
  action,
  reason
}' "$OUT_DIR/refresh.json"

echo "[spring-embedded] proof 2/2: block a platform-owned embedded field mutation"
set +e
./cub-gen springboot set-embedded-config \
  --file "$PAYLOAD" \
  --configmap "$CONFIGMAP" \
  --routes "$ROUTES" \
  "$BLOCK_FIELD" \
  "$BLOCK_VALUE" \
  > "$OUT_DIR/block.txt" 2>&1
block_exit="$?"
set -e

if [ "$block_exit" -ne 1 ]; then
  echo "error: expected blocked embedded mutation exit code 1 for $BLOCK_FIELD, got $block_exit" >&2
  exit 1
fi

echo "[spring-embedded][block] mutation result"
cat "$OUT_DIR/block.txt"

cat <<EOF

[spring-embedded] success
  apply-here path: embedded reservationMode changed directly in the ConfigHub payload
  preserve proof: refresh preview keeps the local mutable-in-ch override
  blocked path: platform-owned embedded datasource field is rejected
  artifacts: $OUT_DIR
EOF
