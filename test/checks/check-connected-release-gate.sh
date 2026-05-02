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
  if ! grep -Eq "^ci-connected: .*\b${dep}\b" Makefile; then
    fail "ci-connected target is missing dependency: $dep"
  fi
}

forbid_make_dep() {
  local dep="$1"
  if grep -Eq "^ci-connected: .*\b${dep}\b" Makefile; then
    fail "ci-connected target should not depend on: $dep"
  fi
}

tmp_json="$(mktemp)"
trap 'rm -f "$tmp_json"' EXIT

go run ./tools/example-truth-matrix --format json >"$tmp_json"

require_make_dep "test-connected-smoke"
forbid_make_dep "test-connected-lifecycles"
forbid_make_dep "test-phase-3-stories"
forbid_make_dep "test-phase-4-stories"
forbid_make_dep "test-flow-a-git-pr-to-mr"
forbid_make_dep "test-flow-b-mr-to-git-pr"
forbid_make_dep "test-connected-governed-reconcile-helm"
forbid_make_dep "test-live-reconcile-flux"
forbid_make_dep "test-live-reconcile-argo"
forbid_make_dep "check-story-evidence"
forbid_make_dep "check-flow-evidence"

if ! grep -q "gate mutation" examples/demo/run-connected-smoke.sh; then
  fail "connected smoke must record one mutation apply gate decision"
fi
if ! grep -q "gate_decision_digest" examples/demo/run-connected-smoke.sh; then
  fail "connected smoke summary must expose the mutation gate decision digest"
fi

assert_jq "$tmp_json" '.summary.generator_fixtures == 8' "expected eight first-class generator fixtures"
assert_jq "$tmp_json" '.summary.connected_release_gated == 2' "expected exactly two flagship examples in the connected smoke lane"
assert_jq "$tmp_json" '[.rows[] | select(.connected_release_gated)] | map(.example) | sort == ["helm-paas", "springboot-paas"]' "connected smoke lane should cover helm-paas and springboot-paas"
assert_jq "$tmp_json" '[.rows[] | select(.connected_release_gated and (.connected_mode_present | not))] | length == 0' "connected smoke examples must expose connected mode entrypoints"
assert_jq "$tmp_json" '[.rows[] | select(.real_live_proof == "paired-harness" and .connected_release_gated)] | length >= 1' "connected smoke lane must keep a paired real-live proof path"
assert_jq "$tmp_json" '[.rows[] | select(.real_live_proof == "standalone" and .connected_release_gated)] | length >= 1' "connected smoke lane must keep a standalone real-live proof path"
assert_jq "$tmp_json" '[.rows[] | select(.example == "live-reconcile" and .real_live_proof == "standalone")] | length == 1' "live-reconcile must remain the standalone runtime proof harness"

echo "ok: connected smoke lane stays small, honest, and anchored to the flagship proof paths"
