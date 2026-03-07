#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-4] Operations workflow: existing operations example"
./examples/demo/simulate-repo-wizard.sh ./examples/ops-workflow ./examples/ops-workflow ops-workflow
