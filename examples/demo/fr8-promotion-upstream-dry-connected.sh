#!/usr/bin/env bash
# FR8: Promotion to Upstream Platform DRY
#
# This demo shows the full promotion flow from live observation through
# WET evidence to upstream DRY promotion:
#
# 1. LIVE → WET: Observe live state, capture delta as WET evidence
# 2. WET → governed: Evaluate delta against policies, reach ALLOW decision
# 3. governed → promotion: After successful rollout, propose promotion
# 4. promotion → upstream DRY: Platform team reviews and merges upstream PR
# 5. app overlay cleanup: Reduce/remove app overlay to avoid drift
#
# This flow implements the "reusable default promotion" pattern from the
# PR-MR linkage contract, where successful app-level changes are promoted
# to the platform's base DRY when they represent a good default.
#
# Usage:
#   ./examples/demo/fr8-promotion-upstream-dry-connected.sh [REPO_PATH]
#
# Required env:
#   - ConfigHub auth (CONFIGHUB_TOKEN or `cub auth login`)
#   - GitHub auth (GH_TOKEN/GITHUB_TOKEN or `gh auth login`)
#
# Optional env:
#   - GIT_REPO: owner/repo (default: detect from remote)
#   - UPSTREAM_REPO: upstream platform repo (default: same as GIT_REPO)
#   - OUT_ROOT: output directory root
#   - SKIP_BUILD: set to 1 to skip cub-gen rebuild

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"
source "$ROOT_DIR/examples/demo/lib/lifecycle-update.sh"

REPO_PATH="${1:-./examples/helm-paas}"
EXAMPLE_SLUG="$(basename "$REPO_PATH")"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/fr8-promotion}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"
SPACE="${SPACE:-}"
VERIFIER="${VERIFIER:-ci-bot}"

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
}

normalize_repo() {
  local value="$1"
  value="${value#https://github.com/}"
  value="${value#http://github.com/}"
  value="${value#git@github.com:}"
  value="${value#github.com/}"
  value="${value%.git}"
  printf '%s' "$value"
}

resolve_default_repo() {
  local remote
  remote="$(git config --get remote.origin.url 2>/dev/null || true)"
  if [ -z "$remote" ]; then
    return 0
  fi
  normalize_repo "$remote"
}

resolve_gh_token() {
  local token
  token="$(printf '%s' "${GH_TOKEN:-${GITHUB_TOKEN:-}}" | tr -d '\r\n')"
  if [ -n "$token" ]; then
    printf '%s' "$token"
    return 0
  fi
  if token="$(gh auth token 2>/dev/null | tr -d '\r\n')" && [ -n "$token" ]; then
    printf '%s' "$token"
    return 0
  fi
  return 1
}

sanitize_slug() {
  local input="$1"
  input="$(printf '%s' "$input" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
  input="$(printf '%s' "$input" | sed -E 's/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "${input:0:63}"
}

is_terminal_decision_state() {
  local state="$1"
  case "$state" in
    ALLOW|ESCALATE|BLOCK) return 0 ;;
    *) return 1 ;;
  esac
}

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

require_cmd gh
require_cmd jq
require_cmd shasum

# Resolve GitHub auth
gh_token="$(resolve_gh_token || true)"
if [ -z "$gh_token" ]; then
  echo "error: missing GitHub auth for FR8 promotion demo." >&2
  echo "remediation: set GH_TOKEN/GITHUB_TOKEN or run 'gh auth login'." >&2
  exit 1
fi
export GH_TOKEN="$gh_token"

# Resolve Git repos
GIT_REPO="${GIT_REPO:-$(resolve_default_repo)}"
UPSTREAM_REPO="${UPSTREAM_REPO:-$GIT_REPO}"

if [ -z "$GIT_REPO" ]; then
  echo "error: unable to resolve Git repository." >&2
  echo "remediation: set GIT_REPO=owner/repo or run from a Git repo with remote.origin.url." >&2
  exit 1
fi

# Require ConfigHub preflight
require_connected_preflight
if [ -z "$SPACE" ]; then
  SPACE="$CONFIGHUB_SPACE"
fi

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[fr8] building cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

mkdir -p "$OUT_DIR"

echo "[fr8] Promotion to Upstream Platform DRY"
echo "[fr8] app repo: $GIT_REPO"
echo "[fr8] upstream repo: $UPSTREAM_REPO"
echo "[fr8] example: $EXAMPLE_SLUG"
echo "[fr8] output: $OUT_DIR"
print_connected_context

# Prepare working copy
work_repo="$OUT_DIR/repo"
mkdir -p "$work_repo"
cp -R "$REPO_PATH"/. "$work_repo"

#############################################################################
# Phase 1: LIVE → WET (Observe and capture current state)
#############################################################################
echo ""
echo "[fr8][phase-1] LIVE → WET: Observe and capture state"

# Discover and import baseline
./cub-gen gitops discover --space "$SPACE" --json "$work_repo" > "$OUT_DIR/discover-baseline.json"
./cub-gen gitops import --space "$SPACE" --json "$work_repo" "$work_repo" > "$OUT_DIR/import-baseline.json"
./cub-gen publish --in "$OUT_DIR/import-baseline.json" > "$OUT_DIR/bundle-baseline.json"

baseline_change_id="$(jq -r .change_id "$OUT_DIR/bundle-baseline.json")"
echo "[fr8] baseline change_id: $baseline_change_id"

#############################################################################
# Phase 2: Apply app-level change (simulate feature rollout)
#############################################################################
echo ""
echo "[fr8][phase-2] Apply app-level change"

apply_update "$EXAMPLE_SLUG" "$work_repo"

# Import after change
./cub-gen gitops import --space "$SPACE" --json "$work_repo" "$work_repo" > "$OUT_DIR/import-changed.json"
./cub-gen publish --in "$OUT_DIR/import-changed.json" > "$OUT_DIR/bundle-changed.json"
./cub-gen verify --json --in "$OUT_DIR/bundle-changed.json" > "$OUT_DIR/verify-changed.json"
./cub-gen attest --in "$OUT_DIR/bundle-changed.json" --verifier "$VERIFIER" > "$OUT_DIR/attestation-changed.json"

changed_change_id="$(jq -r .change_id "$OUT_DIR/bundle-changed.json")"
echo "[fr8] changed change_id: $changed_change_id"

#############################################################################
# Phase 3: WET → governed (Ingest and evaluate)
#############################################################################
echo ""
echo "[fr8][phase-3] WET → governed: Ingest and evaluate"

./cub-gen bridge ingest \
  --in "$OUT_DIR/bundle-changed.json" \
  --base-url "$CONFIGHUB_BASE_URL" \
  --token "$CONFIGHUB_TOKEN" \
  > "$OUT_DIR/ingest.json"

# Query decision
./cub-gen bridge decision query \
  --base-url "$CONFIGHUB_BASE_URL" \
  --token "$CONFIGHUB_TOKEN" \
  --change-id "$changed_change_id" \
  > "$OUT_DIR/decision.json"

decision_state="$(jq -r '.state // "PENDING"' "$OUT_DIR/decision.json")"
echo "[fr8] decision state: $decision_state"

if [ "$decision_state" != "ALLOW" ]; then
  echo "[fr8] warning: decision is $decision_state; promotion requires ALLOW"
  echo "[fr8] continuing demo to show promotion proposal structure"
fi

#############################################################################
# Phase 4: governed → promotion (Propose upstream promotion)
#############################################################################
echo ""
echo "[fr8][phase-4] governed → promotion: Propose upstream DRY promotion"

now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
promotion_slug="$(sanitize_slug "fr8-promotion-${EXAMPLE_SLUG}-${RUN_ID}")"

# Extract what changed (inverse-edit guidance)
jq -r '.inverse_transform_plans // []' "$OUT_DIR/import-changed.json" > "$OUT_DIR/inverse-guidance.json"

# Identify fields suitable for promotion
# (fields that are app-overlay overrides but could be platform defaults)
jq '[.inverse_transform_plans[]?.patches[]? | select(.owner == "app-team" or .owner == null) | {
  dry_file: .dry_file,
  dry_path: .dry_path,
  current_value: .current_value,
  promotion_candidate: true
}]' "$OUT_DIR/import-changed.json" > "$OUT_DIR/promotion-candidates.json"

# Create promotion proposal changeset
cub changeset create \
  --space "$CONFIGHUB_SPACE" \
  --json \
  --allow-exists \
  "$promotion_slug" \
  --description "FR8: Promotion proposal for ${EXAMPLE_SLUG} (change_id=${changed_change_id})" \
  --label "flow=FR8" \
  --label "flow_stage=promotion-proposed" \
  --label "source_change_id=${changed_change_id}" \
  --label "baseline_change_id=${baseline_change_id}" \
  --label "decision_state=${decision_state}" \
  --label "app_repo=$(echo "$GIT_REPO" | tr '/' '-')" \
  --label "upstream_repo=$(echo "$UPSTREAM_REPO" | tr '/' '-')" \
  > "$OUT_DIR/promotion-changeset.json"

promotion_changeset_id="$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$OUT_DIR/promotion-changeset.json")"

#############################################################################
# Phase 5: Generate upstream promotion PR proposal
#############################################################################
echo ""
echo "[fr8][phase-5] Generate upstream promotion PR"

# Determine promotion target files
upstream_branch="confighub/fr8-promotion-${RUN_ID}"
upstream_target_branch="${UPSTREAM_TARGET_BRANCH:-main}"

# Build PR content
promotion_pr_title="[FR8 Promotion] ${EXAMPLE_SLUG}: promote app defaults to platform base"
promotion_pr_body="## FR8: Promotion to Upstream Platform DRY

This PR promotes successful app-level changes to the platform's base DRY,
making them the new default for all deployments.

### Promotion Chain

| Stage | Change ID | Status |
|-------|-----------|--------|
| Baseline | \`${baseline_change_id}\` | captured |
| App Change | \`${changed_change_id}\` | ${decision_state} |
| Promotion | \`${promotion_changeset_id}\` | proposed |

### What's Being Promoted

$(jq -r 'if length > 0 then
  "| DRY File | DRY Path |\n|----------|----------|\n" +
  (map("| \(.dry_file) | \(.dry_path) |") | join("\n"))
else
  "No specific promotion candidates identified."
end' "$OUT_DIR/promotion-candidates.json")

### Promotion Contract

Per the PR-MR linkage contract, this promotion:
- [x] Follows a prior \`ALLOW\` decision (or demo mode)
- [x] Requires separate platform review approval
- [ ] Will reduce/remove app overlay after merge

### Flow

1. **LIVE → WET**: Observed live state, captured delta
2. **WET → governed**: Evaluated against policies → ${decision_state}
3. **governed → promotion**: This promotion PR
4. **promotion → upstream DRY**: Platform team reviews
5. **cleanup**: App overlay reduced after merge

### Linkage

\`\`\`json
{
  \"flow\": \"FR8\",
  \"source_change_id\": \"${changed_change_id}\",
  \"promotion_changeset_id\": \"${promotion_changeset_id}\"
}
\`\`\`

---
Generated by: cub-gen FR8 promotion demo"

jq -n \
  --arg branch "$upstream_branch" \
  --arg title "$promotion_pr_title" \
  --arg body "$promotion_pr_body" \
  --arg target_branch "$upstream_target_branch" \
  --arg upstream_repo "$UPSTREAM_REPO" \
  --slurpfile candidates "$OUT_DIR/promotion-candidates.json" \
  '{
    upstream_repo: $upstream_repo,
    branch: $branch,
    title: $title,
    body: $body,
    target_branch: $target_branch,
    promotion_candidates: $candidates[0]
  }' > "$OUT_DIR/promotion-pr-proposal.json"

#############################################################################
# Phase 6: Create promotion linkage record
#############################################################################
echo ""
echo "[fr8][phase-6] Create promotion linkage record"

linkage_slug="$(sanitize_slug "fr8-linkage-${EXAMPLE_SLUG}-${RUN_ID}")"

jq -n \
  --arg schema "cub.confighub.io/pr-mr-promotion-flow/v1" \
  --arg flow "FR8" \
  --arg flow_stage "promotion-proposed" \
  --arg baseline_change_id "$baseline_change_id" \
  --arg source_change_id "$changed_change_id" \
  --arg decision_state "$decision_state" \
  --arg promotion_changeset_id "$promotion_changeset_id" \
  --arg app_repo "$GIT_REPO" \
  --arg upstream_repo "$UPSTREAM_REPO" \
  --arg upstream_branch "$upstream_branch" \
  --arg upstream_target_branch "$upstream_target_branch" \
  --arg updated_at "$now" \
  '{
    schema: $schema,
    flow: $flow,
    flow_stage: $flow_stage,
    source: {
      baseline_change_id: $baseline_change_id,
      change_id: $source_change_id,
      decision_state: $decision_state,
      app_repo: $app_repo
    },
    promotion: {
      changeset_id: $promotion_changeset_id,
      upstream_repo: $upstream_repo,
      branch: $upstream_branch,
      target_branch: $upstream_target_branch,
      pr_number: null,
      pr_url: null,
      status: "PROPOSED"
    },
    updated_at: $updated_at
  }' > "$OUT_DIR/promotion-linkage.json"

# Record in ConfigHub
cub changeset create \
  --space "$CONFIGHUB_SPACE" \
  --json \
  --allow-exists \
  "$linkage_slug" \
  --description "FR8: Promotion linkage for ${EXAMPLE_SLUG}" \
  --label "flow=FR8" \
  --label "flow_stage=linkage" \
  --label "source_change_id=${changed_change_id}" \
  --label "promotion_changeset_id=${promotion_changeset_id}" \
  > "$OUT_DIR/linkage-changeset.json"

#############################################################################
# Instructions
#############################################################################
echo ""
echo "[fr8] promotion instructions"
echo ""
echo "To complete FR8 promotion:"
echo ""
echo "  1. Review promotion candidates:"
echo "     cat ${OUT_DIR}/promotion-candidates.json"
echo ""
echo "  2. Create upstream promotion PR:"
echo "     cd <upstream-repo>"
echo "     git checkout -b ${upstream_branch}"
echo "     # Apply promotion changes to base DRY files"
echo "     git commit -m \"${promotion_pr_title}"
echo ""
echo "     ConfigHub-Changeset-ID: ${promotion_changeset_id}\""
echo "     git push -u origin ${upstream_branch}"
echo "     gh pr create --title \"${promotion_pr_title}\" --body-file ${OUT_DIR}/promotion-pr-proposal.json"
echo ""
echo "  3. After merge, clean up app overlay:"
echo "     # Remove app-specific overrides that are now platform defaults"
echo ""

#############################################################################
# Summary
#############################################################################
echo "[fr8] summary"
jq -n \
  --arg flow "FR8" \
  --arg flow_name "Promotion to Upstream Platform DRY" \
  --arg baseline_change_id "$baseline_change_id" \
  --arg source_change_id "$changed_change_id" \
  --arg decision_state "$decision_state" \
  --arg promotion_changeset_id "$promotion_changeset_id" \
  --arg app_repo "$GIT_REPO" \
  --arg upstream_repo "$UPSTREAM_REPO" \
  --arg upstream_branch "$upstream_branch" \
  --argjson promotion_candidates "$(jq 'length' "$OUT_DIR/promotion-candidates.json")" \
  '{
    flow: $flow,
    flow_name: $flow_name,
    phases: [
      "LIVE → WET (observe)",
      "WET → governed (evaluate)",
      "governed → promotion (propose)",
      "promotion → upstream DRY (review)",
      "cleanup (reduce overlay)"
    ],
    source: {
      baseline_change_id: $baseline_change_id,
      change_id: $source_change_id,
      decision_state: $decision_state,
      app_repo: $app_repo
    },
    promotion: {
      changeset_id: $promotion_changeset_id,
      upstream_repo: $upstream_repo,
      branch: $upstream_branch,
      candidates: $promotion_candidates,
      status: "PROPOSED"
    }
  }' | tee "$OUT_DIR/fr8-summary.json"
