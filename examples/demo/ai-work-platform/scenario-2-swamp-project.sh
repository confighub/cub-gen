#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-2] Swamp Project: platform Helm package import"
./examples/demo/simulate-repo-wizard.sh ./examples/swamp-project ./examples/swamp-project helm-paas
