#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'USAGE'
Usage:
  ./examples/demo/app-ai-fastpath.sh <repo-path> [render-target]

Environment variables:
  SPACE         ConfigHub space label (default: platform)
  VERIFIER      Attestation verifier label (default: ai-assistant)
  OUT_DIR       Output directory (default: .tmp/app-ai-fastpath/<repo>-<timestamp>)
  SKIP_BUILD    Set to 1 to skip go build

What it does (single command path):
  gitops import -> publish -> verify -> attest -> verify-attestation

Primary output:
  mutation-card.json (compact app/AI programmer handoff)
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

REPO_PATH="${1:-}"
if [ -z "$REPO_PATH" ]; then
  usage >&2
  exit 1
fi

RENDER_TARGET="${2:-$REPO_PATH}"
SPACE="${SPACE:-platform}"
VERIFIER="${VERIFIER:-ai-assistant}"

if [ ! -d "$REPO_PATH" ]; then
  echo "error: repo path not found: $REPO_PATH" >&2
  exit 1
fi
if [ ! -d "$RENDER_TARGET" ]; then
  echo "error: render target path not found: $RENDER_TARGET" >&2
  exit 1
fi

sanitize_slug() {
  local input="$1"
  input="$(printf '%s' "$input" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
  input="$(printf '%s' "$input" | sed -E 's/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "${input:0:63}"
}

repo_slug="$(sanitize_slug "$(basename "$REPO_PATH")")"
if [ -z "$repo_slug" ]; then
  repo_slug="repo"
fi

if [ -n "${OUT_DIR:-}" ]; then
  out_dir="$OUT_DIR"
else
  out_dir=".tmp/app-ai-fastpath/${repo_slug}-$(date +%Y%m%d-%H%M%S)"
fi
mkdir -p "$out_dir"

if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "[fastpath] build cub-gen"
  go build -o ./cub-gen ./cmd/cub-gen
fi

echo "[fastpath] import"
./cub-gen gitops import --space "$SPACE" --json "$REPO_PATH" "$RENDER_TARGET" > "$out_dir/import.json"

echo "[fastpath] publish"
./cub-gen publish --in "$out_dir/import.json" > "$out_dir/bundle.json"

echo "[fastpath] verify bundle"
./cub-gen verify --json --in "$out_dir/bundle.json" > "$out_dir/verify.json"

echo "[fastpath] attest"
./cub-gen attest --in "$out_dir/bundle.json" --verifier "$VERIFIER" > "$out_dir/attestation.json"

echo "[fastpath] verify attestation"
./cub-gen verify-attestation --json --in "$out_dir/attestation.json" --bundle "$out_dir/bundle.json" > "$out_dir/attestation-verify.json"

jq -n \
  --arg repo_path "$REPO_PATH" \
  --arg render_target "$RENDER_TARGET" \
  --arg space "$SPACE" \
  --arg verifier "$VERIFIER" \
  --arg import_path "$out_dir/import.json" \
  --arg bundle_path "$out_dir/bundle.json" \
  --arg verify_path "$out_dir/verify.json" \
  --arg attestation_path "$out_dir/attestation.json" \
  --arg attestation_verify_path "$out_dir/attestation-verify.json" \
  --slurpfile imported "$out_dir/import.json" \
  --slurpfile bundle "$out_dir/bundle.json" \
  --slurpfile verify "$out_dir/verify.json" \
  --slurpfile attestation "$out_dir/attestation.json" \
  --slurpfile attestation_verify "$out_dir/attestation-verify.json" \
  '
    ($imported[0].provenance // []) as $provenance |
    ($provenance | map(.inverse_edit_pointers // []) | flatten | sort_by(-(.confidence // 0)) | .[0]) as $top_inverse |
    ($imported[0].inverse_transform_plans // []) as $plans |
    {
      input: {
        repo_path: $repo_path,
        render_target: $render_target,
        space: $space
      },
      change: {
        change_id: ($bundle[0].change_id // null),
        bundle_digest: ($bundle[0].bundle_digest // null),
        attestation_digest: ($attestation[0].attestation_digest // null)
      },
      discovered_profiles: (($imported[0].discovered // []) | map(.generator_profile) | unique),
      counts: {
        discovered_resources: (($imported[0].discovered // []) | length),
        dry_inputs: (($imported[0].dry_inputs // []) | length),
        wet_targets: (($imported[0].wet_manifest_targets // []) | length),
        inverse_patches: ($plans | map((.patches // []) | length) | add // 0)
      },
      edit_recommendation: {
        owner: ($top_inverse.owner // "unknown"),
        wet_path: ($top_inverse.wet_path // null),
        dry_path: ($top_inverse.dry_path // null),
        edit_hint: ($top_inverse.edit_hint // "No inverse edit hint produced."),
        confidence: ($top_inverse.confidence // null)
      },
      verification: {
        bundle_valid: ($verify[0].valid // false),
        attestation_valid: ($attestation_verify[0].valid // false),
        verifier: $verifier
      },
      artifacts: {
        import: $import_path,
        bundle: $bundle_path,
        verify: $verify_path,
        attestation: $attestation_path,
        attestation_verify: $attestation_verify_path
      }
    }
  ' > "$out_dir/mutation-card.json"

echo "[fastpath] mutation card"
cat "$out_dir/mutation-card.json"

echo "[fastpath] artifacts: $out_dir"
