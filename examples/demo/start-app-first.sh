#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[start-app] step 1: Spring config ownership and provenance"
./examples/springboot-paas/demo-local.sh

cat <<'EOF'

[start-app] next steps
  connected governance:
    cub auth login
    ./examples/springboot-paas/demo-connected.sh

  runtime proof:
    RECONCILER=both ./examples/live-reconcile/demo-local.sh

  cluster-side inspection:
    use cub-scout after the runtime proof when you want to inspect live state
EOF
