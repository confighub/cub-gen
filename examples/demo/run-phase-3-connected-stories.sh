#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"
export CONNECTED_FALLBACK_MODE="${CONNECTED_FALLBACK_MODE:-off}"

echo "[phase-3] preflight (requires cub auth login)"
require_connected_preflight
print_connected_context

echo "[phase-3] running connected stories 1,7,9,12"

./examples/demo/story-1-existing-repo-connected.sh ./examples/helm-paas ./examples/helm-paas helm-paas
./examples/demo/story-7-ci-api-flow-connected.sh
./examples/demo/story-7-agent-tool-call-connected.sh
./examples/demo/story-9-multi-repo-wave-connected.sh
./examples/demo/story-12-unified-actor-evidence.sh ./examples/helm-paas ./examples/helm-paas helm-paas
