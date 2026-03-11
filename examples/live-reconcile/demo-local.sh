#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

RECONCILER="${RECONCILER:-flux}"

case "$RECONCILER" in
  flux)
    ./examples/demo/e2e-live-reconcile-flux.sh
    ;;
  argo)
    ./examples/demo/e2e-live-reconcile-argo.sh
    ;;
  both)
    ./examples/demo/e2e-live-reconcile-flux.sh
    ./examples/demo/e2e-live-reconcile-argo.sh
    ;;
  *)
    echo "error: unsupported RECONCILER value: $RECONCILER (expected flux|argo|both)" >&2
    exit 1
    ;;
esac
