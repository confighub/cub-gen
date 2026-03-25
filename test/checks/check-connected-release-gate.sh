#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "error: $*" >&2
  exit 1
}

assert_jq() {
  local file="$1"
  local expr="$2"
  local label="$3"
  if ! jq -e "$expr" "$file" >/dev/null; then
    fail "$label (expr: $expr)"
  fi
}

require_make_dep() {
  local dep="$1"
  if ! rg -q "^ci-connected: .*\\b${dep}\\b" Makefile; then
    fail "ci-connected target is missing dependency: $dep"
  fi
}

tmp_json="$(mktemp)"
trap 'rm -f "$tmp_json"' EXIT

go run ./tools/example-truth-matrix --format json >"$tmp_json"

require_make_dep "test-connected-lifecycles"
require_make_dep "test-phase-3-stories"
require_make_dep "test-phase-4-stories"
require_make_dep "test-flow-a-git-pr-to-mr"
require_make_dep "test-flow-b-mr-to-git-pr"
require_make_dep "test-connected-governed-reconcile-helm"
require_make_dep "test-live-reconcile-flux"
require_make_dep "test-live-reconcile-argo"
require_make_dep "check-story-evidence"
require_make_dep "check-flow-evidence"

assert_jq "$tmp_json" '.summary.generator_fixtures == 8' "expected eight first-class generator fixtures"
assert_jq "$tmp_json" '.summary.connected_release_gated == 12' "expected every featured example to be connected-release-gated"
assert_jq "$tmp_json" '[.rows[] | select(.generator_fixture and (.connected_release_gated | not))] | length == 0' "all generator fixtures must be in the connected release gate"
assert_jq "$tmp_json" '[.rows[] | select(.connected_release_gated and (.connected_mode_present | not))] | length == 0' "release-gated examples must expose connected mode entrypoints"
assert_jq "$tmp_json" '[.rows[] | select(.real_live_proof == "paired-harness" and .connected_release_gated)] | length >= 1' "release gate must keep a paired real-live proof path"
assert_jq "$tmp_json" '[.rows[] | select(.real_live_proof == "standalone" and .connected_release_gated)] | length >= 1' "release gate must keep a standalone real-live proof path"
assert_jq "$tmp_json" '[.rows[] | select(.example == "helm-paas" and .real_live_proof == "paired-harness")] | length == 1' "helm-paas must remain the paired flagship live-proof example"
assert_jq "$tmp_json" '[.rows[] | select(.example == "live-reconcile" and .real_live_proof == "standalone")] | length == 1' "live-reconcile must remain the standalone runtime proof harness"

echo "ok: connected release gate still covers the flagship connected, flow A/B, and real-live proof lanes"
