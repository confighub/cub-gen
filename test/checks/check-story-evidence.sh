#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "error: $*" >&2
  exit 1
}

latest_json() {
  local search_root="$1"
  local filename="$2"
  find "$search_root" -type f -name "$filename" 2>/dev/null | sort | tail -n1
}

require_json() {
  local search_root="$1"
  local filename="$2"
  local label="$3"
  local path
  path="$(latest_json "$search_root" "$filename")"
  if [ -z "$path" ]; then
    fail "missing evidence for $label ($search_root/**/$filename)"
  fi
  printf '%s' "$path"
}

assert_jq() {
  local file="$1"
  local expr="$2"
  local label="$3"
  if ! jq -e "$expr" "$file" >/dev/null; then
    echo "error: evidence assertion failed for $label" >&2
    echo "  file: $file" >&2
    echo "  expr: $expr" >&2
    exit 1
  fi
}

assert_no_fallback() {
  local file="$1"
  local expr="$2"
  local label="$3"
  if [ "${ALLOW_FALLBACK_INGEST:-0}" = "1" ]; then
    return 0
  fi
  assert_jq "$file" "$expr" "$label"
}

story1_root="$ROOT_DIR/.tmp/story-1"
story7_root="$ROOT_DIR/.tmp/ci-connected"
story8_root="$ROOT_DIR/.tmp/story-8"
story9_root="$ROOT_DIR/.tmp/waves"
story10_root="$ROOT_DIR/.tmp/story-10"
story11_root="$ROOT_DIR/.tmp/story-11"
story12_root="$ROOT_DIR/.tmp/story-12"
story13_root="$ROOT_DIR/.tmp/e2e-connected-governed-reconcile-helm"

if [ -n "${RUN_ID:-}" ]; then
  story1_root="$story1_root/$RUN_ID"
  story7_root="$story7_root/$RUN_ID"
  story8_root="$story8_root/$RUN_ID"
  story10_root="$story10_root/$RUN_ID"
  story11_root="$story11_root/$RUN_ID"
  story12_root="$story12_root/$RUN_ID"
  story13_root="$story13_root/$RUN_ID"
fi
if [ -n "${WAVE_ID:-}" ]; then
  story9_root="$story9_root/$WAVE_ID"
fi

story1="$(require_json "$story1_root" "story-1-summary.json" "story 1")"
assert_jq "$story1" '.change_id | startswith("chg_")' "story 1 change_id"
assert_jq "$story1" '.provenance_summary.wet_targets > 0 and .provenance_summary.provenance_records > 0' "story 1 provenance"
assert_jq "$story1" '.decision_state | IN("ALLOW","ESCALATE","BLOCK")' "story 1 decision state"
assert_no_fallback "$story1" '.ingest_status != "ingested-fallback"' "story 1 no fallback ingest"

story7="$(require_json "$story7_root" "story-7-summary.json" "story 7")"
assert_jq "$story7" '.change_id | startswith("chg_")' "story 7 change_id"
assert_jq "$story7" '.ingest_status != "unknown"' "story 7 ingest status"
assert_jq "$story7" '.decision_state | IN("ALLOW","ESCALATE","BLOCK")' "story 7 decision state"
assert_no_fallback "$story7" '.ingest_status != "ingested-fallback"' "story 7 no fallback ingest"

story8="$(require_json "$story8_root" "story-8-summary.json" "story 8")"
assert_jq "$story8" '.compatibility_query_hits.legacy >= 1 and .compatibility_query_hits.new >= 1 and .compatibility_query_hits.transition_union >= 1' "story 8 query hits"
assert_jq "$story8" '.label_field_origins_count >= 1' "story 8 label field origins"
assert_no_fallback "$story8" '.ingest_status != "ingested-fallback"' "story 8 no fallback ingest"
migration_contract="$(jq -r '.migration_contract // empty' "$story8")"
[ -n "$migration_contract" ] || fail "story 8 migration_contract path missing"
[ -f "$migration_contract" ] || fail "story 8 migration_contract file missing: $migration_contract"
assert_jq "$migration_contract" '.migration.repo_surgery_required == false' "story 8 no repo surgery"
assert_jq "$migration_contract" '.migration.from != "" and .migration.to != "" and .migration.source_value != "" and .migration.target_value != ""' "story 8 migration values"

story9="$(require_json "$story9_root" "wave-summary.json" "story 9")"
assert_jq "$story9" '.totals.repositories > 0' "story 9 repository count"
assert_jq "$story9" '(.totals.allow + .totals.escalate + .totals.block) == .totals.repositories' "story 9 decision accounting"
assert_no_fallback "$story9" '[.targets[].ingest_status] | all(. != "ingested-fallback")' "story 9 no fallback ingest"

story10="$(latest_json "$story10_root" "story-10-summary.json")"
if [ -n "$story10" ]; then
  assert_jq "$story10" '.signatures_verified == true and .branch_protection_preserved == true' "story 10 signed write-back proof"
  assert_no_fallback "$story10" '.ingest_status != "ingested-fallback"' "story 10 no fallback ingest"
elif [ "${ALLOW_STORY_10_SKIP:-0}" = "1" ]; then
  echo "[story-evidence] warning: story 10 summary not found; skipping because ALLOW_STORY_10_SKIP=1."
else
  fail "missing evidence for story 10 (.tmp/story-10/**/story-10-summary.json). set APP_PR_REPO/APP_PR_NUMBER/PROMOTION_PR_REPO/PROMOTION_PR_NUMBER (and GH_TOKEN/GITHUB_TOKEN) or use ALLOW_STORY_10_SKIP=1 for local troubleshooting"
fi

story11="$(require_json "$story11_root" "story-11-summary.json" "story 11")"
assert_jq "$story11" '.proposals | length == 2' "story 11 proposal count"
assert_jq "$story11" '[.proposals[].backend_query_hits] | all(. >= 1)' "story 11 proposal query hits"
proposal_file="$(jq -r '.live_decision_proposal // empty' "$story11")"
[ -n "$proposal_file" ] || fail "story 11 live_decision_proposal path missing"
[ -f "$proposal_file" ] || fail "story 11 live_decision_proposal file missing: $proposal_file"
assert_jq "$proposal_file" '.observation.before != .observation.after' "story 11 live observation delta"
story11_dir="$(dirname "$story11")"
story11_ingest="$story11_dir/update/ingest.json"
[ -f "$story11_ingest" ] || fail "story 11 ingest evidence missing: $story11_ingest"
assert_no_fallback "$story11_ingest" '.status != "ingested-fallback"' "story 11 no fallback ingest"

story12="$(require_json "$story12_root" "story-12-summary.json" "story 12")"
assert_jq "$story12" '.actor_chain | length == 3' "story 12 actor chain length"
assert_jq "$story12" '[.actor_chain[].attestation_digest] | all(. != "")' "story 12 actor attestations"
assert_jq "$story12" '.decision_state | IN("ALLOW","ESCALATE","BLOCK")' "story 12 decision state"
assert_no_fallback "$story12" '.ingest_status != "ingested-fallback"' "story 12 no fallback ingest"

connected_e2e="$(require_json "$story13_root" "summary.json" "story 13 connected e2e")"
assert_jq "$connected_e2e" '.governed_change.decision_state == "ALLOW"' "story 13 connected decision"
assert_jq "$connected_e2e" '.live_reconcile.flux_ok == true and .live_reconcile.argo_ok == true' "story 13 live reconcile"
assert_no_fallback "$connected_e2e" '.governed_change.ingest_status != "ingested-fallback"' "story 13 no fallback ingest"

echo "ok: connected story evidence artifacts are present and valid"
