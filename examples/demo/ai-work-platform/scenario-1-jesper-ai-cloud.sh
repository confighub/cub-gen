#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-1] Jesper AI Cloud: repo-first wizard simulation"
./examples/demo/simulate-repo-wizard.sh ./examples/jesper-ai-cloud ./examples/jesper-ai-cloud backstage-idp
