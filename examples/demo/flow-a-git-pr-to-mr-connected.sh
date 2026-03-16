#!/usr/bin/env bash
# Flow A: Git PR → ConfigHub MR
#
# This demo shows the most common governed change flow:
# 1. Developer makes changes and opens Git PR
# 2. CI/webhook triggers cub-gen import
# 3. cub-gen creates/updates ConfigHub MR with evidence bundle
# 4. ConfigHub evaluates and returns governed decision (ALLOW/ESCALATE/BLOCK)
# 5. Decision is posted back to Git PR as status check
#
# Usage:
#   ./examples/demo/flow-a-git-pr-to-mr-connected.sh [REPO_PATH] [PR_NUMBER]
#
# Required env:
#   - ConfigHub auth (CONFIGHUB_TOKEN or `cub auth login`)
#   - GitHub auth (GH_TOKEN/GITHUB_TOKEN or `gh auth login`)
#
# Optional env:
#   - GIT_REPO: owner/repo (default: detect from remote)
#   - PR_NUMBER: pull request number to link
#   - OUT_ROOT: output directory root
#   - SKIP_BUILD: set to 1 to skip cub-gen rebuild

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

REPO_PATH="${1:-./examples/helm-paas}"
PR_NUMBER="${2:-${PR_NUMBER:-}}"
EXAMPLE_SLUG="$(basename "$REPO_PATH")"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/flow-a}"
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
  echo "error: missing GitHub auth for Flow A demo." >&2
  echo "remediation: set GH_TOKEN/GITHUB_TOKEN or run 'gh auth login'." >&2
  exit 1
fi
export GH_TOKEN="$gh_token"

# Resolve Git repo
GIT_REPO="${GIT_REPO:-$(resolve_default_repo)}"
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
  echo "[flow-a] building cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

mkdir -p "$OUT_DIR"

echo "[flow-a] Git PR → ConfigHub MR governed flow"
echo "[flow-a] repo: $GIT_REPO"
echo "[flow-a] example: $EXAMPLE_SLUG"
echo "[flow-a] output: $OUT_DIR"
print_connected_context

# Step 1: Discover and import
echo "[flow-a][step-1] discover and import DRY source"
./cub-gen gitops discover --space "$SPACE" --json "$REPO_PATH" > "$OUT_DIR/discover.json"
./cub-gen gitops import --space "$SPACE" --json "$REPO_PATH" "$REPO_PATH" > "$OUT_DIR/import.json"

# Step 2: Publish evidence bundle
echo "[flow-a][step-2] publish evidence bundle"
./cub-gen publish --in "$OUT_DIR/import.json" > "$OUT_DIR/bundle.json"
./cub-gen verify --json --in "$OUT_DIR/bundle.json" > "$OUT_DIR/verify.json"
./cub-gen attest --in "$OUT_DIR/bundle.json" --verifier "$VERIFIER" > "$OUT_DIR/attestation.json"

change_id="$(jq -r .change_id "$OUT_DIR/bundle.json")"
bundle_digest="$(jq -r .bundle_digest "$OUT_DIR/bundle.json")"

# Step 3: Ingest into ConfigHub (creates/updates MR)
echo "[flow-a][step-3] ingest bundle into ConfigHub"
./cub-gen bridge ingest \
  --in "$OUT_DIR/bundle.json" \
  --base-url "$CONFIGHUB_BASE_URL" \
  --token "$CONFIGHUB_TOKEN" \
  > "$OUT_DIR/ingest.json"

ingest_status="$(jq -r '.status // "unknown"' "$OUT_DIR/ingest.json")"
artifact_id="$(jq -r '.artifact_id // empty' "$OUT_DIR/ingest.json")"

# Step 4: Query backend decision
echo "[flow-a][step-4] query ConfigHub decision"
./cub-gen bridge decision query \
  --base-url "$CONFIGHUB_BASE_URL" \
  --token "$CONFIGHUB_TOKEN" \
  --change-id "$change_id" \
  > "$OUT_DIR/decision.json"

decision_state="$(jq -r '.state // "PENDING"' "$OUT_DIR/decision.json")"
policy_ref="$(jq -r '.policy_decision_ref // ""' "$OUT_DIR/decision.json")"

# Step 5: Create PR-MR linkage record
echo "[flow-a][step-5] create PR-MR linkage record"
now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
linkage_slug="$(sanitize_slug "flow-a-${change_id}")"

pr_url=""
pr_state=""
if [ -n "$PR_NUMBER" ]; then
  pr_json="$OUT_DIR/github-pr.json"
  if gh api -H "Accept: application/vnd.github+json" "repos/${GIT_REPO}/pulls/${PR_NUMBER}" > "$pr_json" 2>/dev/null; then
    pr_url="$(jq -r '.html_url // ""' "$pr_json")"
    pr_state="$(jq -r '.state // ""' "$pr_json")"
  fi
fi

jq -n \
  --arg schema "cub.confighub.io/pr-mr-promotion-flow/v1" \
  --arg change_id "$change_id" \
  --arg git_repo "$GIT_REPO" \
  --arg git_pr_number "${PR_NUMBER:-}" \
  --arg git_pr_url "$pr_url" \
  --arg git_pr_state "$pr_state" \
  --arg confighub_mr_id "$artifact_id" \
  --arg confighub_mr_status "$ingest_status" \
  --arg decision_state "$decision_state" \
  --arg policy_ref "$policy_ref" \
  --arg flow "A" \
  --arg flow_direction "git-pr-to-confighub-mr" \
  --arg updated_at "$now" \
  '{
    schema: $schema,
    change_id: $change_id,
    flow: $flow,
    flow_direction: $flow_direction,
    git_pr: {
      repo: $git_repo,
      number: (if $git_pr_number == "" then null else ($git_pr_number | tonumber) end),
      url: (if $git_pr_url == "" then null else $git_pr_url end),
      state: (if $git_pr_state == "" then null else $git_pr_state end)
    },
    confighub_mr: {
      id: $confighub_mr_id,
      status: $confighub_mr_status
    },
    decision: {
      state: $decision_state,
      policy_ref: (if $policy_ref == "" then null else $policy_ref end)
    },
    updated_at: $updated_at
  }' > "$OUT_DIR/pr-mr-linkage.json"

# Step 6: Create ConfigHub changeset for audit trail
echo "[flow-a][step-6] record linkage in ConfigHub changeset"
cub changeset create \
  --space "$CONFIGHUB_SPACE" \
  --json \
  --allow-exists \
  "$linkage_slug" \
  --description "Flow A: Git PR #${PR_NUMBER:-none} → ConfigHub MR for change_id=${change_id}" \
  --label "flow=A" \
  --label "flow_direction=git-pr-to-confighub-mr" \
  --label "change_id=${change_id}" \
  --label "git_repo=$(echo "$GIT_REPO" | tr '/' '-')" \
  --label "git_pr_number=${PR_NUMBER:-none}" \
  --label "decision_state=${decision_state}" \
  > "$OUT_DIR/linkage-changeset.json"

# Step 7: Post status to GitHub PR (if PR_NUMBER provided)
if [ -n "$PR_NUMBER" ]; then
  echo "[flow-a][step-7] post status to GitHub PR #${PR_NUMBER}"

  case "$decision_state" in
    ALLOW)
      gh_state="success"
      gh_description="ConfigHub: ALLOW - change approved"
      ;;
    ESCALATE)
      gh_state="pending"
      gh_description="ConfigHub: ESCALATE - requires additional review"
      ;;
    BLOCK)
      gh_state="failure"
      gh_description="ConfigHub: BLOCK - change rejected"
      ;;
    *)
      gh_state="pending"
      gh_description="ConfigHub: decision pending"
      ;;
  esac

  # Get head SHA from PR
  head_sha="$(jq -r '.head.sha // empty' "$pr_json")"
  if [ -n "$head_sha" ]; then
    gh api -X POST "repos/${GIT_REPO}/statuses/${head_sha}" \
      -f state="$gh_state" \
      -f description="$gh_description" \
      -f context="confighub/governed-change" \
      -f target_url="${CONFIGHUB_BASE_URL}/spaces/${CONFIGHUB_SPACE}/changes/${change_id}" \
      > "$OUT_DIR/github-status.json" 2>/dev/null || echo "[flow-a] warning: failed to post GitHub status"
  fi
else
  echo "[flow-a][step-7] skipped GitHub status (no PR_NUMBER provided)"
fi

# Summary
echo ""
echo "[flow-a] summary"
jq -n \
  --arg flow "A" \
  --arg direction "Git PR → ConfigHub MR" \
  --arg change_id "$change_id" \
  --arg bundle_digest "$bundle_digest" \
  --arg ingest_status "$ingest_status" \
  --arg decision_state "$decision_state" \
  --arg policy_ref "$policy_ref" \
  --arg git_repo "$GIT_REPO" \
  --arg git_pr_number "${PR_NUMBER:-none}" \
  --arg confighub_mr_id "$artifact_id" \
  --argjson wet_targets "$(jq '.wet_manifest_targets | length' "$OUT_DIR/import.json")" \
  '{
    flow: $flow,
    direction: $direction,
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    ingest_status: $ingest_status,
    decision_state: $decision_state,
    policy_ref: $policy_ref,
    git: {
      repo: $git_repo,
      pr_number: $git_pr_number
    },
    confighub: {
      mr_id: $confighub_mr_id
    },
    wet_targets: $wet_targets
  }' | tee "$OUT_DIR/flow-a-summary.json"
