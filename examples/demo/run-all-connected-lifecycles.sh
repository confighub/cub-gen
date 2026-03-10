#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

echo "[connected-lifecycle] preflight (requires cub auth login)"
require_connected_preflight
print_connected_context

echo "[connected-lifecycle] build cub-gen once"
go build -o ./cub-gen ./cmd/cub-gen

examples=(
  "helm-paas"
  "scoredev-paas"
  "springboot-paas"
  "backstage-idp"
  "just-apps-no-platform-config"
  "ops-workflow"
  "c3agent"
  "ai-ops-paas"
  "swamp-automation"
  "swamp-project"
  "confighub-actions"
)

echo "[connected-lifecycle] starting run for ${#examples[@]} examples"

declare -a failed=()
for example in "${examples[@]}"; do
  echo
  echo "============================================================"
  echo "[connected-lifecycle] running: $example"
  echo "============================================================"

  if SKIP_BUILD=1 ./examples/demo/simulate-confighub-lifecycle-connected.sh "./examples/$example" "./examples/$example" "$example"; then
    echo "[connected-lifecycle] PASS: $example"
  else
    echo "[connected-lifecycle] FAIL: $example"
    failed+=("$example")
  fi
done

echo
if [ "${#failed[@]}" -eq 0 ]; then
  echo "[connected-lifecycle] all examples passed"
  exit 0
fi

echo "[connected-lifecycle] failures (${#failed[@]}): ${failed[*]}" >&2
exit 1
