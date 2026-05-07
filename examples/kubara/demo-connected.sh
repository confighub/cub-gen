#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

require_connected_preflight

echo "[kubara] connected context"
print_connected_context
echo "[kubara] running local deterministic proof; this wrapper does not deploy"

./examples/kubara/demo-local.sh
