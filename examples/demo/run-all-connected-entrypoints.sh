#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

echo "[connected-entrypoints] preflight (requires cub auth login)"
require_connected_preflight
print_connected_context

echo "[connected-entrypoints] build cub-gen once"
go build -o ./cub-gen ./cmd/cub-gen

INCLUDE_LIVE_RECONCILE="${INCLUDE_LIVE_RECONCILE:-0}"
RECONCILER="${RECONCILER:-both}"

mapfile -t connected_scripts < <(find "$ROOT_DIR/examples" -mindepth 2 -maxdepth 2 -type f -name demo-connected.sh | sort)

declare -a failed=()
ran=0
skipped=0

for script in "${connected_scripts[@]}"; do
  example_dir="$(dirname "$script")"
  example="$(basename "$example_dir")"

  if [ "$example" = "demo" ]; then
    continue
  fi

  if [ "$example" = "live-reconcile" ] && [ "$INCLUDE_LIVE_RECONCILE" != "1" ]; then
    echo "[connected-entrypoints] SKIP: $example (set INCLUDE_LIVE_RECONCILE=1 to include)"
    skipped=$((skipped + 1))
    continue
  fi

  echo
  echo "============================================================"
  echo "[connected-entrypoints] running: $example"
  echo "============================================================"

  if [ "$example" = "live-reconcile" ]; then
    if RECONCILER="$RECONCILER" "$script"; then
      echo "[connected-entrypoints] PASS: $example"
    else
      echo "[connected-entrypoints] FAIL: $example"
      failed+=("$example")
    fi
  else
    if SKIP_BUILD=1 "$script"; then
      echo "[connected-entrypoints] PASS: $example"
    else
      echo "[connected-entrypoints] FAIL: $example"
      failed+=("$example")
    fi
  fi

  ran=$((ran + 1))
done

echo
echo "[connected-entrypoints] summary: ran=$ran skipped=$skipped failed=${#failed[@]}"
if [ "${#failed[@]}" -eq 0 ]; then
  echo "[connected-entrypoints] all selected example entrypoints passed"
  exit 0
fi

echo "[connected-entrypoints] failures: ${failed[*]}" >&2
exit 1
