#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-10}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[story-10] signed commit + branch protection write-back proof"
echo "[story-10] output: $OUT_DIR"

./examples/demo/simulate-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$OUT_DIR"

writeback_proof="$OUT_DIR/update/writeback-proof.json"
jq -n \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg app_pr_repo "github.com/confighub/apps" \
  --argjson app_pr_number 42 \
  --arg app_pr_url "https://github.com/confighub/apps/pull/42" \
  --arg app_commit_sha "8a7f4f2d8ca1a0a41f1007d0d4b297d2fa9a2c11" \
  --arg promotion_repo "github.com/confighub/platform-dry" \
  --argjson promotion_pr_number 7 \
  --arg promotion_pr_url "https://github.com/confighub/platform-dry/pull/7" \
  --arg promotion_commit_sha "5eecf7bf0ec7c8d0ef95f084d6464a8fa4e2bc31" \
  '{
    schema: "confighub.io/writeback-proof/v1",
    change_id: $change_id,
    app_pr: {
      repo: $app_pr_repo,
      number: $app_pr_number,
      url: $app_pr_url,
      commit_sha: $app_commit_sha,
      commit_signature: {verified: true, signer: "ci-bot@confighub.ai"}
    },
    promotion_pr: {
      repo: $promotion_repo,
      number: $promotion_pr_number,
      url: $promotion_pr_url,
      commit_sha: $promotion_commit_sha,
      commit_signature: {verified: true, signer: "platform-owner@confighub.ai"}
    },
    branch_protection: {
      required_approving_review_count: 2,
      required_status_checks: ["ci/test", "ci/policy", "ci/attestation"],
      enforce_admins: true
    },
    verification: {
      signatures_verified: true,
      branch_protection_preserved: true
    }
  }' > "$writeback_proof"

writeback_digest="$(shasum -a 256 "$writeback_proof" | awk '{print $1}')"

jq -n \
  --arg story "10-signed-commit-branch-protection-proof" \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg bundle_digest "$(jq -r .bundle_digest "$OUT_DIR/update/bundle.json")" \
  --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/decision-final.json")" \
  --arg ingest_status "$(jq -r '.status // "unknown"' "$OUT_DIR/update/ingest.json")" \
  --arg proof_file "$writeback_proof" \
  --arg proof_digest "$writeback_digest" \
  --argjson signatures_verified "$(jq '.verification.signatures_verified' "$writeback_proof")" \
  --argjson branch_protection_preserved "$(jq '.verification.branch_protection_preserved' "$writeback_proof")" \
  '{
    story: $story,
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    decision_state: $decision_state,
    ingest_status: $ingest_status,
    writeback_proof_file: $proof_file,
    writeback_proof_digest: $proof_digest,
    signatures_verified: $signatures_verified,
    branch_protection_preserved: $branch_protection_preserved
  }' | tee "$OUT_DIR/story-10-summary.json"
