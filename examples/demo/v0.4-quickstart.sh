#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required for the v0.4 quickstart" >&2
  echo "hint: install jq, then rerun ./examples/demo/v0.4-quickstart.sh" >&2
  exit 1
fi

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  echo "[v0.4] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

OUT_DIR="${OUT_DIR:-$ROOT_DIR/.tmp/v0.4-quickstart}"
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

echo
echo "[v0.4] question"
echo "If this deployed field is wrong, where do I fix it?"

echo
echo "[v0.4] 1. Component and Variant topology"
TOPOLOGY="$OUT_DIR/variant-topology.json"
./cub-gen platform import --json ./testdata/variant-topology/platform.yaml >"$TOPOLOGY"

jq -e '((.diagnostics // []) | length) == 0' "$TOPOLOGY" >/dev/null
jq -e '.components[] | select(.id == "checkout-api")' "$TOPOLOGY" >/dev/null
jq -e '.variants[] | select(.id == "checkout-api/base" and .variant_kind == "base")' "$TOPOLOGY" >/dev/null
jq -e '.variants[] | select(.id == "checkout-api/prod-us" and .variant_kind == "deployment" and .target == "prod-us")' "$TOPOLOGY" >/dev/null

jq -r '
  "Component: " + (.components[0].id) + " owner=" + (.components[0].owner // "-"),
  "Generators: " + ([.generators[].profile] | unique | join(", ")),
  (.variants[] | "Variant: " + .id + " kind=" + .variant_kind + " target=" + (.target // "-") + " owner=" + (.owner // "-"))
' "$TOPOLOGY"

echo
echo "[v0.4] 2. Rendered field provenance"
IMPORT="$OUT_DIR/helm-import.json"
./cub-gen gitops import --space platform --json ./examples/helm-paas >"$IMPORT"

jq -e '.provenance[0].field_origin_map[0].source_path != null' "$IMPORT" >/dev/null
jq -r '
  .provenance[0].field_origin_map[0] as $origin |
  .provenance[0].inverse_edit_pointers[0] as $edit |
  "Generator: " + .discovered[0].generator_profile,
  "Rendered field: " + $origin.wet_path,
  "Source file/path: " + $origin.source_path + "#" + $origin.dry_path,
  "Owner: " + $edit.owner,
  "Edit hint: " + $edit.edit_hint
' "$IMPORT"

echo
echo "[v0.4] 3. Change route proof"
ALLOW="$OUT_DIR/spring-allow.json"
BLOCK="$OUT_DIR/spring-block.json"
./cub-gen springboot validate-mutation \
  --routes ./examples/springboot-paas/operational/field-routes.yaml \
  --json feature.inventory.reservationMode >"$ALLOW"

set +e
./cub-gen springboot validate-mutation \
  --routes ./examples/springboot-paas/operational/field-routes.yaml \
  --json spring.datasource.url >"$BLOCK" 2>"$OUT_DIR/spring-block.err"
block_status=$?
set -e

if [[ "$block_status" -eq 0 ]]; then
  echo "error: expected spring.datasource.url to be blocked" >&2
  exit 1
fi

jq -e '.allowed == true and .action == "mutable-in-ch"' "$ALLOW" >/dev/null
jq -e '.allowed == false and .action == "generator-owned"' "$BLOCK" >/dev/null
jq -r '
  "Apply here: " + .field_path + " action=" + .action + " owner=" + .owner + " reason=" + .reason
' "$ALLOW"
jq -r '
  "Block/escalate: " + .field_path + " action=" + .action + " owner=" + .owner + " reason=" + .reason
' "$BLOCK"

echo
echo "[v0.4] 4. Proof bundle"
BUNDLE="$OUT_DIR/helm-bundle.json"
VERIFY="$OUT_DIR/helm-verify.txt"
./cub-gen publish --space platform --pretty ./examples/helm-paas >"$BUNDLE"
./cub-gen verify --in "$BUNDLE" >"$VERIFY"

jq -e '.change_id != "" and .bundle_digest != ""' "$BUNDLE" >/dev/null
jq -r '
  "Change: " + .change_id,
  "Proof bundle: " + .bundle_digest,
  "Proof file: '"$BUNDLE"'"
' "$BUNDLE"
cat "$VERIFY"

echo
echo "[v0.4] done"
echo "Local proof is in $OUT_DIR"
echo "Connected handoff: cub auth login, then run ./examples/demo/run-connected-smoke.sh"
