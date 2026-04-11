#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_ROOT="${OUT_ROOT:-$ROOT_DIR/.tmp/helm-paas-layered-trace}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
WORK_REPO="$OUT_DIR/repo"

mkdir -p "$OUT_DIR"
rm -rf "$WORK_REPO"

echo "[helm-layered] copying working tree into $WORK_REPO"
mkdir -p "$WORK_REPO"
rsync -a \
  --exclude '.git' \
  --exclude '.tmp' \
  --exclude '/cub-gen' \
  "$ROOT_DIR"/ "$WORK_REPO"/

cd "$WORK_REPO"
go build -o ./cub-gen ./cmd/cub-gen

echo "[helm-layered] proof 1/2: layered import attributes cluster selection and overlay choice"
./cub-gen gitops import --space platform --json ./examples/helm-paas > "$OUT_DIR/import-allow.json"

jq -e '
  .provenance[0].helm_layered_analysis.generation_decision_state == "attributed" and
  .provenance[0].helm_layered_analysis.cluster_selector == "env=prod, region=eu" and
  (.provenance[0].helm_layered_analysis.matched_clusters | index("prod-eu")) != null and
  (.provenance[0].helm_layered_analysis.selected_value_files | index("values-prod.yaml")) != null and
  .provenance[0].helm_layered_analysis.security_decision_state == "allow"
' "$OUT_DIR/import-allow.json" >/dev/null

echo "[helm-layered][allow] layered analysis"
jq '{
  generation_state: .provenance[0].helm_layered_analysis.generation_decision_state,
  generation_reason: .provenance[0].helm_layered_analysis.generation_decision_reason,
  matched_clusters: .provenance[0].helm_layered_analysis.matched_clusters,
  selected_value_files: .provenance[0].helm_layered_analysis.selected_value_files,
  security_state: .provenance[0].helm_layered_analysis.security_decision_state,
  security_reason: .provenance[0].helm_layered_analysis.security_decision_reason
}' "$OUT_DIR/import-allow.json"

echo "[helm-layered] proof 2/2: customer overlay weakens a managed security control and is classified as blocked"
perl -0pi -e 's/enabled: true/enabled: false/' ./examples/helm-paas/platform/catalogs/customer-service-catalog/payments-api-prod.yaml
./cub-gen gitops import --space platform --json ./examples/helm-paas > "$OUT_DIR/import-block.json"

jq -e '
  .provenance[0].helm_layered_analysis.security_decision_state == "blocked" and
  (.provenance[0].helm_layered_analysis.security_decision_reason | contains("oauth2Proxy"))
' "$OUT_DIR/import-block.json" >/dev/null

echo "[helm-layered][block] layered analysis"
jq '{
  security_state: .provenance[0].helm_layered_analysis.security_decision_state,
  security_reason: .provenance[0].helm_layered_analysis.security_decision_reason,
  security_control_path: .provenance[0].helm_layered_analysis.security_control_path,
  security_override_path: .provenance[0].helm_layered_analysis.security_override_path
}' "$OUT_DIR/import-block.json"

cat <<EOF

[helm-layered] success
  attributed path: cluster labels -> ApplicationSet selector -> values overlay -> rendered field
  blocked path: customer service catalog tries to weaken a platform-required security control
  artifacts: $OUT_DIR
EOF
