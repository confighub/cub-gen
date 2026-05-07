#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

go build -o ./cub-gen ./cmd/cub-gen

OUT_DIR="$ROOT_DIR/.tmp/examples/kubara"
mkdir -p "$OUT_DIR"

./cub-gen gitops import --space platform --json ./examples/just-apps-no-platform-config \
  > "$OUT_DIR/no-config-platform-import.json"

./cub-gen platform import --json ./testdata/platform-estate/platform.yaml \
  > "$OUT_DIR/platform-estate.json"

./cub-gen platform import --json ./testdata/variant-topology/platform.yaml \
  > "$OUT_DIR/variant-topology.json"

jq '{
  app_config_generator: .discovered[0].generator_profile,
  first_origin: .provenance[0].field_origin_map[0]
}' "$OUT_DIR/no-config-platform-import.json"

jq '{
  components: .summary.component_count,
  variants: .summary.variant_count,
  generators: .summary.generator_count,
  diagnostics: .summary.diagnostic_count
}' "$OUT_DIR/platform-estate.json"

jq '{
  base_variants: [.variants[] | select(.variant_kind == "base") | .id],
  deployment_variants: [.variants[] | select(.variant_kind == "deployment") | .id]
}' "$OUT_DIR/variant-topology.json"

echo
echo "Kubara-like platform proof written to: $OUT_DIR"
