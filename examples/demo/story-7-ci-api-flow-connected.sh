#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

# CI defaults: non-interactive auth via env vars.
EXAMPLE_SLUG="${EXAMPLE_SLUG:-helm-paas}"
REPO_PATH="${REPO_PATH:-./examples/$EXAMPLE_SLUG}"
RENDER_TARGET="${RENDER_TARGET:-$REPO_PATH}"
RUN_ID="${RUN_ID:-${GITHUB_RUN_ID:-local}}"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/.tmp/ci-connected/$RUN_ID/$EXAMPLE_SLUG}"

if [ -z "${CONFIGHUB_TOKEN:-}" ]; then
  echo "error: CONFIGHUB_TOKEN is required for non-interactive CI mode." >&2
  echo "remediation: set CONFIGHUB_TOKEN, CONFIGHUB_BASE_URL, and CONFIGHUB_SPACE in CI secrets/env." >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[story-7] CI-centric connected flow"
echo "[story-7] example: $EXAMPLE_SLUG"
echo "[story-7] output: $OUT_DIR"

VERIFIER="${VERIFIER:-ci-bot}" \
./examples/demo/run-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$OUT_DIR"

jq -n \
  --arg story "7-ci-centric-api-flow" \
  --arg run_id "$RUN_ID" \
  --arg example "$EXAMPLE_SLUG" \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/decision-final.json")" \
  --arg policy_ref "$(jq -r '.policy_decision_ref // ""' "$OUT_DIR/update/decision-final.json")" \
  --arg verifier "${VERIFIER:-ci-bot}" \
  --arg ingest_status "$(jq -r '.status // "unknown"' "$OUT_DIR/update/ingest.json")" \
  '{
    story: $story,
    run_id: $run_id,
    example: $example,
    change_id: $change_id,
    decision_state: $decision_state,
    policy_decision_ref: $policy_ref,
    verifier: $verifier,
    ingest_status: $ingest_status
  }' | tee "$OUT_DIR/story-7-summary.json"
