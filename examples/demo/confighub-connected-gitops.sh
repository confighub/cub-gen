#!/usr/bin/env bash
set -euo pipefail

# Connected-mode demo for users with a running ConfigHub backend and cub CLI auth.
# This script uses the real backend-connected cub gitops flow (discover/import/cleanup).

usage() {
  cat <<'EOF'
Usage:
  SPACE=<space-slug> DISCOVERY_TARGET=<target-slug> RENDER_TARGET=<target-slug> ./examples/demo/confighub-connected-gitops.sh

Optional env vars:
  CUB_BIN=<path-to-cub-cli>   default: cub
  CLEANUP=1                   run "cub gitops cleanup" at the end

Preconditions:
  1. ConfigHub backend is running and reachable.
  2. cub CLI is installed and authenticated (e.g., "cub auth login --server <url>").
  3. SPACE exists and contains both DISCOVERY_TARGET and RENDER_TARGET.
EOF
}

CUB_BIN="${CUB_BIN:-cub}"
SPACE="${SPACE:-}"
DISCOVERY_TARGET="${DISCOVERY_TARGET:-}"
RENDER_TARGET="${RENDER_TARGET:-}"
CLEANUP="${CLEANUP:-0}"

if [[ -z "$SPACE" || -z "$DISCOVERY_TARGET" || -z "$RENDER_TARGET" ]]; then
  usage
  exit 1
fi

if ! command -v "$CUB_BIN" >/dev/null 2>&1; then
  echo "[connected] error: cub CLI not found at '$CUB_BIN'"
  exit 1
fi

if ! "$CUB_BIN" auth get-token >/dev/null 2>&1; then
  echo "[connected] error: cub CLI is not authenticated. Run:"
  echo "  $CUB_BIN auth login --server <confighub-url>"
  exit 1
fi

echo "[connected] verifying space and targets"
"$CUB_BIN" space get "$SPACE" >/dev/null
"$CUB_BIN" target get --space "$SPACE" "$DISCOVERY_TARGET" >/dev/null
"$CUB_BIN" target get --space "$SPACE" "$RENDER_TARGET" >/dev/null

echo "[connected] discover"
"$CUB_BIN" gitops discover --space "$SPACE" "$DISCOVERY_TARGET"

echo "[connected] import"
"$CUB_BIN" gitops import --space "$SPACE" "$DISCOVERY_TARGET" "$RENDER_TARGET"

echo "[connected] imported dry/wet units"
"$CUB_BIN" unit list --space "$SPACE" --no-header --names | grep -E '(-dry|-wet|-crds)$' || true

FIRST_WET="$("$CUB_BIN" unit list --space "$SPACE" --no-header --names | grep -E '\-wet$' | head -1 || true)"
if [[ -n "$FIRST_WET" ]]; then
  echo "[connected] mutation history for sample wet unit: $FIRST_WET"
  "$CUB_BIN" mutation list --space "$SPACE" "$FIRST_WET" || true
fi

if [[ "$CLEANUP" == "1" ]]; then
  echo "[connected] cleanup discover unit"
  "$CUB_BIN" gitops cleanup --space "$SPACE" "$DISCOVERY_TARGET"
fi

echo "[connected] done"
