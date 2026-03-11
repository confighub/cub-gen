#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'USAGE'
Usage:
  ./examples/demo/change-api-http-e2e.sh [target-slug] [render-target-slug]

Default:
  target-slug: ./examples/scoredev-paas
  render-target-slug: ./examples/scoredev-paas
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

TARGET_SLUG="${1:-./examples/scoredev-paas}"
RENDER_TARGET_SLUG="${2:-$TARGET_SLUG}"
LISTEN_ADDR="${CHANGE_API_LISTEN_ADDR:-127.0.0.1:18787}"
BASE_URL="http://${LISTEN_ADDR}"

for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
done

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  go build -o ./cub-gen ./cmd/cub-gen
fi

tmpdir="$(mktemp -d)"
cleanup() {
  if [ -n "${api_pid:-}" ]; then
    kill "$api_pid" >/dev/null 2>&1 || true
    wait "$api_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmpdir"
}
trap cleanup EXIT

./cub-gen change api serve --listen "$LISTEN_ADDR" --space platform --ref HEAD --verifier ci-bot >"$tmpdir/server.log" 2>&1 &
api_pid=$!

for _ in {1..40}; do
  if curl -fsS "$BASE_URL/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

if ! curl -fsS "$BASE_URL/healthz" >/dev/null 2>&1; then
  echo "error: API server did not become ready" >&2
  echo "--- server log ---" >&2
  sed -n '1,120p' "$tmpdir/server.log" >&2
  exit 1
fi

cat > "$tmpdir/run-request.json" <<JSON
{
  "action": "run",
  "mode": "local",
  "input": {
    "target_slug": "${TARGET_SLUG}",
    "render_target_slug": "${RENDER_TARGET_SLUG}",
    "space": "platform",
    "ref": "HEAD"
  }
}
JSON

curl -fsS \
  -H 'Content-Type: application/json' \
  -X POST \
  --data-binary "@$tmpdir/run-request.json" \
  "$BASE_URL/v1/changes" > "$tmpdir/run-response.json"

change_id="$(jq -r '.change.change_id // empty' "$tmpdir/run-response.json")"
if [ -z "$change_id" ]; then
  echo "error: missing change_id in run response" >&2
  cat "$tmpdir/run-response.json" >&2
  exit 1
fi

curl -fsS "$BASE_URL/v1/changes/$change_id" > "$tmpdir/get-response.json"
curl -fsS "$BASE_URL/v1/changes/$change_id/explanations?owner=app-team" > "$tmpdir/explain-response.json"

jq -e '.decision.state == "ALLOW" and .promotion_ready == true and (.verification.bundle_valid == true) and (.verification.attestation_valid == true)' "$tmpdir/run-response.json" >/dev/null
jq -e --arg id "$change_id" '.change.change_id == $id and .decision.state == "ALLOW"' "$tmpdir/get-response.json" >/dev/null
jq -e --arg id "$change_id" '.change.change_id == $id and .query.match_count >= 1 and (.explanation.owner == "app-team")' "$tmpdir/explain-response.json" >/dev/null

jq -n \
  --arg change_id "$change_id" \
  --arg decision_state "$(jq -r '.decision.state' "$tmpdir/run-response.json")" \
  --arg decision_source "$(jq -r '.decision.source' "$tmpdir/run-response.json")" \
  --arg dry_path "$(jq -r '.explanation.dry_path' "$tmpdir/explain-response.json")" \
  --arg wet_path "$(jq -r '.explanation.wet_path' "$tmpdir/explain-response.json")" \
  '{
    status: "ok",
    endpoint_contract: [
      "POST /v1/changes",
      "GET /v1/changes/{change_id}",
      "GET /v1/changes/{change_id}/explanations"
    ],
    change_id: $change_id,
    decision: {
      state: $decision_state,
      source: $decision_source
    },
    explanation: {
      dry_path: $dry_path,
      wet_path: $wet_path
    }
  }'
