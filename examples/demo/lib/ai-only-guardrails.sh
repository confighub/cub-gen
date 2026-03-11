#!/usr/bin/env bash
set -euo pipefail

# Guardrails for AI-only pilot lanes.
# These checks intentionally run before any render/import work.

ai_only_normalize_csv() {
  local csv="$1"
  printf '%s\n' "$csv" | tr '[:upper:]' '[:lower:]' | tr ',' '\n' | sed -E 's/^[[:space:]]+|[[:space:]]+$//g' | sed '/^$/d'
}

ai_only_is_allowed_repo() {
  local repo_basename="$1"
  local allowed_csv="$2"
  local entry
  while IFS= read -r entry; do
    if [ "$entry" = "$repo_basename" ]; then
      return 0
    fi
  done < <(ai_only_normalize_csv "$allowed_csv")
  return 1
}

enforce_ai_only_scope() {
  local repo_path="$1"
  local render_target="${2:-$repo_path}"

  local allowed_repos
  allowed_repos="${AI_ONLY_ALLOWED_REPOS:-swamp-automation,ops-workflow}"

  local repo_abs target_abs repo_name
  repo_abs="$(cd "$repo_path" && pwd)"
  target_abs="$(cd "$render_target" && pwd)"
  repo_name="$(basename "$repo_abs" | tr '[:upper:]' '[:lower:]')"

  if ! ai_only_is_allowed_repo "$repo_name" "$allowed_repos"; then
    echo "error: $repo_name is outside allowed AI-only scope." >&2
    echo "allowed AI-only repos: $allowed_repos" >&2
    echo "remediation: use swamp-automation/ops-workflow for AI-only pilot lanes, or run non-AI-only flows." >&2
    return 1
  fi

  # AI-only pilot lanes must stay within one repo/render boundary.
  if [ "$target_abs" != "$repo_abs" ]; then
    echo "error: AI-only scope requires render target to match repo path." >&2
    echo "repo: $repo_abs" >&2
    echo "render_target: $target_abs" >&2
    return 1
  fi

  local hard_deny_regex
  hard_deny_regex="${AI_ONLY_HARD_DENY_REGEX:-cluster-admin|system:masters|\bdeleteEverything\b|\bdelete\s+namespace\b}"
  if rg -n -i "$hard_deny_regex" "$repo_abs" -g '*.yaml' -g '*.yml' >/dev/null 2>&1; then
    echo "error: AI-only scope hard deny rule matched in $repo_name." >&2
    echo "matched regex: $hard_deny_regex" >&2
    return 1
  fi

  local require_rollback
  require_rollback="${AI_ONLY_REQUIRE_ROLLBACK_HOOK:-1}"
  if [ "$require_rollback" = "1" ]; then
    if ! rg -n -i "rollback|revert" "$repo_abs" -g '*.yaml' -g '*.yml' >/dev/null 2>&1; then
      echo "error: AI-only scope requires at least one rollback/revert hook in workflow files." >&2
      echo "remediation: add an explicit rollback/revert step before using AI-only lane." >&2
      return 1
    fi
  fi

  echo "[ai-only] scope check passed for repo=$repo_name (allowed=$allowed_repos, rollback_hook_required=$require_rollback)"
}
