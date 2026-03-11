#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'USAGE'
Usage:
  ./test/checks/pr-dry-ownership-gate.sh <repo-path> <base-ref> <head-ref> [actor-role] [--report-json <path>]

Example:
  ./test/checks/pr-dry-ownership-gate.sh ./examples/helm-paas origin/main HEAD app-team
  ./test/checks/pr-dry-ownership-gate.sh ./examples/helm-paas origin/main HEAD app-team --report-json .tmp/pr-gate/report.json

Behavior:
  - Loads DRY inputs via `cub-gen gitops import`.
  - Scans changed YAML/JSON files in <repo-path> between refs.
  - Fails if a changed file is not a recognized DRY input.
  - Optionally enforces owner role match for DRY files unless actor-role is `any`.
  - Optionally writes a JSON report with failures + inverse-edit suggestions.
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

REPO_PATH="${1:-}"
BASE_REF="${2:-}"
HEAD_REF="${3:-}"
ACTOR_ROLE="${4:-any}"
SPACE="${SPACE:-platform}"
REPORT_JSON="${REPORT_JSON:-}"

if [ "${5:-}" = "--report-json" ]; then
  REPORT_JSON="${6:-}"
elif [ -n "${5:-}" ]; then
  echo "error: unsupported argument: ${5:-}" >&2
  usage >&2
  exit 1
fi

if [ "${5:-}" = "--report-json" ] && [ -z "$REPORT_JSON" ]; then
  echo "error: --report-json requires a path" >&2
  usage >&2
  exit 1
fi

if [ -z "$REPO_PATH" ] || [ -z "$BASE_REF" ] || [ -z "$HEAD_REF" ]; then
  usage >&2
  exit 1
fi

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi

if ! git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  echo "error: base ref not found: $BASE_REF" >&2
  exit 1
fi
if ! git rev-parse --verify "$HEAD_REF" >/dev/null 2>&1; then
  echo "error: head ref not found: $HEAD_REF" >&2
  exit 1
fi

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  go build -o ./cub-gen ./cmd/cub-gen
fi

repo_norm="${REPO_PATH#./}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

./cub-gen gitops import --space "$SPACE" --json "$REPO_PATH" "$REPO_PATH" > "$tmpdir/import.json"

jq -r '.dry_inputs[]? | "\(.path)\t\(.owner)"' "$tmpdir/import.json" > "$tmpdir/dry-inputs.tsv"
jq '[.dry_inputs[]? | {path, owner, role}]' "$tmpdir/import.json" > "$tmpdir/dry-inputs.json"
jq '
  (.dry_inputs // []) as $inputs
  |
  [.provenance[]?.inverse_edit_pointers[]? as $ptr
   | {
       wet_path: $ptr.wet_path,
       dry_path: $ptr.dry_path,
       dry_files: ($inputs | map(select(.owner == $ptr.owner) | .path) | unique),
       owner: $ptr.owner,
       confidence: $ptr.confidence,
       edit_hint: $ptr.edit_hint
     }]
  | sort_by(-(.confidence // 0))
  | .[:10]
' "$tmpdir/import.json" > "$tmpdir/inverse-pointers.json"

write_report() {
  local status="$1"

  : > "$tmpdir/checked-files.txt"
  for file in "${checked_files[@]:-}"; do
    [ -n "$file" ] || continue
    printf '%s\n' "$file" >> "$tmpdir/checked-files.txt"
  done

  : > "$tmpdir/failures.txt"
  for failure in "${failures[@]:-}"; do
    [ -n "$failure" ] || continue
    printf '%s\n' "$failure" >> "$tmpdir/failures.txt"
  done

  jq -Rs 'split("\n") | map(select(length > 0))' "$tmpdir/checked-files.txt" > "$tmpdir/checked-files.json"
  jq -Rs 'split("\n") | map(select(length > 0))' "$tmpdir/failures.txt" > "$tmpdir/failures.json"

  if [ -n "$REPORT_JSON" ]; then
    mkdir -p "$(dirname "$REPORT_JSON")"
    jq -n \
      --arg status "$status" \
      --arg repo_path "$repo_norm" \
      --arg base_ref "$BASE_REF" \
      --arg head_ref "$HEAD_REF" \
      --arg actor_role "$ACTOR_ROLE" \
      --arg space "$SPACE" \
      --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
      --argjson changed_files "$(cat "$tmpdir/checked-files.json")" \
      --argjson failures "$(cat "$tmpdir/failures.json")" \
      --argjson dry_inputs "$(cat "$tmpdir/dry-inputs.json")" \
      --argjson suggestions "$(cat "$tmpdir/inverse-pointers.json")" \
      '{
        status: $status,
        repo_path: $repo_path,
        base_ref: $base_ref,
        head_ref: $head_ref,
        actor_role: $actor_role,
        space: $space,
        generated_at: $generated_at,
        changed_files: $changed_files,
        failures: $failures,
        dry_inputs: $dry_inputs,
        suggestions: $suggestions
      }' > "$REPORT_JSON"
  fi
}

declare -a changed_files=()
while IFS= read -r changed; do
  changed_files+=("$changed")
done < <(git diff --name-only "$BASE_REF" "$HEAD_REF" -- "$repo_norm")

declare -a checked_files=()
declare -a failures=()

if [ "${#changed_files[@]}" -eq 0 ]; then
  write_report "pass"
  echo "ok: no changed files under $repo_norm"
  exit 0
fi

for changed in "${changed_files[@]}"; do
  case "$changed" in
    *.yaml|*.yml|*.json) ;;
    *) continue ;;
  esac
  checked_files+=("$changed")

  rel="$changed"
  if [[ "$rel" == "$repo_norm/"* ]]; then
    rel="${rel#"$repo_norm/"}"
  fi

  dry_line="$(awk -F'\t' -v path="$rel" '$1 == path {print $0}' "$tmpdir/dry-inputs.tsv" || true)"
  if [ -z "$dry_line" ]; then
    failures+=("$changed: changed file is not a recognized DRY input for this generator")
    continue
  fi

  owner="$(printf '%s' "$dry_line" | cut -f2)"
  if [ "$ACTOR_ROLE" != "any" ] && [ "$owner" != "$ACTOR_ROLE" ]; then
    failures+=("$changed: owner mismatch (file owner=$owner, actor role=$ACTOR_ROLE)")
  fi
done

if [ "${#failures[@]}" -gt 0 ]; then
  write_report "fail"
  echo "error: PR dry-ownership gate failed for $repo_norm" >&2
  for failure in "${failures[@]}"; do
    echo "  - $failure" >&2
  done

  top_hint="$(jq -r '
    [.provenance[]?.inverse_edit_pointers[]?]
    | sort_by(-(.confidence // 0))
    | .[0].edit_hint // "Use `cub-gen change explain` to locate correct DRY edit path."
  ' "$tmpdir/import.json")"
  echo "hint: $top_hint" >&2
  exit 2
fi

write_report "pass"
echo "ok: changed YAML/JSON files are DRY inputs and pass ownership role check"
