#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[start-platform] step 1: source-side provenance and governance"
./examples/helm-paas/demo-local.sh

cat <<'EOF'

[start-platform] next steps
  connected governance:
    cub auth login
    ./examples/helm-paas/demo-connected.sh

  runtime proof:
    RECONCILER=both ./examples/live-reconcile/demo-local.sh

  cluster-side inspection:
    use cub-scout against the reconciled workload after the runtime proof
EOF
