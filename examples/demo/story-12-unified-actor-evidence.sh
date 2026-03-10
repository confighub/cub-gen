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
"${ingest_cmd[@]}" > "$OUT_DIR/ingest.json"

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
