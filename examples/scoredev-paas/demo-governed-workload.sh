#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/scoredev-governed-workload}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
WORK_REPO="$OUT_DIR/repo"
ALLOW_WET_PATH="${ALLOW_WET_PATH:-Deployment/spec/template/spec/containers[name=main]/image}"

mkdir -p "$OUT_DIR"
rm -rf "$WORK_REPO"

echo "[score-governed] cloning repo into $WORK_REPO"
git clone --quiet "$ROOT_DIR" "$WORK_REPO"

cd "$WORK_REPO"
git config user.name "cub-gen demo"
git config user.email "cub-gen-demo@example.invalid"

echo "[score-governed] build cub-gen once for both workload checks"
go build -o ./cub-gen ./cmd/cub-gen

echo "[score-governed] proof 1/2: app team updates the image and stays within the approved contract"
perl -0pi -e 's#ghcr.io/example/checkout-api:v1\.0\.0#ghcr.io/example/checkout-api:v1.0.1#' ./examples/scoredev-paas/score.yaml

./cub-gen change explain --space platform --wet-path "$ALLOW_WET_PATH" ./examples/scoredev-paas > "$OUT_DIR/allow-explain.json"
./cub-gen score validate-workload \
  --score ./examples/scoredev-paas/score.yaml \
  --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml \
  > "$OUT_DIR/allow.txt"

echo "[score-governed][allow] inverse edit guidance"
jq '{
  owner: .explanation.owner,
  wet_path: .explanation.wet_path,
  dry_path: .explanation.dry_path,
  source_path: .explanation.source_path,
  confidence: .explanation.confidence,
  edit_hint: .explanation.edit_hint
}' "$OUT_DIR/allow-explain.json"

echo "[score-governed][allow] workload contract result"
cat "$OUT_DIR/allow.txt"

echo "[score-governed] proof 2/2: app team adds an unapproved resource type"
perl -0pi -e 's/  dns:\n    type: dns/  dns:\n    type: dns\n  ml-gpu:\n    type: gpu-pool/' ./examples/scoredev-paas/score.yaml

set +e
./cub-gen score validate-workload \
  --score ./examples/scoredev-paas/score.yaml \
  --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml \
  > "$OUT_DIR/escalate.txt" 2>&1
escalate_exit="$?"
set -e

if [ "$escalate_exit" -ne 1 ]; then
  echo "error: expected ESCALATE path to exit 1, got $escalate_exit" >&2
  exit 1
fi

echo "[score-governed][escalate] workload contract result"
cat "$OUT_DIR/escalate.txt"

cat <<EOF

[score-governed] success
  allowed path: image change stays app-owned and workload contract returns ALLOW
  escalate path: unapproved resource type requires platform review
  artifacts: $OUT_DIR
EOF
