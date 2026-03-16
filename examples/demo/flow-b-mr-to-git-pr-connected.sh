#!/usr/bin/env bash
# Flow B: ConfigHub MR → Git PR
#
# This demo shows the reverse governed flow:
# 1. ConfigHub detects a change (live observation, platform update, or promotion)
# 2. ConfigHub creates an MR with proposed changes
# 3. After MR approval, cub-gen generates a Git PR
# 4. Git PR is linked back to ConfigHub MR
# 5. Platform team reviews and merges Git PR
#
# This flow is common for:
# - Live-origin proposals (story 11 accept path)
# - Platform-initiated changes
# - Upstream DRY promotions
#
# Usage:
#   ./examples/demo/flow-b-mr-to-git-pr-connected.sh [REPO_PATH] [CHANGESET_SLUG]
#
# Required env:
#   - ConfigHub auth (CONFIGHUB_TOKEN or `cub auth login`)
#   - GitHub auth (GH_TOKEN/GITHUB_TOKEN or `gh auth login`)
#
# Optional env:
#   - GIT_REPO: owner/repo (default: detect from remote)
#   - CHANGESET_SLUG: ConfigHub changeset to use as MR source
#   - OUT_ROOT: output directory root
#   - SKIP_BUILD: set to 1 to skip cub-gen rebuild

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/examples/demo/lib/connected-preflight.sh"

REPO_PATH="${1:-./examples/helm-paas}"
CHANGESET_SLUG="${2:-${CHANGESET_SLUG:-}}"
EXAMPLE_SLUG="$(basename "$REPO_PATH")"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/flow-b}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID/$EXAMPLE_SLUG"
SPACE="${SPACE:-}"
TARGET_BRANCH="${TARGET_BRANCH:-main}"
PR_TITLE_PREFIX="${PR_TITLE_PREFIX:-[ConfigHub]}"

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

# Resolve GitHub auth
gh_token="$(resolve_gh_token || true)"
if [ -z "$gh_token" ]; then
  echo "error: missing GitHub auth for Flow B demo." >&2
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
  echo "[flow-b] building cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

mkdir -p "$OUT_DIR"

echo "[flow-b] ConfigHub MR → Git PR governed flow"
echo "[flow-b] repo: $GIT_REPO"
echo "[flow-b] example: $EXAMPLE_SLUG"
echo "[flow-b] output: $OUT_DIR"
print_connected_context

# Step 1: Create or retrieve ConfigHub changeset (simulating MR)
echo "[flow-b][step-1] create/retrieve ConfigHub MR changeset"
now="$(date -u +%Y-%m-%dT%H%M%SZ)"

if [ -z "$CHANGESET_SLUG" ]; then
  CHANGESET_SLUG="$(sanitize_slug "flow-b-mr-${EXAMPLE_SLUG}-${RUN_ID}")"
  echo "[flow-b] creating new changeset: $CHANGESET_SLUG"

  # Create a changeset representing a ConfigHub-initiated change
  cub changeset create \
    --space "$CONFIGHUB_SPACE" \
    --json \
    --allow-exists \
    "$CHANGESET_SLUG" \
    --description "Flow B: ConfigHub-initiated change for ${EXAMPLE_SLUG}" \
    --label "flow=B" \
    --label "flow_direction=confighub-mr-to-git-pr" \
    --label "origin=confighub-mr" \
    --label "target_repo=$(echo "$GIT_REPO" | tr '/' '-')" \
    --label "target_branch=${TARGET_BRANCH}" \
    > "$OUT_DIR/mr-changeset.json"
else
  echo "[flow-b] using existing changeset: $CHANGESET_SLUG"
  cub changeset get --space "$CONFIGHUB_SPACE" --json "$CHANGESET_SLUG" > "$OUT_DIR/mr-changeset.json"
fi

changeset_id="$(jq -r '.ChangeSetID // .ChangeSet.ChangeSetID // empty' "$OUT_DIR/mr-changeset.json")"
if [ -z "$changeset_id" ]; then
  echo "error: unable to resolve changeset ID." >&2
  exit 1
fi

# Step 2: Discover current state and generate proposed changes
echo "[flow-b][step-2] discover current state"
./cub-gen gitops discover --space "$SPACE" --json "$REPO_PATH" > "$OUT_DIR/discover.json"
./cub-gen gitops import --space "$SPACE" --json "$REPO_PATH" "$REPO_PATH" > "$OUT_DIR/import.json"

# Step 3: Extract inverse-edit guidance for proposed change
echo "[flow-b][step-3] extract inverse-edit guidance"
jq '.inverse_transform_plans // []' "$OUT_DIR/import.json" > "$OUT_DIR/inverse-guidance.json"

# Get a representative DRY file to edit
dry_file="$(jq -r '[.inverse_transform_plans[]?.patches[]?.dry_file // empty][0] // "values.yaml"' "$OUT_DIR/import.json")"
dry_path="$(jq -r '[.inverse_transform_plans[]?.patches[]?.dry_path // empty][0] // "image.tag"' "$OUT_DIR/import.json")"

# Step 4: Simulate ConfigHub MR approval
echo "[flow-b][step-4] simulate ConfigHub MR approval"
approval_slug="$(sanitize_slug "flow-b-approval-${EXAMPLE_SLUG}-${RUN_ID}")"
cub changeset create \
  --space "$CONFIGHUB_SPACE" \
  --json \
  --allow-exists \
  "$approval_slug" \
  --description "Flow B: MR approval for changeset ${CHANGESET_SLUG}" \
  --label "flow=B" \
  --label "flow_stage=mr-approved" \
  --label "parent_changeset_id=${changeset_id}" \
  --label "approved_by=platform-lead" \
  --label "approved_at=${now}" \
  > "$OUT_DIR/approval-changeset.json"

# Step 5: Generate Git PR content
echo "[flow-b][step-5] generate Git PR proposal"
branch_name="confighub/${CHANGESET_SLUG}"
pr_title="${PR_TITLE_PREFIX} ${EXAMPLE_SLUG}: ConfigHub-initiated change"
pr_body="## ConfigHub MR → Git PR

This PR was generated from ConfigHub MR approval.

### Source
- **ConfigHub Changeset**: \`${CHANGESET_SLUG}\`
- **Changeset ID**: \`${changeset_id}\`
- **Space**: \`${CONFIGHUB_SPACE}\`

### Proposed Changes
Based on inverse-edit guidance:
- **DRY File**: \`${dry_file}\`
- **DRY Path**: \`${dry_path}\`

### Flow
This is **Flow B** (ConfigHub MR → Git PR):
1. ConfigHub detected/initiated a change
2. ConfigHub MR was approved
3. This Git PR was generated for review
4. Platform team reviews and merges

### Linkage
\`\`\`json
$(jq -c '{flow: "B", changeset_slug: "'"$CHANGESET_SLUG"'", changeset_id: "'"$changeset_id"'"}')
\`\`\`

---
Generated by: cub-gen Flow B demo"

jq -n \
  --arg branch "$branch_name" \
  --arg title "$pr_title" \
  --arg body "$pr_body" \
  --arg target_branch "$TARGET_BRANCH" \
  --arg dry_file "$dry_file" \
  --arg dry_path "$dry_path" \
  '{
    branch: $branch,
    title: $title,
    body: $body,
    target_branch: $target_branch,
    proposed_edit: {
      dry_file: $dry_file,
      dry_path: $dry_path
    }
  }' > "$OUT_DIR/pr-proposal.json"

# Step 6: Create PR-MR linkage record
echo "[flow-b][step-6] create PR-MR linkage record"
linkage_slug="$(sanitize_slug "flow-b-linkage-${EXAMPLE_SLUG}-${RUN_ID}")"

jq -n \
  --arg schema "cub.confighub.io/pr-mr-promotion-flow/v1" \
  --arg changeset_id "$changeset_id" \
  --arg changeset_slug "$CHANGESET_SLUG" \
  --arg git_repo "$GIT_REPO" \
  --arg git_branch "$branch_name" \
  --arg git_target_branch "$TARGET_BRANCH" \
  --arg flow "B" \
  --arg flow_direction "confighub-mr-to-git-pr" \
  --arg status "PR_PROPOSED" \
  --arg updated_at "$now" \
  '{
    schema: $schema,
    flow: $flow,
    flow_direction: $flow_direction,
    confighub_mr: {
      changeset_id: $changeset_id,
      changeset_slug: $changeset_slug,
      status: "APPROVED"
    },
    git_pr: {
      repo: $git_repo,
      branch: $git_branch,
      target_branch: $git_target_branch,
      status: "PROPOSED",
      number: null,
      url: null
    },
    status: $status,
    updated_at: $updated_at
  }' > "$OUT_DIR/pr-mr-linkage.json"

# Record linkage in ConfigHub
cub changeset create \
  --space "$CONFIGHUB_SPACE" \
  --json \
  --allow-exists \
  "$linkage_slug" \
  --description "Flow B: PR-MR linkage for ${CHANGESET_SLUG} → Git PR" \
  --label "flow=B" \
  --label "flow_stage=pr-proposed" \
  --label "confighub_changeset_id=${changeset_id}" \
  --label "git_repo=$(echo "$GIT_REPO" | tr '/' '-')" \
  --label "git_branch=${branch_name}" \
  > "$OUT_DIR/linkage-changeset.json"

# Step 7: Instructions for creating actual Git PR
echo ""
echo "[flow-b][step-7] Git PR creation instructions"
echo ""
echo "To complete Flow B, create the Git PR:"
echo ""
echo "  1. Create branch: git checkout -b ${branch_name}"
echo "  2. Apply changes from inverse-edit guidance"
echo "  3. Commit with linkage metadata:"
echo "     git commit -m \"${pr_title}"
echo ""
echo "     ConfigHub-Changeset-ID: ${changeset_id}\""
echo "  4. Push and create PR:"
echo "     git push -u origin ${branch_name}"
echo "     gh pr create --title \"${pr_title}\" --body-file ${OUT_DIR}/pr-proposal.json"
echo ""

# Summary
echo "[flow-b] summary"
jq -n \
  --arg flow "B" \
  --arg direction "ConfigHub MR → Git PR" \
  --arg changeset_slug "$CHANGESET_SLUG" \
  --arg changeset_id "$changeset_id" \
  --arg git_repo "$GIT_REPO" \
  --arg git_branch "$branch_name" \
  --arg target_branch "$TARGET_BRANCH" \
  --arg dry_file "$dry_file" \
  --arg dry_path "$dry_path" \
  --arg status "PR_PROPOSED" \
  '{
    flow: $flow,
    direction: $direction,
    confighub: {
      changeset_slug: $changeset_slug,
      changeset_id: $changeset_id,
      status: "APPROVED"
    },
    git: {
      repo: $git_repo,
      branch: $git_branch,
      target_branch: $target_branch,
      status: "PROPOSED"
    },
    proposed_edit: {
      dry_file: $dry_file,
      dry_path: $dry_path
    },
    status: $status
  }' | tee "$OUT_DIR/flow-b-summary.json"
