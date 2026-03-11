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

guide="docs/workflows/operation-registry-real-apps.md"
if [ ! -f "$guide" ]; then
  echo "error: missing operation-registry guide: $guide" >&2
  exit 1
fi

declare -a failures=()
while IFS= read -r registry; do
  example_dir="$(dirname "$(dirname "$registry")")"
  example_name="$(basename "$example_dir")"
  readme="$example_dir/README.md"
  rel_registry="${registry#$ROOT_DIR/}"

  if [ ! -f "$readme" ]; then
    failures+=("$example_name: missing README.md for registry-bearing example")
    continue
  fi

  if ! file_has_pattern 'platform/registry\.yaml' "$readme"; then
    failures+=("$example_name: README missing platform/registry.yaml discoverability")
  fi

  if ! file_has_pattern "$rel_registry" "$guide"; then
    failures+=("$example_name: operation-registry guide missing $rel_registry")
  fi
done < <(find "$ROOT_DIR/examples" -mindepth 3 -maxdepth 3 -type f -path '*/platform/registry.yaml' | sort)

if [ "${#failures[@]}" -gt 0 ]; then
  echo "error: registry discoverability check failed:" >&2
  for failure in "${failures[@]}"; do
    echo "  - $failure" >&2
  done
  exit 1
fi

echo "ok: registry-bearing examples are discoverable in example READMEs and central guide"
