#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[module-1] build cub-gen"
go build -o ./cub-gen ./cmd/cub-gen

echo "[module-1] discover helm app"
./cub-gen gitops discover --space platform ./examples/helm-paas

echo "[module-1] import and show DRY/WET summary"
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'

