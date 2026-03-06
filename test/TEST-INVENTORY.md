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

## Mandatory validation commands

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^TestGitOpsParity' -count=1 -v
```
