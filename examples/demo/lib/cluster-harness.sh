#!/usr/bin/env bash
# Shared real-cluster harness for connected tests
#
# This library provides unified cluster management for cub-gen connected tests.
# It supports both Flux and Argo CD reconcilers on kind clusters.
#
# Usage:
#   source examples/demo/lib/cluster-harness.sh
#   setup_connected_cluster flux   # or: setup_connected_cluster argo
#   ... run connected tests ...
#   cleanup_connected_cluster      # optional, or use CLEANUP_CLUSTER=1

set -euo pipefail

# Cluster configuration (overridable via env vars)
CLUSTER_NAME="${CLUSTER_NAME:-cub-gen-connected}"
KUBE_CONTEXT="kind-${CLUSTER_NAME}"
CLEANUP_CLUSTER="${CLEANUP_CLUSTER:-0}"

# Flux configuration
FLUX_NS="${FLUX_NS:-flux-system}"

# Argo configuration
ARGO_NS="${ARGO_NS:-argocd}"

# App configuration
TARGET_NS="${TARGET_NS:-demo-live}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-aiops-demo}"

# Timeouts
RECONCILE_TIMEOUT="${RECONCILE_TIMEOUT:-300}"
HEALTH_CHECK_RETRIES="${HEALTH_CHECK_RETRIES:-30}"

# State tracking
_CLUSTER_HARNESS_INITIALIZED="${_CLUSTER_HARNESS_INITIALIZED:-0}"
_CLUSTER_HARNESS_RECONCILER=""

#######################################
# Require a command to be available
# Arguments:
#   Command name
# Returns:
#   Exits 1 if command not found
#######################################
require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    echo "hint: install $cmd before running connected cluster tests" >&2
    exit 1
  fi
}

#######################################
# Verify Docker is running
# Returns:
#   Exits 1 if Docker not available
#######################################
require_docker() {
  require_cmd docker
  if ! docker info >/dev/null 2>&1; then
    echo "error: docker is not running or not reachable" >&2
    echo "hint: start Docker Desktop or the docker daemon" >&2
    exit 1
  fi
}

#######################################
# Check if cluster exists and is healthy
# Arguments:
#   Cluster name
# Returns:
#   0 if healthy, 1 otherwise
#######################################
cluster_is_healthy() {
  local name="${1:-$CLUSTER_NAME}"
  local ctx="kind-${name}"

  # Check cluster exists
  if ! kind get clusters 2>/dev/null | grep -qx "$name"; then
    return 1
  fi

  # Check API server responds
  if ! kubectl --context "$ctx" cluster-info >/dev/null 2>&1; then
    return 1
  fi

  return 0
}

#######################################
# Create or reuse a kind cluster
# Arguments:
#   None (uses CLUSTER_NAME)
# Returns:
#   Sets KUBE_CONTEXT
#######################################
setup_kind_cluster() {
  require_cmd kind
  require_cmd kubectl
  require_docker

  if cluster_is_healthy "$CLUSTER_NAME"; then
    echo "[cluster-harness] reusing existing cluster: $CLUSTER_NAME"
  else
    echo "[cluster-harness] creating kind cluster: $CLUSTER_NAME"
    kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
    kind create cluster --name "$CLUSTER_NAME" --wait 60s
  fi

  KUBE_CONTEXT="kind-${CLUSTER_NAME}"
  export KUBE_CONTEXT

  # Wait for cluster to be ready
  local retries=$HEALTH_CHECK_RETRIES
  while ! kubectl --context "$KUBE_CONTEXT" get nodes >/dev/null 2>&1; do
    retries=$((retries - 1))
    if [ $retries -le 0 ]; then
      echo "error: cluster did not become ready in time" >&2
      exit 1
    fi
    echo "[cluster-harness] waiting for cluster to be ready..."
    sleep 2
  done

  echo "[cluster-harness] cluster ready: $CLUSTER_NAME (context: $KUBE_CONTEXT)"
}

#######################################
# Install Flux on the cluster
# Returns:
#   0 on success
#######################################
install_flux() {
  require_cmd flux

  if kubectl --context "$KUBE_CONTEXT" get namespace "$FLUX_NS" >/dev/null 2>&1; then
    echo "[cluster-harness] flux namespace exists, verifying readiness..."
  else
    echo "[cluster-harness] installing flux controllers"
    flux --context "$KUBE_CONTEXT" install --timeout="${RECONCILE_TIMEOUT}s"
  fi

  # Always verify flux is ready (handles partial installs)
  flux --context "$KUBE_CONTEXT" check --timeout="${RECONCILE_TIMEOUT}s"
  echo "[cluster-harness] flux installed and ready"
}

#######################################
# Install Argo CD on the cluster
# Returns:
#   0 on success
#######################################
install_argo() {
  if kubectl --context "$KUBE_CONTEXT" get namespace "$ARGO_NS" >/dev/null 2>&1; then
    echo "[cluster-harness] argo cd namespace exists, verifying readiness..."
  else
    echo "[cluster-harness] installing argo cd"
    kubectl --context "$KUBE_CONTEXT" create namespace "$ARGO_NS"
    kubectl --context "$KUBE_CONTEXT" apply -n "$ARGO_NS" \
      -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
  fi

  # Always verify argo is ready (handles partial installs)
  echo "[cluster-harness] waiting for argo cd to be ready..."
  kubectl --context "$KUBE_CONTEXT" wait --for=condition=available \
    --timeout="${RECONCILE_TIMEOUT}s" \
    -n "$ARGO_NS" deployment/argocd-server

  echo "[cluster-harness] argo cd installed and ready"
}

#######################################
# Set up a connected cluster with specified reconciler
# Arguments:
#   Reconciler type: flux | argo | both
# Returns:
#   0 on success, exports KUBE_CONTEXT
#######################################
setup_connected_cluster() {
  local reconciler="${1:-flux}"

  # If already initialized, check if we need to add a reconciler
  if [ "$_CLUSTER_HARNESS_INITIALIZED" = "1" ]; then
    if [ "$reconciler" = "$_CLUSTER_HARNESS_RECONCILER" ]; then
      echo "[cluster-harness] cluster already initialized with $reconciler"
      return 0
    fi
    # Handle adding additional reconciler to existing cluster
    if [ "$reconciler" = "both" ]; then
      echo "[cluster-harness] adding missing reconciler to existing cluster"
      if [ "$_CLUSTER_HARNESS_RECONCILER" = "flux" ]; then
        install_argo
      elif [ "$_CLUSTER_HARNESS_RECONCILER" = "argo" ]; then
        install_flux
      fi
      _CLUSTER_HARNESS_RECONCILER="both"
      return 0
    fi
    # Different single reconciler requested - install it
    echo "[cluster-harness] adding $reconciler to existing cluster (was: $_CLUSTER_HARNESS_RECONCILER)"
    case "$reconciler" in
      flux) install_flux ;;
      argo) install_argo ;;
    esac
    _CLUSTER_HARNESS_RECONCILER="both"
    return 0
  fi

  setup_kind_cluster

  case "$reconciler" in
    flux)
      install_flux
      ;;
    argo)
      install_argo
      ;;
    both)
      install_flux
      install_argo
      ;;
    *)
      echo "error: unknown reconciler: $reconciler (expected: flux, argo, both)" >&2
      exit 1
      ;;
  esac

  # Create target namespace if needed
  if ! kubectl --context "$KUBE_CONTEXT" get namespace "$TARGET_NS" >/dev/null 2>&1; then
    kubectl --context "$KUBE_CONTEXT" create namespace "$TARGET_NS"
  fi

  _CLUSTER_HARNESS_INITIALIZED="1"
  _CLUSTER_HARNESS_RECONCILER="$reconciler"

  echo "[cluster-harness] connected cluster ready"
  echo "  cluster: $CLUSTER_NAME"
  echo "  context: $KUBE_CONTEXT"
  echo "  reconciler: $reconciler"
  echo "  target namespace: $TARGET_NS"
}

#######################################
# Clean up the connected cluster
# Arguments:
#   None
# Returns:
#   0 on success
#######################################
cleanup_connected_cluster() {
  if [ "$CLEANUP_CLUSTER" = "1" ] || [ "${1:-}" = "force" ]; then
    echo "[cluster-harness] deleting kind cluster: $CLUSTER_NAME"
    kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
    _CLUSTER_HARNESS_INITIALIZED="0"
    _CLUSTER_HARNESS_RECONCILER=""
  else
    echo "[cluster-harness] keeping cluster (set CLEANUP_CLUSTER=1 to delete)"
  fi
}

#######################################
# Wait for a deployment to be ready
# Arguments:
#   Deployment name
#   Namespace (optional, defaults to TARGET_NS)
# Returns:
#   0 on success
#######################################
wait_for_deployment() {
  local name="$1"
  local ns="${2:-$TARGET_NS}"

  echo "[cluster-harness] waiting for deployment: $name (namespace: $ns)"
  kubectl --context "$KUBE_CONTEXT" wait --for=condition=available \
    --timeout="${RECONCILE_TIMEOUT}s" \
    -n "$ns" "deployment/$name"
}

#######################################
# Get deployment replica count
# Arguments:
#   Deployment name
#   Namespace (optional, defaults to TARGET_NS)
# Returns:
#   Replica count on stdout
#######################################
get_deployment_replicas() {
  local name="$1"
  local ns="${2:-$TARGET_NS}"

  kubectl --context "$KUBE_CONTEXT" get deployment "$name" \
    -n "$ns" -o jsonpath='{.spec.replicas}'
}

#######################################
# Print cluster state for debugging
# Arguments:
#   None
# Returns:
#   Prints cluster state to stdout
#######################################
print_cluster_state() {
  echo "[cluster-harness] cluster state snapshot"
  echo "=== Nodes ==="
  kubectl --context "$KUBE_CONTEXT" get nodes -o wide 2>/dev/null || echo "(no nodes)"
  echo ""
  echo "=== Namespaces ==="
  kubectl --context "$KUBE_CONTEXT" get namespaces 2>/dev/null || echo "(no namespaces)"
  echo ""
  echo "=== Deployments in $TARGET_NS ==="
  kubectl --context "$KUBE_CONTEXT" get deployments -n "$TARGET_NS" 2>/dev/null || echo "(none)"
  echo ""
  if [ "$_CLUSTER_HARNESS_RECONCILER" = "flux" ] || [ "$_CLUSTER_HARNESS_RECONCILER" = "both" ]; then
    echo "=== Flux Kustomizations ==="
    kubectl --context "$KUBE_CONTEXT" get kustomizations -n "$FLUX_NS" 2>/dev/null || echo "(none)"
    echo ""
  fi
  if [ "$_CLUSTER_HARNESS_RECONCILER" = "argo" ] || [ "$_CLUSTER_HARNESS_RECONCILER" = "both" ]; then
    echo "=== Argo Applications ==="
    kubectl --context "$KUBE_CONTEXT" get applications -n "$ARGO_NS" 2>/dev/null || echo "(none)"
    echo ""
  fi
}

# Export key variables for downstream scripts
export CLUSTER_NAME KUBE_CONTEXT CLEANUP_CLUSTER
export FLUX_NS ARGO_NS TARGET_NS DEPLOYMENT_NAME
export RECONCILE_TIMEOUT HEALTH_CHECK_RETRIES
