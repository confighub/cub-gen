#!/usr/bin/env bash
# AI Ops PaaS — Demo Script
#
# Shows the full detect -> import -> explain path for an AI agent fleet.
# Proves: 2 DRY inputs -> 11 WET targets, all governed.
#
# Usage: ./examples/ai-ops-paas/demo.sh
# Prereq: go build -o ./cub-gen ./cmd/cub-gen

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CUB_GEN="${REPO_ROOT}/cub-gen"
EXAMPLE="${REPO_ROOT}/examples/ai-ops-paas"

if [ ! -f "$CUB_GEN" ]; then
  echo "Building cub-gen..."
  (cd "$REPO_ROOT" && go build -o ./cub-gen ./cmd/cub-gen)
fi

echo "============================================"
echo "  AI Ops PaaS — Governed Agent Fleet Demo"
echo "============================================"
echo ""
echo "This demo shows how a platform team provides self-service"
echo "AI agent fleet provisioning with full governance."
echo ""
echo "DRY input:  c3agent.yaml (base) + c3agent-prod.yaml (prod overlay)"
echo "Generator:  c3agent (detected automatically)"
echo "WET output: 11 Kubernetes resource targets"
echo ""

# Step 1: Discover
echo "--- Step 1: Discover generators ---"
echo "\$ cub-gen gitops discover --space ai-ops --json $EXAMPLE"
echo ""
"$CUB_GEN" gitops discover --space ai-ops --json "$EXAMPLE" 2>/dev/null | python3 -m json.tool 2>/dev/null || "$CUB_GEN" gitops discover --space ai-ops --json "$EXAMPLE"
echo ""

# Step 2: Import with full provenance
echo "--- Step 2: Import with provenance ---"
echo "\$ cub-gen gitops import --space ai-ops --json $EXAMPLE $EXAMPLE"
echo ""
"$CUB_GEN" gitops import --space ai-ops --json "$EXAMPLE" "$EXAMPLE" 2>/dev/null | python3 -m json.tool 2>/dev/null || "$CUB_GEN" gitops import --space ai-ops --json "$EXAMPLE" "$EXAMPLE"
echo ""

# Step 3: Show the generator triple details
echo "--- Step 3: Generator triple (contract + provenance + inverse) ---"
echo "\$ cub-gen generators --json --details | jq '.families[] | select(.kind == \"c3agent\")'"
echo ""
"$CUB_GEN" generators --json --details 2>/dev/null | python3 -c "
import json, sys
data = json.load(sys.stdin)
for f in data.get('families', []):
    if f.get('kind') == 'c3agent':
        print(json.dumps(f, indent=2))
        break
" 2>/dev/null || echo "(install python3 for pretty-printed output)"
echo ""

echo "============================================"
echo "  Demo complete."
echo ""
echo "  What you just saw:"
echo "    1. Automatic c3agent detection from c3agent.yaml"
echo "    2. DRY/WET import with field-origin provenance"
echo "    3. 11 WET targets across Deployment, Service,"
echo "       ConfigMap, Secret, PVC, SA, ClusterRole, ClusterRoleBinding"
echo "    4. Inverse edit hints for every DRY section"
echo ""
echo "  The platform/ directory contains illustrative specs:"
echo "    - registry.yaml:    FrameworkRegistry (typed operations)"
echo "    - constraints.yaml: Platform policy guardrails"
echo ""
echo "  These specs show what operations the platform offers."
echo "  In ConfigHub, they become the governance contract."
echo "============================================"
