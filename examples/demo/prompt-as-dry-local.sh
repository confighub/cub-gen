#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/ai-only-guardrails.sh"

REPO_PATH="${1:-./examples/swamp-automation}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

enforce_ai_only_scope "$REPO_PATH" "$REPO_PATH"

echo "[prompt-as-dry][local] intent artifact: $REPO_PATH/workflow-deploy.yaml"
./examples/demo/app-ai-change-run.sh "$REPO_PATH" "$REPO_PATH"
