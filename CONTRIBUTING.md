# Contributing to cub-gen

Thank you for contributing to `cub-gen`.

`cub-gen` is a deterministic Git-native generator importer. Contributions must preserve this identity.

## Getting started

```bash
git clone https://github.com/confighub/cub-gen.git
cd cub-gen
go build ./cmd/cub-gen
go test ./... -v
```

## Development rules

1. Keep behavior deterministic.
2. Keep command contracts stable unless explicitly changed.
3. Prefer small, test-backed PRs.
4. Update docs and parity notes for user-visible changes.

## Required test coverage by change type

| Change type | Required tests |
|---|---|
| Detection/import logic | Unit tests in `internal/...` |
| CLI output/flags | Contract/golden tests in `cmd/cub-gen/*_parity_test.go` |
| Command contract change | Follow `docs/testing/contract-drift-checklist.md` + `PARITY.md` update |
| New user-visible flow | `make test-examples` + docs/example update |

## Pull request checklist

1. Explain user problem solved.
2. Include deterministic success criteria.
3. Include commands run and outcome summary.
4. Confirm docs/parity updates when relevant.
5. For contract drift, confirm checklist completion in `docs/testing/contract-drift-checklist.md`.

## Mandatory commands before PR

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
```
