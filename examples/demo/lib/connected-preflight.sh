#!/usr/bin/env bash
set -euo pipefail

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    return 1
  fi
}

resolve_confighub_base_url() {
  if [ -n "${CONFIGHUB_BASE_URL:-}" ]; then
    printf '%s' "$CONFIGHUB_BASE_URL"
    return 0
  fi
  cub context get --json 2>/dev/null | jq -r '.coordinate.serverURL // empty'
}

resolve_confighub_space() {
  if [ -n "${CONFIGHUB_SPACE:-}" ]; then
    printf '%s' "$CONFIGHUB_SPACE"
    return 0
  fi
  cub context get --json 2>/dev/null | jq -r '.settings.defaultSpace // empty'
}

require_connected_preflight() {
  require_cmd cub
  require_cmd jq

  local token
  token="$(printf '%s' "${CONFIGHUB_TOKEN:-}" | tr -d '\r\n')"
  if [ -z "$token" ]; then
    if ! token="$(cub auth get-token 2>/dev/null | tr -d '\r\n')" || [ -z "$token" ]; then
      echo "error: ConfigHub authentication missing." >&2
      echo "remediation: run 'cub auth login' (interactive) or export CONFIGHUB_TOKEN (CI)." >&2
      return 1
    fi
  fi

  local base_url
  base_url="$(resolve_confighub_base_url)"
  if [ -z "$base_url" ]; then
    echo "error: unable to resolve ConfigHub base URL." >&2
    echo "remediation: set CONFIGHUB_BASE_URL or configure a cub context with coordinate.serverURL." >&2
    return 1
  fi

  local space
  space="$(resolve_confighub_space)"
  if [ -z "$space" ]; then
    echo "error: unable to resolve ConfigHub space." >&2
    echo "remediation: set CONFIGHUB_SPACE or run 'cub context set --space <space-name>'." >&2
    return 1
  fi

  if ! cub info >/dev/null 2>&1; then
    require_cmd curl
    if ! curl -fsS --max-time 5 "$base_url" >/dev/null 2>&1; then
      echo "error: unable to reach ConfigHub server at $base_url." >&2
      echo "remediation: verify CONFIGHUB_BASE_URL / context server URL and credentials." >&2
      return 1
    fi
  fi

  export CONFIGHUB_TOKEN="$token"
  export CONFIGHUB_BASE_URL="$base_url"
  export CONFIGHUB_SPACE="$space"
}

print_connected_context() {
  local user
  user="$( (cub context get --json 2>/dev/null || true) | jq -r '.coordinate.user // empty' 2>/dev/null || true )"
  if [ -z "$user" ]; then
    user="${CONFIGHUB_USER:-token-auth}"
  fi
  echo "[connected] user: $user"
  echo "[connected] base_url: $CONFIGHUB_BASE_URL"
  echo "[connected] space: $CONFIGHUB_SPACE"
  echo "[connected] token: present"
}
