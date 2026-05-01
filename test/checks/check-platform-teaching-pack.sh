#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

declare -a failures=()

require_file() {
  local file="$1"
  if [ ! -f "$file" ]; then
    failures+=("missing required file: $file")
  fi
}

require_pattern() {
  local file="$1"
  local pattern="$2"
  local message="$3"
  if ! grep -Eq -- "$pattern" "$file"; then
    failures+=("$message")
  fi
}

require_min_count() {
  local file="$1"
  local pattern="$2"
  local min_count="$3"
  local message="$4"
  local count
  count="$(grep -Ec -- "$pattern" "$file" || true)"
  if [ "$count" -lt "$min_count" ]; then
    failures+=("$message")
  fi
}

openchoreo_doc="docs/agentic-gitops/03-worked-examples/05-openchoreo-generator-worked-example.md"
argo_doc="docs/agentic-gitops/03-worked-examples/06-argo-generators-worked-example.md"
manifesto_doc="docs/agentic-gitops/02-design/90-platform-generators-manifesto.md"
appset_contract="docs/contracts/applicationset-generator-boundary.md"
appofapps_contract="docs/contracts/app-of-apps-generator-boundary.md"

for file in "$openchoreo_doc" "$argo_doc" "$manifesto_doc" "$appset_contract" "$appofapps_contract" "README.md" "docs/index.md" "examples/README.md" "mkdocs.yml"; do
  require_file "$file"
done

require_min_count "$openchoreo_doc" '^```mermaid$' 2 "OpenChoreo worked example must stay diagram-heavy"
require_min_count "$argo_doc" '^```mermaid$' 2 "Argo worked example must include ApplicationSet and app-of-apps diagrams"

require_pattern "$openchoreo_doc" 'fixture-backed hardgate support' "OpenChoreo doc must state fixture-backed support status"
require_pattern "$openchoreo_doc" 'not (the same as )?full upstream OpenChoreo conformance' "OpenChoreo doc must not overclaim upstream conformance"
require_pattern "$openchoreo_doc" 'testdata/openchoreo-hardgate' "OpenChoreo doc must point at the real hardgate fixture"
require_pattern "$openchoreo_doc" 'apply-here|lift-upstream|block/escalate|temporary overlay' "OpenChoreo doc must explain edit routing"

require_pattern "$argo_doc" 'Status: worked example plus fixture-backed adapters' "Argo doc must state current support status"
require_pattern "$argo_doc" './cub-gen gitops import --space platform --json ./testdata/applicationset-standalone' "Argo doc must prove ApplicationSet against a real fixture"
require_pattern "$argo_doc" './cub-gen gitops import --space platform --json ./testdata/app-of-apps-standalone' "Argo doc must prove app-of-apps against a real fixture"
require_pattern "$argo_doc" 'unsupported generator types.*observed-only|observed-only or degraded' "Argo doc must describe graceful degradation"

require_pattern "README.md" 'A \*\*Generator\*\* is a function on config data' "README must define Generator in the first-minute pitch"
require_pattern "README.md" 'repo in, explanation out' "README must keep the first-minute repo-in explanation-out line"
require_pattern "README.md" 'It does not deploy\. Flux and Argo still reconcile|Not a Flux or Argo replacement' "README must explain generator, not replacement"
require_pattern "docs/index.md" 'not by replacing the platform' "Docs index must explain generator, not replacement"
require_pattern "examples/README.md" 'fixture-backed hardgate; not full upstream conformance' "Examples index must keep OpenChoreo claims conservative"

require_pattern "mkdocs.yml" '05-openchoreo-generator-worked-example.md' "MkDocs nav must include OpenChoreo worked example"
require_pattern "mkdocs.yml" '06-argo-generators-worked-example.md' "MkDocs nav must include Argo worked example"
require_pattern "mkdocs.yml" 'applicationset-generator-boundary.md' "MkDocs nav must include ApplicationSet contract"
require_pattern "mkdocs.yml" 'app-of-apps-generator-boundary.md' "MkDocs nav must include app-of-apps contract"

if [ "${#failures[@]}" -gt 0 ]; then
  echo "error: platform teaching pack check failed:" >&2
  for failure in "${failures[@]}"; do
    echo "  - $failure" >&2
  done
  exit 1
fi

echo "ok: platform generator teaching pack has conservative claims, diagrams, fixture proof, and nav coverage"
