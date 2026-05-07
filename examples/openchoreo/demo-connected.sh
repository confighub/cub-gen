#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

require_connected_preflight

echo "[openchoreo] connected context"
print_connected_context
echo "[openchoreo] running local deterministic proof; this wrapper does not deploy"

./examples/openchoreo/demo-local.sh
