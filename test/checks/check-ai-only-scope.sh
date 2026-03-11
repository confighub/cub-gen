#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

DOC="docs/workflows/ai-only-guardrails.md"
if [ ! -f "$DOC" ]; then
  echo "error: missing AI-only guardrails doc: $DOC" >&2
  exit 1
fi

require_doc_text() {
  local pattern="$1"
  local label="$2"
  if ! rg -q "$pattern" "$DOC"; then
    echo "error: AI-only guardrails doc missing $label" >&2
    exit 1
  fi
}

require_doc_text "Allowed Scope Matrix" "Allowed Scope Matrix section"
require_doc_text "Hard Deny List" "Hard Deny List section"
require_doc_text "Mandatory Rollback Hooks" "Mandatory Rollback Hooks section"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

# Positive lane: default swamp workflow should pass AI-only guardrails.
SKIP_BUILD=1 ./examples/demo/prompt-as-dry-local.sh ./examples/swamp-automation >"$TMP_DIR/allowed.log" 2>&1

# Negative lane: out-of-scope repo must fail.
if SKIP_BUILD=1 ./examples/demo/prompt-as-dry-local.sh ./examples/helm-paas >"$TMP_DIR/outscope.log" 2>"$TMP_DIR/outscope.err"; then
  echo "error: expected AI-only out-of-scope check to fail for helm-paas" >&2
  exit 1
fi
if ! rg -q "outside allowed AI-only scope" "$TMP_DIR/outscope.err"; then
  echo "error: out-of-scope failure did not report expected reason" >&2
  cat "$TMP_DIR/outscope.err" >&2
  exit 1
fi

# Negative lane: even allowed repo names must fail without rollback/revert hook.
NO_ROLLBACK_REPO="$TMP_DIR/no-rollback-workflow"
mkdir -p "$NO_ROLLBACK_REPO"
cat > "$NO_ROLLBACK_REPO/workflow-deploy.yaml" <<'YAML'
jobs:
  - name: deploy
    steps:
      - task: app-deployer.apply
YAML

if AI_ONLY_ALLOWED_REPOS="no-rollback-workflow" SKIP_BUILD=1 ./examples/demo/prompt-as-dry-local.sh "$NO_ROLLBACK_REPO" >"$TMP_DIR/no-rollback.log" 2>"$TMP_DIR/no-rollback.err"; then
  echo "error: expected missing rollback hook check to fail" >&2
  exit 1
fi
if ! rg -q "requires at least one rollback/revert hook" "$TMP_DIR/no-rollback.err"; then
  echo "error: rollback guard failure did not report expected reason" >&2
  cat "$TMP_DIR/no-rollback.err" >&2
  exit 1
fi

echo "ok: AI-only guardrails enforce allowed scope and rollback hooks"
