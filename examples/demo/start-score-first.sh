#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[start-score] step 1: keep score.yaml as the app-team contract and trace it to runtime fields"
./examples/demo/module-2-score-field-map.sh

echo
echo "[start-score] step 2: prove app-owned workload changes stay allowed and new resource types escalate"
./examples/scoredev-paas/demo-governed-workload.sh

echo
echo "[start-score] step 3: run the merged Score workload as a real app on a local cluster"
./examples/scoredev-paas/demo-runtime.sh

cat <<'EOF'

[start-score] next steps
  connected governance:
    cub auth login
    ./examples/scoredev-paas/demo-connected.sh

  what to inspect:
    - local output: score.yaml field origin + inverse edit map
    - governed output: ALLOW for safe image change, ESCALATE for unapproved resource type
    - runtime output: live checkout-api Deployment + Service on kind with /healthz and / verified
    - connected output: change_id, bundle digest, and decision/query evidence

  cleanup when finished:
    kind delete cluster --name scoredev-runtime
    or set CLUSTER_NAME to keep this proof separate from other examples
EOF
