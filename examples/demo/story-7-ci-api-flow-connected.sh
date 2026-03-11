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
SPACE="${SPACE:-${CONFIGHUB_SPACE:-platform}}"
VERIFIER="${VERIFIER:-ci-bot}"

if [ -z "${CONFIGHUB_TOKEN:-}" ]; then
  echo "error: CONFIGHUB_TOKEN is required for non-interactive CI mode." >&2
  echo "remediation: set CONFIGHUB_TOKEN, CONFIGHUB_BASE_URL, and CONFIGHUB_SPACE in CI secrets/env." >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[story-7] CI-centric connected flow"
echo "[story-7] example: $EXAMPLE_SLUG"
echo "[story-7] output: $OUT_DIR"
echo "[story-7] mode: cub-gen change run --mode connected"

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  go build -o ./cub-gen ./cmd/cub-gen
fi

./cub-gen change run \
  --mode connected \
  --space "$SPACE" \
  --base-url "$CONFIGHUB_BASE_URL" \
  --token "$CONFIGHUB_TOKEN" \
  --verifier "$VERIFIER" \
  "$REPO_PATH" "$RENDER_TARGET" > "$OUT_DIR/change-run.json"

WET_PATH="$(jq -r '.preview.edit_recommendation.wet_path // empty' "$OUT_DIR/change-run.json")"
if [ -n "$WET_PATH" ]; then
  ./cub-gen change explain \
    --space "$SPACE" \
    --wet-path "$WET_PATH" \
    "$REPO_PATH" "$RENDER_TARGET" > "$OUT_DIR/change-explain.json"
fi

jq -n \
  --arg story "7-ci-centric-api-flow" \
  --arg run_id "$RUN_ID" \
  --arg example "$EXAMPLE_SLUG" \
  --arg change_id "$(jq -r '.preview.change.change_id' "$OUT_DIR/change-run.json")" \
  --arg decision_state "$(jq -r '.decision.state // "UNKNOWN"' "$OUT_DIR/change-run.json")" \
  --arg decision_authority "$(jq -r '.decision.authority // ""' "$OUT_DIR/change-run.json")" \
  --arg decision_source "$(jq -r '.decision.source // ""' "$OUT_DIR/change-run.json")" \
  --argjson promotion_ready "$(jq '.promotion_ready // false' "$OUT_DIR/change-run.json")" \
  --arg verifier "$VERIFIER" \
  --arg wet_path "$(jq -r '.preview.edit_recommendation.wet_path // ""' "$OUT_DIR/change-run.json")" \
  --arg dry_path "$(jq -r '.preview.edit_recommendation.dry_path // ""' "$OUT_DIR/change-run.json")" \
  --arg edit_hint "$(jq -r '.preview.edit_recommendation.edit_hint // ""' "$OUT_DIR/change-run.json")" \
  --argjson wet_targets "$(jq '.preview.counts.wet_targets // 0' "$OUT_DIR/change-run.json")" \
  '{
    story: $story,
    run_id: $run_id,
    example: $example,
    change_id: $change_id,
    decision_state: $decision_state,
    decision_authority: $decision_authority,
    decision_source: $decision_source,
    promotion_ready: $promotion_ready,
    verifier: $verifier,
    wet_targets: $wet_targets,
    top_edit_recommendation: {
      wet_path: $wet_path,
      dry_path: $dry_path,
      edit_hint: $edit_hint
    }
  }' | tee "$OUT_DIR/story-7-summary.json"
