#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_DIR="${1:-$ROOT_DIR/examples/demo/output/persona-value}"
PLATFORM_DIR="$OUT_DIR/platform-engineer"
APP_DIR="$OUT_DIR/app-team"
GITOPS_DIR="$OUT_DIR/gitops-team"

mkdir -p "$PLATFORM_DIR" "$APP_DIR" "$GITOPS_DIR"

echo "[setup] building cub-gen"
go build -o ./cub-gen ./cmd/cub-gen

echo "[platform-engineer] capturing generator model visibility"
./cub-gen generators --markdown --details > "$PLATFORM_DIR/generators-details.md"
./cub-gen gitops discover --space platform ./examples/helm-paas > "$PLATFORM_DIR/helm-discover.txt"
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      dry_inputs,
      wet_manifest_targets,
      rendered_lineage: .provenance[0].rendered_object_lineage
    }' > "$PLATFORM_DIR/helm-import-summary.json"

echo "[app-team] capturing inverse edit guidance"
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      field_origin_map: .provenance[0].field_origin_map,
      inverse_edit_pointers: .provenance[0].inverse_edit_pointers
    }' > "$APP_DIR/score-edit-map.json"
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      inverse_edit_pointers: .provenance[0].inverse_edit_pointers
    }' > "$APP_DIR/spring-edit-map.json"

echo "[gitops-team] capturing publish/verify/attest evidence"
./cub-gen gitops discover --space platform ./examples/helm-paas > "$GITOPS_DIR/helm-discover.txt"
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > "$GITOPS_DIR/helm-bundle.json"
./cub-gen verify --in "$GITOPS_DIR/helm-bundle.json" > "$GITOPS_DIR/helm-verify.json"
./cub-gen attest --in "$GITOPS_DIR/helm-bundle.json" --verifier ci-bot > "$GITOPS_DIR/helm-attestation.json"
./cub-gen verify-attestation --in "$GITOPS_DIR/helm-attestation.json" --bundle "$GITOPS_DIR/helm-bundle.json" > "$GITOPS_DIR/helm-attestation-verify.json"

echo "[proof] running make ci"
make ci > "$OUT_DIR/ci-proof.log"

echo "[done] persona bundles written to: $OUT_DIR"
echo "[done] key outputs:"
echo "  - $PLATFORM_DIR/generators-details.md"
echo "  - $APP_DIR/score-edit-map.json"
echo "  - $APP_DIR/spring-edit-map.json"
echo "  - $GITOPS_DIR/helm-attestation.json"
echo "  - $OUT_DIR/ci-proof.log"
