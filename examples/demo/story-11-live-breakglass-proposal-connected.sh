#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"
source "$ROOT_DIR/examples/demo/lib/helm-live-reconcile-manifests.sh"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-11}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"
CLEANUP_STORY_11_CHANGESETS="${CLEANUP_STORY_11_CHANGESETS:-0}"

sanitize_slug() {
  local input="$1"
  input="$(printf '%s' "$input" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
  input="$(printf '%s' "$input" | sed -E 's/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "${input:0:63}"
}

create_backend_changeset() {
  local slug="$1"
  local description="$2"
  local output="$3"
  shift 3
  cub changeset create --space "$CONFIGHUB_SPACE" --json "$slug" --description "$description" "$@" > "$output"
}

query_changesets() {
  local where_expr="$1"
  local output="$2"
  cub changeset list --space "$CONFIGHUB_SPACE" --json --where "$where_expr" > "$output"
}

slugify_label() {
  local value="$1"
  value="$(printf '%s' "$value" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-')"
  value="$(printf '%s' "$value" | sed -E 's/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "$value"
}

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

require_connected_preflight

mkdir -p "$OUT_DIR"

echo "[story-11] LIVE break-glass accept/revert proposal flow"
echo "[story-11] output: $OUT_DIR"

./examples/demo/run-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$OUT_DIR"

change_id="$(jq -r .change_id "$OUT_DIR/update/bundle.json")"
backend_decision_state="$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/decision-final.json")"
deployment_name="$(jq -r '[.wet_manifest_targets[] | select(.kind=="Deployment") | .name][0] // "payments-api"' "$OUT_DIR/update/import.json")"
workload="Deployment/${deployment_name}"
path="$(jq -r '[.inverse_transform_plans[]?.patches[]? | select((.wet_path // "") | test("image")) | .wet_path][0] // "Deployment/spec/template/spec/containers[0]/image"' "$OUT_DIR/update/import.json")"

before_repo="$(resolve_helm_value "$OUT_DIR/create/repo" "image.repository")"
before_tag="$(resolve_helm_value "$OUT_DIR/create/repo" "image.tag")"
after_repo="$(resolve_helm_value "$OUT_DIR/update/repo" "image.repository")"
after_tag="$(resolve_helm_value "$OUT_DIR/update/repo" "image.tag")"

if [ -z "$before_repo" ] || [ -z "$before_tag" ] || [ -z "$after_repo" ] || [ -z "$after_tag" ]; then
  echo "error: failed to resolve image values from connected lifecycle repo snapshots for Story 11." >&2
  exit 1
fi

before="${before_repo}:${before_tag}"
after="${after_repo}:${after_tag}"
reason="connected lifecycle image delta detected for ${deployment_name} (${before} -> ${after})"
field_path_label="$(slugify_label "$path")"
workload_label="$(slugify_label "$deployment_name")"
before_label="$(slugify_label "$before_tag")"
after_label="$(slugify_label "$after_tag")"

observation_slug="$(sanitize_slug "story11-live-observation-${EXAMPLE_SLUG}-${RUN_ID}")"
live_observation="$OUT_DIR/update/live-observation-backend.json"
create_backend_changeset \
  "$observation_slug" \
  "Story 11 live break-glass observation for change_id=${change_id}: ${reason}" \
  "$live_observation" \
  --label "story=11" \
  --label "change_id=${change_id}" \
  --label "proposal_role=live_observation" \
  --label "workload=${workload_label}" \
  --label "field_path=${field_path_label}" \
  --label "from=${before_label}" \
  --label "to=${after_label}"

observation_changeset_id="$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$live_observation")"
if [ -z "$observation_changeset_id" ]; then
  echo "error: unable to resolve backend observation ChangeSetID for Story 11." >&2
  exit 1
fi

accept_slug="$(sanitize_slug "story11-accept-${EXAMPLE_SLUG}-${RUN_ID}")"
accept_proposal="$OUT_DIR/update/proposal-accept-backend.json"
create_backend_changeset \
  "$accept_slug" \
  "Story 11 ACCEPT proposal for change_id=${change_id}: persist break-glass image tag." \
  "$accept_proposal" \
  --label "story=11" \
  --label "change_id=${change_id}" \
  --label "proposal_role=live_breakglass" \
  --label "proposal_action=accept" \
  --label "observation_changeset_id=${observation_changeset_id}" \
  --label "dry_file=examples-helm-paas-values-prod-yaml" \
  --label "dry_path=image-tag" \
  --label "dry_value=${after_label}"

revert_slug="$(sanitize_slug "story11-revert-${EXAMPLE_SLUG}-${RUN_ID}")"
revert_proposal="$OUT_DIR/update/proposal-revert-backend.json"
create_backend_changeset \
  "$revert_slug" \
  "Story 11 REVERT proposal for change_id=${change_id}: rollback break-glass image tag." \
  "$revert_proposal" \
  --label "story=11" \
  --label "change_id=${change_id}" \
  --label "proposal_role=live_breakglass" \
  --label "proposal_action=revert" \
  --label "observation_changeset_id=${observation_changeset_id}" \
  --label "dry_file=examples-helm-paas-values-prod-yaml" \
  --label "dry_path=image-tag" \
  --label "dry_value=${before_label}"

accept_where="Labels.story = '11' AND Labels.change_id = '${change_id}' AND Labels.proposal_action = 'accept'"
revert_where="Labels.story = '11' AND Labels.change_id = '${change_id}' AND Labels.proposal_action = 'revert'"
accept_query="$OUT_DIR/update/proposal-accept-query.json"
revert_query="$OUT_DIR/update/proposal-revert-query.json"
query_changesets "$accept_where" "$accept_query"
query_changesets "$revert_where" "$revert_query"

accept_hits="$(jq 'length' "$accept_query")"
revert_hits="$(jq 'length' "$revert_query")"
if [ "$accept_hits" -lt 1 ] || [ "$revert_hits" -lt 1 ]; then
  echo "error: backend proposal queries did not return expected Story 11 proposals." >&2
  echo "  accept_hits=$accept_hits revert_hits=$revert_hits" >&2
  exit 1
fi

live_proposal="$OUT_DIR/update/live-decision-proposal.json"
jq -n \
  --arg change_id "$change_id" \
  --arg current_state "$backend_decision_state" \
  --arg current_policy_ref "$(jq -r '.policy_decision_ref // ""' "$OUT_DIR/update/decision-final.json")" \
  --arg proposed_policy_ref "policy/live-break-glass-review" \
  --arg workload "$workload" \
  --arg field_path "$path" \
  --arg before "$before" \
  --arg after "$after" \
  --arg observation_reason "$reason" \
  --arg observation_changeset_id "$observation_changeset_id" \
  --arg accept_changeset_id "$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$accept_proposal")" \
  --arg revert_changeset_id "$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$revert_proposal")" \
  --arg accept_query "$accept_query" \
  --arg revert_query "$revert_query" \
  --argjson accept_hits "$accept_hits" \
  --argjson revert_hits "$revert_hits" \
  --arg proposal_reason "explicit proposal required for live break-glass mutation" \
  '{
    schema: "confighub.io/live-decision-proposal/v2",
    change_id: $change_id,
    current_decision_state: $current_state,
    current_policy_decision_ref: $current_policy_ref,
    proposed_decision_state: "ESCALATE",
    proposed_policy_decision_ref: $proposed_policy_ref,
    reason: $proposal_reason,
    observation: {
      workload: $workload,
      field_path: $field_path,
      before: $before,
      after: $after,
      reason: $observation_reason,
      backend_changeset_id: $observation_changeset_id
    },
    proposals: [
      {
        action: "accept",
        backend_changeset_id: $accept_changeset_id,
        query_file: $accept_query,
        hits: $accept_hits
      },
      {
        action: "revert",
        backend_changeset_id: $revert_changeset_id,
        query_file: $revert_query,
        hits: $revert_hits
      }
    ]
  }' > "$live_proposal"

if [ "$CLEANUP_STORY_11_CHANGESETS" = "1" ]; then
  cub changeset delete --space "$CONFIGHUB_SPACE" --quiet "$accept_slug" || true
  cub changeset delete --space "$CONFIGHUB_SPACE" --quiet "$revert_slug" || true
  cub changeset delete --space "$CONFIGHUB_SPACE" --quiet "$observation_slug" || true
fi

jq -n \
  --arg story "11-live-breakglass-accept-revert" \
  --arg change_id "$change_id" \
  --arg decision_state "$(jq -r '.current_decision_state // "UNKNOWN"' "$live_proposal")" \
  --arg proposed_decision_state "$(jq -r '.proposed_decision_state // "ESCALATE"' "$live_proposal")" \
  --arg policy_ref "$(jq -r '.proposed_policy_decision_ref // ""' "$live_proposal")" \
  --arg observation "$live_observation" \
  --arg accept "$accept_proposal" \
  --arg revert "$revert_proposal" \
  --arg decision_proposal "$live_proposal" \
  --argjson accept_hits "$accept_hits" \
  --argjson revert_hits "$revert_hits" \
  '{
    story: $story,
    change_id: $change_id,
    backend_decision_state: $decision_state,
    proposed_decision_state: $proposed_decision_state,
    policy_decision_ref: $policy_ref,
    live_decision_proposal: $decision_proposal,
    live_observation: $observation,
    proposals: [
      {action: "accept", file: $accept, backend_query_hits: $accept_hits},
      {action: "revert", file: $revert, backend_query_hits: $revert_hits}
    ],
    note: "LIVE mutation is converted into explicit governed proposals instead of silent overwrite."
  }' | tee "$OUT_DIR/story-11-summary.json"
