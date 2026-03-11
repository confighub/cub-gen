#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

USE_RG=0
if command -v rg >/dev/null 2>&1; then
  USE_RG=1
fi

has_pattern() {
  local pattern="$1"
  local file="$2"
  if [ "$USE_RG" -eq 1 ]; then
    rg -q -- "$pattern" "$file"
  else
    grep -Eq -- "$pattern" "$file"
  fi
}

require_file_pattern() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if [ ! -f "$file" ]; then
    echo "error: missing file: $file" >&2
    exit 1
  fi
  if ! has_pattern "$pattern" "$file"; then
    echo "error: $file missing $label" >&2
    exit 1
  fi
}

# Main README must keep clear connected onboarding and backend reality.
require_file_pattern "README.md" "cub auth login" "connected auth step (cub auth login)"
require_file_pattern "README.md" "confighubai/confighub" "ConfigHub OSS backend link"

# Docs-site landing pages must preserve the same message.
require_file_pattern "docs/index.md" "cub auth login" "connected auth step (cub auth login)"
require_file_pattern "docs/index.md" "confighubai/confighub" "ConfigHub OSS backend link"
require_file_pattern "docs/platform.md" "cub auth login" "connected auth step (cub auth login)"
require_file_pattern "docs/platform.md" "confighubai/confighub" "ConfigHub OSS backend link"

echo "ok: core docs keep explicit local-vs-connected onboarding and backend availability"
