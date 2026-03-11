#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

./examples/demo/run-confighub-lifecycle-connected.sh "./examples/c3agent" "./examples/c3agent" "c3agent"
