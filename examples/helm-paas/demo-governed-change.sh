#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/helm-paas-governed-change}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
WORK_REPO="$OUT_DIR/repo"
ALLOW_WET_PATH="${ALLOW_WET_PATH:-Deployment/spec/template/spec/containers[0]/image}"

mkdir -p "$OUT_DIR"
rm -rf "$WORK_REPO"

echo "[helm-governed] cloning repo into $WORK_REPO"
git clone --quiet "$ROOT_DIR" "$WORK_REPO"

cd "$WORK_REPO"
git config user.name "cub-gen demo"
git config user.email "cub-gen-demo@example.invalid"

echo "[helm-governed] build cub-gen once for both ownership-gate checks"
go build -o ./cub-gen ./cmd/cub-gen

BASE_SHA="$(git rev-parse HEAD)"

echo "[helm-governed] proof 1/2: app-team change stays on the allowed DRY path"
./cub-gen change explain --space platform --wet-path "$ALLOW_WET_PATH" ./examples/helm-paas > "$OUT_DIR/allow-explain.json"

git checkout --quiet -b demo-allow "$BASE_SHA"
perl -0pi -e 's/tag: v1\.0\.0/tag: v1.0.1/' ./examples/helm-paas/values.yaml
git add ./examples/helm-paas/values.yaml
git commit --quiet -m "demo: app team updates image tag"

SKIP_BUILD=1 ./test/checks/pr-dry-ownership-gate.sh \
  ./examples/helm-paas \
  "$BASE_SHA" \
  HEAD \
  app-team \
  --report-json "$OUT_DIR/allow-report.json" \
  >"$OUT_DIR/allow-gate.log" 2>&1

echo "[helm-governed][allow] inverse edit guidance"
jq '{
  owner: .explanation.owner,
  wet_path: .explanation.wet_path,
  dry_path: .explanation.dry_path,
  source_path: .explanation.source_path,
  confidence: .explanation.confidence,
  edit_hint: .explanation.edit_hint
}' "$OUT_DIR/allow-explain.json"

echo "[helm-governed][allow] ownership gate result"
jq '{
  status,
  changed_files,
  failures
}' "$OUT_DIR/allow-report.json"

echo "[helm-governed] proof 2/2: app-team edits the platform-owned runtime contract directly"
git checkout --quiet -B demo-block "$BASE_SHA"
perl -0pi -e 's#path: /healthz#path: /readyz#' ./examples/helm-paas/templates/deployment.yaml
git add ./examples/helm-paas/templates/deployment.yaml
git commit --quiet -m "demo: app team edits platform runtime contract"

set +e
SKIP_BUILD=1 ./test/checks/pr-dry-ownership-gate.sh \
  ./examples/helm-paas \
  "$BASE_SHA" \
  HEAD \
  app-team \
  --report-json "$OUT_DIR/block-report.json" \
  >"$OUT_DIR/block-gate.log" 2>&1
block_exit="$?"
set -e

if [ "$block_exit" -eq 0 ]; then
  echo "error: expected the platform-owned template edit to fail the ownership gate." >&2
  exit 1
fi
if [ "$block_exit" -ne 2 ]; then
  echo "error: expected ownership gate exit code 2 for the blocked change, got $block_exit." >&2
  exit 1
fi

echo "[helm-governed][block] ownership gate result"
jq '{
  status,
  changed_files,
  failures
}' "$OUT_DIR/block-report.json"

echo "[helm-governed][block] gate hint"
grep '^hint:' "$OUT_DIR/block-gate.log" || true

cat <<EOF

[helm-governed] success
  allowed path: app-team edits the chart values file and passes the ownership gate
  blocked path: app-team edits the platform-owned deployment template and is rejected
  artifacts: $OUT_DIR
EOF
