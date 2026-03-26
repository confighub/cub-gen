#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

JSON_OUT="$ROOT_DIR/docs/testing/example-truth-matrix.json"
MD_OUT="$ROOT_DIR/docs/testing/example-truth-matrix.md"

tmp_json="$(mktemp)"
tmp_md="$(mktemp)"
trap 'rm -f "$tmp_json" "$tmp_md"' EXIT

go run ./tools/example-truth-matrix --format json >"$tmp_json"
go run ./tools/example-truth-matrix --format markdown >"$tmp_md"

if ! diff -u "$JSON_OUT" "$tmp_json"; then
  echo "error: docs/testing/example-truth-matrix.json is out of date. Regenerate with: go run ./tools/example-truth-matrix --format json" >&2
  exit 1
fi

if ! diff -u "$MD_OUT" "$tmp_md"; then
  echo "error: docs/testing/example-truth-matrix.md is out of date. Regenerate with: go run ./tools/example-truth-matrix --format markdown" >&2
  exit 1
fi

echo "ok: example truth matrix artifacts are current"

