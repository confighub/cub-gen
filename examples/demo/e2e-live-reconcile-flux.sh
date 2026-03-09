#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CLUSTER_NAME="${CLUSTER_NAME:-cub-gen-live}"
KUBE_CONTEXT="kind-${CLUSTER_NAME}"
FLUX_NS="${FLUX_NS:-flux-system}"
SOURCE_NAME="${SOURCE_NAME:-cub-gen-live}"
KUSTOM_NAME="${KUSTOM_NAME:-cub-gen-live}"
REPO_URL="${REPO_URL:-https://github.com/confighub/cub-gen}"
REPO_BRANCH="${REPO_BRANCH:-main}"
PATH_V1="${PATH_V1:-./examples/live-reconcile/flux/manifests-v1}"
PATH_V2="${PATH_V2:-./examples/live-reconcile/flux/manifests-v2}"
TARGET_NS="${TARGET_NS:-demo-live}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-aiops-demo}"
CLEANUP_CLUSTER="${CLEANUP_CLUSTER:-0}"

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
}

require_cmd docker
require_cmd kind
require_cmd kubectl
require_cmd flux
require_cmd jq

if ! docker info >/dev/null 2>&1; then
  echo "error: docker is not running or not reachable" >&2
  exit 1
fi

cleanup() {
  if [ "$CLEANUP_CLUSTER" = "1" ]; then
    echo "[cleanup] deleting kind cluster: $CLUSTER_NAME"
    kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

if ! kind get clusters | grep -qx "$CLUSTER_NAME"; then
  echo "[setup] creating kind cluster: $CLUSTER_NAME"
  kind create cluster --name "$CLUSTER_NAME"
else
  echo "[setup] kind cluster already exists: $CLUSTER_NAME"
fi

KUBECTL=(kubectl --context "$KUBE_CONTEXT")
FLUX=(flux --context "$KUBE_CONTEXT")

if ! "${KUBECTL[@]}" get namespace "$FLUX_NS" >/dev/null 2>&1; then
  echo "[setup] installing flux controllers"
  "${FLUX[@]}" install
else
  echo "[setup] flux already installed in $FLUX_NS"
fi

apply_flux_objects() {
  local path="$1"
  cat <<EOF | "${KUBECTL[@]}" apply -f -
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: ${SOURCE_NAME}
  namespace: ${FLUX_NS}
spec:
  interval: 1m0s
  url: ${REPO_URL}
  ref:
    branch: ${REPO_BRANCH}
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: ${KUSTOM_NAME}
  namespace: ${FLUX_NS}
spec:
  interval: 1m0s
  path: ${path}
  prune: true
  wait: true
  timeout: 3m0s
  sourceRef:
    kind: GitRepository
    name: ${SOURCE_NAME}
  targetNamespace: ${TARGET_NS}
EOF
}

reconcile_and_wait() {
  local phase="$1"
  local ts
  ts="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  echo "[reconcile][$phase] request source + kustomization reconciliation"
  "${KUBECTL[@]}" -n "$FLUX_NS" annotate "gitrepository/${SOURCE_NAME}" reconcile.fluxcd.io/requestedAt="$ts" --overwrite >/dev/null
  "${KUBECTL[@]}" -n "$FLUX_NS" annotate "kustomization/${KUSTOM_NAME}" reconcile.fluxcd.io/requestedAt="$ts" --overwrite >/dev/null

  "${KUBECTL[@]}" -n "$FLUX_NS" wait "gitrepository/${SOURCE_NAME}" --for=condition=ready --timeout=3m
  if ! "${KUBECTL[@]}" -n "$FLUX_NS" wait "kustomization/${KUSTOM_NAME}" --for=condition=ready --timeout=5m; then
    echo "error: kustomization did not become ready" >&2
    "${KUBECTL[@]}" -n "$FLUX_NS" get "kustomization/${KUSTOM_NAME}" -o yaml >&2 || true
    exit 1
  fi

  "${KUBECTL[@]}" -n "$TARGET_NS" rollout status "deployment/${DEPLOYMENT_NAME}" --timeout=3m
}

deployment_snapshot() {
  "${KUBECTL[@]}" -n "$TARGET_NS" get "deployment/${DEPLOYMENT_NAME}" -o json \
    | jq '{replicas: .spec.replicas, image: .spec.template.spec.containers[0].image, revision: (.spec.template.metadata.labels["demo.confighub.io/revision"] // "unknown")}'
}

echo "[phase:create] apply desired state v1"
apply_flux_objects "$PATH_V1"
reconcile_and_wait "create-v1"
SNAPSHOT_V1="$(deployment_snapshot)"
echo "$SNAPSHOT_V1" | jq '{phase: "create", state: .}'

echo "[phase:update] switch desired state to v2"
apply_flux_objects "$PATH_V2"
reconcile_and_wait "update-v2"
SNAPSHOT_V2="$(deployment_snapshot)"
echo "$SNAPSHOT_V2" | jq '{phase: "update", state: .}'

echo "[phase:drift] force live drift (replicas=5), then reconcile back to desired"
"${KUBECTL[@]}" -n "$TARGET_NS" scale "deployment/${DEPLOYMENT_NAME}" --replicas=5
"${KUBECTL[@]}" -n "$TARGET_NS" get "deployment/${DEPLOYMENT_NAME}" -o json | jq '{phase: "drifted-live", replicas: .spec.replicas}'
reconcile_and_wait "drift-correction"
SNAPSHOT_CORRECTED="$(deployment_snapshot)"
echo "$SNAPSHOT_CORRECTED" | jq '{phase: "corrected", state: .}'

echo "[evidence] flux reconciliation status"
"${FLUX[@]}" get sources git --all-namespaces
"${FLUX[@]}" get kustomizations --all-namespaces

echo "[summary]"
jq -n \
  --argjson v1 "$SNAPSHOT_V1" \
  --argjson v2 "$SNAPSHOT_V2" \
  --argjson corrected "$SNAPSHOT_CORRECTED" \
  '{
    create: $v1,
    update: $v2,
    corrected: $corrected,
    update_applied: (($v1.image != $v2.image) and ($v1.replicas != $v2.replicas)),
    drift_corrected: ($corrected.replicas == $v2.replicas and $corrected.image == $v2.image)
  }'
