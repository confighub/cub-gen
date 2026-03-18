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

require_heading() {
  local file="$1"
  local heading="$2"
  if ! grep -Eq "^## ${heading}$" "$file"; then
    echo "error: heading not found in $file: ## ${heading}" >&2
    exit 1
  fi
}

require_absent() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "$pattern" "$file"; then
    echo "error: stale status language found in $file: $pattern" >&2
    exit 1
  fi
}

require_heading "$README_MAIN" "Execution status"
require_heading "$README_DEMO" "PRD execution status"

require_absent "$README_MAIN" "Met/strong in current demos"
require_absent "$README_DEMO" "Met/strong in current demos"

main_tracked="$(extract_stories "$README_MAIN" "Actively tracked")"
demo_tracked="$(extract_stories "$README_DEMO" "Actively tracked")"

if [ "$main_tracked" != "$demo_tracked" ]; then
  echo "error: active execution issue list drift between README.md and examples/demo/README.md" >&2
  echo "  README.md   tracked=$main_tracked" >&2
  echo "  demo README tracked=$demo_tracked" >&2
  exit 1
fi

extract_stories "$README_MAIN" "Strong now" >/dev/null
extract_stories "$README_MAIN" "In progress" >/dev/null
extract_stories "$README_DEMO" "Strong now" >/dev/null
extract_stories "$README_DEMO" "In progress" >/dev/null

echo "ok: execution-status tables are consistent"
