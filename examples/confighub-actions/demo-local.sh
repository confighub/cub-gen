#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

./examples/demo/simulate-confighub-lifecycle.sh "./examples/confighub-actions" "./examples/confighub-actions" "confighub-actions"
