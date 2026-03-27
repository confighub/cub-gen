#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBECONFIG="${KUBECONFIG:-$ROOT_DIR/var/springboot-platform.kubeconfig}"
NAMESPACE="${NAMESPACE:-inventory-api}"
DEPLOYMENT="inventory-api"
SERVICE="inventory-api"
LOCAL_PORT="${LOCAL_PORT:-18080}"

export KUBECONFIG

fail() { echo "FAIL: $*" >&2; exit 1; }

echo "Checking cluster..."
kubectl cluster-info >/dev/null 2>&1 || fail "Cluster is not reachable"
echo "  OK: Cluster is reachable"

echo "Checking namespace..."
kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || fail "Namespace $NAMESPACE does not exist"
echo "  OK: Namespace $NAMESPACE exists"

echo "Checking deployment..."
kubectl -n "$NAMESPACE" get deployment "$DEPLOYMENT" >/dev/null 2>&1 || fail "Deployment $DEPLOYMENT not found"
READY=$(kubectl -n "$NAMESPACE" get deployment "$DEPLOYMENT" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
[[ "${READY:-0}" -ge 1 ]] || fail "Deployment $DEPLOYMENT has no ready replicas"
echo "  OK: Deployment $DEPLOYMENT has $READY ready replica(s)"

echo "Checking pods..."
POD=$(kubectl -n "$NAMESPACE" get pods -l "app.kubernetes.io/name=$DEPLOYMENT" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
[[ -n "$POD" ]] || fail "No pods found for $DEPLOYMENT"
PHASE=$(kubectl -n "$NAMESPACE" get pod "$POD" -o jsonpath='{.status.phase}')
[[ "$PHASE" == "Running" ]] || fail "Pod $POD is $PHASE, expected Running"
echo "  OK: Pod $POD is Running"

echo "Port-forwarding to $LOCAL_PORT..."
kubectl -n "$NAMESPACE" port-forward "svc/$SERVICE" "${LOCAL_PORT}:80" >/dev/null 2>&1 &
PF_PID=$!
trap 'kill $PF_PID 2>/dev/null || true' EXIT
sleep 3

echo "Checking /api/inventory/summary..."
SUMMARY=$(curl -sf "http://localhost:${LOCAL_PORT}/api/inventory/summary" 2>/dev/null) || fail "Could not reach /api/inventory/summary"
echo "  Response: $SUMMARY"

SVC=$(echo "$SUMMARY" | jq -r '.service')
[[ "$SVC" == "inventory-api" ]] || fail "Expected service=inventory-api, got $SVC"
echo "  OK: service=$SVC"

ITEMS=$(echo "$SUMMARY" | jq '.items | length')
[[ "$ITEMS" -ge 1 ]] || fail "Expected at least 1 item, got $ITEMS"
echo "  OK: $ITEMS inventory items returned"

echo "Checking /api/inventory/items..."
ITEMS_RESP=$(curl -sf "http://localhost:${LOCAL_PORT}/api/inventory/items" 2>/dev/null) || fail "Could not reach /api/inventory/items"
ITEM_COUNT=$(echo "$ITEMS_RESP" | jq 'length')
[[ "$ITEM_COUNT" -ge 1 ]] || fail "Expected items, got $ITEM_COUNT"
echo "  OK: $ITEM_COUNT items from /api/inventory/items"

echo "Checking /actuator/health..."
HEALTH=$(curl -sf "http://localhost:${LOCAL_PORT}/actuator/health" 2>/dev/null) || fail "Could not reach /actuator/health"
STATUS=$(echo "$HEALTH" | jq -r '.status')
[[ "$STATUS" == "UP" ]] || fail "Health status is $STATUS, expected UP"
echo "  OK: actuator health is UP"

echo ""
echo "====================================="
echo "E2E verification PASSED"
echo "====================================="
