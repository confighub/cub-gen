#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

WAVE_ID="${WAVE_ID:-wave-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/waves/$WAVE_ID}"
WAVE_FAIL_ON_BLOCK="${WAVE_FAIL_ON_BLOCK:-0}"

# Default wave set focuses on common platform-app patterns.
EXAMPLES=(
  "helm-paas"
  "scoredev-paas"
  "springboot-paas"
  "c3agent"
  "confighub-actions"
)

mkdir -p "$OUT_ROOT"

echo "[story-9] governed multi-repo wave"
echo "[story-9] wave id: $WAVE_ID"
echo "[story-9] output root: $OUT_ROOT"

summaries=()
blocked=0

for example in "${EXAMPLES[@]}"; do
  repo="./examples/$example"
  out_dir="$OUT_ROOT/$example"
  mkdir -p "$out_dir"

  decision_state="ALLOW"
  policy_ref="policy/wave/default-allow"

  case "$example" in
    c3agent)
      decision_state="ESCALATE"
      policy_ref="policy/wave/ai-review-required"
      ;;
    confighub-actions)
      decision_state="BLOCK"
      policy_ref="policy/wave/recursive-governance-hold"
      ;;
  esac

  echo "[story-9] $example decision=$decision_state"

  DECISION_STATE="$decision_state" \
  DECISION_POLICY_REF="$policy_ref" \
  DECISION_APPROVED_BY="" \
  DECISION_REASON_PREFIX="wave:$WAVE_ID" \
  VERIFIER="wave-bot" \
  ./examples/demo/simulate-confighub-lifecycle-connected.sh "$repo" "$repo" "$example" "$out_dir"

  summary_file="$out_dir/story-9-summary.json"
  jq -n \
    --arg wave_id "$WAVE_ID" \
    --arg repo "$example" \
    --arg change_id "$(jq -r .change_id "$out_dir/update/bundle.json")" \
    --arg bundle_digest "$(jq -r .bundle_digest "$out_dir/update/bundle.json")" \
    --arg decision_state "$(jq -r '.state // "UNKNOWN"' "$out_dir/update/decision-final.json")" \
    --arg policy_ref "$(jq -r '.policy_decision_ref // ""' "$out_dir/update/decision-final.json")" \
    --arg ingest_status "$(jq -r '.status // "unknown"' "$out_dir/update/ingest.json")" \
    --argjson wet_targets "$(jq '.wet_manifest_targets | length' "$out_dir/update/import.json")" \
    '{
      wave_id: $wave_id,
      repository: $repo,
      change_id: $change_id,
      bundle_digest: $bundle_digest,
      decision_state: $decision_state,
      policy_decision_ref: $policy_ref,
      ingest_status: $ingest_status,
      wet_targets: $wet_targets
    }' | tee "$summary_file"

  summaries+=("$summary_file")

  if [ "$decision_state" = "BLOCK" ]; then
    blocked=$((blocked + 1))
  fi
done

jq -s \
  --arg wave_id "$WAVE_ID" \
  '{
    story: "9-governed-multi-repo-wave",
    wave_id: $wave_id,
    targets: ., 
    totals: {
      repositories: length,
      allow: ([.[] | select(.decision_state == "ALLOW")] | length),
      escalate: ([.[] | select(.decision_state == "ESCALATE")] | length),
      block: ([.[] | select(.decision_state == "BLOCK")] | length)
    }
  }' "${summaries[@]}" | tee "$OUT_ROOT/wave-summary.json"

if [ "$blocked" -gt 0 ] && [ "$WAVE_FAIL_ON_BLOCK" = "1" ]; then
  echo "error: wave contains BLOCK decisions ($blocked) and WAVE_FAIL_ON_BLOCK=1" >&2
  exit 1
fi
