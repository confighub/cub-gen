#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-2] Swamp Automation: governed ALLOW/BLOCK workflow proof"
./examples/swamp-automation/demo-governed-structure.sh
