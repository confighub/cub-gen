# cub-gen

`cub-gen` is a prototype for Git-native generator import based on `cub gitops`.

## Current scope

- Detect generator source types in a repo (`helm`, `score.dev`, `springboot`)
- Per `cub gitops` the flow stages are:
  - `gitops discover`
  - `gitops import`
  - `gitops cleanup`
- Keep all behavior local for now

## Quick start

```bash
go build ./cmd/cub-gen

./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform ./examples/helm-paas ./examples/helm-paas
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

## Examples

- `examples/helm-paas`
- `examples/scoredev-paas`
- `examples/springboot-paas`

## Score.dev import contract (MVP-01)

For score.dev repos, `gitops import --json` now emits:

- `generator_profile: "scoredev-paas"`
- provenance `field_origin_map` entries
- provenance `inverse_edit_pointers` entries

This makes app-level edits explicit: each important rendered field points back to a DRY score path and edit hint.

## Helm import contract (MVP-02)

For Helm repos, `gitops import --json` now emits explicit DRY and WET structures:

- top-level `dry_inputs` (chart + values files)
- top-level `wet_manifest_targets` (HelmRelease/Deployment/Service targets)
- provenance `chart_path`
- provenance `values_paths`
- provenance `rendered_object_lineage`

## Quality model (inherited from cub-scout, adapted)

- Deterministic behavior: same input => same output
- Contract parity tests for CLI outputs (JSON + table goldens)
- Proof-first delivery: define test matrix before implementation
- Example-backed validation for user-visible behavior

See:

- `CLAUDE.md`
- `CONTRIBUTING.md`
- `docs/testing/README.md`
- `docs/workflows/proof-first-delivery.md`
- `PARITY.md`

## Test

```bash
go test ./...
go test ./cmd/cub-gen -run '^TestGitOpsParity' -count=1 -v
```
