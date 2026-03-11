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

declare -a scripts=()
while IFS= read -r script; do
  scripts+=("$script")
done < <(
  {
    find "$ROOT_DIR/examples/demo" -maxdepth 1 -type f -name '*connected*.sh'
    find "$ROOT_DIR/examples" -mindepth 2 -maxdepth 2 -type f -name 'demo-connected.sh'
  } | sort
)

declare -a failures=()
for script in "${scripts[@]}"; do
  # Accept either:
  # 1) shared connected preflight path, or
  # 2) call into connected lifecycle wrapper that enforces preflight, or
  # 3) explicit CI non-interactive auth contract.
  if file_has_pattern 'connected-preflight\.sh|require_connected_preflight|run-confighub-lifecycle-connected\.sh' "$script"; then
    continue
  fi

  if file_has_pattern 'CONFIGHUB_TOKEN.*required|non-interactive CI mode|--mode connected' "$script"; then
    continue
  fi

  failures+=("${script#$ROOT_DIR/}: missing connected auth/preflight contract")
done

if [ "${#failures[@]}" -gt 0 ]; then
  echo "error: connected auth contract check failed:" >&2
  for failure in "${failures[@]}"; do
    echo "  - $failure" >&2
  done
  exit 1
fi

echo "ok: all connected scripts use shared preflight/lifecycle or explicit CI auth contract"
