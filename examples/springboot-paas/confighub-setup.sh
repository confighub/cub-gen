#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CUB="${CUB:-cub}"
EXAMPLE_LABEL="springboot-paas"

ENVS=("dev" "stage" "prod")
SPACE_PREFIX="inventory-api"

case "${1:-}" in
  --explain)
    cat <<EOF
springboot-paas ConfigHub setup

This script creates ConfigHub spaces and units for the inventory-api
example across dev, stage, and prod environments.

Spaces: inventory-api-dev, inventory-api-stage, inventory-api-prod
Units:  inventory-api (one per space)
Data:   from confighub/inventory-api-{dev,stage,prod}.yaml

Labels: ExampleName=$EXAMPLE_LABEL

Cleanup: ./confighub-cleanup.sh
EOF
    exit 0
    ;;
esac

echo "=== ConfigHub setup for springboot-paas ==="

for env in "${ENVS[@]}"; do
  space="${SPACE_PREFIX}-${env}"
  unit_file="$ROOT_DIR/confighub/inventory-api-${env}.yaml"

  echo ""
  echo "--- Setting up $space ---"

  ${CUB} space create "$space" \
    --label "ExampleName=${EXAMPLE_LABEL}" \
    --label "Environment=${env}" \
    2>/dev/null || echo "  (space may already exist)"

  ${CUB} unit create "$space" inventory-api \
    --data-file "$unit_file" \
    2>/dev/null || ${CUB} unit update "$space" inventory-api \
    --data-file "$unit_file"

  echo "  OK: $space/inventory-api configured"
done

echo ""
echo "=== Setup complete ==="
echo "Verify with: ./confighub-verify.sh"
