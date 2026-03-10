#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[module-5] build cub-gen"
go build -o ./cub-gen ./cmd/cub-gen

echo "[module-5] import no-config-platform provider config and show ownership/provenance"
./cub-gen gitops import --space platform --json ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
