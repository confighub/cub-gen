#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

go build -o ./cub-gen ./cmd/cub-gen

OUT_DIR="$ROOT_DIR/.tmp/examples/openchoreo"
mkdir -p "$OUT_DIR"

FIXTURE="./testdata/openchoreo-hardgate"

./cub-gen gitops discover --space platform --adoption-report --json "$FIXTURE" \
  > "$OUT_DIR/discover.json"

./cub-gen gitops import --space platform --json "$FIXTURE" \
  > "$OUT_DIR/import.json"

./cub-gen publish --space platform "$FIXTURE" \
  | ./cub-gen verify --in - \
  > "$OUT_DIR/verify.txt"

jq '{
  generator: (.detections[0].profile // .resources[0].generator_profile),
  adoption: .adoption_report.summary,
  diagnostics: ([.adoption_report.diagnostics[]?.code, .diagnostics[]?.code] | unique)
}' "$OUT_DIR/discover.json"

echo
echo "OpenChoreo example proof written to: $OUT_DIR"
