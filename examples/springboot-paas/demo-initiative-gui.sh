#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_DIR="${OUT_DIR:-$ROOT_DIR/.tmp/springboot-initiative-gui}"
ROUTES="$ROOT_DIR/examples/springboot-paas/operational/field-routes.yaml"
CUB_GEN="${CUB_GEN:-$ROOT_DIR/cub-gen}"
AT="${AT:-2026-05-03T12:00:00Z}"

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required" >&2
  exit 1
fi

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  go build -o "$CUB_GEN" ./cmd/cub-gen
elif [ ! -x "$CUB_GEN" ]; then
  echo "error: $CUB_GEN is not executable and SKIP_BUILD=1" >&2
  exit 1
fi

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

common=(
  gate mutation
  --routes "$ROUTES"
  --json
  --pretty=false
  --at "$AT"
  --space inventory-api-prod
  --component inventory-api
  --variant inventory-api-prod
  --target prod-us
  --resource-type ConfigMap
  --resource-name inventory-api-config
  --origin confighub-initiative
  --attempted-layer rendered-config
)

"$CUB_GEN" "${common[@]}" \
  --change-id chg_inventory_reservation_mode \
  --new-value optimistic \
  feature.inventory.reservationMode \
  >"$OUT_DIR/01-allow-reservation-mode.json"

"$CUB_GEN" "${common[@]}" \
  --change-id chg_inventory_redis_cache \
  --new-value redis \
  --github-pr-repo github.com/confighub/cub-gen \
  spring.cache.type \
  >"$OUT_DIR/02-escalate-redis-cache.json"

"$CUB_GEN" "${common[@]}" \
  --change-id chg_inventory_datasource_override \
  --new-value jdbc:postgresql://shadow.platform.svc:5432/inventory \
  spring.datasource.url \
  >"$OUT_DIR/03-block-datasource.json"

jq -s '
  def scenario($id; $title; $question; $unit; $old; $new; $decision):
    {
      id: $id,
      title: $title,
      user_question: $question,
      confighub_unit: $unit,
      changed_field: $decision.mutation.rendered_field,
      old_value: $old,
      new_value: $new,
      route: $decision.route.kind,
      decision: $decision.decision.state,
      owner: $decision.proof.owner,
      generator: $decision.proof.generator,
      source_file: $decision.proof.source_file,
      source_field: $decision.proof.source_field,
      gate: $decision.gate.name,
      decision_digest: $decision.decision_digest,
      next_actions: $decision.next_actions,
      proof_event_count: ($decision.proof_events | length)
    };

  {
    schema_version: "cub.confighub.io/initiative-gui-example/v1",
    generated_at: "2026-05-03T12:00:00Z",
    title: "ConfigHub Initiative: inventory-api prod AppConfig change",
    component: "inventory-api",
    variant: "inventory-api-prod",
    target: "prod-us",
    current_confighub_mapping: {
      gui_object: "Initiative",
      cli_backing: "ChangeSet + Unit revision + Function/Trigger result + ApplyGates",
      note: "The current public cub CLI exposes ChangeSets and ApplyGates. This example produces the gate cards the GUI should show on an Initiative."
    },
    gui_panels: [
      "changed AppConfig field",
      "Generator proof",
      "mutation apply gate",
      "next action",
      "proof events and digest"
    ],
    scenarios: [
      scenario(
        "allow-feature-flag";
        "ALLOW: app-owned feature flag";
        "Can I flip reservation mode in prod?";
        "inventory-api-prod/inventory-api";
        "normal";
        "optimistic";
        .[0]
      ),
      scenario(
        "escalate-redis-cache";
        "ESCALATE: source change required";
        "Can I add Redis caching by editing rendered AppConfig?";
        "inventory-api-prod/inventory-api";
        "none";
        "redis";
        .[1]
      ),
      scenario(
        "block-datasource";
        "BLOCK: platform-owned connection";
        "Can I point prod at a shadow datasource?";
        "inventory-api-prod/inventory-api";
        "managed connection";
        "shadow database URL";
        .[2]
      )
    ]
  }
' \
  "$OUT_DIR/01-allow-reservation-mode.json" \
  "$OUT_DIR/02-escalate-redis-cache.json" \
  "$OUT_DIR/03-block-datasource.json" \
  >"$OUT_DIR/initiative-card.json"

jq -r '
  "ConfigHub Initiative GUI example written to: " + input_filename,
  "",
  (.scenarios[] | "- " + .title + ": field=" + .changed_field + " route=" + .route + " decision=" + .decision + " owner=" + .owner)
' "$OUT_DIR/initiative-card.json"
