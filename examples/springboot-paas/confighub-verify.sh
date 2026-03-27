#!/usr/bin/env bash
set -euo pipefail

CUB="${CUB:-cub}"
EXAMPLE_LABEL="springboot-paas"
SPACE_PREFIX="inventory-api"
ENVS=("dev" "stage" "prod")

fail() { echo "FAIL: $*" >&2; exit 1; }

echo "=== ConfigHub verification for springboot-paas ==="

for env in "${ENVS[@]}"; do
  space="${SPACE_PREFIX}-${env}"
  echo ""
  echo "--- Checking $space ---"

  ${CUB} space get "$space" --json 2>/dev/null | jq -e ".Space.Labels.ExampleName == \"${EXAMPLE_LABEL}\"" >/dev/null \
    || fail "Space $space missing or wrong label"
  echo "  OK: Space $space exists with correct label"

  UNIT_DATA=$(${CUB} unit get "$space" inventory-api --json 2>/dev/null) \
    || fail "Unit inventory-api not found in $space"
  echo "  OK: Unit inventory-api exists"

  echo "$UNIT_DATA" | jq -e '.Unit.Data | length > 0' >/dev/null \
    || fail "Unit inventory-api in $space has no data"
  echo "  OK: Unit has data content"
done

echo ""
echo "====================================="
echo "ConfigHub verification PASSED"
echo "====================================="
