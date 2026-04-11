#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

RECONCILER="${RECONCILER:-both}" # flux|argo|both
TARGET_NS="${TARGET_NS:-demo-live-helm}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-payments-api}"
LIVE_NAME="${LIVE_NAME:-helm-paas-live}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/helm-paas-runtime}"
FLUX_CONTEXT="${FLUX_CONTEXT:-kind-${FLUX_CLUSTER_NAME:-${CLUSTER_NAME:-cub-gen-live}}}"
ARGO_CONTEXT="${ARGO_CONTEXT:-kind-${ARGO_CLUSTER_NAME:-${CLUSTER_NAME:-cub-gen-live-argo}}}"

print_flux_snapshot() {
  echo
  echo "[helm-runtime][flux] live deployment snapshot"
  kubectl --context "$FLUX_CONTEXT" -n "$TARGET_NS" get deploy,pods,svc
  kubectl --context "$FLUX_CONTEXT" -n flux-system get kustomization "$LIVE_NAME" -o json \
    | jq '{
        name: .metadata.name,
        ready: (([.status.conditions[]? | select(.type=="Ready") | .status] | first) // "Unknown")
      }'
}

print_argo_snapshot() {
  echo
  echo "[helm-runtime][argo] live deployment snapshot"
  kubectl --context "$ARGO_CONTEXT" -n "$TARGET_NS" get deploy,pods,svc
  kubectl --context "$ARGO_CONTEXT" -n argocd get application "$LIVE_NAME" -o json \
    | jq '{
        name: .metadata.name,
        sync: (.status.sync.status // "Unknown"),
        health: (.status.health.status // "Unknown")
      }'
}

echo "[helm-runtime] connected governance + live reconcile proof for helm-paas"
APP_NAME="$LIVE_NAME" \
SOURCE_NAME="$LIVE_NAME" \
KUSTOM_NAME="$LIVE_NAME" \
EXAMPLE_SLUG="helm-paas" \
REPO_PATH="./examples/helm-paas" \
RENDER_TARGET="./examples/helm-paas" \
TARGET_NS="$TARGET_NS" \
DEPLOYMENT_NAME="$DEPLOYMENT_NAME" \
OUT_ROOT="$OUT_ROOT" \
RECONCILER="$RECONCILER" \
./examples/demo/e2e-connected-governed-reconcile-helm.sh

case "$RECONCILER" in
  flux)
    print_flux_snapshot
    NEXT_CONTEXT_LINES="  kubectl --context ${FLUX_CONTEXT} -n ${TARGET_NS} get deployment ${DEPLOYMENT_NAME} -o yaml"
    ;;
  argo)
    print_argo_snapshot
    NEXT_CONTEXT_LINES="  kubectl --context ${ARGO_CONTEXT} -n ${TARGET_NS} get deployment ${DEPLOYMENT_NAME} -o yaml"
    ;;
  both)
    print_flux_snapshot
    print_argo_snapshot
    NEXT_CONTEXT_LINES="$(cat <<EOF_BOTH
  kubectl --context ${FLUX_CONTEXT} -n ${TARGET_NS} get deployment ${DEPLOYMENT_NAME} -o yaml
  kubectl --context ${ARGO_CONTEXT} -n ${TARGET_NS} get deployment ${DEPLOYMENT_NAME} -o yaml
EOF_BOTH
)"
    ;;
  *)
    echo "error: unsupported RECONCILER value: $RECONCILER (expected flux|argo|both)" >&2
    exit 1
    ;;
esac

cat <<EOF

[helm-runtime] next checks
${NEXT_CONTEXT_LINES}
  use cub-scout after this when you want the cluster-side ownership/drift view
EOF
