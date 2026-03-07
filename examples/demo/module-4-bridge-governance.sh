#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[module-4] build cub-gen"
go build -o ./cub-gen ./cmd/cub-gen

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "[module-4] publish + attest"
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > "$tmpdir/bundle.json"
./cub-gen attest --in "$tmpdir/bundle.json" --verifier ci-bot > "$tmpdir/attestation.json"

echo "[module-4] create decision (local bridge simulation)"
jq -n \
  --arg change_id "$(jq -r .change_id "$tmpdir/bundle.json")" \
  --arg bundle_digest "$(jq -r .bundle_digest "$tmpdir/bundle.json")" \
  '{
    status_code: 201,
    artifact_id: "wet_art_123",
    status: "created",
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    idempotency_key: ($change_id + ":" + $bundle_digest)
  }' > "$tmpdir/ingest.json"

./cub-gen bridge decision create --ingest "$tmpdir/ingest.json" > "$tmpdir/decision.json"
./cub-gen bridge decision attach --decision "$tmpdir/decision.json" --attestation "$tmpdir/attestation.json" > "$tmpdir/decision-attested.json"
./cub-gen bridge decision apply --decision "$tmpdir/decision-attested.json" --state ALLOW --approved-by platform-owner --reason "policy checks passed" > "$tmpdir/decision-allow.json"

echo "[module-4] run promotion guardrail flow"
./cub-gen bridge promote init \
  --change-id "$(jq -r .change_id "$tmpdir/decision-allow.json")" \
  --app-pr-repo github.com/confighub/apps \
  --app-pr-number 42 \
  --app-pr-url https://github.com/confighub/apps/pull/42 \
  --mr-id mr_123 \
  --mr-url https://confighub.example/mr/123 > "$tmpdir/flow.json"

./cub-gen bridge promote govern --flow "$tmpdir/flow.json" --state ALLOW --decision-ref decision_123 > "$tmpdir/flow-allow.json"
./cub-gen bridge promote verify --flow "$tmpdir/flow-allow.json" > "$tmpdir/flow-verified.json"
./cub-gen bridge promote open --flow "$tmpdir/flow-verified.json" --repo github.com/confighub/platform-dry --number 7 --url https://github.com/confighub/platform-dry/pull/7 > "$tmpdir/flow-open.json"
./cub-gen bridge promote approve --flow "$tmpdir/flow-open.json" --by platform-owner > "$tmpdir/flow-approved.json"
./cub-gen bridge promote merge --flow "$tmpdir/flow-approved.json" --by platform-owner > "$tmpdir/flow-promoted.json"

echo "[module-4] final decision + flow"
jq '{change_id,state,approved_by,decision_reason}' "$tmpdir/decision-allow.json"
jq '{change_id,state,promotion_merged,platform_review_approved}' "$tmpdir/flow-promoted.json"

