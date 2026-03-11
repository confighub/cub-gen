#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

USE_RG=0
if command -v rg >/dev/null 2>&1; then
  USE_RG=1
fi

file_has_pattern() {
  local pattern="$1"
  local file="$2"
  if [ "$USE_RG" -eq 1 ]; then
    rg -q -- "$pattern" "$file"
  else
    grep -Eq -- "$pattern" "$file"
  fi
}

stdin_has_pattern() {
  local pattern="$1"
  if [ "$USE_RG" -eq 1 ]; then
    rg -q -- "$pattern"
  else
    grep -Eq -- "$pattern"
  fi
}

stdin_first_match_line() {
  local pattern="$1"
  if [ "$USE_RG" -eq 1 ]; then
    rg -n -- "$pattern" | head -n1 | cut -d: -f1
  else
    grep -En -- "$pattern" | head -n1 | cut -d: -f1
  fi
}

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
    if ! file_has_pattern 'run-confighub-lifecycle-connected\.sh|connected-preflight\.sh' "$connected_script"; then
      failures+=("$example_name: demo-connected.sh must use shared connected preflight path")
    fi
  fi

  if ! file_has_pattern 'demo-local\.sh' "$readme"; then
    failures+=("$example_name: README missing demo-local.sh usage")
  fi

  if ! file_has_pattern 'demo-connected\.sh' "$readme"; then
    failures+=("$example_name: README missing demo-connected.sh usage")
  fi

  if ! file_has_pattern 'cub auth login' "$readme"; then
    failures+=("$example_name: README missing connected login step (cub auth login)")
  fi

  if ! file_has_pattern '^## If you already' "$readme"; then
    failures+=("$example_name: README missing expert-user viewpoint section (## If you already ...)")
  fi

  if ! file_has_pattern '^## Why this maps' "$readme"; then
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

  if ! printf '%s\n' "$entrypoint_section" | stdin_has_pattern 'demo-local\.sh'; then
    failures+=("$example_name: entrypoint section missing demo-local.sh command")
  fi

  if ! printf '%s\n' "$entrypoint_section" | stdin_has_pattern 'demo-connected\.sh'; then
    failures+=("$example_name: entrypoint section missing demo-connected.sh command")
  fi

  if ! printf '%s\n' "$entrypoint_section" | stdin_has_pattern 'cub auth login'; then
    failures+=("$example_name: entrypoint section missing cub auth login command")
  else
    login_line="$(printf '%s\n' "$entrypoint_section" | stdin_first_match_line 'cub auth login')"
    connected_line="$(printf '%s\n' "$entrypoint_section" | stdin_first_match_line 'demo-connected\.sh')"
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
