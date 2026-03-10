#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}" 
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-1}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[story-1] existing repo import + connected query"
echo "[story-1] output: $OUT_DIR"

./examples/demo/simulate-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$OUT_DIR"

jq -n \
  --arg repo "$EXAMPLE_SLUG" \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg bundle_digest "$(jq -r .bundle_digest "$OUT_DIR/update/bundle.json")" \
  --arg ingest_status "$(jq -r '.status // "unknown"' "$OUT_DIR/update/ingest.json")" \
  --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/decision-final.json")" \
  --argjson dry_inputs "$(jq '.dry_inputs | length' "$OUT_DIR/update/import.json")" \
  --argjson provenance_records "$(jq '.provenance | length' "$OUT_DIR/update/bundle.json")" \
  --argjson wet_targets "$(jq '.wet_manifest_targets | length' "$OUT_DIR/update/import.json")" \
  '{
    story: "1-existing-repo-import-query",
    repository: $repo,
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    ingest_status: $ingest_status,
    decision_state: $decision_state,
    provenance_summary: {
      dry_inputs: $dry_inputs,
      provenance_records: $provenance_records,
      wet_targets: $wet_targets
    },
    notes: "Use change_id to query governed decision state and correlate to bundle provenance."
  }' | tee "$OUT_DIR/story-1-summary.json"
