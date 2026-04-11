#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[start-score] step 1: keep score.yaml as the app-team contract and trace it to runtime fields"
./examples/demo/module-2-score-field-map.sh

cat <<'EOF'

[start-score] next steps
  connected governance:
    cub auth login
    ./examples/scoredev-paas/demo-connected.sh

  what to inspect:
    - local output: score.yaml field origin + inverse edit map
    - connected output: change_id, bundle digest, and decision/query evidence

  runtime proof today:
    standalone Score live-cluster proof is still open work
    use RECONCILER=both ./examples/live-reconcile/demo-local.sh for reconciler proof
    and ./examples/scoredev-paas/README.md for the current Score-specific truth
EOF
