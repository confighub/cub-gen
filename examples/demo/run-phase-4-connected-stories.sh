#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

echo "[phase-4] preflight (requires cub auth login)"
require_connected_preflight
print_connected_context

echo "[phase-4] running connected stories 8,10,11"

./examples/demo/story-8-label-evolution-connected.sh ./examples/backstage-idp ./examples/backstage-idp backstage-idp
./examples/demo/story-10-signed-writeback-proof-connected.sh ./examples/helm-paas ./examples/helm-paas helm-paas
./examples/demo/story-11-live-breakglass-proposal-connected.sh ./examples/helm-paas ./examples/helm-paas helm-paas
