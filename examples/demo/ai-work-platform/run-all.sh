#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

./examples/demo/ai-work-platform/scenario-1-c3agent.sh
./examples/demo/ai-work-platform/scenario-2-swamp.sh
./examples/demo/ai-work-platform/scenario-3-confighub-actions.sh
./examples/demo/ai-work-platform/scenario-4-operations.sh
