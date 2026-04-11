#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-scoredev-runtime}"
KUBECONFIG="${KUBECONFIG:-$ROOT_DIR/var/${CLUSTER_NAME}.kubeconfig}"
NAMESPACE="${NAMESPACE:-checkout-api}"
DEPLOYMENT="${DEPLOYMENT:-checkout-api}"
SERVICE="${SERVICE:-checkout-api}"
LOCAL_PORT="${LOCAL_PORT:-18081}"
EXPECTED_IMAGE="${EXPECTED_IMAGE:-ghcr.io/example/checkout-api:v2.1.0}"

export KUBECONFIG

fail() { echo "FAIL: $*" >&2; exit 1; }

echo "Checking cluster..."
kubectl cluster-info >/dev/null 2>&1 || fail "Cluster is not reachable"
echo "  OK: Cluster is reachable"

echo "Checking namespace..."
kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1 || fail "Namespace ${NAMESPACE} does not exist"
echo "  OK: Namespace ${NAMESPACE} exists"

echo "Checking deployment..."
kubectl -n "${NAMESPACE}" get deployment "${DEPLOYMENT}" >/dev/null 2>&1 || fail "Deployment ${DEPLOYMENT} not found"
READY="$(kubectl -n "${NAMESPACE}" get deployment "${DEPLOYMENT}" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")"
[[ "${READY:-0}" -ge 1 ]] || fail "Deployment ${DEPLOYMENT} has no ready replicas"
echo "  OK: Deployment ${DEPLOYMENT} has ${READY} ready replica(s)"

echo "Checking image..."
IMAGE="$(kubectl -n "${NAMESPACE}" get deployment "${DEPLOYMENT}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
[[ "${IMAGE}" == "${EXPECTED_IMAGE}" ]] || fail "Expected image ${EXPECTED_IMAGE}, got ${IMAGE}"
echo "  OK: Deployment image is ${IMAGE}"

echo "Checking pods..."
POD="$(kubectl -n "${NAMESPACE}" get pods -l "app.kubernetes.io/name=${DEPLOYMENT}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)"
[[ -n "${POD}" ]] || fail "No pods found for ${DEPLOYMENT}"
PHASE="$(kubectl -n "${NAMESPACE}" get pod "${POD}" -o jsonpath='{.status.phase}')"
[[ "${PHASE}" == "Running" ]] || fail "Pod ${POD} is ${PHASE}, expected Running"
echo "  OK: Pod ${POD} is Running"

echo "Port-forwarding to ${LOCAL_PORT}..."
kubectl -n "${NAMESPACE}" port-forward "svc/${SERVICE}" "${LOCAL_PORT}:8080" >/dev/null 2>&1 &
PF_PID=$!
trap 'kill ${PF_PID} 2>/dev/null || true' EXIT
sleep 3

echo "Checking /healthz..."
HEALTHZ="$(curl -sf "http://localhost:${LOCAL_PORT}/healthz" 2>/dev/null)" || fail "Could not reach /healthz"
[[ "${HEALTHZ}" == "ok" ]] || fail "Expected /healthz to return ok, got ${HEALTHZ}"
echo "  OK: /healthz returned ok"

echo "Checking /..."
ROOT_JSON="$(curl -sf "http://localhost:${LOCAL_PORT}/" 2>/dev/null)" || fail "Could not reach /"
echo "  Response: ${ROOT_JSON}"

SERVICE_NAME="$(echo "${ROOT_JSON}" | jq -r '.service')"
[[ "${SERVICE_NAME}" == "checkout-api" ]] || fail "Expected service=checkout-api, got ${SERVICE_NAME}"
echo "  OK: service=${SERVICE_NAME}"

LOG_LEVEL="$(echo "${ROOT_JSON}" | jq -r '.logLevel')"
[[ "${LOG_LEVEL}" == "warn" ]] || fail "Expected logLevel=warn, got ${LOG_LEVEL}"
echo "  OK: logLevel=${LOG_LEVEL}"

echo
echo "====================================="
echo "E2E verification PASSED"
echo "====================================="
