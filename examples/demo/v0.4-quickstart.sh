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
LIFT="$OUT_DIR/spring-lift.json"
BLOCK="$OUT_DIR/spring-block.json"
./cub-gen gate mutation \
  --routes ./examples/springboot-paas/operational/field-routes.yaml \
  --json \
  --at 2026-05-02T12:00:00Z \
  feature.inventory.reservationMode >"$ALLOW"

./cub-gen gate mutation \
  --routes ./examples/springboot-paas/operational/field-routes.yaml \
  --json \
  --at 2026-05-02T12:00:00Z \
  spring.cache.type >"$LIFT"

set +e
./cub-gen gate mutation \
  --routes ./examples/springboot-paas/operational/field-routes.yaml \
  --json \
  --enforce \
  --at 2026-05-02T12:00:00Z \
  spring.datasource.url >"$BLOCK" 2>"$OUT_DIR/spring-block.err"
block_status=$?
set -e

if [[ "$block_status" -eq 0 ]]; then
  echo "error: expected spring.datasource.url to be blocked" >&2
  exit 1
fi

jq -e '.route.kind == "apply-here" and .decision.state == "ALLOW" and (.proof_events | length) == 1' "$ALLOW" >/dev/null
jq -e '.route.kind == "lift-upstream" and .decision.state == "ESCALATE" and (.proof_events | length) == 1' "$LIFT" >/dev/null
jq -e '.route.kind == "block/escalate" and .decision.state == "BLOCK" and (.proof_events | length) == 1' "$BLOCK" >/dev/null
jq -r '
  "Apply here: " + .mutation.rendered_field + " route=" + .route.kind + " decision=" + .decision.state + " owner=" + .proof.owner
' "$ALLOW"
jq -r '
  "Lift upstream: " + .mutation.rendered_field + " route=" + .route.kind + " decision=" + .decision.state + " next=" + .next_actions[0].kind
' "$LIFT"
jq -r '
  "Block/escalate: " + .mutation.rendered_field + " route=" + .route.kind + " decision=" + .decision.state + " owner=" + .proof.owner
' "$BLOCK"

echo
echo "[v0.4] 4. Deployment adaptation gate"
ADAPT="$OUT_DIR/deployment-adapt.json"
./cub-gen platform adapt --json ./testdata/deployment-adaptation/platform.yaml >"$ADAPT"

jq -e '.deployments[] | select(.id == "checkout-api/prod-us" and .variant_kind == "deployment" and .target == "prod-us")' "$ADAPT" >/dev/null
jq -e '.deployments[0].apply_gate.state == "blocked-before-adaptation"' "$ADAPT" >/dev/null
jq -e '.summary.placeholder_count == 3 and .summary.proposed_replacement_count == 3' "$ADAPT" >/dev/null

jq -r '
  .deployments[0] as $deployment |
  "Apply gate: " + $deployment.apply_gate.name + " state=" + $deployment.apply_gate.state + " unresolved=" + ($deployment.apply_gate.unresolved_count | tostring),
  ($deployment.placeholders[] | "Adapt: " + .token + " -> " + .value + " route=" + .route + " owner=" + .owner)
' "$ADAPT"

echo
echo "[v0.4] 5. Proof bundle"
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
