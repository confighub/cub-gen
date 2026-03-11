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
    if ! rg -q 'run-confighub-lifecycle-connected\.sh|connected-preflight\.sh' "$connected_script"; then
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

  entrypoint_section="$(awk '
    /^## Local and Connected Entrypoints$/ {capture=1; next}
    /^## / && capture {exit}
    capture {print}
  ' "$readme")"

  if [ -z "$entrypoint_section" ]; then
    failures+=("$example_name: README missing Local and Connected Entrypoints section")
    continue
  fi

  if ! printf '%s\n' "$entrypoint_section" | rg -q 'demo-local\.sh'; then
    failures+=("$example_name: entrypoint section missing demo-local.sh command")
  fi

  if ! printf '%s\n' "$entrypoint_section" | rg -q 'demo-connected\.sh'; then
    failures+=("$example_name: entrypoint section missing demo-connected.sh command")
  fi

  if ! printf '%s\n' "$entrypoint_section" | rg -q 'cub auth login'; then
    failures+=("$example_name: entrypoint section missing cub auth login command")
  else
    login_line="$(printf '%s\n' "$entrypoint_section" | rg -n 'cub auth login' | head -n1 | cut -d: -f1)"
    connected_line="$(printf '%s\n' "$entrypoint_section" | rg -n 'demo-connected\.sh' | head -n1 | cut -d: -f1)"
    if [ -n "$login_line" ] && [ -n "$connected_line" ] && [ "$login_line" -gt "$connected_line" ]; then
      failures+=("$example_name: entrypoint section must list cub auth login before demo-connected.sh")
    fi
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
