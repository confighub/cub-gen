#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

usage() {
  cat <<'USAGE'
Usage:
  ./examples/demo/story-7-agent-tool-call-connected.sh [example-slug]

Purpose:
  Prove one shared change lifecycle across adapter tool calls:
  preview -> run (connected) -> explain (by change_id + bundle).

Environment variables:
  EXAMPLE_SLUG         Example under examples/ (default: helm-paas)
  REPO_PATH            Override repo path (default: ./examples/$EXAMPLE_SLUG)
  RENDER_TARGET        Override render target (default: $REPO_PATH)
  OUT_DIR              Output directory (default: .tmp/agent-tool-call/<run>/<slug>)
  SPACE                ConfigHub space label (default: CONFIGHUB_SPACE or platform)
  RUN_ID               Run identifier (default: GITHUB_RUN_ID or local)
  SKIP_BUILD           Set to 1 to skip go build
  BRIDGE_INGEST_ENDPOINT   Optional ingest endpoint override
  BRIDGE_DECISION_ENDPOINT Optional decision endpoint override
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

EXAMPLE_SLUG="${1:-${EXAMPLE_SLUG:-helm-paas}}"
REPO_PATH="${REPO_PATH:-./examples/$EXAMPLE_SLUG}"
RENDER_TARGET="${RENDER_TARGET:-$REPO_PATH}"
RUN_ID="${RUN_ID:-${GITHUB_RUN_ID:-local}}"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/.tmp/agent-tool-call/$RUN_ID/$EXAMPLE_SLUG}"
SPACE="${SPACE:-${CONFIGHUB_SPACE:-platform}}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi
if [ ! -d "$RENDER_TARGET" ]; then
  echo "error: render target path not found: $RENDER_TARGET" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[story-7-agent] preflight (requires cub auth login or CONFIGHUB_TOKEN)"
require_connected_preflight
print_connected_context

echo "[story-7-agent] example: $EXAMPLE_SLUG"
echo "[story-7-agent] output: $OUT_DIR"

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  go build -o ./cub-gen ./cmd/cub-gen
fi

# 1) Agent requests preview via adapter.
jq -n \
  --arg action "preview" \
  --arg target_slug "$REPO_PATH" \
  --arg render_target_slug "$RENDER_TARGET" \
  --arg space "$SPACE" \
  '{
    action: $action,
    input: {
      target_slug: $target_slug,
      render_target_slug: $render_target_slug,
      space: $space
    }
  }' > "$OUT_DIR/preview-request.json"

SKIP_BUILD=1 ./examples/demo/change-api-adapter.sh \
  --request "$OUT_DIR/preview-request.json" \
  --out "$OUT_DIR/preview.json"

# 2) Agent requests connected run via adapter.
jq -n \
  --arg action "run" \
  --arg mode "connected" \
  --arg target_slug "$REPO_PATH" \
  --arg render_target_slug "$RENDER_TARGET" \
  --arg space "$SPACE" \
  --arg base_url "$CONFIGHUB_BASE_URL" \
  --arg token "$CONFIGHUB_TOKEN" \
  --arg ingest_endpoint "${BRIDGE_INGEST_ENDPOINT:-}" \
  --arg decision_endpoint "${BRIDGE_DECISION_ENDPOINT:-}" \
  '{
    action: $action,
    mode: $mode,
    input: {
      target_slug: $target_slug,
      render_target_slug: $render_target_slug,
      space: $space
    },
    connected: ({
      base_url: $base_url,
      token: $token
    }
    + (if $ingest_endpoint == "" then {} else {ingest_endpoint: $ingest_endpoint} end)
    + (if $decision_endpoint == "" then {} else {decision_endpoint: $decision_endpoint} end))
  }' > "$OUT_DIR/run-request.json"

SKIP_BUILD=1 ./examples/demo/change-api-adapter.sh \
  --request "$OUT_DIR/run-request.json" \
  --out "$OUT_DIR/run.json"

RUN_CHANGE_ID="$(jq -r '.preview.change.change_id // empty' "$OUT_DIR/run.json")"
if [ -z "$RUN_CHANGE_ID" ]; then
  echo "error: run response missing preview.change.change_id" >&2
  exit 1
fi

# Build canonical bundle for explain-by-change-id drilldown.
./cub-gen publish --space "$SPACE" "$REPO_PATH" "$RENDER_TARGET" > "$OUT_DIR/bundle.json"
BUNDLE_CHANGE_ID="$(jq -r '.change_id // empty' "$OUT_DIR/bundle.json")"
if [ -z "$BUNDLE_CHANGE_ID" ]; then
  echo "error: bundle missing change_id" >&2
  exit 1
fi
if [ "$RUN_CHANGE_ID" != "$BUNDLE_CHANGE_ID" ]; then
  echo "error: run change_id ($RUN_CHANGE_ID) does not match bundle change_id ($BUNDLE_CHANGE_ID)" >&2
  exit 1
fi

WET_PATH="$(jq -r '.preview.edit_recommendation.wet_path // empty' "$OUT_DIR/run.json")"

# 3) Agent requests explain by existing lifecycle id via adapter.
jq -n \
  --arg action "explain" \
  --arg change_id "$RUN_CHANGE_ID" \
  --arg bundle "$OUT_DIR/bundle.json" \
  --arg wet_path "$WET_PATH" \
  '{
    action: $action,
    change: {
      change_id: $change_id,
      bundle: $bundle
    },
    filters: (if $wet_path == "" then {} else {wet_path: $wet_path} end)
  }' > "$OUT_DIR/explain-request.json"

SKIP_BUILD=1 ./examples/demo/change-api-adapter.sh \
  --request "$OUT_DIR/explain-request.json" \
  --out "$OUT_DIR/explain.json"

PREVIEW_CHANGE_ID="$(jq -r '.change.change_id // empty' "$OUT_DIR/preview.json")"
EXPLAIN_CHANGE_ID="$(jq -r '.change.change_id // empty' "$OUT_DIR/explain.json")"

jq -n \
  --arg story "7-agent-tool-call-flow" \
  --arg run_id "$RUN_ID" \
  --arg example "$EXAMPLE_SLUG" \
  --arg preview_change_id "$PREVIEW_CHANGE_ID" \
  --arg run_change_id "$RUN_CHANGE_ID" \
  --arg explain_change_id "$EXPLAIN_CHANGE_ID" \
  --arg bundle_change_id "$BUNDLE_CHANGE_ID" \
  --arg decision_state "$(jq -r '.decision.state // "UNKNOWN"' "$OUT_DIR/run.json")" \
  --arg decision_authority "$(jq -r '.decision.authority // ""' "$OUT_DIR/run.json")" \
  --arg decision_source "$(jq -r '.decision.source // ""' "$OUT_DIR/run.json")" \
  --arg wet_path "$WET_PATH" \
  --arg dry_path "$(jq -r '.explanation.dry_path // ""' "$OUT_DIR/explain.json")" \
  --arg edit_hint "$(jq -r '.explanation.edit_hint // ""' "$OUT_DIR/explain.json")" \
  '{
    story: $story,
    run_id: $run_id,
    example: $example,
    lifecycle: {
      preview_change_id: $preview_change_id,
      run_change_id: $run_change_id,
      explain_change_id: $explain_change_id,
      bundle_change_id: $bundle_change_id,
      preview_matches_run: ($preview_change_id == $run_change_id),
      run_matches_explain: ($run_change_id == $explain_change_id),
      run_matches_bundle: ($run_change_id == $bundle_change_id),
      all_equal: ($preview_change_id == $run_change_id and $run_change_id == $explain_change_id and $run_change_id == $bundle_change_id)
    },
    decision: {
      state: $decision_state,
      authority: $decision_authority,
      source: $decision_source
    },
    top_edit_recommendation: {
      wet_path: $wet_path,
      dry_path: $dry_path,
      edit_hint: $edit_hint
    }
  }' | tee "$OUT_DIR/story-7-agent-summary.json"

jq -e '.lifecycle.all_equal == true' "$OUT_DIR/story-7-agent-summary.json" >/dev/null

echo "[story-7-agent] summary: $OUT_DIR/story-7-agent-summary.json"
