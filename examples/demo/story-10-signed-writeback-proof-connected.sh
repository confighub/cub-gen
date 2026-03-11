#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPO_PATH="${1:-./examples/helm-paas}"
RENDER_TARGET="${2:-$REPO_PATH}"
EXAMPLE_SLUG="${3:-$(basename "$REPO_PATH")}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/story-10}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"

APP_PR_REPO="${APP_PR_REPO:-}"
APP_PR_NUMBER="${APP_PR_NUMBER:-}"
PROMOTION_PR_REPO="${PROMOTION_PR_REPO:-}"
PROMOTION_PR_NUMBER="${PROMOTION_PR_NUMBER:-}"
STRICT_PROOF="${STRICT_PROOF:-1}"

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

require_positive_int() {
  local label="$1"
  local value="$2"
  if ! [[ "$value" =~ ^[1-9][0-9]*$ ]]; then
    echo "error: $label must be a positive integer, got: $value" >&2
    exit 1
  fi
}

fetch_pr_json() {
  local repo="$1"
  local pr_number="$2"
  local output="$3"
  if ! gh api -H "Accept: application/vnd.github+json" "repos/${repo}/pulls/${pr_number}" > "$output"; then
    echo "error: failed to fetch PR metadata for ${repo}#${pr_number}" >&2
    echo "remediation: ensure GH/GITHUB token can read the repository and pull request." >&2
    exit 1
  fi
}

fetch_commit_json() {
  local repo="$1"
  local sha="$2"
  local output="$3"
  if ! gh api -H "Accept: application/vnd.github+json" "repos/${repo}/commits/${sha}" > "$output"; then
    echo "error: failed to fetch commit metadata for ${repo}@${sha}" >&2
    echo "remediation: ensure commit SHA exists and token has repo read access." >&2
    exit 1
  fi
}

fetch_branch_protection_json() {
  local repo="$1"
  local branch="$2"
  local output="$3"
  local encoded_branch
  encoded_branch="$(jq -rn --arg v "$branch" '$v|@uri')"
  if ! gh api -H "Accept: application/vnd.github+json" "repos/${repo}/branches/${encoded_branch}/protection" > "$output"; then
    echo "error: failed to fetch branch protection for ${repo}:${branch}" >&2
    echo "remediation: token must have permission to read branch protection on the target repository." >&2
    exit 1
  fi
}

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

require_cmd gh
require_cmd jq
require_cmd shasum

token="$(resolve_gh_token || true)"
if [ -z "$token" ]; then
  echo "error: missing GitHub auth for Story 10 proof capture." >&2
  echo "remediation: set GH_TOKEN/GITHUB_TOKEN or run 'gh auth login' first." >&2
  exit 1
fi
export GH_TOKEN="$token"

default_repo="$(resolve_default_repo)"
if [ -n "$default_repo" ]; then
  APP_PR_REPO="${APP_PR_REPO:-$default_repo}"
  PROMOTION_PR_REPO="${PROMOTION_PR_REPO:-$default_repo}"
fi

if [ -z "$APP_PR_REPO" ] || [ -z "$APP_PR_NUMBER" ] || [ -z "$PROMOTION_PR_REPO" ] || [ -z "$PROMOTION_PR_NUMBER" ]; then
  echo "error: Story 10 requires PR coordinates for app and promotion write-back proofs." >&2
  echo "required env: APP_PR_REPO, APP_PR_NUMBER, PROMOTION_PR_REPO, PROMOTION_PR_NUMBER" >&2
  exit 1
fi

APP_PR_REPO="$(normalize_repo "$APP_PR_REPO")"
PROMOTION_PR_REPO="$(normalize_repo "$PROMOTION_PR_REPO")"
require_positive_int "APP_PR_NUMBER" "$APP_PR_NUMBER"
require_positive_int "PROMOTION_PR_NUMBER" "$PROMOTION_PR_NUMBER"

mkdir -p "$OUT_DIR"

echo "[story-10] signed commit + branch protection write-back proof"
echo "[story-10] output: $OUT_DIR"
echo "[story-10] app PR: ${APP_PR_REPO}#${APP_PR_NUMBER}"
echo "[story-10] promotion PR: ${PROMOTION_PR_REPO}#${PROMOTION_PR_NUMBER}"

./examples/demo/run-confighub-lifecycle-connected.sh "$REPO_PATH" "$RENDER_TARGET" "$EXAMPLE_SLUG" "$OUT_DIR"

decision_state="$(jq -r '.state // "UNKNOWN"' "$OUT_DIR/update/decision-final.json")"
if [ "$decision_state" != "ALLOW" ]; then
  echo "error: backend decision is $decision_state; Story 10 write-back proof requires ALLOW." >&2
  exit 1
fi

app_pr_json="$OUT_DIR/update/github-app-pr.json"
promotion_pr_json="$OUT_DIR/update/github-promotion-pr.json"
fetch_pr_json "$APP_PR_REPO" "$APP_PR_NUMBER" "$app_pr_json"
fetch_pr_json "$PROMOTION_PR_REPO" "$PROMOTION_PR_NUMBER" "$promotion_pr_json"

app_commit_sha="$(jq -r '.merge_commit_sha // .head.sha // empty' "$app_pr_json")"
promotion_commit_sha="$(jq -r '.merge_commit_sha // .head.sha // empty' "$promotion_pr_json")"
if [ -z "$app_commit_sha" ] || [ -z "$promotion_commit_sha" ]; then
  echo "error: unable to resolve commit SHA from PR metadata." >&2
  exit 1
fi

app_base_branch="$(jq -r '.base.ref // empty' "$app_pr_json")"
promotion_base_branch="$(jq -r '.base.ref // empty' "$promotion_pr_json")"
if [ -z "$app_base_branch" ] || [ -z "$promotion_base_branch" ]; then
  echo "error: unable to resolve base branch from PR metadata." >&2
  exit 1
fi

app_commit_json="$OUT_DIR/update/github-app-commit.json"
promotion_commit_json="$OUT_DIR/update/github-promotion-commit.json"
app_branch_protection_json="$OUT_DIR/update/github-app-branch-protection.json"
promotion_branch_protection_json="$OUT_DIR/update/github-promotion-branch-protection.json"

fetch_commit_json "$APP_PR_REPO" "$app_commit_sha" "$app_commit_json"
fetch_commit_json "$PROMOTION_PR_REPO" "$promotion_commit_sha" "$promotion_commit_json"
fetch_branch_protection_json "$APP_PR_REPO" "$app_base_branch" "$app_branch_protection_json"
fetch_branch_protection_json "$PROMOTION_PR_REPO" "$promotion_base_branch" "$promotion_branch_protection_json"

writeback_proof="$OUT_DIR/update/writeback-proof.json"
jq -n \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg fetched_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg app_repo "$APP_PR_REPO" \
  --arg promotion_repo "$PROMOTION_PR_REPO" \
  --slurpfile app_pr "$app_pr_json" \
  --slurpfile promotion_pr "$promotion_pr_json" \
  --slurpfile app_commit "$app_commit_json" \
  --slurpfile promotion_commit "$promotion_commit_json" \
  --slurpfile app_bp "$app_branch_protection_json" \
  --slurpfile promotion_bp "$promotion_branch_protection_json" \
  '
    def pr_view($pr):
      {
        number: $pr.number,
        url: $pr.html_url,
        state: $pr.state,
        merged: ($pr.merged // false),
        base_branch: $pr.base.ref,
        head_sha: $pr.head.sha,
        merge_commit_sha: ($pr.merge_commit_sha // ""),
        commit_sha_used: ($pr.merge_commit_sha // $pr.head.sha)
      };
    def sig_view($commit):
      {
        verified: ($commit.commit.verification.verified // false),
        reason: ($commit.commit.verification.reason // "unknown"),
        signer: ($commit.commit.committer.email // $commit.commit.author.email // ""),
        verified_at: ($commit.commit.verification.verified_at // null)
      };
    def bp_view($bp):
      {
        required_approving_review_count: ($bp.required_pull_request_reviews.required_approving_review_count // 0),
        required_status_checks: ($bp.required_status_checks.contexts // []),
        required_status_checks_count: ($bp.required_status_checks.contexts // [] | length),
        strict_status_checks: ($bp.required_status_checks.strict // false),
        enforce_admins: ($bp.enforce_admins.enabled // false),
        required_linear_history: ($bp.required_linear_history.enabled // false),
        allow_force_pushes: ($bp.allow_force_pushes.enabled // false),
        allow_deletions: ($bp.allow_deletions.enabled // false)
      };
    $app_pr[0] as $appPr
    | $promotion_pr[0] as $promotionPr
    | $app_commit[0] as $appCommit
    | $promotion_commit[0] as $promotionCommit
    | $app_bp[0] as $appBpRaw
    | $promotion_bp[0] as $promotionBpRaw
    | pr_view($appPr) as $appPrView
    | pr_view($promotionPr) as $promotionPrView
    | sig_view($appCommit) as $appSig
    | sig_view($promotionCommit) as $promotionSig
    | bp_view($appBpRaw) as $appBp
    | bp_view($promotionBpRaw) as $promotionBp
    | ($appSig.verified and $promotionSig.verified) as $signatures_verified
    | (
        ($appBp.required_approving_review_count > 0)
        and ($promotionBp.required_approving_review_count >= $appBp.required_approving_review_count)
        and ($appBp.required_status_checks_count > 0)
        and ($promotionBp.required_status_checks_count >= $appBp.required_status_checks_count)
        and (if $appBp.enforce_admins then $promotionBp.enforce_admins else true end)
      ) as $branch_protection_preserved
    | {
        schema: "confighub.io/writeback-proof/v2",
        change_id: $change_id,
        fetched_at: $fetched_at,
        app_pr: ({repo: $app_repo} + $appPrView + {
          commit_signature: $appSig,
          branch_protection: $appBp
        }),
        promotion_pr: ({repo: $promotion_repo} + $promotionPrView + {
          commit_signature: $promotionSig,
          branch_protection: $promotionBp
        }),
        verification: {
          signatures_verified: $signatures_verified,
          branch_protection_preserved: $branch_protection_preserved,
          proof_complete: true
        }
      }
  ' > "$writeback_proof"

writeback_digest="$(shasum -a 256 "$writeback_proof" | awk '{print $1}')"

signatures_verified="$(jq -r '.verification.signatures_verified' "$writeback_proof")"
branch_protection_preserved="$(jq -r '.verification.branch_protection_preserved' "$writeback_proof")"

if [ "$STRICT_PROOF" = "1" ] && { [ "$signatures_verified" != "true" ] || [ "$branch_protection_preserved" != "true" ]; }; then
  echo "error: Story 10 proof verification failed." >&2
  echo "  signatures_verified=$signatures_verified" >&2
  echo "  branch_protection_preserved=$branch_protection_preserved" >&2
  echo "remediation: ensure both PR commits are signed/verified and branch protection is active on write-back branches." >&2
  exit 1
fi

jq -n \
  --arg story "10-signed-commit-branch-protection-proof" \
  --arg change_id "$(jq -r .change_id "$OUT_DIR/update/bundle.json")" \
  --arg bundle_digest "$(jq -r .bundle_digest "$OUT_DIR/update/bundle.json")" \
  --arg decision_state "$decision_state" \
  --arg ingest_status "$(jq -r '.status // "unknown"' "$OUT_DIR/update/ingest.json")" \
  --arg proof_file "$writeback_proof" \
  --arg proof_digest "$writeback_digest" \
  --argjson signatures_verified "$(jq '.verification.signatures_verified' "$writeback_proof")" \
  --argjson branch_protection_preserved "$(jq '.verification.branch_protection_preserved' "$writeback_proof")" \
  '{
    story: $story,
    change_id: $change_id,
    bundle_digest: $bundle_digest,
    decision_state: $decision_state,
    ingest_status: $ingest_status,
    writeback_proof_file: $proof_file,
    writeback_proof_digest: $proof_digest,
    signatures_verified: $signatures_verified,
    branch_protection_preserved: $branch_protection_preserved
  }' | tee "$OUT_DIR/story-10-summary.json"
