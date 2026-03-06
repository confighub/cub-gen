# cub-gen Test Inventory

## Current automated tests

### CLI parity tests

- `cmd/cub-gen/gitops_parity_test.go`
  - golden discover JSON (Helm)
  - golden discover JSON (Score)
  - golden discover JSON (Spring Boot)
  - golden discover JSON (Backstage IDP)
  - golden discover JSON (Ably app-config)
  - golden discover JSON (Ops workflow)
  - golden import JSON (Helm)
  - golden import JSON (Score)
  - golden import JSON (Spring Boot)
  - golden import JSON (Backstage IDP)
  - golden import JSON (Ably app-config)
  - golden import JSON (Ops workflow)
  - golden cleanup JSON
  - golden discover/import table output
  - error mode coverage
- `cmd/cub-gen/generators_parity_test.go`
  - golden generator family JSON contract
  - golden generator family filtered JSON contracts (single + multi-value filters for kind/capability + profile + combined + empty match)
  - golden generator family table contracts (full + filtered + empty match)
  - golden generators help output contract
- `cmd/cub-gen/examples_smoke_test.go`
  - path-mode discover/import for Helm, Score, Spring, Backstage, Ably, Ops (`./examples/...` without alias config)
- `cmd/cub-gen/examples_bridge_smoke_test.go`
  - path-mode publish/verify/attest/verify-attestation flow for Helm, Score, Spring, Backstage, Ably, Ops (no alias config)
- `cmd/cub-gen/publish_command_test.go`
  - direct publish mode validated for Helm, Score, Spring, Backstage, Ably, Ops
- `cmd/cub-gen/verify_command_test.go`
  - verify JSON path validated for Helm, Score, Spring, Backstage, Ably, Ops bundles
- `cmd/cub-gen/attest_command_test.go`
  - attest path validated for Helm, Score, Spring, Backstage, Ably, Ops bundles
- `cmd/cub-gen/verify_attestation_command_test.go`
  - verify-attestation JSON and linked-bundle JSON paths validated for Helm, Score, Spring, Backstage, Ably, Ops attestation records

### Internal logic tests

- `internal/detect/detect_test.go`
- `internal/importer/importer_test.go` (includes generator capability contract assertions)
- `internal/gitops/flow_test.go`
- `internal/registry/registry_test.go`

## Golden artifacts

- `cmd/cub-gen/testdata/parity/gitops-discover.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover-score.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover-spring.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover-backstage.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover-ably.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover-ops.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-score.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-spring.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-backstage.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-ably.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-ops.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-cleanup.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-discover.table.golden.txt`
- `cmd/cub-gen/testdata/parity/gitops-import.table.golden.txt`
- `cmd/cub-gen/testdata/parity/publish-from-import.golden.json`
- `cmd/cub-gen/testdata/parity/publish-from-import-score.golden.json`
- `cmd/cub-gen/testdata/parity/publish-from-import-spring.golden.json`
- `cmd/cub-gen/testdata/parity/publish-from-import-backstage.golden.json`
- `cmd/cub-gen/testdata/parity/publish-from-import-ably.golden.json`
- `cmd/cub-gen/testdata/parity/publish-from-import-ops.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-helm.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-score.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-spring.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-backstage.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-ably.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-ops.golden.json`
- `cmd/cub-gen/testdata/parity/verify-attestation.json.golden.json`
- `cmd/cub-gen/testdata/parity/verify-attestation-linked-helm.json.golden.json`
- `cmd/cub-gen/testdata/parity/verify-attestation-linked-score.json.golden.json`
- `cmd/cub-gen/testdata/parity/verify-attestation-linked-spring.json.golden.json`
- `cmd/cub-gen/testdata/parity/verify-attestation-linked-backstage.json.golden.json`
- `cmd/cub-gen/testdata/parity/verify-attestation-linked-ably.json.golden.json`
- `cmd/cub-gen/testdata/parity/verify-attestation-linked-ops.json.golden.json`
- `cmd/cub-gen/testdata/parity/generators.golden.json`
- `cmd/cub-gen/testdata/parity/generators-kind-helm.golden.json`
- `cmd/cub-gen/testdata/parity/generators-kind-helm-score.golden.json`
- `cmd/cub-gen/testdata/parity/generators-capability-ops.golden.json`
- `cmd/cub-gen/testdata/parity/generators-capability-helm-score.golden.json`
- `cmd/cub-gen/testdata/parity/generators-profile-spring.golden.json`
- `cmd/cub-gen/testdata/parity/generators-combined-score.golden.json`
- `cmd/cub-gen/testdata/parity/generators-empty.golden.json`
- `cmd/cub-gen/testdata/parity/generators.table.golden.txt`
- `cmd/cub-gen/testdata/parity/generators-kind-helm.table.golden.txt`
- `cmd/cub-gen/testdata/parity/generators-empty.table.golden.txt`
- `cmd/cub-gen/testdata/parity/generators-help.stderr.golden.txt`

## Mandatory validation commands

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
make update-goldens # only when intended CLI contract changes
```
