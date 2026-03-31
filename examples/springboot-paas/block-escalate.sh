#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${1:-}" in
  --explain)
    cat <<'EOF'
block-escalate: datasource override boundary

spring.datasource.* fields are platform-owned per field-routes.yaml.
Direct mutation of these fields should be blocked or escalated.

The runtime-policy.yaml declares managedDatasource: postgres-shared,
meaning the platform controls the datasource connection.

Use --render-attempt to see a dry-run of what a datasource override
would look like and why it should be blocked.
EOF
    ;;
  --render-attempt)
    echo "=== Dry-run datasource override attempt ==="
    echo ""
    echo "Field: spring.datasource.url"
    echo "Current: jdbc:postgresql://postgres.platform.svc:5432/inventory"
    echo "Attempted: jdbc:postgresql://custom-db.team.svc:5432/inventory"
    echo ""
    echo "Route rule: spring.datasource.* -> generator-owned (platform-engineering)"
    echo "Policy: managedDatasource = postgres-shared (from runtime-policy.yaml)"
    echo ""
    echo "Decision: BLOCK or ESCALATE"
    echo "Reason: Datasource connectivity is part of the managed platform boundary."
    echo ""
    echo "Env var on Deployment:"
    grep 'SPRING_DATASOURCE_URL' "$ROOT_DIR/operational/deployment.yaml" || true
    ;;
  *)
    echo "Usage: $0 [--explain | --render-attempt]" >&2
    exit 2
    ;;
esac
