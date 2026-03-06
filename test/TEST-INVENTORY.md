# cub-gen Test Inventory

## Current automated tests

### CLI parity tests

- `cmd/cub-gen/gitops_parity_test.go`
  - golden discover JSON (Helm)
  - golden discover JSON (Score)
  - golden discover JSON (Spring Boot)
  - golden import JSON (Helm)
  - golden import JSON (Score)
  - golden import JSON (Spring Boot)
  - golden cleanup JSON
  - golden discover/import table output
  - error mode coverage
- `cmd/cub-gen/examples_smoke_test.go`
  - path-mode discover/import for Helm, Score, Spring (`./examples/...` without alias config)
- `cmd/cub-gen/examples_bridge_smoke_test.go`
  - path-mode publish/verify/attest/verify-attestation flow for Helm, Score, Spring (no alias config)
- `cmd/cub-gen/publish_command_test.go`
  - direct publish mode validated for Helm, Score, Spring
- `cmd/cub-gen/verify_command_test.go`
  - verify JSON path validated for Helm, Score, Spring bundles
- `cmd/cub-gen/attest_command_test.go`
  - attest path validated for Helm, Score, Spring bundles
- `cmd/cub-gen/verify_attestation_command_test.go`
  - verify-attestation JSON and linked-bundle JSON paths validated for Helm, Score, Spring attestation records

### Internal logic tests

- `internal/detect/detect_test.go`
- `internal/importer/importer_test.go`
- `internal/gitops/flow_test.go`

## Golden artifacts

- `cmd/cub-gen/testdata/parity/gitops-discover.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover-score.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover-spring.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-score.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-spring.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-cleanup.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover.table.golden.txt`
- `cmd/cub-gen/testdata/parity/gitops-import.table.golden.txt`
- `cmd/cub-gen/testdata/parity/publish-from-import.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-score.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-spring.golden.json`

## Mandatory validation commands

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
make update-goldens # only when intended CLI contract changes
```
