#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}" 
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-12}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"

mkdir -p "$OUT_DIR"

echo "[story-12] unified human/ci/ai mutation evidence"
echo "[story-12] output: $OUT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"
require_connected_preflight
print_connected_context

go build -o ./cub-gen ./cmd/cub-gen

BRIDGE_INGEST_ENDPOINT="${BRIDGE_INGEST_ENDPOINT:-}"
BRIDGE_DECISION_ENDPOINT="${BRIDGE_DECISION_ENDPOINT:-}"
CONNECTED_FALLBACK_MODE="${CONNECTED_FALLBACK_MODE:-auto}"
FALLBACK_DECISION_STATE="${FALLBACK_DECISION_STATE:-ALLOW}"
FALLBACK_POLICY_REF="${FALLBACK_POLICY_REF:-policy/fallback-changeset-allow}"
FALLBACK_APPROVED_BY="${FALLBACK_APPROVED_BY:-fallback-platform-owner}"
FALLBACK_DECISION_REASON="${FALLBACK_DECISION_REASON:-bridge endpoint unavailable; explicit fallback decision recorded in ConfigHub changeset}"

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
  local outdir="$1"
  local bridge_error="$2"

  local change_id bundle_digest idempotency_key now digest_short slug backend_changeset artifact_id
  change_id="$(jq -r .change_id "$outdir/bundle.json")"
  bundle_digest="$(jq -r .bundle_digest "$outdir/bundle.json")"
  idempotency_key="${change_id}:${bundle_digest}"
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  digest_short="${bundle_digest#sha256:}"
  digest_short="${digest_short:0:16}"
  slug="$(sanitize_slug "cubgen-story12-${change_id}")"
  backend_changeset="$outdir/fallback-changeset.json"

  cub changeset create \
    --space "$CONFIGHUB_SPACE" \
    --json \
    --allow-exists \
    "$slug" \
    --description "cub-gen fallback ingest for story-12, change_id=${change_id}" \
    --label "cubgen_mode=changeset_fallback" \
    --label "cubgen_story=12" \
    --label "cubgen_change_id=${change_id}" \
    --label "cubgen_bundle_sha=${digest_short}" \
    --label "cubgen_decision_state=${FALLBACK_DECISION_STATE}" \
    --label "cubgen_policy_ref=${FALLBACK_POLICY_REF}" \
    > "$backend_changeset"

  artifact_id="$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$backend_changeset")"
  if [ -z "$artifact_id" ]; then
    echo "error: fallback ingest could not resolve backend changeset ID for story-12." >&2
    exit 1
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
    }' > "$outdir/decision-query.json"
}

./cub-gen gitops import --space "$CONFIGHUB_SPACE" --json "$REPO_PATH" "$RENDER_TARGET" > "$OUT_DIR/import.json"
./cub-gen publish --in "$OUT_DIR/import.json" > "$OUT_DIR/bundle.json"
./cub-gen verify --json --in "$OUT_DIR/bundle.json" > "$OUT_DIR/verify.json"

# Three actor classes under one change_id.
./cub-gen attest --in "$OUT_DIR/bundle.json" --verifier "human:platform-owner" > "$OUT_DIR/attestation-human.json"
./cub-gen attest --in "$OUT_DIR/bundle.json" --verifier "ci:github-actions" > "$OUT_DIR/attestation-ci.json"
./cub-gen attest --in "$OUT_DIR/bundle.json" --verifier "ai:deploy-agent" > "$OUT_DIR/attestation-ai.json"

./cub-gen verify-attestation --json --in "$OUT_DIR/attestation-human.json" --bundle "$OUT_DIR/bundle.json" > "$OUT_DIR/attestation-human-verify.json"
./cub-gen verify-attestation --json --in "$OUT_DIR/attestation-ci.json" --bundle "$OUT_DIR/bundle.json" > "$OUT_DIR/attestation-ci-verify.json"
./cub-gen verify-attestation --json --in "$OUT_DIR/attestation-ai.json" --bundle "$OUT_DIR/bundle.json" > "$OUT_DIR/attestation-ai-verify.json"

ingest_cmd=(
  ./cub-gen bridge ingest
  --in "$OUT_DIR/bundle.json"
  --base-url "$CONFIGHUB_BASE_URL"
  --token "$CONFIGHUB_TOKEN"
)
if [ -n "$BRIDGE_INGEST_ENDPOINT" ]; then
  ingest_cmd+=(--endpoint "$BRIDGE_INGEST_ENDPOINT")
fi
fallback_used=0
bridge_ingest_error=""
if ! "${ingest_cmd[@]}" > "$OUT_DIR/ingest.json" 2>"$OUT_DIR/ingest.error"; then
  bridge_ingest_error="$(tr '\n' ' ' < "$OUT_DIR/ingest.error" | sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//')"
  if should_use_fallback "$bridge_ingest_error"; then
    echo "[story-12] bridge ingest unavailable; using changeset-backed fallback"
    run_changeset_fallback "$OUT_DIR" "$bridge_ingest_error"
    fallback_used=1
  else
    echo "error: story-12 ingest failed." >&2
    echo "details: $bridge_ingest_error" >&2
    echo "remediation: ensure bridge ingest endpoint is enabled, set BRIDGE_INGEST_ENDPOINT, or use CONNECTED_FALLBACK_MODE=changeset." >&2
    exit 1
  fi
fi

if [ "$fallback_used" -eq 0 ]; then
  decision_query_cmd=(
    ./cub-gen bridge decision query
    --base-url "$CONFIGHUB_BASE_URL"
    --token "$CONFIGHUB_TOKEN"
    --change-id "$(jq -r .change_id "$OUT_DIR/bundle.json")"
  )
  if [ -n "$BRIDGE_DECISION_ENDPOINT" ]; then
    decision_query_cmd+=(--endpoint "$BRIDGE_DECISION_ENDPOINT")
  fi
  "${decision_query_cmd[@]}" > "$OUT_DIR/decision-query.json"
fi

jq -n \
  --arg story "12-unified-human-ci-ai-mutation" \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/bundle.json")" \
  --arg bundle_digest "$(jq -r .bundle_digest "$OUT_DIR/bundle.json")" \
  --arg ingest_status "$(jq -r '.status // "unknown"' "$OUT_DIR/ingest.json")" \
  --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/decision-query.json")" \
  --arg human_digest "$(jq -r .attestation_digest "$OUT_DIR/attestation-human.json")" \
  --arg ci_digest "$(jq -r .attestation_digest "$OUT_DIR/attestation-ci.json")" \
  --arg ai_digest "$(jq -r .attestation_digest "$OUT_DIR/attestation-ai.json")" \
  '{
    story: $story,
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    ingest_status: $ingest_status,
    decision_state: $decision_state,
    actor_chain: [
      {actor_type: "human", verifier: "human:platform-owner", attestation_digest: $human_digest},
      {actor_type: "ci", verifier: "ci:github-actions", attestation_digest: $ci_digest},
      {actor_type: "ai", verifier: "ai:deploy-agent", attestation_digest: $ai_digest}
    ]
  }' | tee "$OUT_DIR/story-12-summary.json"
