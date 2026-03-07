#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-3] ConfigHub Actions: governed execution intent import"
./examples/demo/simulate-repo-wizard.sh ./examples/confighub-actions ./examples/confighub-actions ops-workflow
