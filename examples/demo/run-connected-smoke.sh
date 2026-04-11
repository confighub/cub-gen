#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

RUN_ID="${RUN_ID:-${GITHUB_RUN_ID:-local}}"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/.tmp/connected-smoke/$RUN_ID}"
SPACE="${SPACE:-}"
VERIFIER="${VERIFIER:-ci-bot}"

examples=(
  "helm-paas"
  "springboot-paas"
)

run_example() {
  local example="$1"
  local repo_path="./examples/$example"
  local example_out="$OUT_DIR/$example"

  mkdir -p "$example_out"

  ./cub-gen gitops discover --space "$SPACE" --json "$repo_path" > "$example_out/discover.json"
  ./cub-gen gitops import --space "$SPACE" --json "$repo_path" "$repo_path" > "$example_out/import.json"
  ./cub-gen publish --in "$example_out/import.json" > "$example_out/bundle.json"
  ./cub-gen verify --json --in "$example_out/bundle.json" > "$example_out/verify.json"
  ./cub-gen attest --in "$example_out/bundle.json" --verifier "$VERIFIER" > "$example_out/attestation.json"
  ./cub-gen verify-attestation --json --in "$example_out/attestation.json" --bundle "$example_out/bundle.json" > "$example_out/attestation-verify.json"

  jq -n \
    --arg example "$example" \
    --arg space "$SPACE" \
    --arg generator_profile "$(jq -r '.summary.generator_profiles[0] // ""' "$example_out/bundle.json")" \
    --arg change_id "$(jq -r '.change_id // ""' "$example_out/bundle.json")" \
    --arg bundle_digest "$(jq -r '.bundle_digest // ""' "$example_out/bundle.json")" \
    --argjson dry_inputs "$(jq '.summary.dry_inputs // 0' "$example_out/bundle.json")" \
    --argjson wet_targets "$(jq '.summary.wet_manifest_targets // 0' "$example_out/bundle.json")" \
    --argjson verify_valid "$(jq '.valid // false' "$example_out/verify.json")" \
    --argjson attestation_valid "$(jq '.valid // false' "$example_out/attestation-verify.json")" \
    --argjson linked_bundle_check "$(jq '.linked_bundle_check // false' "$example_out/attestation-verify.json")" \
    '{
      example: $example,
      space: $space,
      generator_profile: $generator_profile,
      change_id: $change_id,
      bundle_digest: $bundle_digest,
      dry_inputs: $dry_inputs,
      wet_targets: $wet_targets,
      verify_valid: $verify_valid,
      attestation_valid: $attestation_valid,
      linked_bundle_check: $linked_bundle_check
    }' > "$example_out/summary.json"

  jq -e '.change_id != "" and .verify_valid == true and .attestation_valid == true and .linked_bundle_check == true' "$example_out/summary.json" >/dev/null
}

echo "[connected-smoke] preflight (requires ConfigHub auth login or CONFIGHUB_* env)"
require_connected_preflight
if [ -z "$SPACE" ]; then
  SPACE="$CONFIGHUB_SPACE"
fi
print_connected_context

mkdir -p "$OUT_DIR"
if cub context get --json > "$OUT_DIR/context.json" 2>/dev/null; then
  :
fi
cub info > "$OUT_DIR/cub-info.txt"

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[connected-smoke] build cub-gen once"
  go build -o ./cub-gen ./cmd/cub-gen
fi

echo "[connected-smoke] starting run for ${#examples[@]} flagship examples"

declare -a failed=()
for example in "${examples[@]}"; do
  echo
  echo "============================================================"
  echo "[connected-smoke] running: $example"
  echo "============================================================"

  if run_example "$example"; then
    echo "[connected-smoke] PASS: $example"
  else
    echo "[connected-smoke] FAIL: $example"
    failed+=("$example")
  fi
done

echo
if [ "${#failed[@]}" -eq 0 ]; then
  echo "[connected-smoke] all flagship smoke examples passed"
  exit 0
fi

echo "[connected-smoke] failures (${#failed[@]}): ${failed[*]}" >&2
exit 1
