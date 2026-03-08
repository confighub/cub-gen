#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-2] Swamp Automation: AI-native workflow import"
./examples/demo/simulate-repo-wizard.sh ./examples/swamp-automation ./examples/swamp-automation swamp
