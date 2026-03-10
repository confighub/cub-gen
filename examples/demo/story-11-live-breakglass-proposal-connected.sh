#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-11}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[story-11] LIVE break-glass accept/revert proposal flow"
echo "[story-11] output: $OUT_DIR"

./examples/demo/simulate-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$OUT_DIR"

change_id="$(jq -r .change_id "$OUT_DIR/update/bundle.json")"

live_observation="$OUT_DIR/update/live-observation.json"
jq -n \
  --arg change_id "$change_id" \
  --arg workload "Deployment/helm-paas-sample" \
  --arg path "spec.template.spec.containers[0].image" \
  --arg before "ghcr.io/acme-corp/sample-app:v1.2.3" \
  --arg after "ghcr.io/acme-corp/sample-app:v1.2.4-hotfix" \
  --arg reason "on-call break-glass patch to mitigate incident INC-4821" \
  '{
    schema: "confighub.io/live-observation/v1",
    change_id: $change_id,
    source: "kubectl-live-patch",
    observation: {
      workload: $workload,
      field_path: $path,
      before: $before,
      after: $after,
      reason: $reason
    }
  }' > "$live_observation"

accept_proposal="$OUT_DIR/update/proposal-accept.json"
jq -n \
  --arg change_id "$change_id" \
  --arg based_on "$live_observation" \
  --arg dry_file "examples/helm-paas/values-prod.yaml" \
  --arg dry_path "image.tag" \
  --arg value "v1.2.4-hotfix" \
  '{
    schema: "confighub.io/live-proposal/v1",
    proposal_id: ("accept-" + $change_id),
    change_id: $change_id,
    action: "accept",
    based_on_observation: $based_on,
    dry_edit: {
      file: $dry_file,
      path: $dry_path,
      value: $value
    }
  }' > "$accept_proposal"

revert_proposal="$OUT_DIR/update/proposal-revert.json"
jq -n \
  --arg change_id "$change_id" \
  --arg based_on "$live_observation" \
  --arg dry_file "examples/helm-paas/values-prod.yaml" \
  --arg dry_path "image.tag" \
  --arg value "v1.2.3" \
  '{
    schema: "confighub.io/live-proposal/v1",
    proposal_id: ("revert-" + $change_id),
    change_id: $change_id,
    action: "revert",
    based_on_observation: $based_on,
    dry_edit: {
      file: $dry_file,
      path: $dry_path,
      value: $value
    }
  }' > "$revert_proposal"

./cub-gen bridge decision create --ingest "$OUT_DIR/update/ingest.json" > "$OUT_DIR/update/live-decision-base.json"

./cub-gen bridge decision apply \
  --decision "$OUT_DIR/update/live-decision-base.json" \
  --state ESCALATE \
  --policy-ref "policy/live-break-glass-review" \
  --reason "explicit proposal required for live break-glass mutation" > "$OUT_DIR/update/live-decision-proposal.json"

jq -n \
  --arg story "11-live-breakglass-accept-revert" \
  --arg change_id "$change_id" \
  --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/live-decision-proposal.json")" \
  --arg policy_ref "$(jq -r '.policy_decision_ref // ""' "$OUT_DIR/update/live-decision-proposal.json")" \
  --arg observation "$live_observation" \
  --arg accept "$accept_proposal" \
  --arg revert "$revert_proposal" \
  '{
    story: $story,
    change_id: $change_id,
    decision_state: $decision_state,
    policy_decision_ref: $policy_ref,
    live_observation: $observation,
    proposals: [
      {action: "accept", file: $accept},
      {action: "revert", file: $revert}
    ],
    note: "LIVE mutation is converted into explicit governed proposals instead of silent overwrite."
  }' | tee "$OUT_DIR/story-11-summary.json"
