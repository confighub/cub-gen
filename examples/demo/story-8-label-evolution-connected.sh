#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

REPO_PATH="${1:-./examples/backstage-idp}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-8}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"
CLEANUP_STORY_8_CHANGESET="${CLEANUP_STORY_8_CHANGESET:-0}"

sanitize_slug() {
  local input="$1"
  input="$(printf '%s' "$input" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
  input="$(printf '%s' "$input" | sed -E 's/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "${input:0:63}"
}

query_changesets() {
  local where_expr="$1"
  local output="$2"
  cub changeset list --space "$CONFIGHUB_SPACE" --json --where "$where_expr" > "$output"
}

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

require_connected_preflight

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

change_id="$(jq -r .change_id "$OUT_DIR/update/bundle.json")"
migration_slug="$(sanitize_slug "story8-taxonomy-${EXAMPLE_SLUG}-${RUN_ID}")"
backend_changeset="$OUT_DIR/update/migration-changeset.json"

cub changeset create \
  --space "$CONFIGHUB_SPACE" \
  --json \
  "$migration_slug" \
  --description "Story 8 taxonomy migration anchor for change_id=${change_id}" \
  --label "story=8" \
  --label "change_id=${change_id}" \
  --label "lifecycle=production" \
  --label "environment_tier=prod" \
  --label "migration_phase=dual_read" \
  > "$backend_changeset"

migration_changeset_id="$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$backend_changeset")"
if [ -z "$migration_changeset_id" ]; then
  echo "error: unable to resolve ChangeSetID from backend Story 8 migration anchor." >&2
  exit 1
fi

legacy_where="ChangeSetID = '${migration_changeset_id}' AND Labels.lifecycle = 'production'"
new_where="ChangeSetID = '${migration_changeset_id}' AND Labels.environment_tier = 'prod'"

legacy_query="$OUT_DIR/update/migration-query-legacy.json"
new_query="$OUT_DIR/update/migration-query-new.json"
union_query="$OUT_DIR/update/migration-query-transition-union.json"
query_changesets "$legacy_where" "$legacy_query"
query_changesets "$new_where" "$new_query"
jq -s 'add | unique_by(.ChangeSet.ChangeSetID)' "$legacy_query" "$new_query" > "$union_query"

legacy_hits="$(jq 'length' "$legacy_query")"
new_hits="$(jq 'length' "$new_query")"
union_hits="$(jq 'length' "$union_query")"
if [ "$legacy_hits" -lt 1 ] || [ "$new_hits" -lt 1 ] || [ "$union_hits" -lt 1 ]; then
  echo "error: taxonomy compatibility queries did not return backend results." >&2
  echo "  legacy_hits=$legacy_hits new_hits=$new_hits transition_union_hits=$union_hits" >&2
  exit 1
fi

migration_contract="$OUT_DIR/update/label-migration-contract.json"
jq -n \
  --arg change_id "$change_id" \
  --arg changeset_id "$migration_changeset_id" \
  --arg changeset_slug "$migration_slug" \
  --arg changeset_file "$backend_changeset" \
  --arg from_key "metadata.labels.lifecycle" \
  --arg to_key "metadata.labels.environment_tier" \
  --arg legacy_where "$legacy_where" \
  --arg new_where "$new_where" \
  --arg union_where "legacy-query-union + new-query-union (computed in script)" \
  --arg legacy_query "$legacy_query" \
  --arg new_query "$new_query" \
  --arg union_query "$union_query" \
  --argjson legacy_hits "$legacy_hits" \
  --argjson new_hits "$new_hits" \
  --argjson union_hits "$union_hits" \
  '{
    schema: "confighub.io/taxonomy-migration/v2",
    change_id: $change_id,
    backend_anchor: {
      changeset_id: $changeset_id,
      changeset_slug: $changeset_slug,
      changeset_file: $changeset_file
    },
    migration: {
      from: $from_key,
      to: $to_key,
      strategy: "dual-read-then-cutover",
      repo_surgery_required: false
    },
    compatibility_queries: {
      legacy: {
        where: $legacy_where,
        result_file: $legacy_query,
        hits: $legacy_hits
      },
      new: {
        where: $new_where,
        result_file: $new_query,
        hits: $new_hits
      },
      transition_union: {
        where: $union_where,
        result_file: $union_query,
        hits: $union_hits
      }
    }
  }' > "$migration_contract"

if [ "$CLEANUP_STORY_8_CHANGESET" = "1" ]; then
  cub changeset delete --space "$CONFIGHUB_SPACE" --quiet "$migration_slug" || true
fi

jq -n \
  --arg story "8-label-taxonomy-evolution" \
  --arg change_id "$change_id" \
  --arg bundle_digest "$(jq -r .bundle_digest "$OUT_DIR/update/bundle.json")" \
  --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/decision-final.json")" \
  --arg ingest_status "$(jq -r '.status // "unknown"' "$OUT_DIR/update/ingest.json")" \
  --arg contract "$migration_contract" \
  --arg backend_changeset "$backend_changeset" \
  --arg labels_file "$label_fields" \
  --argjson legacy_hits "$legacy_hits" \
  --argjson new_hits "$new_hits" \
  --argjson transition_union_hits "$union_hits" \
  --argjson label_fields_count "$(jq 'length' "$label_fields")" \
  '{
    story: $story,
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    decision_state: $decision_state,
    ingest_status: $ingest_status,
    migration_contract: $contract,
    backend_migration_changeset: $backend_changeset,
    compatibility_query_hits: {
      legacy: $legacy_hits,
      new: $new_hits,
      transition_union: $transition_union_hits
    },
    label_field_origins: $labels_file,
    label_field_origins_count: $label_fields_count
  }' | tee "$OUT_DIR/story-8-summary.json"
