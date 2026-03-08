#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-1] C3 Agent: repo-first wizard simulation"
./examples/demo/simulate-repo-wizard.sh ./examples/c3agent ./examples/c3agent c3agent
