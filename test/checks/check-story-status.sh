#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

README_MAIN="$ROOT_DIR/README.md"
README_DEMO="$ROOT_DIR/examples/demo/README.md"

extract_stories() {
  local file="$1"
  local status="$2"
  local value
  value="$(awk -F'|' -v status="$status" '
    $0 ~ /^\|/ {
      label=$2
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", label)
      if (label == status) {
        stories=$3
        gsub(/[[:space:]]/, "", stories)
        print stories
        exit
      }
    }
  ' "$file")"

  if [ -z "$value" ]; then
    echo "error: status row not found in $file: $status" >&2
    exit 1
  fi

  printf '%s' "$value"
}

main_met="$(extract_stories "$README_MAIN" "Met/strong in current demos")"
main_partial="$(extract_stories "$README_MAIN" "Partial (simulated/local-first, not full backend/runtime integration)")"
main_deferred="$(extract_stories "$README_MAIN" "Deferred")"

demo_met="$(extract_stories "$README_DEMO" "Met/strong in current demos")"
demo_partial="$(extract_stories "$README_DEMO" "Partial (simulated/local-first, not full backend/runtime integration)")"
demo_deferred="$(extract_stories "$README_DEMO" "Deferred")"

if [ "$main_met" != "$demo_met" ] || [ "$main_partial" != "$demo_partial" ] || [ "$main_deferred" != "$demo_deferred" ]; then
  echo "error: story-status table drift between README.md and examples/demo/README.md" >&2
  echo "  README.md     met=$main_met partial=$main_partial deferred=$main_deferred" >&2
  echo "  demo README   met=$demo_met partial=$demo_partial deferred=$demo_deferred" >&2
  exit 1
fi

echo "ok: story-status tables are consistent"
