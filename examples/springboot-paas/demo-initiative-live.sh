#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

CUB="${CUB:-cub}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SPACE="${SPACE:-springboot-initiative-live}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/springboot-initiative-live}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
CARD_DIR="$OUT_DIR/card"
EXAMPLE_LABEL="springboot-paas-initiative-live"
INITIATIVE_LABEL="inventory-api-appconfig-gates"
FILTER_SLUG="inventory-api-gate-units-${RUN_ID}"
VIEW_SLUG="inventory-api-appconfig-gates-${RUN_ID}"
CARD_UNIT="initiative-card-$(printf '%s' "$RUN_ID" | tr '[:upper:]' '[:lower:]')"
RENDERED_UNIT="inventory-api-prod"

usage() {
  cat <<EOF
Usage: ./examples/springboot-paas/demo-initiative-live.sh [--cleanup|--explain]

Creates live ConfigHub evidence for the Spring cub-gen Initiative proof.

Default space: ${SPACE}

Environment overrides:
  SPACE       ConfigHub space to create/use
  RUN_ID      Suffix for run-specific live objects
  OUT_ROOT    Local output root
  CUB         cub binary

The default run mutates ConfigHub by creating:
  - one Space
  - one rendered inventory-api Unit
  - one compact Initiative card Unit
  - three per-scenario decision Units
  - three ChangeSets
  - one Filter and one View labeled as an Initiative
EOF
}

sanitize_slug() {
  local input="$1"
  input="$(printf '%s' "$input" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
  input="$(printf '%s' "$input" | sed -E 's/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "${input:0:63}"
}

write_configmap_unit() {
  local name="$1"
  local source="$2"
  local dest="$3"

  {
    printf 'apiVersion: v1\n'
    printf 'kind: ConfigMap\n'
    printf 'metadata:\n'
    printf '  name: %s\n' "$name"
    printf '  namespace: apps\n'
    printf '  labels:\n'
    printf '    app.kubernetes.io/name: inventory-api\n'
    printf '    cubgen.confighub.io/initiative: %s\n' "$INITIATIVE_LABEL"
    printf 'data:\n'
    printf '  gate.json: |-\n'
    sed 's/^/    /' "$source"
  } > "$dest"
}

case "${1:-}" in
  --cleanup)
    echo "[spring-initiative-live] deleting spaces with ExampleName=${EXAMPLE_LABEL}"
    "$CUB" space delete --where "Labels.ExampleName = '${EXAMPLE_LABEL}'" --recursive
    exit 0
    ;;
  --explain)
    usage
    exit 0
    ;;
  "")
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

require_connected_preflight
require_cmd go

mkdir -p "$OUT_DIR"

echo "[spring-initiative-live] connected context"
print_connected_context
"$CUB" context get -o json > "$OUT_DIR/context.json"
ORG_ID="$(jq -r '.coordinate.organizationID // empty' "$OUT_DIR/context.json")"
ORG_NAME="$(jq -r '.metadata.organizationName // empty' "$OUT_DIR/context.json")"
UI_BASE_URL="${CONFIGHUB_BASE_URL%/}"
UI_ORG_QUERY=""
if [ -n "$ORG_ID" ]; then
  UI_ORG_QUERY="?org=${ORG_ID}"
fi
echo "[spring-initiative-live] output: $OUT_DIR"

echo "[spring-initiative-live] generate local gate cards"
OUT_DIR="$CARD_DIR" ./examples/springboot-paas/demo-initiative-gui.sh \
  | tee "$OUT_DIR/local-card.stdout"

echo "[spring-initiative-live] create or reuse ConfigHub space: $SPACE"
"$CUB" space create "$SPACE" \
  --allow-exists \
  -o json \
  --label "ExampleName=${EXAMPLE_LABEL}" \
  --label "Component=inventory-api" \
  --label "Purpose=mutation-apply-gate-live-proof" \
  --annotation "Description=Live cub-gen Spring Initiative proof" \
  > "$OUT_DIR/space.json"

echo "[spring-initiative-live] publish rendered inventory-api Unit"
"$CUB" unit create \
  --space "$SPACE" \
  --allow-exists \
  -o json \
  -t "Kubernetes/YAML" \
  --label "ExampleName=${EXAMPLE_LABEL}" \
  --label "CubGenInitiative=${INITIATIVE_LABEL}" \
  --label "RunID=${RUN_ID}" \
  --label "Component=inventory-api" \
  --label "Variant=inventory-api-prod" \
  --label "Target=prod-us" \
  --label "UnitRole=rendered-config" \
  --annotation "cubgen-note=Rendered Kubernetes config reviewed by the live Initiative proof" \
  "$RENDERED_UNIT" \
  ./examples/springboot-paas/confighub/inventory-api-prod.yaml \
  > "$OUT_DIR/unit-rendered.json"

"$CUB" unit update \
  --space "$SPACE" \
  --patch \
  -o json \
  --label "ExampleName=${EXAMPLE_LABEL}" \
  --label "CubGenInitiative=${INITIATIVE_LABEL}" \
  --label "RunID=${RUN_ID}" \
  --label "Component=inventory-api" \
  --label "Variant=inventory-api-prod" \
  --label "Target=prod-us" \
  --label "UnitRole=rendered-config" \
  --annotation "cubgen-note=Rendered Kubernetes config reviewed by the live Initiative proof" \
  "$RENDERED_UNIT" \
  > "$OUT_DIR/unit-rendered-patch.json"

echo "[spring-initiative-live] create run ChangeSets and decision Units"
declare -a scenario_specs=(
  "allow-feature-flag|01-allow-reservation-mode.json|allow-feature-flag"
  "escalate-redis-cache|02-escalate-redis-cache.json|redis-cache"
  "block-datasource|03-block-datasource.json|datasource-override"
)

for spec in "${scenario_specs[@]}"; do
  IFS="|" read -r scenario_id decision_file suffix <<< "$spec"
  decision_path="$CARD_DIR/$decision_file"
  decision_state="$(jq -r '.decision.state' "$decision_path")"
  route_kind="$(jq -r '.route.kind' "$decision_path")"
  route_label="$(sanitize_slug "$route_kind")"
  owner="$(jq -r '.proof.owner' "$decision_path")"
  field="$(jq -r '.mutation.rendered_field' "$decision_path")"
  field_label="$(sanitize_slug "$field")"
  source_file="$(jq -r '.proof.source_file' "$decision_path")"
  digest="$(jq -r '.decision_digest' "$decision_path")"
  change_id="$(jq -r '.change_id' "$decision_path")"
  next_action_kind="$(jq -r '.next_actions[0].kind // "none"' "$decision_path")"
  proposal_files="$(jq -r '.next_actions[0].files // [] | join(";")' "$decision_path")"
  changeset_slug="$(sanitize_slug "initiative-${suffix}-${RUN_ID}")"
  decision_unit="$(sanitize_slug "decision-${suffix}-${RUN_ID}")"
  decision_unit_file="$OUT_DIR/${decision_unit}.yaml"

  write_configmap_unit "$decision_unit" "$decision_path" "$decision_unit_file"

  "$CUB" changeset create \
    --space "$SPACE" \
    --allow-exists \
    -o json \
    "$changeset_slug" \
    --description "${decision_state}: ${field} via ${route_kind} (${change_id})" \
    --label "ExampleName=${EXAMPLE_LABEL}" \
    --label "cubgen_initiative=${INITIATIVE_LABEL}" \
    --label "run_id=${RUN_ID}" \
    --label "scenario=${scenario_id}" \
    --label "decision=${decision_state}" \
    --label "route=${route_label}" \
    --label "field=${field_label}" \
    --annotation "cubgen-change-id=${change_id}" \
    --annotation "cubgen-field=${field}" \
    --annotation "cubgen-route=${route_kind}" \
    --annotation "cubgen-decision=${decision_state}" \
    --annotation "cubgen-owner=${owner}" \
    --annotation "cubgen-source-file=${source_file}" \
    --annotation "cubgen-next-action=${next_action_kind}" \
    --annotation "cubgen-proposal-files=${proposal_files}" \
    --annotation "cubgen-decision-digest=${digest}" \
    --annotation "cubgen-decision-unit=${decision_unit}" \
    > "$OUT_DIR/changeset-${scenario_id}.json"

  "$CUB" unit create \
    --space "$SPACE" \
    --allow-exists \
    -o json \
    -t "Kubernetes/YAML" \
    --changeset "$changeset_slug" \
    --change-desc "Record cub-gen mutation apply gate decision for ${field}" \
    --label "ExampleName=${EXAMPLE_LABEL}" \
    --label "CubGenInitiative=${INITIATIVE_LABEL}" \
    --label "RunID=${RUN_ID}" \
    --label "Component=inventory-api" \
    --label "Variant=inventory-api-prod" \
    --label "Target=prod-us" \
    --label "UnitRole=mutation-apply-gate-decision" \
    --label "Scenario=${scenario_id}" \
    --label "Decision=${decision_state}" \
    --label "Route=${route_label}" \
    --annotation "cubgen-change-id=${change_id}" \
    --annotation "cubgen-field=${field}" \
    --annotation "cubgen-source-file=${source_file}" \
    --annotation "cubgen-decision-digest=${digest}" \
    "$decision_unit" \
    "$decision_unit_file" \
    > "$OUT_DIR/unit-${scenario_id}.json"
done

echo "[spring-initiative-live] publish compact Initiative card Unit"
card_unit_file="$OUT_DIR/${CARD_UNIT}.yaml"
write_configmap_unit "$(sanitize_slug "$CARD_UNIT")" "$CARD_DIR/initiative-card.json" "$card_unit_file"

"$CUB" unit create \
  --space "$SPACE" \
  --allow-exists \
  -o json \
  -t "Kubernetes/YAML" \
  --label "ExampleName=${EXAMPLE_LABEL}" \
  --label "CubGenInitiative=${INITIATIVE_LABEL}" \
  --label "RunID=${RUN_ID}" \
  --label "Component=inventory-api" \
  --label "Variant=inventory-api-prod" \
  --label "Target=prod-us" \
  --label "UnitRole=initiative-card" \
  --annotation "cubgen-note=Compact card for the current ConfigHub Initiative UI proof" \
  "$CARD_UNIT" \
  "$card_unit_file" \
  > "$OUT_DIR/unit-card.json"

echo "[spring-initiative-live] create Initiative filter and view"
"$CUB" filter create \
  --space "$SPACE" \
  --allow-exists \
  -o json \
  "$FILTER_SLUG" \
  Unit \
  --where-field "Labels.CubGenInitiative = '${INITIATIVE_LABEL}' AND Labels.RunID = '${RUN_ID}'" \
  --label "ExampleName=${EXAMPLE_LABEL}" \
  --label "cubgen_initiative=${INITIATIVE_LABEL}" \
  --label "run_id=${RUN_ID}" \
  > "$OUT_DIR/filter.json"

check_summary="$(jq -r '
  {
    allow: ([.scenarios[] | select(.decision == "ALLOW")] | length),
    escalate: ([.scenarios[] | select(.decision == "ESCALATE")] | length),
    block: ([.scenarios[] | select(.decision == "BLOCK")] | length),
    total: (.scenarios | length)
  } | "allow:\(.allow);escalate:\(.escalate);block:\(.block);total:\(.total)"
' "$CARD_DIR/initiative-card.json")"

"$CUB" view create \
  --space "$SPACE" \
  --allow-exists \
  -o json \
  "$VIEW_SLUG" \
  "$FILTER_SLUG" \
  --column Unit.Slug \
  --column Labels.UnitRole \
  --column Labels.Decision \
  --column Labels.Route \
  --column Labels.Scenario \
  --label "initiative=true" \
  --label "initiative-priority=HIGH" \
  --label "initiative-status=in_progress" \
  --label "ExampleName=${EXAMPLE_LABEL}" \
  --label "cubgen_initiative=${INITIATIVE_LABEL}" \
  --label "run_id=${RUN_ID}" \
  --annotation "initiative-description=Review cub-gen mutation apply gates for inventory-api prod AppConfig edits" \
  --annotation "initiative-deadline=2026-05-31" \
  --annotation "initiative-check-summary=${check_summary}" \
  --annotation "cubgen-card-unit=${CARD_UNIT}" \
  --annotation "cubgen-rendered-unit=${RENDERED_UNIT}" \
  --annotation "cubgen-doc=examples/springboot-paas/docs/initiative-gui.md" \
  > "$OUT_DIR/view.json"

echo "[spring-initiative-live] verify live queries"
"$CUB" unit list \
  --space "$SPACE" \
  -o json \
  --where "Labels.CubGenInitiative = '${INITIATIVE_LABEL}' AND Labels.RunID = '${RUN_ID}'" \
  > "$OUT_DIR/unit-query.json"

"$CUB" changeset list \
  --space "$SPACE" \
  -o json \
  --where "Labels.cubgen_initiative = '${INITIATIVE_LABEL}' AND Labels.run_id = '${RUN_ID}'" \
  > "$OUT_DIR/changeset-query.json"

unit_hits="$(jq 'length' "$OUT_DIR/unit-query.json")"
changeset_hits="$(jq 'length' "$OUT_DIR/changeset-query.json")"
if [ "$unit_hits" -lt 5 ] || [ "$changeset_hits" -lt 3 ]; then
  echo "error: live query proof incomplete (unit_hits=$unit_hits changeset_hits=$changeset_hits)" >&2
  exit 1
fi

SPACE_ID="$(jq -r '.Space.SpaceID // .SpaceID // empty' "$OUT_DIR/space.json")"
CARD_UNIT_ID="$(jq -r '.Unit.UnitID // .UnitID // empty' "$OUT_DIR/unit-card.json")"
VIEW_ID="$(jq -r '.View.ViewID // .ViewID // empty' "$OUT_DIR/view.json")"
SPACE_URL=""
CARD_UNIT_URL=""
if [ -n "$SPACE_ID" ]; then
  SPACE_URL="${UI_BASE_URL}/spaces/${SPACE_ID}${UI_ORG_QUERY}"
fi
if [ -n "$SPACE_ID" ] && [ -n "$CARD_UNIT_ID" ]; then
  CARD_UNIT_URL="${UI_BASE_URL}/units/${SPACE_ID}/${CARD_UNIT_ID}${UI_ORG_QUERY}"
fi

jq -n \
  --arg schema_version "cub.confighub.io/springboot-initiative-live/v1" \
  --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg base_url "$CONFIGHUB_BASE_URL" \
  --arg organization_id "$ORG_ID" \
  --arg organization_name "$ORG_NAME" \
  --arg space "$SPACE" \
  --arg space_id "$SPACE_ID" \
  --arg run_id "$RUN_ID" \
  --arg initiative "$INITIATIVE_LABEL" \
  --arg view "$VIEW_SLUG" \
  --arg view_id "$VIEW_ID" \
  --arg filter "$FILTER_SLUG" \
  --arg card_unit "$CARD_UNIT" \
  --arg card_unit_id "$CARD_UNIT_ID" \
  --arg rendered_unit "$RENDERED_UNIT" \
  --arg space_url "$SPACE_URL" \
  --arg card_unit_url "$CARD_UNIT_URL" \
  --arg local_card "$CARD_DIR/initiative-card.json" \
  --arg unit_query "$OUT_DIR/unit-query.json" \
  --arg changeset_query "$OUT_DIR/changeset-query.json" \
  --argjson unit_hits "$unit_hits" \
  --argjson changeset_hits "$changeset_hits" \
  --slurpfile card "$CARD_DIR/initiative-card.json" \
  '{
    schema_version: $schema_version,
    generated_at: $generated_at,
    confighub: {
      base_url: $base_url,
      organization_id: $organization_id,
      organization_name: $organization_name,
      space: $space,
      space_id: $space_id,
      initiative_view: $view,
      initiative_view_id: $view_id,
      filter: $filter,
      card_unit: $card_unit,
      card_unit_id: $card_unit_id,
      rendered_unit: $rendered_unit,
      space_url: $space_url,
      card_unit_url: $card_unit_url
    },
    run: {
      run_id: $run_id,
      initiative: $initiative,
      local_card: $local_card,
      unit_query: $unit_query,
      changeset_query: $changeset_query,
      unit_hits: $unit_hits,
      changeset_hits: $changeset_hits
    },
    decisions: $card[0].scenarios,
    current_ui_path: [
      "Open ConfigHub",
      ("Open space " + $space),
      ("Open Views/Initiatives and select " + $view),
      ("Open unit " + $card_unit + " for the compact gate card"),
      "Open the three ChangeSets for route, owner, next action, and digest metadata"
    ],
    note: "This is live ConfigHub evidence using current Space, Unit, ChangeSet, Filter, and View primitives. First-class native card rendering is the next ConfigHub UI/backend integration."
  }' > "$OUT_DIR/live-summary.json"

REDIS_CHANGESET_SLUG="$(sanitize_slug "initiative-redis-cache-${RUN_ID}")"

cat <<EOF

Live ConfigHub Initiative evidence is ready.

Organization:   ${ORG_NAME:-unknown} (${ORG_ID:-unknown})
Space:          $SPACE
Initiative view:$VIEW_SLUG
Card unit:      $CARD_UNIT
Run ID:         $RUN_ID
Summary:        $OUT_DIR/live-summary.json

Inspect with:
  cub view get --space $SPACE -o json $VIEW_SLUG
  cub unit get --space $SPACE -o json $CARD_UNIT
  cub changeset list --space $SPACE -o json --where "Labels.cubgen_initiative = '$INITIATIVE_LABEL' AND Labels.run_id = '$RUN_ID'"
  cub changeset get --space $SPACE -o json $REDIS_CHANGESET_SLUG

Open in the current UI:
  ${SPACE_URL:-run: cub space get --web $SPACE}
  ${CARD_UNIT_URL:-run: cub unit get --space $SPACE --web $CARD_UNIT}
  Then open Views/Initiatives in the $SPACE space and select $VIEW_SLUG.

Cleanup:
  SPACE=$SPACE ./examples/springboot-paas/demo-initiative-live.sh --cleanup
EOF
