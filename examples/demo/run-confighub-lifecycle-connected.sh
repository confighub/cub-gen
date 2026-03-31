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
BRIDGE_INGEST_ENDPOINT="${BRIDGE_INGEST_ENDPOINT:-}"
BRIDGE_DECISION_ENDPOINT="${BRIDGE_DECISION_ENDPOINT:-}"
DECISION_POLL_TIMEOUT_SECS="${DECISION_POLL_TIMEOUT_SECS:-120}"
DECISION_POLL_INTERVAL_SECS="${DECISION_POLL_INTERVAL_SECS:-3}"
REQUIRE_TERMINAL_DECISION="${REQUIRE_TERMINAL_DECISION:-1}"
CONNECTED_FALLBACK_MODE="${CONNECTED_FALLBACK_MODE:-off}"
FALLBACK_DECISION_STATE="${FALLBACK_DECISION_STATE:-ALLOW}"
FALLBACK_POLICY_REF="${FALLBACK_POLICY_REF:-policy/fallback-changeset-allow}"
FALLBACK_APPROVED_BY="${FALLBACK_APPROVED_BY:-fallback-platform-owner}"
FALLBACK_DECISION_REASON="${FALLBACK_DECISION_REASON:-bridge endpoint unavailable; explicit fallback decision recorded in ConfigHub changeset}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

if [ ! -d "$RENDER_TARGET" ]; then
  echo "error: render target path not found: $RENDER_TARGET" >&2
  exit 1
fi

require_connected_preflight
if [ -z "$SPACE" ]; then
  SPACE="$CONFIGHUB_SPACE"
fi
print_connected_context

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[lifecycle][connected] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

is_terminal_decision_state() {
  local state="$1"
  case "$state" in
    ALLOW|ESCALATE|BLOCK) return 0 ;;
    *) return 1 ;;
  esac
}

query_backend_decision() {
  local change_id="$1"
  local output="$2"

  decision_query_cmd=(
    ./cub-gen bridge decision query
    --base-url "$CONFIGHUB_BASE_URL"
    --token "$CONFIGHUB_TOKEN"
    --change-id "$change_id"
  )
  if [ -n "$BRIDGE_DECISION_ENDPOINT" ]; then
    decision_query_cmd+=(--endpoint "$BRIDGE_DECISION_ENDPOINT")
  fi
  "${decision_query_cmd[@]}" > "$output"
}

sanitize_slug() {
  local input="$1"
  input="$(printf '%s' "$input" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
  input="$(printf '%s' "$input" | sed -E 's/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "${input:0:63}"
}

should_use_fallback() {
  local bridge_error="$1"
  case "$CONNECTED_FALLBACK_MODE" in
    off) return 1 ;;
    changeset) return 0 ;;
    auto)
      if printf '%s' "$bridge_error" | grep -Eq "status=404|Not Found"; then
        return 0
      fi
      return 1
      ;;
    *)
      echo "error: unsupported CONNECTED_FALLBACK_MODE=$CONNECTED_FALLBACK_MODE (expected off|auto|changeset)" >&2
      return 1
      ;;
  esac
}

run_changeset_fallback() {
  local phase="$1"
  local outdir="$2"
  local bridge_error="$3"

  local change_id bundle_digest idempotency_key now digest_short slug fallback_changeset
  change_id="$(jq -r .change_id "$outdir/bundle.json")"
  bundle_digest="$(jq -r .bundle_digest "$outdir/bundle.json")"
  idempotency_key="${change_id}:${bundle_digest}"
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  digest_short="${bundle_digest#sha256:}"
  digest_short="${digest_short:0:16}"
  slug="$(sanitize_slug "cubgen-${phase}-${change_id}")"
  fallback_changeset="$outdir/fallback-changeset.json"

  cub changeset create \
    --space "$CONFIGHUB_SPACE" \
    --json \
    --allow-exists \
    "$slug" \
    --description "cub-gen fallback ingest for phase=${phase}, change_id=${change_id}" \
    --label "cubgen_mode=changeset_fallback" \
    --label "cubgen_phase=${phase}" \
    --label "cubgen_change_id=${change_id}" \
    --label "cubgen_bundle_sha=${digest_short}" \
    --label "cubgen_decision_state=${FALLBACK_DECISION_STATE}" \
    --label "cubgen_policy_ref=${FALLBACK_POLICY_REF}" \
    > "$fallback_changeset"

  local artifact_id
  artifact_id="$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$fallback_changeset")"
  if [ -z "$artifact_id" ]; then
    echo "error: fallback ingest could not resolve backend changeset ID." >&2
    return 1
  fi

  jq -n \
    --argjson status_code 202 \
    --arg artifact_id "$artifact_id" \
    --arg status "ingested-fallback" \
    --arg change_id "$change_id" \
    --arg bundle_digest "$bundle_digest" \
    --arg idempotency_key "$idempotency_key" \
    --arg backend_mode "changeset-fallback" \
    --arg changeset_slug "$slug" \
    --arg bridge_error "$bridge_error" \
    '{
      status_code: $status_code,
      artifact_id: $artifact_id,
      status: $status,
      idempotent: false,
      change_id: $change_id,
      bundle_digest: $bundle_digest,
      idempotency_key: $idempotency_key,
      backend_mode: $backend_mode,
      changeset_slug: $changeset_slug,
      bridge_error: $bridge_error
    }' > "$outdir/ingest.json"

  jq -n \
    --arg schema_version "cub.confighub.io/governed-decision-state/v1" \
    --arg source "confighub-backend-changeset-fallback" \
    --arg change_id "$change_id" \
    --arg bundle_digest "$bundle_digest" \
    --arg artifact_id "$artifact_id" \
    --arg idempotency_key "$idempotency_key" \
    --arg state "$FALLBACK_DECISION_STATE" \
    --arg policy_decision_ref "$FALLBACK_POLICY_REF" \
    --arg approved_by "$FALLBACK_APPROVED_BY" \
    --arg decision_reason "$FALLBACK_DECISION_REASON" \
    --arg decided_at "$now" \
    --arg updated_at "$now" \
    --arg fallback_mode "changeset" \
    '{
      schema_version: $schema_version,
      source: $source,
      change_id: $change_id,
      bundle_digest: $bundle_digest,
      artifact_id: $artifact_id,
      idempotency_key: $idempotency_key,
      state: $state,
      policy_decision_ref: $policy_decision_ref,
      approved_by: $approved_by,
      decision_reason: $decision_reason,
      decided_at: $decided_at,
      updated_at: $updated_at,
      fallback_mode: $fallback_mode
    }' > "$outdir/decision-query-initial.json"
  cp "$outdir/decision-query-initial.json" "$outdir/decision-query.json"
  cp "$outdir/decision-query-initial.json" "$outdir/decision-final.json"
}

run_governed_cycle_connected() {
  local phase="$1"
  local repo="$2"
  local render_target="$3"
  local outdir="$4"

  mkdir -p "$outdir"
  rm -rf "$outdir/repo" "$outdir/render-target"
  mkdir -p "$outdir/repo" "$outdir/render-target"
  cp -R "$repo"/. "$outdir/repo"
  cp -R "$render_target"/. "$outdir/render-target"

  ./cub-gen gitops discover --space "$SPACE" --json "$repo" > "$outdir/discover.json"
  ./cub-gen gitops import --space "$SPACE" --json "$repo" "$render_target" > "$outdir/import.json"
  ./cub-gen publish --in "$outdir/import.json" > "$outdir/bundle.json"
  ./cub-gen verify --json --in "$outdir/bundle.json" > "$outdir/verify.json"
  ./cub-gen attest --in "$outdir/bundle.json" --verifier "$VERIFIER" > "$outdir/attestation.json"
  ./cub-gen verify-attestation --json --in "$outdir/attestation.json" --bundle "$outdir/bundle.json" > "$outdir/attestation-verify.json"

  echo "[connected][$phase] ingest bundle into ConfigHub"
  ingest_cmd=(
    ./cub-gen bridge ingest
    --in "$outdir/bundle.json"
    --base-url "$CONFIGHUB_BASE_URL"
    --token "$CONFIGHUB_TOKEN"
  )
  if [ -n "$BRIDGE_INGEST_ENDPOINT" ]; then
    ingest_cmd+=(--endpoint "$BRIDGE_INGEST_ENDPOINT")
  fi
  local used_fallback=0
  local bridge_ingest_error=""
  if ! "${ingest_cmd[@]}" > "$outdir/ingest.json" 2>"$outdir/ingest.error"; then
    bridge_ingest_error="$(tr '\n' ' ' < "$outdir/ingest.error" | sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//')"
    if should_use_fallback "$bridge_ingest_error"; then
      echo "[connected][$phase] bridge ingest unavailable; using changeset-backed fallback"
      run_changeset_fallback "$phase" "$outdir" "$bridge_ingest_error"
      used_fallback=1
    else
      echo "error: connected ingest failed for $phase." >&2
      echo "details: $bridge_ingest_error" >&2
      echo "remediation: ensure CONFIGHUB_BASE_URL points to a ConfigHub backend exposing ingest, set BRIDGE_INGEST_ENDPOINT to the correct path, or use CONNECTED_FALLBACK_MODE=changeset." >&2
      return 1
    fi
  fi

  local change_id
  change_id="$(jq -r .change_id "$outdir/bundle.json")"

  local decision_state
  if [ "$used_fallback" -eq 1 ]; then
    decision_state="$(jq -r '.state // "UNKNOWN"' "$outdir/decision-final.json")"
  else
    echo "[connected][$phase] query decision state from ConfigHub (authoritative)"
    query_backend_decision "$change_id" "$outdir/decision-query-initial.json"
    cp "$outdir/decision-query-initial.json" "$outdir/decision-query.json"

    decision_state="$(jq -r '.state // "UNKNOWN"' "$outdir/decision-query-initial.json")"
    if ! is_terminal_decision_state "$decision_state"; then
      local waited=0
      while [ "$waited" -lt "$DECISION_POLL_TIMEOUT_SECS" ]; do
        sleep "$DECISION_POLL_INTERVAL_SECS"
        waited=$((waited + DECISION_POLL_INTERVAL_SECS))
        query_backend_decision "$change_id" "$outdir/decision-query.json"
        decision_state="$(jq -r '.state // "UNKNOWN"' "$outdir/decision-query.json")"
        if is_terminal_decision_state "$decision_state"; then
          break
        fi
      done
    fi

    if ! is_terminal_decision_state "$decision_state"; then
      if [ "$REQUIRE_TERMINAL_DECISION" = "1" ]; then
        echo "error: backend decision did not reach terminal ALLOW|ESCALATE|BLOCK state within ${DECISION_POLL_TIMEOUT_SECS}s (state=$decision_state)." >&2
        echo "remediation: ensure ConfigHub decision evaluation workers are running and policy evaluation is enabled for governed-wet ingest." >&2
        return 1
      fi
      echo "[connected][$phase] warning: non-terminal backend decision state after timeout: $decision_state"
    fi
    cp "$outdir/decision-query.json" "$outdir/decision-final.json"
  fi

  local decision_source="confighub-backend"
  local flow_note="Promotion state is backend-owned in connected mode; no local bridge promote simulation performed."
  if [ "$used_fallback" -eq 1 ]; then
    decision_source="confighub-backend-changeset-fallback"
    flow_note="Bridge ingest endpoint unavailable; fallback stores ingest+decision evidence in ConfigHub changesets."
  fi
  jq -n \
    --arg change_id "$change_id" \
    --arg decision_state "$decision_state" \
    --arg source "$decision_source" \
    --arg note "$flow_note" \
    '{change_id: $change_id, decision_state: $decision_state, source: $source, note: $note}' > "$outdir/flow-final.json"
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

echo "[phase:create][connected] discover -> import -> publish -> verify -> attest -> ingest -> backend decision query"
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
