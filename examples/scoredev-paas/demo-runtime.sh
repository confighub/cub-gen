#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-scoredev-runtime}"
KUBECONFIG_PATH="${ROOT_DIR}/var/${CLUSTER_NAME}.kubeconfig"
NAMESPACE="${NAMESPACE:-checkout-api}"
MANIFESTS_PATH="${MANIFESTS_PATH:-${ROOT_DIR}/var/runtime-manifests.yaml}"
DEPLOYMENT_NAME="checkout-api"

"${ROOT_DIR}/bin/create-cluster" "${CLUSTER_NAME}"
export KUBECONFIG="${KUBECONFIG_PATH}"

"${ROOT_DIR}/bin/build-image" "${CLUSTER_NAME}"
"${ROOT_DIR}/bin/render-runtime" "${MANIFESTS_PATH}"

echo "Applying rendered runtime manifests..."
kubectl apply -f "${MANIFESTS_PATH}" >/dev/null

echo "Waiting for deployment/${DEPLOYMENT_NAME} to become ready..."
kubectl -n "${NAMESPACE}" rollout status deployment/"${DEPLOYMENT_NAME}" --timeout=180s

"${ROOT_DIR}/verify-e2e.sh"

echo
echo "[score-runtime] success"
echo "  cluster: ${CLUSTER_NAME}"
echo "  kubeconfig: ${KUBECONFIG_PATH}"
echo "  manifests: ${MANIFESTS_PATH}"
