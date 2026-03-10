#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/lifecycle-update.sh"
source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}" 
OUTPUT_DIR="${4:-}"
SPACE="${SPACE:-}"
VERIFIER="${VERIFIER:-ci-bot}"
DECISION_STATE="${DECISION_STATE:-ALLOW}"
DECISION_APPROVED_BY="${DECISION_APPROVED_BY:-platform-owner}"
DECISION_POLICY_REF="${DECISION_POLICY_REF:-}"
DECISION_REASON_PREFIX="${DECISION_REASON_PREFIX:-governance}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

if [ ! -d "$RENDER_TARGET" ]; then
  echo "error: render target path not found: $RENDER_TARGET" >&2
  exit 1
fi

case "$DECISION_STATE" in
  ALLOW|ESCALATE|BLOCK) ;;
  *)
    echo "error: unsupported DECISION_STATE: $DECISION_STATE (expected ALLOW|ESCALATE|BLOCK)" >&2
    exit 1
    ;;
esac

require_connected_preflight
if [ -z "$SPACE" ]; then
  SPACE="$CONFIGHUB_SPACE"
fi
print_connected_context

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[lifecycle][connected] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

run_governed_cycle_connected() {
  local phase="$1"
  local repo="$2"
  local render_target="$3"
  local outdir="$4"

  mkdir -p "$outdir"

  ./cub-gen gitops discover --space "$SPACE" --json "$repo" > "$outdir/discover.json"
  ./cub-gen gitops import --space "$SPACE" --json "$repo" "$render_target" > "$outdir/import.json"
  ./cub-gen publish --in "$outdir/import.json" > "$outdir/bundle.json"
  ./cub-gen verify --json --in "$outdir/bundle.json" > "$outdir/verify.json"
  ./cub-gen attest --in "$outdir/bundle.json" --verifier "$VERIFIER" > "$outdir/attestation.json"
  ./cub-gen verify-attestation --json --in "$outdir/attestation.json" --bundle "$outdir/bundle.json" > "$outdir/attestation-verify.json"

  echo "[connected][$phase] ingest bundle into ConfigHub"
  if ! ./cub-gen bridge ingest \
    --in "$outdir/bundle.json" \
    --base-url "$CONFIGHUB_BASE_URL" \
    --token "$CONFIGHUB_TOKEN" \
    > "$outdir/ingest.json"; then
    echo "error: connected ingest failed for $phase." >&2
    echo "remediation: ensure CONFIGHUB_BASE_URL points to a ConfigHub backend that exposes /api/v1/governed-wet-artifacts:ingest." >&2
    return 1
  fi

  local change_id
  change_id="$(jq -r .change_id "$outdir/bundle.json")"

  echo "[connected][$phase] query decision state from ConfigHub"
  ./cub-gen bridge decision query \
    --base-url "$CONFIGHUB_BASE_URL" \
    --token "$CONFIGHUB_TOKEN" \
    --change-id "$change_id" \
    > "$outdir/decision-query.json"

  # Keep local contract artifacts for continuity with existing promotion demos.
  ./cub-gen bridge decision create --ingest "$outdir/ingest.json" > "$outdir/decision.json"
  ./cub-gen bridge decision attach --decision "$outdir/decision.json" --attestation "$outdir/attestation.json" > "$outdir/decision-attested.json"

  local reason
  reason="$DECISION_REASON_PREFIX: $phase"
  local decision_output
  decision_output="$outdir/decision-final.json"
  if [ -n "$DECISION_POLICY_REF" ]; then
    ./cub-gen bridge decision apply \
      --decision "$outdir/decision-attested.json" \
      --state "$DECISION_STATE" \
      --policy-ref "$DECISION_POLICY_REF" \
      --reason "$reason" > "$decision_output"
  else
    ./cub-gen bridge decision apply \
      --decision "$outdir/decision-attested.json" \
      --state "$DECISION_STATE" \
      --approved-by "$DECISION_APPROVED_BY" \
      --reason "$reason" > "$decision_output"
  fi

  if [ "$DECISION_STATE" = "ALLOW" ]; then
    ./cub-gen bridge promote init \
      --change-id "$(jq -r .change_id "$decision_output")" \
      --app-pr-repo github.com/confighub/apps \
      --app-pr-number 42 \
      --app-pr-url https://github.com/confighub/apps/pull/42 \
      --mr-id mr_123 \
      --mr-url "$CONFIGHUB_BASE_URL/mr/123" > "$outdir/flow.json"

    ./cub-gen bridge promote govern --flow "$outdir/flow.json" --state ALLOW --decision-ref decision_123 > "$outdir/flow-allow.json"
    ./cub-gen bridge promote verify --flow "$outdir/flow-allow.json" > "$outdir/flow-verified.json"
    ./cub-gen bridge promote open --flow "$outdir/flow-verified.json" --repo github.com/confighub/platform-dry --number 7 --url https://github.com/confighub/platform-dry/pull/7 > "$outdir/flow-open.json"
    ./cub-gen bridge promote approve --flow "$outdir/flow-open.json" --by platform-owner > "$outdir/flow-approved.json"
    ./cub-gen bridge promote merge --flow "$outdir/flow-approved.json" --by platform-owner > "$outdir/flow-final.json"
  else
    jq -n \
      --arg change_id "$(jq -r .change_id "$decision_output")" \
      --arg state "$DECISION_STATE" \
      --arg reason "$reason" \
      '{change_id: $change_id, state: $state, promotion: "skipped", reason: $reason}' > "$outdir/flow-final.json"
  fi
}

show_surface_views() {
  local repo="$1"
  local outdir="$2"

  echo "[surface][oci]"
  jq '{change_id, bundle_digest, output_uris: [.provenance[].outputs[].uri] | unique}' "$outdir/bundle.json"

  echo "[surface][confighub-ingest]"
  jq '{status_code, status, change_id, bundle_digest, idempotency_key, artifact_id}' "$outdir/ingest.json"

  echo "[surface][confighub-decision-query]"
  jq '.' "$outdir/decision-query.json"

  echo "[surface][flux]"
  if [ -d "$repo/gitops/flux" ]; then
    find "$repo/gitops/flux" -type f | sort | sed "s|^$repo/|  |"
  else
    echo "  no flux fixture files in this example"
  fi

  echo "[surface][argo]"
  if [ -d "$repo/gitops/argo" ]; then
    find "$repo/gitops/argo" -type f | sort | sed "s|^$repo/|  |"
  else
    echo "  no argo fixture files in this example"
  fi

  echo "[surface][cub-scout-watchlist]"
  jq '[.wet_manifest_targets[] | {kind, name, namespace}] | .[:6]' "$outdir/import.json"
}

summary_line() {
  local phase="$1"
  local outdir="$2"
  jq -n \
    --arg phase "$phase" \
    --arg change_id "$(jq -r .change_id "$outdir/bundle.json")" \
    --arg bundle_digest "$(jq -r .bundle_digest "$outdir/bundle.json")" \
    --arg ingest_status "$(jq -r '.status // "unknown"' "$outdir/ingest.json")" \
    --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$outdir/decision-final.json")" \
    --arg decision_authority "$(jq -r '.approved_by // .policy_decision_ref // "none"' "$outdir/decision-final.json")" \
    --argjson wet_targets "$(jq '.wet_manifest_targets | length' "$outdir/import.json")" \
    --argjson inverse_patches "$(jq '[.inverse_transform_plans[].patches | length] | add' "$outdir/import.json")" \
    --argjson attested_valid "$(jq '.valid' "$outdir/attestation-verify.json")" \
    '{phase: $phase, change_id: $change_id, bundle_digest: $bundle_digest, ingest_status: $ingest_status, decision_state: $decision_state, decision_authority: $decision_authority, wet_targets: $wet_targets, inverse_patches: $inverse_patches, attestation_valid: $attested_valid}'
}

echo "[lifecycle][connected] example: $EXAMPLE_SLUG"
echo "[lifecycle][connected] source: $REPO_PATH"

if [ -n "$OUTPUT_DIR" ]; then
  tmpdir="$OUTPUT_DIR"
  mkdir -p "$tmpdir"
  echo "[lifecycle][connected] output dir: $tmpdir"
else
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT
fi

work_repo="$tmpdir/repo"
work_target="$tmpdir/target"
mkdir -p "$work_repo" "$work_target"
cp -R "$REPO_PATH"/. "$work_repo"
cp -R "$RENDER_TARGET"/. "$work_target"

echo "[phase:create][connected] discover -> import -> publish -> verify -> attest -> ingest -> decision-query -> promote"
run_governed_cycle_connected "create" "$work_repo" "$work_target" "$tmpdir/create"
summary_line "create" "$tmpdir/create" | tee "$tmpdir/create/summary.json"
show_surface_views "$work_repo" "$tmpdir/create"

echo "[phase:update][connected] apply source change"
apply_update "$EXAMPLE_SLUG" "$work_repo"

echo "[phase:update][connected] rerun connected governance chain"
run_governed_cycle_connected "update" "$work_repo" "$work_target" "$tmpdir/update"
summary_line "update" "$tmpdir/update" | tee "$tmpdir/update/summary.json"

echo "[phase:diff][connected]"
jq -n \
  --arg create_change_id "$(jq -r .change_id "$tmpdir/create/bundle.json")" \
  --arg update_change_id "$(jq -r .change_id "$tmpdir/update/bundle.json")" \
  --arg create_digest "$(jq -r .bundle_digest "$tmpdir/create/bundle.json")" \
  --arg update_digest "$(jq -r .bundle_digest "$tmpdir/update/bundle.json")" \
  --arg create_ingest_status "$(jq -r '.status // "unknown"' "$tmpdir/create/ingest.json")" \
  --arg update_ingest_status "$(jq -r '.status // "unknown"' "$tmpdir/update/ingest.json")" \
  --argjson create_wet "$(jq '.wet_manifest_targets | length' "$tmpdir/create/import.json")" \
  --argjson update_wet "$(jq '.wet_manifest_targets | length' "$tmpdir/update/import.json")" \
  '{
    create_change_id: $create_change_id,
    update_change_id: $update_change_id,
    change_id_changed: ($create_change_id != $update_change_id),
    create_bundle_digest: $create_digest,
    update_bundle_digest: $update_digest,
    bundle_digest_changed: ($create_digest != $update_digest),
    create_ingest_status: $create_ingest_status,
    update_ingest_status: $update_ingest_status,
    create_wet_targets: $create_wet,
    update_wet_targets: $update_wet
  }' | tee "$tmpdir/diff-summary.json"

echo "[lifecycle][connected] artifacts: $tmpdir"
