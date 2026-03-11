# Getting Started

Get from zero to provenance-traced import in under 10 minutes.

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- Git

## Build

```bash
git clone https://github.com/confighub/cub-gen.git
cd cub-gen
go build -o cub-gen ./cmd/cub-gen
```

## Use your own repo in 3 commands

```bash
REPO=/path/to/your/repo
./cub-gen change preview --space platform "$REPO" "$REPO"
./cub-gen change run --mode local --space platform "$REPO" "$REPO"
./cub-gen change explain --space platform --owner app-team "$REPO" "$REPO"
```

Connected mode for the same repo:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"
./cub-gen change run --mode connected --base-url "$BASE_URL" --token "$TOKEN" --space platform "$REPO" "$REPO"
```

## Your first import (Helm)

The three core commands mirror `cub gitops`:

### 1. Discover generator roots

```bash
./cub-gen gitops discover --space platform ./examples/helm-paas
```

This scans the repo and classifies it as a Helm generator (`helm-paas` profile).

### 2. Import with provenance

```bash
./cub-gen gitops import --space platform --json \
  ./examples/helm-paas ./examples/helm-paas | jq .
```

The import output includes:

- **`generator_profile`** — which generator family was detected
- **`dry_inputs`** — the human-editable source files (Chart.yaml, values.yaml)
- **`wet_manifest_targets`** — the rendered deployment artifacts
- **`provenance`** — field-origin map and inverse-edit pointers

### 3. Clean up discover state

```bash
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

## Inspect provenance

The key value of cub-gen is the provenance trail. Focus on the inverse-edit pointers:

```bash
./cub-gen gitops import --space platform --json \
  ./examples/helm-paas ./examples/helm-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      dry_inputs,
      wet_manifest_targets,
      provenance: .provenance[0] | {
        chart_path,
        values_paths,
        rendered_object_lineage
      }
    }'
```

This tells you: for any WET field, where is the DRY source to edit it safely.

## Try other generators

Each generator follows the same three-command flow:

=== "Score.dev"

    ```bash
    ./cub-gen gitops discover --space platform ./examples/scoredev-paas
    ./cub-gen gitops import --space platform --json \
      ./examples/scoredev-paas ./examples/scoredev-paas | jq .
    ./cub-gen gitops cleanup --space platform ./examples/scoredev-paas
    ```

=== "Spring Boot"

    ```bash
    ./cub-gen gitops discover --space platform ./examples/springboot-paas
    ./cub-gen gitops import --space platform --json \
      ./examples/springboot-paas ./examples/springboot-paas | jq .
    ./cub-gen gitops cleanup --space platform ./examples/springboot-paas
    ```

=== "Backstage IDP"

    ```bash
    ./cub-gen gitops discover --space platform ./examples/backstage-idp
    ./cub-gen gitops import --space platform --json \
      ./examples/backstage-idp ./examples/backstage-idp | jq .
    ./cub-gen gitops cleanup --space platform ./examples/backstage-idp
    ```

## Bridge artifacts (publish, verify, attest)

Generate a ConfigHub-ready change bundle with digest verification:

```bash
./cub-gen publish --space platform \
  ./examples/helm-paas ./examples/helm-paas \
  | jq '{schema_version, source, change_id, summary}'
```

Verify bundle integrity:

```bash
./cub-gen publish --space platform \
  ./examples/helm-paas ./examples/helm-paas \
  | ./cub-gen verify --in -
```

Emit an attestation record:

```bash
./cub-gen publish --space platform \
  ./examples/helm-paas ./examples/helm-paas \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{status, verifier, bundle_digest, attestation_digest}'
```

## Run demo modules

Five self-contained demo modules, each runnable independently:

```bash
./examples/demo/module-1-helm-import.sh
./examples/demo/module-2-score-field-map.sh
./examples/demo/module-3-spring-ownership.sh
./examples/demo/module-4-bridge-governance.sh
./examples/demo/module-5-no-config-platform.sh
```

Or all at once:

```bash
./examples/demo/run-all-modules.sh
```

## What stays unchanged

- Flux/Argo remains the reconciler for WET to LIVE
- Git/OCI remains the transport path
- Existing cluster/controller permissions and PR workflow stay in place

## Next steps

- [The ConfigHub Platform](platform.md) — how cub-gen connects to ConfigHub, bridge workers, and Flux/ArgoCD
- [CLI Reference](cli-reference.md) — full command and flag documentation
- [Architecture](agentic-gitops/02-design/00-agentic-gitops-design.md) — DRY/WET model, contract triples, governed execution
- [Worked Examples](agentic-gitops/03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md) — end-to-end Score.dev walkthrough
- [Adoption Path & FAQ](agentic-gitops/05-rollout/40-adoption-and-reference.md) — progressive adoption ladder
