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

flow_a_root="$ROOT_DIR/.tmp/flow-a"
flow_b_root="$ROOT_DIR/.tmp/flow-b"

if [ -n "${RUN_ID:-}" ]; then
  flow_a_root="$flow_a_root/$RUN_ID"
  flow_b_root="$flow_b_root/$RUN_ID"
fi

flow_a="$(require_json "$flow_a_root" "flow-a-summary.json" "Flow A")"
assert_jq "$flow_a" '.flow == "A"' "flow A identifier"
assert_jq "$flow_a" '.change_id | startswith("chg_")' "flow A change_id"
assert_jq "$flow_a" '.ingest_status != "unknown"' "flow A ingest status"
assert_jq "$flow_a" '.decision_state | IN("ALLOW","ESCALATE","BLOCK")' "flow A decision state"
assert_jq "$flow_a" '.wet_targets > 0' "flow A wet target count"
assert_jq "$flow_a" '.git.repo != "" and .git.pr_number != "none"' "flow A Git linkage"
assert_jq "$flow_a" '.confighub.mr_id != ""' "flow A ConfigHub MR linkage"

flow_b="$(require_json "$flow_b_root" "flow-b-summary.json" "Flow B")"
assert_jq "$flow_b" '.flow == "B"' "flow B identifier"
assert_jq "$flow_b" '.confighub.changeset_slug != "" and .confighub.changeset_id != ""' "flow B ConfigHub MR source"
assert_jq "$flow_b" '.confighub.status == "APPROVED"' "flow B approval state"
assert_jq "$flow_b" '(.git.repo != "") and (.git.branch | startswith("confighub/"))' "flow B Git branch proposal"
assert_jq "$flow_b" '.git.status == "PROPOSED"' "flow B Git PR proposal state"
assert_jq "$flow_b" '.proposed_edit.dry_file != "" and .proposed_edit.dry_path != ""' "flow B inverse-edit guidance"

echo "ok: Flow A and Flow B evidence artifacts are present and valid"
