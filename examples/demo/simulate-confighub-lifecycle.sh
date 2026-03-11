#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}"
SPACE="${SPACE:-platform}"
VERIFIER="${VERIFIER:-ci-bot}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

if [ ! -d "$RENDER_TARGET" ]; then
  echo "error: render target path not found: $RENDER_TARGET" >&2
  exit 1
fi

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[lifecycle] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

source "$ROOT_DIR/examples/demo/lib/lifecycle-update.sh"

run_governed_cycle() {
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

  jq -n \
    --arg change_id "$(jq -r .change_id "$outdir/bundle.json")" \
    --arg bundle_digest "$(jq -r .bundle_digest "$outdir/bundle.json")" \
    '{
      status_code: 201,
      artifact_id: "wet_art_123",
      status: "created",
      change_id: $change_id,
      bundle_digest: $bundle_digest,
      idempotency_key: ($change_id + ":" + $bundle_digest)
    }' > "$outdir/ingest.json"

  ./cub-gen bridge decision create --ingest "$outdir/ingest.json" > "$outdir/decision.json"
  ./cub-gen bridge decision attach --decision "$outdir/decision.json" --attestation "$outdir/attestation.json" > "$outdir/decision-attested.json"
  ./cub-gen bridge decision apply --decision "$outdir/decision-attested.json" --state ALLOW --approved-by platform-owner --reason "$phase approved" > "$outdir/decision-allow.json"

  ./cub-gen bridge promote init \
    --change-id "$(jq -r .change_id "$outdir/decision-allow.json")" \
    --app-pr-repo github.com/confighub/apps \
    --app-pr-number 42 \
    --app-pr-url https://github.com/confighub/apps/pull/42 \
    --mr-id mr_123 \
    --mr-url https://confighub.example/mr/123 > "$outdir/flow.json"

  ./cub-gen bridge promote govern --flow "$outdir/flow.json" --state ALLOW --decision-ref decision_123 > "$outdir/flow-allow.json"
  ./cub-gen bridge promote verify --flow "$outdir/flow-allow.json" > "$outdir/flow-verified.json"
  ./cub-gen bridge promote open --flow "$outdir/flow-verified.json" --repo github.com/confighub/platform-dry --number 7 --url https://github.com/confighub/platform-dry/pull/7 > "$outdir/flow-open.json"
  ./cub-gen bridge promote approve --flow "$outdir/flow-open.json" --by platform-owner > "$outdir/flow-approved.json"
  ./cub-gen bridge promote merge --flow "$outdir/flow-approved.json" --by platform-owner > "$outdir/flow-promoted.json"
}

show_surface_views() {
  local repo="$1"
  local outdir="$2"

  echo "[surface][oci]"
  jq '{change_id, bundle_digest, output_uris: [.provenance[].outputs[].uri] | unique}' "$outdir/bundle.json"

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
    --argjson wet_targets "$(jq '.wet_manifest_targets | length' "$outdir/import.json")" \
    --argjson inverse_patches "$(jq '[.inverse_transform_plans[].patches | length] | add' "$outdir/import.json")" \
    --argjson attested_valid "$(jq '.valid' "$outdir/attestation-verify.json")" \
    '{phase: $phase, change_id: $change_id, bundle_digest: $bundle_digest, wet_targets: $wet_targets, inverse_patches: $inverse_patches, attestation_valid: $attested_valid}'
}

echo "[lifecycle] example: $EXAMPLE_SLUG"
echo "[lifecycle] source: $REPO_PATH"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

work_repo="$tmpdir/repo"
work_target="$tmpdir/target"
mkdir -p "$work_repo" "$work_target"
cp -R "$REPO_PATH"/. "$work_repo"
cp -R "$RENDER_TARGET"/. "$work_target"

echo "[phase:create] discover -> import -> publish -> verify -> attest -> decision -> promote"
run_governed_cycle "create" "$work_repo" "$work_target" "$tmpdir/create"
summary_line "create" "$tmpdir/create"
show_surface_views "$work_repo" "$tmpdir/create"

echo "[phase:update] apply source change"
apply_update "$EXAMPLE_SLUG" "$work_repo"

echo "[phase:update] rerun governance chain"
run_governed_cycle "update" "$work_repo" "$work_target" "$tmpdir/update"
summary_line "update" "$tmpdir/update"

echo "[phase:diff]"
jq -n \
  --arg create_change_id "$(jq -r .change_id "$tmpdir/create/bundle.json")" \
  --arg update_change_id "$(jq -r .change_id "$tmpdir/update/bundle.json")" \
  --arg create_digest "$(jq -r .bundle_digest "$tmpdir/create/bundle.json")" \
  --arg update_digest "$(jq -r .bundle_digest "$tmpdir/update/bundle.json")" \
  --argjson create_wet "$(jq '.wet_manifest_targets | length' "$tmpdir/create/import.json")" \
  --argjson update_wet "$(jq '.wet_manifest_targets | length' "$tmpdir/update/import.json")" \
  '{
    create_change_id: $create_change_id,
    update_change_id: $update_change_id,
    change_id_changed: ($create_change_id != $update_change_id),
    create_bundle_digest: $create_digest,
    update_bundle_digest: $update_digest,
    bundle_digest_changed: ($create_digest != $update_digest),
    create_wet_targets: $create_wet,
    update_wet_targets: $update_wet
  }'
