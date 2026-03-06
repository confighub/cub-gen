# cub-gen Testing Strategy

This repo inherits cub-scout quality discipline with scope adjusted for `cub-gen`.

## Test tiers

### Tier 0: smoke

- Build CLI binary.

### Tier 1: unit/contract

- `go test ./...`
- Core detection/import/flow logic tests in `internal/...`.
- Bridge bundle tests in `internal/publish/...`.

### Tier 2: parity/golden

- CLI output contract tests in `cmd/cub-gen/gitops_parity_test.go`.
- Bridge publish golden lock in `cmd/cub-gen/publish_parity_test.go`.
- Verify command behavior tests in `cmd/cub-gen/verify_command_test.go`.
- Verify command golden locks in `cmd/cub-gen/verify_parity_test.go`.
- Golden files under `cmd/cub-gen/testdata/parity/`.
- Includes command help/usage goldens to lock human-facing CLI contract.
- Helm example is covered across discover/import/cleanup parity paths.
- Spring Boot import JSON contract is golden-locked (`gitops-import-spring.golden.json`).

### Tier 3: examples proof

- Run at least one command per example for changed behavior.
- For bridge changes, validate import -> publish pipeline output.

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
