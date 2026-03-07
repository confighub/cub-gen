#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
SPACE="${SPACE:-platform}"
PROFILE_HINT="${3:-auto}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

echo "[wizard] step 1/5: select source"
echo "  repo: $REPO_PATH"
echo "  render-target: $RENDER_TARGET"
echo "  profile-hint: $PROFILE_HINT"
echo "  space: $SPACE"

echo "[wizard] build cub-gen"
go build -o ./cub-gen ./cmd/cub-gen

echo "[wizard] step 2/5: discover generators"
DISCOVER_JSON="$(./cub-gen gitops discover --space "$SPACE" --json "$REPO_PATH")"
echo "$DISCOVER_JSON" | jq '{repo: .target_path, discovered_count: (.resources|length), resource_types: [.resources[].resource_type] | unique, resource_names: [.resources[].resource_name]}'

echo "[wizard] step 3/5: preview import graph (DRY -> GEN -> WET)"
IMPORT_JSON="$(./cub-gen gitops import --space "$SPACE" --json "$REPO_PATH" "$RENDER_TARGET")"
echo "$IMPORT_JSON" | jq '{profiles: [.discovered[].generator_profile] | unique, dry_units: (.dry_units|length), generator_units: (.generator_units|length), wet_units: (.wet_units|length), links: (.links|length), dry_inputs: (.dry_inputs|length), wet_manifest_targets: (.wet_manifest_targets|length)}'

echo "[wizard] step 4/5: preview provenance and inverse edit hints"
echo "$IMPORT_JSON" | jq '{sample_field_origin: (.provenance[0].field_origin_map[0] // null), sample_inverse_hint: (.provenance[0].inverse_edit_pointers[0] // null)}'

echo "[wizard] step 5/5: simulate import confirmation + governed bundle"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "$IMPORT_JSON" > "$TMPDIR/import.json"
./cub-gen publish --in "$TMPDIR/import.json" > "$TMPDIR/bundle.json"
./cub-gen verify --json --in "$TMPDIR/bundle.json" > "$TMPDIR/verify.json"
./cub-gen attest --in "$TMPDIR/bundle.json" --verifier ci-bot > "$TMPDIR/attestation.json"

echo "[wizard] result summary"
jq '{change_id, bundle_digest, summary}' "$TMPDIR/bundle.json"
jq '{valid, change_id, bundle_digest}' "$TMPDIR/verify.json"
