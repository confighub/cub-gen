#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/swamp-governed-structure}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
WORK_REPO="$OUT_DIR/repo"
ALLOW_MODEL="${ALLOW_MODEL:-app-healthcheck}"
ALLOW_METHOD="${ALLOW_METHOD:-verify}"
ALLOW_STEP_NAME="${ALLOW_STEP_NAME:-healthcheck}"
BLOCK_MODEL="${BLOCK_MODEL:-untrusted-model}"
BLOCK_METHOD="${BLOCK_METHOD:-do_something}"
BLOCK_STEP_NAME="${BLOCK_STEP_NAME:-risky-step}"
REQUIRED_STEP="${REQUIRED_STEP:-validate}"

mkdir -p "$OUT_DIR"
rm -rf "$WORK_REPO"

echo "[swamp-governed] cloning repo into $WORK_REPO"
git clone --quiet "$ROOT_DIR" "$WORK_REPO"

cd "$WORK_REPO"
git config user.name "cub-gen demo"
git config user.email "cub-gen-demo@example.invalid"

echo "[swamp-governed] build cub-gen once for both workflow policy checks"
go build -o ./cub-gen ./cmd/cub-gen

BASE_SHA="$(git rev-parse HEAD)"
WORKFLOW_PATH="./examples/swamp-automation/workflow-deploy.yaml"

echo "[swamp-governed] proof 1/2: approved model-method addition stays inside policy"
git checkout --quiet -b demo-allow "$BASE_SHA"
perl -0pi -e 's/(      - name: apply\n)/      - name: '"$ALLOW_STEP_NAME"'\n        task:\n          type: model_method\n          modelIdOrName: '"$ALLOW_MODEL"'\n          methodName: '"$ALLOW_METHOD"'\n$1/' "$WORKFLOW_PATH"
git add "$WORKFLOW_PATH"
git commit --quiet -m "demo: add approved healthcheck step"

./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation > "$OUT_DIR/allow-import.json"

allow_model_method="$ALLOW_MODEL.$ALLOW_METHOD"
if ! jq -e \
  --arg step "$ALLOW_STEP_NAME" \
  --arg model_method "$allow_model_method" \
  '.provenance[0].swamp_workflow_analysis
  | (.missing_required_steps | length == 0)
    and (.unapproved_models | length == 0)
    and (.unapproved_model_methods | length == 0)
    and (.step_names | index($step) != null)
    and (.model_method_refs | index($model_method) != null)' \
  "$OUT_DIR/allow-import.json" >/dev/null; then
  echo "error: expected approved workflow addition to stay inside policy." >&2
  exit 1
fi

jq '{
  decision_state: "ALLOW",
  inverse_hint: (.provenance[0].inverse_edit_pointers[] | select(.dry_path == "jobs[].steps[].task.modelIdOrName")),
  policy: (.provenance[0].swamp_workflow_analysis | {
    required_steps,
    missing_required_steps,
    approved_models,
    approved_model_methods,
    step_names,
    model_method_refs,
    unapproved_models,
    unapproved_model_methods
  })
}' "$OUT_DIR/allow-import.json" > "$OUT_DIR/allow-summary.json"

echo "[swamp-governed][allow] policy summary"
cat "$OUT_DIR/allow-summary.json"

echo "[swamp-governed] proof 2/2: unapproved model and missing required step are surfaced immediately"
git checkout --quiet -B demo-block "$BASE_SHA"
perl -0pi -e 's/- name: validate\n        task:\n          type: model_method\n          modelIdOrName: app-validator\n          methodName: check\n        retries: 2/- name: '"$BLOCK_STEP_NAME"'\n        task:\n          type: model_method\n          modelIdOrName: '"$BLOCK_MODEL"'\n          methodName: '"$BLOCK_METHOD"'\n        retries: 2/' "$WORKFLOW_PATH"
git add "$WORKFLOW_PATH"
git commit --quiet -m "demo: add unapproved workflow step and remove validate"

./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation > "$OUT_DIR/block-import.json"

block_model_method="$BLOCK_MODEL.$BLOCK_METHOD"
if ! jq -e \
  --arg required_step "$REQUIRED_STEP" \
  --arg model "$BLOCK_MODEL" \
  --arg model_method "$block_model_method" \
  '.provenance[0].swamp_workflow_analysis
  | (.missing_required_steps | index($required_step) != null)
    and (.unapproved_models | index($model) != null)
    and (.unapproved_model_methods | index($model_method) != null)' \
  "$OUT_DIR/block-import.json" >/dev/null; then
  echo "error: expected blocked workflow mutation to surface missing required step and unapproved model metadata." >&2
  exit 1
fi

jq '{
  decision_state: "BLOCK",
  policy: (.provenance[0].swamp_workflow_analysis | {
    required_steps,
    missing_required_steps,
    forbidden_step_names,
    forbidden_steps_present,
    step_names,
    model_method_refs,
    unapproved_models,
    unapproved_model_methods
  })
}' "$OUT_DIR/block-import.json" > "$OUT_DIR/block-summary.json"

echo "[swamp-governed][block] policy summary"
cat "$OUT_DIR/block-summary.json"

cat <<EOF

[swamp-governed] success
  allowed path: approved model-method '$allow_model_method' is added while required step '$REQUIRED_STEP' stays present
  blocked path: missing required step '$REQUIRED_STEP' and unapproved model-method '$block_model_method' are surfaced together
  artifacts: $OUT_DIR
EOF
