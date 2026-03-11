#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"
export CONNECTED_FALLBACK_MODE="${CONNECTED_FALLBACK_MODE:-off}"

echo "[phase-4] preflight (requires cub auth login)"
require_connected_preflight
print_connected_context

echo "[phase-4] running connected stories 8,10,11"

./examples/demo/story-8-label-evolution-connected.sh ./examples/backstage-idp ./examples/backstage-idp backstage-idp

story10_ready=1
ALLOW_STORY_10_SKIP="${ALLOW_STORY_10_SKIP:-0}"
for required_var in APP_PR_REPO APP_PR_NUMBER PROMOTION_PR_REPO PROMOTION_PR_NUMBER; do
  if [ -z "${!required_var:-}" ]; then
    story10_ready=0
    break
  fi
done

if [ "$story10_ready" -eq 1 ]; then
  ./examples/demo/story-10-signed-writeback-proof-connected.sh ./examples/helm-paas ./examples/helm-paas helm-paas
elif [ "$ALLOW_STORY_10_SKIP" = "1" ]; then
  echo "[phase-4] skipping story 10 (missing APP_PR_REPO/APP_PR_NUMBER/PROMOTION_PR_REPO/PROMOTION_PR_NUMBER)"
  echo "[phase-4] set ALLOW_STORY_10_SKIP=0 (default) to enforce story 10."
else
  echo "error: story 10 requires APP_PR_REPO, APP_PR_NUMBER, PROMOTION_PR_REPO, PROMOTION_PR_NUMBER." >&2
  echo "remediation: export these env vars (and GH_TOKEN/GITHUB_TOKEN if needed), or set ALLOW_STORY_10_SKIP=1 for local troubleshooting only." >&2
  exit 1
fi

./examples/demo/story-11-live-breakglass-proposal-connected.sh ./examples/helm-paas ./examples/helm-paas helm-paas
