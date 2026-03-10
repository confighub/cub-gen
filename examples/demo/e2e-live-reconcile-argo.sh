#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CLUSTER_NAME="${CLUSTER_NAME:-cub-gen-live-argo}"
KUBE_CONTEXT="kind-${CLUSTER_NAME}"
ARGO_NS="${ARGO_NS:-argocd}"
APP_NAME="${APP_NAME:-cub-gen-live}"
REPO_URL="${REPO_URL:-https://github.com/confighub/cub-gen}"
REPO_BRANCH="${REPO_BRANCH:-main}"
PATH_V1="${PATH_V1:-examples/live-reconcile/flux/manifests-v1}"
PATH_V2="${PATH_V2:-examples/live-reconcile/flux/manifests-v2}"
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
require_cmd jq
require_cmd curl

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

install_argocd_if_needed() {
  if ! "${KUBECTL[@]}" get namespace "$ARGO_NS" >/dev/null 2>&1; then
    echo "[setup] installing Argo CD"
    "${KUBECTL[@]}" create namespace "$ARGO_NS"
    local install_output
    if ! install_output="$("${KUBECTL[@]}" --request-timeout=5m apply --server-side -n "$ARGO_NS" -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml 2>&1)"; then
      echo "$install_output"
      if echo "$install_output" | grep -qi 'timeout'; then
        echo "[setup] warning: Argo manifest apply timed out; continuing because partial apply often succeeds on slow local clusters."
      else
        return 1
      fi
    else
      echo "$install_output"
    fi
  else
    echo "[setup] Argo CD namespace already exists: $ARGO_NS"
  fi

  "${KUBECTL[@]}" wait --for=condition=Established crd/applications.argoproj.io --timeout=2m
  "${KUBECTL[@]}" -n "$ARGO_NS" rollout status deployment/argocd-redis --timeout=5m
  "${KUBECTL[@]}" -n "$ARGO_NS" rollout status deployment/argocd-repo-server --timeout=5m
  "${KUBECTL[@]}" -n "$ARGO_NS" rollout status deployment/argocd-server --timeout=5m
  "${KUBECTL[@]}" -n "$ARGO_NS" rollout status statefulset/argocd-application-controller --timeout=5m
}

apply_application() {
  local path="$1"
  cat <<EOF_APP | "${KUBECTL[@]}" apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ${APP_NAME}
  namespace: ${ARGO_NS}
spec:
  project: default
  source:
    repoURL: ${REPO_URL}
    targetRevision: ${REPO_BRANCH}
    path: ${path}
  destination:
    server: https://kubernetes.default.svc
    namespace: ${TARGET_NS}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF_APP
}

update_application_path() {
  local path="$1"
  "${KUBECTL[@]}" -n "$ARGO_NS" patch "application/${APP_NAME}" --type merge -p "{\"spec\":{\"source\":{\"path\":\"${path}\"}}}"
}

request_refresh() {
  "${KUBECTL[@]}" -n "$ARGO_NS" annotate "application/${APP_NAME}" argocd.argoproj.io/refresh=hard --overwrite >/dev/null
}

wait_for_sync() {
  local phase="$1"
  local timeout_sec=300
  local step=5
  local elapsed=0

  echo "[reconcile][$phase] waiting for Argo Application to be Synced + Healthy"
  while [ "$elapsed" -lt "$timeout_sec" ]; do
    local app_json
    app_json="$("${KUBECTL[@]}" -n "$ARGO_NS" get "application/${APP_NAME}" -o json)"
    local sync health op_phase
    sync="$(jq -r '.status.sync.status // "Unknown"' <<<"$app_json")"
    health="$(jq -r '.status.health.status // "Unknown"' <<<"$app_json")"
    op_phase="$(jq -r '.status.operationState.phase // "Unknown"' <<<"$app_json")"

    if [ "$sync" = "Synced" ] && [ "$health" = "Healthy" ]; then
      "${KUBECTL[@]}" -n "$TARGET_NS" rollout status "deployment/${DEPLOYMENT_NAME}" --timeout=3m
      return 0
    fi

    if [ "$op_phase" = "Error" ] || [ "$op_phase" = "Failed" ]; then
      echo "error: Argo operation failed during $phase" >&2
      "${KUBECTL[@]}" -n "$ARGO_NS" get "application/${APP_NAME}" -o yaml >&2 || true
      return 1
    fi

    sleep "$step"
    elapsed=$((elapsed + step))
  done

  echo "error: timed out waiting for Argo sync/health during $phase" >&2
  "${KUBECTL[@]}" -n "$ARGO_NS" get "application/${APP_NAME}" -o yaml >&2 || true
  return 1
}

deployment_snapshot() {
  "${KUBECTL[@]}" -n "$TARGET_NS" get "deployment/${DEPLOYMENT_NAME}" -o json \
    | jq '{replicas: .spec.replicas, image: .spec.template.spec.containers[0].image, revision: (.spec.template.metadata.labels["demo.confighub.io/revision"] // "unknown")}'
}

install_argocd_if_needed

echo "[phase:create] apply desired state v1"
apply_application "$PATH_V1"
request_refresh
wait_for_sync "create-v1"
SNAPSHOT_V1="$(deployment_snapshot)"
echo "$SNAPSHOT_V1" | jq '{phase: "create", state: .}'

echo "[phase:update] switch desired state to v2"
update_application_path "$PATH_V2"
request_refresh
wait_for_sync "update-v2"
SNAPSHOT_V2="$(deployment_snapshot)"
echo "$SNAPSHOT_V2" | jq '{phase: "update", state: .}'

echo "[phase:drift] force live drift (replicas=5), then reconcile back to desired"
"${KUBECTL[@]}" -n "$TARGET_NS" scale "deployment/${DEPLOYMENT_NAME}" --replicas=5
"${KUBECTL[@]}" -n "$TARGET_NS" get "deployment/${DEPLOYMENT_NAME}" -o json | jq '{phase: "drifted-live", replicas: .spec.replicas}'
request_refresh
wait_for_sync "drift-correction"
SNAPSHOT_CORRECTED="$(deployment_snapshot)"
echo "$SNAPSHOT_CORRECTED" | jq '{phase: "corrected", state: .}'

echo "[evidence] argo application status"
"${KUBECTL[@]}" -n "$ARGO_NS" get "application/${APP_NAME}" -o json \
  | jq '{name: .metadata.name, sync: .status.sync.status, health: .status.health.status, revision: (.status.sync.revision // "unknown")}'

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
