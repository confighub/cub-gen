#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-.}"

if [[ ! -d "$ROOT" ]]; then
  echo "error: root path not found: $ROOT" >&2
  exit 2
fi

failures=0
checked=0
files=()

add_markdown_from_dir() {
  local d="$1"
  [[ -d "$d" ]] || return 0
  while IFS= read -r -d '' f; do
    files+=("$f")
  done < <(find "$d" -type f -name '*.md' -print0)
}

# Core, normative scope for term enforcement.
add_markdown_from_dir "$ROOT/00-index"
add_markdown_from_dir "$ROOT/01-vision"
add_markdown_from_dir "$ROOT/02-design"
add_markdown_from_dir "$ROOT/03-worked-examples"
add_markdown_from_dir "$ROOT/04-schemas"

[[ -f "$ROOT/README.md" ]] && files+=("$ROOT/README.md")
[[ -f "$ROOT/agentic-gitops-design.md" ]] && files+=("$ROOT/agentic-gitops-design.md")
[[ -f "$ROOT/gitops-checkpoint-schemas.md" ]] && files+=("$ROOT/gitops-checkpoint-schemas.md")
[[ -f "$ROOT/05-rollout/90-today-demo-plan.md" ]] && files+=("$ROOT/05-rollout/90-today-demo-plan.md")
[[ -f "$ROOT/05-rollout/91-speaker-sheet-today.md" ]] && files+=("$ROOT/05-rollout/91-speaker-sheet-today.md")
[[ -f "$ROOT/05-rollout/92-two-minute-modules.md" ]] && files+=("$ROOT/05-rollout/92-two-minute-modules.md")
[[ -f "$ROOT/05-rollout/93-live-run-checklist.md" ]] && files+=("$ROOT/05-rollout/93-live-run-checklist.md")

if (( ${#files[@]} == 0 )); then
  echo "error: no core markdown files found under $ROOT" >&2
  exit 2
fi

for file in "${files[@]}"; do
  if ! grep -Eq 'Agentic GitOps' "$file"; then
    continue
  fi

  checked=$((checked + 1))
  missing=()

  if ! grep -Eq 'WET *-> *LIVE|inner loop' "$file"; then
    missing+=("inner-loop rule (WET -> LIVE or inner loop)")
  fi

  if ! grep -Eq 'Flux/Argo|reconciler' "$file"; then
    missing+=("reconciler reference (Flux/Argo or reconciler)")
  fi

  if ! grep -Eq 'governed config automation' "$file"; then
    missing+=("reclassification term (governed config automation)")
  fi

  if (( ${#missing[@]} > 0 )); then
    failures=$((failures + 1))
    echo "[FAIL] $file"
    for item in "${missing[@]}"; do
      echo "  - missing: $item"
    done
  fi
done

if (( checked == 0 )); then
  echo "[WARN] no markdown files with 'Agentic GitOps' found in core scope under $ROOT"
fi

if (( failures > 0 )); then
  echo "lint failed: $failures file(s) violate Agentic GitOps naming rule"
  exit 1
fi

echo "lint passed: checked $checked file(s) with Agentic GitOps references"
