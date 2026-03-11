#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

USE_RG=0
if command -v rg >/dev/null 2>&1; then
  USE_RG=1
fi

search_pattern='\\bA[b]ly\\b|\\ba[b]ly\\b'
exclude_globs=(
  --glob '!.git/**'
  --glob '!**/*.png'
  --glob '!**/*.jpg'
  --glob '!**/*.jpeg'
  --glob '!**/*.gif'
  --glob '!**/*.svg'
)

if [ "$USE_RG" -eq 1 ]; then
  if rg -n -S --pcre2 "$search_pattern" "${exclude_globs[@]}" \
    README.md docs examples cmd internal .github mkdocs.yml; then
    echo "error: found legacy provider terminology. Use no-config-platform naming instead." >&2
    exit 1
  fi
else
  if grep -RInE '\bA[b]ly\b|\ba[b]ly\b' README.md docs examples cmd internal .github mkdocs.yml; then
    echo "error: found legacy provider terminology. Use no-config-platform naming instead." >&2
    exit 1
  fi
fi

echo "ok: no legacy provider terminology found; naming is consistently no-config-platform"
