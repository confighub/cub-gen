#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

REPO_PATH="${1:-./examples/swamp-automation}"
RENDER_TARGET="${2:-$REPO_PATH}"
SLUG="${3:-prompt-dry-swamp}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi
if [ ! -d "$RENDER_TARGET" ]; then
  echo "error: render target path not found: $RENDER_TARGET" >&2
  exit 1
fi

require_connected_preflight
print_connected_context

echo "[prompt-as-dry][connected] intent artifact: $REPO_PATH/workflow-deploy.yaml"
./examples/demo/run-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$SLUG"
