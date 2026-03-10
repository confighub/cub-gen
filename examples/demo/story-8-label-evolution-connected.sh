#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/backstage-idp}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-8}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[story-8] label/taxonomy evolution without repo surgery"
echo "[story-8] output: $OUT_DIR"

./examples/demo/simulate-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$OUT_DIR"

label_fields="$OUT_DIR/update/label-field-origins.json"
jq '
  [.provenance[]
   | .field_origin_map[]
   | select(.wet_path | test("labels\\["))
   | {
       wet_path,
       dry_file: .dry_source.file,
       dry_path: .dry_source.path,
       owner: .ownership.owner,
       confidence
     }]
' "$OUT_DIR/update/import.json" > "$label_fields"

migration_contract="$OUT_DIR/update/label-migration-contract.json"
jq -n \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg from_key "metadata.labels.lifecycle" \
  --arg to_key "metadata.labels.environment_tier" \
  '{
    schema: "confighub.io/taxonomy-migration/v1",
    change_id: $change_id,
    migration: {
      from: $from_key,
      to: $to_key,
      strategy: "dual-read-then-cutover",
      repo_surgery_required: false
    },
    compatibility_queries: {
      legacy: "label.lifecycle=production",
      new: "label.environment_tier=prod",
      transition_union: "(label.lifecycle=production) OR (label.environment_tier=prod)"
    }
  }' > "$migration_contract"

jq -n \
  --arg story "8-label-taxonomy-evolution" \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg bundle_digest "$(jq -r .bundle_digest "$OUT_DIR/update/bundle.json")" \
  --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/decision-final.json")" \
  --arg ingest_status "$(jq -r '.status // "unknown"' "$OUT_DIR/update/ingest.json")" \
  --arg contract "$migration_contract" \
  --arg labels_file "$label_fields" \
  --argjson label_fields_count "$(jq 'length' "$label_fields")" \
  '{
    story: $story,
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    decision_state: $decision_state,
    ingest_status: $ingest_status,
    migration_contract: $contract,
    label_field_origins: $labels_file,
    label_field_origins_count: $label_fields_count
  }' | tee "$OUT_DIR/story-8-summary.json"
