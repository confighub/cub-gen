#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

RECONCILER="${RECONCILER:-both}"  # flux|argo|both
EXAMPLE_SLUG="${EXAMPLE_SLUG:-helm-paas}"
REPO_PATH="${REPO_PATH:-./examples/helm-paas}"
RENDER_TARGET="${RENDER_TARGET:-$REPO_PATH}"
TARGET_NS="${TARGET_NS:-demo-live-helm}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-payments-api}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/e2e-connected-governed-reconcile-helm}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
REPO_URL="${REPO_URL:-https://github.com/confighub/cub-gen}"
REPO_BRANCH="${REPO_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}"

case "$RECONCILER" in
  flux|argo|both) ;;
  *)
    echo "error: unsupported RECONCILER value: $RECONCILER (expected flux|argo|both)" >&2
    exit 1
    ;;
esac

if ! git ls-remote --exit-code --heads origin "$REPO_BRANCH" >/dev/null 2>&1; then
  echo "[e2e] branch '$REPO_BRANCH' not found on origin; falling back to 'main' for reconciler source"
  REPO_BRANCH="main"
fi

mkdir -p "$OUT_DIR"

echo "[e2e] connected preflight (requires cub auth login)"
require_connected_preflight
print_connected_context

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[e2e] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

echo "[e2e] phase 1/3: run real connected governed lifecycle for $EXAMPLE_SLUG"
LIFECYCLE_DIR="$OUT_DIR/lifecycle"
SKIP_BUILD=1 ./examples/demo/simulate-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$LIFECYCLE_DIR"

CHANGE_ID="$(jq -r .change_id "$LIFECYCLE_DIR/update/bundle.json")"
BUNDLE_DIGEST="$(jq -r .bundle_digest "$LIFECYCLE_DIR/update/bundle.json")"
INGEST_STATUS="$(jq -r '.status // "unknown"' "$LIFECYCLE_DIR/update/ingest.json")"
DECISION_STATE="$(jq -r '.state // "UNKNOWN"' "$LIFECYCLE_DIR/update/decision-final.json")"

echo "[e2e] phase 2/3: run live reconciler proof(s) from helm-paas rendered fixture manifests"
flux_ok=false
argo_ok=false

if [ "$RECONCILER" = "flux" ] || [ "$RECONCILER" = "both" ]; then
  echo "[e2e][flux] create -> update -> drift-correction"
  if REPO_URL="$REPO_URL" \
    REPO_BRANCH="$REPO_BRANCH" \
    PATH_V1="./examples/live-reconcile/helm-paas/manifests-v1" \
    PATH_V2="./examples/live-reconcile/helm-paas/manifests-v2" \
    TARGET_NS="$TARGET_NS" \
    DEPLOYMENT_NAME="$DEPLOYMENT_NAME" \
    SOURCE_NAME="cub-gen-live-helm" \
    KUSTOM_NAME="cub-gen-live-helm" \
    ./examples/demo/e2e-live-reconcile-flux.sh | tee "$OUT_DIR/flux.log"; then
    flux_ok=true
  fi
fi

if [ "$RECONCILER" = "argo" ] || [ "$RECONCILER" = "both" ]; then
  echo "[e2e][argo] create -> update -> drift-correction"
  if REPO_URL="$REPO_URL" \
    REPO_BRANCH="$REPO_BRANCH" \
    PATH_V1="examples/live-reconcile/helm-paas/manifests-v1" \
    PATH_V2="examples/live-reconcile/helm-paas/manifests-v2" \
    TARGET_NS="$TARGET_NS" \
    DEPLOYMENT_NAME="$DEPLOYMENT_NAME" \
    APP_NAME="cub-gen-live-helm" \
    ./examples/demo/e2e-live-reconcile-argo.sh | tee "$OUT_DIR/argo.log"; then
    argo_ok=true
  fi
fi

echo "[e2e] phase 3/3: summary"
jq -n \
  --arg story "connected-governed-reconcile-helm" \
  --arg example "$EXAMPLE_SLUG" \
  --arg reconciler "$RECONCILER" \
  --arg change_id "$CHANGE_ID" \
  --arg bundle_digest "$BUNDLE_DIGEST" \
  --arg ingest_status "$INGEST_STATUS" \
  --arg decision_state "$DECISION_STATE" \
  --arg repo_url "$REPO_URL" \
  --arg repo_branch "$REPO_BRANCH" \
  --arg target_ns "$TARGET_NS" \
  --arg deployment "$DEPLOYMENT_NAME" \
  --argjson flux_ok "$flux_ok" \
  --argjson argo_ok "$argo_ok" \
  '{
    story: $story,
    example: $example,
    reconciler: $reconciler,
    governed_change: {
      change_id: $change_id,
      bundle_digest: $bundle_digest,
      ingest_status: $ingest_status,
      decision_state: $decision_state
    },
    live_reconcile: {
      repo_url: $repo_url,
      repo_branch: $repo_branch,
      target_namespace: $target_ns,
      deployment: $deployment,
      flux_ok: $flux_ok,
      argo_ok: $argo_ok
    }
  }' | tee "$OUT_DIR/summary.json"

if [ "$RECONCILER" = "flux" ] && [ "$flux_ok" != "true" ]; then
  exit 1
fi
if [ "$RECONCILER" = "argo" ] && [ "$argo_ok" != "true" ]; then
  exit 1
fi
if [ "$RECONCILER" = "both" ] && { [ "$flux_ok" != "true" ] || [ "$argo_ok" != "true" ]; }; then
  exit 1
fi

echo "[e2e] success: connected governance + live reconcile proved"
echo "[e2e] artifacts: $OUT_DIR"
