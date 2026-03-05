# cub-gen

`cub-gen` is a standalone, local-first prototype for Git-native generator import.

It is intentionally separate from `cub-scout`:

1. `cub-scout` focuses on observation/exploration.
2. `cub-gen` focuses on deterministic generator ingestion, lineage, and parity with `cub gitops` command shape.

## Current scope

- Detect generator source types in a repo (`helm`, `score.dev`, `springboot`)
- Mirror `cub gitops` flow stages:
  - `gitops discover`
  - `gitops import`
  - `gitops cleanup`
- Keep all behavior local (no ConfigHub API dependency yet)

## Quick start

```bash
go build ./cmd/cub-gen

./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform ./examples/scoredev-paas ./examples/scoredev-paas
./cub-gen gitops cleanup --space platform ./examples/springboot-paas
```

## Examples

- `examples/helm-paas`
- `examples/scoredev-paas`
- `examples/springboot-paas`

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
