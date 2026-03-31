#!/usr/bin/env bash
set -euo pipefail

CUB="${CUB:-cub}"
EXAMPLE_LABEL="springboot-paas"

echo "=== ConfigHub cleanup for springboot-paas ==="
echo ""
echo "This will delete ALL spaces with label ExampleName=${EXAMPLE_LABEL}"
echo ""

${CUB} space delete --where "Labels.ExampleName = '${EXAMPLE_LABEL}'" --recursive

echo ""
echo "=== Cleanup complete ==="
