#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[lifecycle] build cub-gen once"
go build -o ./cub-gen ./cmd/cub-gen

examples=(
  "helm-paas"
  "scoredev-paas"
  "springboot-paas"
  "backstage-idp"
  "ably-config"
  "ops-workflow"
  "c3agent"
  "ai-ops-paas"
  "swamp-automation"
  "confighub-actions"
)

for example in "${examples[@]}"; do
  echo
  echo "============================================================"
  echo "[lifecycle] running: $example"
  echo "============================================================"
  SKIP_BUILD=1 ./examples/demo/simulate-confighub-lifecycle.sh "./examples/$example" "./examples/$example" "$example"
done
