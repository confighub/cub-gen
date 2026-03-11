#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[deprecated] app-ai-fastpath.sh is deprecated; use ./examples/demo/app-ai-change-run.sh" >&2
exec ./examples/demo/app-ai-change-run.sh "$@"
