#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

./examples/demo/run-confighub-lifecycle-connected.sh "./examples/just-apps-no-platform-config" "./examples/just-apps-no-platform-config" "just-apps-no-platform-config"
