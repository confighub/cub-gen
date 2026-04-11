#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[start-platform] step 1: source-side provenance and governance"
./examples/helm-paas/demo-local.sh

cat <<'EOF'

[start-platform] next steps
  governed ownership proof from the same example:
    ./examples/helm-paas/demo-governed-change.sh

  connected governance only:
    cub auth login
    ./examples/helm-paas/demo-connected.sh

  connected + live proof in the helm flagship:
    cub auth login
    RECONCILER=argo ./examples/helm-paas/demo-runtime.sh
    RECONCILER=both ./examples/helm-paas/demo-runtime.sh

  cluster-side inspection:
    use cub-scout against the reconciled workload after the runtime proof
EOF
