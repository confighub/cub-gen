# cub-gen Testing Strategy

This repo inherits cub-scout quality discipline with scope adjusted for `cub-gen`.

## Test tiers

### Tier 0: smoke

- Build CLI binary.

### Tier 1: unit/contract

- `go test ./...`
- Core detection/import/flow logic tests in `internal/...`.

### Tier 2: parity/golden

- CLI output contract tests in `cmd/cub-gen/gitops_parity_test.go`.
- Golden files under `cmd/cub-gen/testdata/parity/`.

### Tier 3: examples proof

- Run at least one command per example for changed behavior.

## Required commands

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^TestGitOpsParity' -count=1 -v
```

## Rules

1. Same input => same output.
2. Golden output is treated as contract surface.
3. Unsupported behavior must fail explicitly, never silently.
4. Every user-visible change needs either a new golden or an explicit reason why not.
