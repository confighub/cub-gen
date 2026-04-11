#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[ai-demo-4] Operations workflow: governed ALLOW/BLOCK policy proof"
./examples/ops-workflow/demo-governed-policy.sh
