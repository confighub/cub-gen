#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/ops-workflow-governed-policy}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
WORK_REPO="$OUT_DIR/repo"
ALLOW_SCHEDULE="${ALLOW_SCHEDULE:-0 4 * * *}"
BLOCK_ACTION="${BLOCK_ACTION:-destroy}"

mkdir -p "$OUT_DIR"
rm -rf "$WORK_REPO"

echo "[ops-governed] cloning repo into $WORK_REPO"
git clone --quiet "$ROOT_DIR" "$WORK_REPO"

cd "$WORK_REPO"
git config user.name "cub-gen demo"
git config user.email "cub-gen-demo@example.invalid"

echo "[ops-governed] build cub-gen once for both workflow policy checks"
go build -o ./cub-gen ./cmd/cub-gen

BASE_SHA="$(git rev-parse HEAD)"
POLICY_PATH="./examples/ops-workflow/platform/execution-policy.yaml"
WORKFLOW_PATH="./examples/ops-workflow/operations-prod.yaml"

echo "[ops-governed] proof 1/2: schedule change stays within the documented production window"
git checkout --quiet -b demo-allow "$BASE_SHA"
perl -0pi -e 's/schedule: "0 3 \* \* \*"/schedule: "'"$ALLOW_SCHEDULE"'"/' "$WORKFLOW_PATH"
git add "$WORKFLOW_PATH"
git commit --quiet -m "demo: move maintenance window inside the documented production window"

./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow > "$OUT_DIR/allow-import.json"

allow_schedule="$(sed -n 's/.*schedule: "\(.*\)"/\1/p' "$WORKFLOW_PATH" | head -n1)"
production_window="$(sed -n '/production:/,/reason:/ s/.*window: "\(.*\)"/\1/p' "$POLICY_PATH" | head -n1)"
set -- $allow_schedule
allow_hour="${2:-}"
if [ -z "$allow_hour" ]; then
  echo "error: could not parse allowed schedule hour from $allow_schedule" >&2
  exit 1
fi
if [ "$allow_hour" -lt 0 ] || [ "$allow_hour" -gt 5 ]; then
  echo "error: expected allowed schedule hour to stay inside 00:00-06:00 UTC, got $allow_schedule" >&2
  exit 1
fi
if ! jq -e '.provenance[0].ops_workflow_analysis.blocked_actions_used | length == 0' "$OUT_DIR/allow-import.json" >/dev/null; then
  echo "error: expected no blocked actions in the allowed workflow change." >&2
  exit 1
fi

jq '{
  schedule_inverse: (.provenance[0].inverse_edit_pointers[] | select(.dry_path == "triggers.schedule")),
  policy: .provenance[0].ops_workflow_analysis | {
    allowed_actions,
    blocked_actions,
    approval_gates,
    schedule_overrides,
    blocked_actions_used
  }
}' "$OUT_DIR/allow-import.json" > "$OUT_DIR/allow-summary.json"

echo "[ops-governed][allow] policy summary"
jq --arg production_window "$production_window" --arg allow_schedule "$allow_schedule" '{
  production_window: $production_window,
  candidate_schedule: $allow_schedule,
  schedule_inverse: .schedule_inverse,
  policy: .policy
}' "$OUT_DIR/allow-summary.json"

echo "[ops-governed] proof 2/2: blocked action is surfaced immediately from the policy metadata"
git checkout --quiet -B demo-block "$BASE_SHA"
perl -0pi -e 's/(actions:\n  deploy:\n    image_tag: v1\.2\.3-prod\n)/$1  destroy:\n    service: checkout-api\n/' "$WORKFLOW_PATH"
git add "$WORKFLOW_PATH"
git commit --quiet -m "demo: introduce blocked destroy action"

./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow > "$OUT_DIR/block-import.json"

if ! jq -e --arg blocked "$BLOCK_ACTION" '.provenance[0].ops_workflow_analysis.blocked_actions_used | index($blocked) != null' "$OUT_DIR/block-import.json" >/dev/null; then
  echo "error: expected blocked action $BLOCK_ACTION to appear in blocked_actions_used." >&2
  exit 1
fi

jq '{
  blocked_actions_used: .provenance[0].ops_workflow_analysis.blocked_actions_used,
  blocked_actions: .provenance[0].ops_workflow_analysis.blocked_actions,
  approval_gates: .provenance[0].ops_workflow_analysis.approval_gates
}' "$OUT_DIR/block-import.json" > "$OUT_DIR/block-summary.json"

echo "[ops-governed][block] policy summary"
cat "$OUT_DIR/block-summary.json"

cat <<EOF

[ops-governed] success
  allowed path: schedule moves to $allow_schedule and stays inside the documented production window ($production_window)
  blocked path: blocked action '$BLOCK_ACTION' is surfaced in blocked_actions_used
  artifacts: $OUT_DIR
EOF
