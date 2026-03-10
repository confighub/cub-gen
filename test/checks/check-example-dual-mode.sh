#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if ! command -v rg >/dev/null 2>&1; then
  echo "error: required command not found: rg" >&2
  exit 1
fi

declare -a failures=()

while IFS= read -r readme; do
  example_dir="$(dirname "$readme")"
  example_name="$(basename "$example_dir")"
  if [ "$example_name" = "demo" ]; then
    continue
  fi

  local_script="$example_dir/demo-local.sh"
  connected_script="$example_dir/demo-connected.sh"

  if [ ! -x "$local_script" ]; then
    failures+=("$example_name: missing executable demo-local.sh")
  fi

  if [ ! -x "$connected_script" ]; then
    failures+=("$example_name: missing executable demo-connected.sh")
  fi

  if [ -x "$connected_script" ]; then
    if ! rg -q 'simulate-confighub-lifecycle-connected\.sh|connected-preflight\.sh' "$connected_script"; then
      failures+=("$example_name: demo-connected.sh must use shared connected preflight path")
    fi
  fi

  if ! rg -q 'demo-local\.sh' "$readme"; then
    failures+=("$example_name: README missing demo-local.sh usage")
  fi

  if ! rg -q 'demo-connected\.sh' "$readme"; then
    failures+=("$example_name: README missing demo-connected.sh usage")
  fi

  if ! rg -q 'cub auth login' "$readme"; then
    failures+=("$example_name: README missing connected login step (cub auth login)")
  fi

  if ! rg -q '^## If you already' "$readme"; then
    failures+=("$example_name: README missing expert-user viewpoint section (## If you already ...)")
  fi

  if ! rg -q '^## Why this maps' "$readme"; then
    failures+=("$example_name: README missing model-mapping section (## Why this maps ...)")
  fi
done < <(find "$ROOT_DIR/examples" -mindepth 2 -maxdepth 2 -type f -name README.md | sort)

if [ "${#failures[@]}" -gt 0 ]; then
  echo "error: dual-mode docs/scripts check failed:" >&2
  for failure in "${failures[@]}"; do
    echo "  - $failure" >&2
  done
  exit 1
fi

echo "ok: every example has local+connected entrypoints, shared connected preflight wiring, login guidance, expert viewpoint, and model mapping"
