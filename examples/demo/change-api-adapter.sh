#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'USAGE'
Usage:
  ./examples/demo/change-api-adapter.sh --request <request.json> [--out <response.json>]

Request shape (see docs/contracts/change-api-v1.md):
  {
    "action": "preview" | "run" | "explain",
    "mode": "local" | "connected",          # required for run
    "input": {
      "target_slug": "./examples/scoredev-paas",
      "render_target_slug": "./examples/scoredev-paas",
      "space": "platform",
      "ref": "HEAD",
      "where_resource": "..."
    },
    "connected": {
      "base_url": "https://confighub.example",
      "token": "...",
      "ingest_endpoint": "/api/v1/...",
      "decision_endpoint": "/api/v1/..."
    },
    "filters": {
      "wet_path": "...",
      "dry_path": "...",
      "owner": "..."
    }
  }
USAGE
}

REQUEST=""
OUT="-"

while [ $# -gt 0 ]; do
  case "$1" in
    --request)
      REQUEST="${2:-}"
      shift 2
      ;;
    --out)
      OUT="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown arg: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ -z "$REQUEST" ] || [ ! -f "$REQUEST" ]; then
  echo "error: --request <file> is required" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required" >&2
  exit 1
fi

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  go build -o ./cub-gen ./cmd/cub-gen
fi

action="$(jq -r '.action // empty' "$REQUEST")"
target_slug="$(jq -r '.input.target_slug // empty' "$REQUEST")"
render_target_slug="$(jq -r '.input.render_target_slug // empty' "$REQUEST")"
space="$(jq -r '.input.space // "platform"' "$REQUEST")"
ref="$(jq -r '.input.ref // "HEAD"' "$REQUEST")"
where_resource="$(jq -r '.input.where_resource // empty' "$REQUEST")"

if [ -z "$action" ] || [ -z "$target_slug" ] || [ -z "$render_target_slug" ]; then
  echo "error: request must include action + input.target_slug + input.render_target_slug" >&2
  exit 1
fi

cmd=(./cub-gen change)

case "$action" in
  preview)
    cmd+=(preview --space "$space" --ref "$ref")
    if [ -n "$where_resource" ]; then
      cmd+=(--where-resource "$where_resource")
    fi
    cmd+=("$target_slug" "$render_target_slug")
    ;;
  run)
    mode="$(jq -r '.mode // empty' "$REQUEST")"
    if [ -z "$mode" ]; then
      echo "error: action=run requires mode" >&2
      exit 1
    fi
    cmd+=(run --mode "$mode" --space "$space" --ref "$ref")
    if [ -n "$where_resource" ]; then
      cmd+=(--where-resource "$where_resource")
    fi
    if [ "$mode" = "connected" ]; then
      base_url="$(jq -r '.connected.base_url // env.CONFIGHUB_BASE_URL // empty' "$REQUEST")"
      token="$(jq -r '.connected.token // env.CONFIGHUB_TOKEN // empty' "$REQUEST")"
      ingest_endpoint="$(jq -r '.connected.ingest_endpoint // empty' "$REQUEST")"
      decision_endpoint="$(jq -r '.connected.decision_endpoint // empty' "$REQUEST")"
      if [ -n "$base_url" ]; then
        cmd+=(--base-url "$base_url")
      fi
      if [ -n "$token" ]; then
        cmd+=(--token "$token")
      fi
      if [ -n "$ingest_endpoint" ]; then
        cmd+=(--ingest-endpoint "$ingest_endpoint")
      fi
      if [ -n "$decision_endpoint" ]; then
        cmd+=(--decision-endpoint "$decision_endpoint")
      fi
    fi
    cmd+=("$target_slug" "$render_target_slug")
    ;;
  explain)
    wet_path="$(jq -r '.filters.wet_path // empty' "$REQUEST")"
    dry_path="$(jq -r '.filters.dry_path // empty' "$REQUEST")"
    owner="$(jq -r '.filters.owner // empty' "$REQUEST")"
    cmd+=(explain --space "$space" --ref "$ref")
    if [ -n "$where_resource" ]; then
      cmd+=(--where-resource "$where_resource")
    fi
    if [ -n "$wet_path" ]; then
      cmd+=(--wet-path "$wet_path")
    fi
    if [ -n "$dry_path" ]; then
      cmd+=(--dry-path "$dry_path")
    fi
    if [ -n "$owner" ]; then
      cmd+=(--owner "$owner")
    fi
    cmd+=("$target_slug" "$render_target_slug")
    ;;
  *)
    echo "error: unsupported action: $action" >&2
    exit 1
    ;;
esac

if [ "$OUT" = "-" ]; then
  "${cmd[@]}"
else
  mkdir -p "$(dirname "$OUT")"
  "${cmd[@]}" > "$OUT"
  echo "[change-api-adapter] wrote response: $OUT"
fi
