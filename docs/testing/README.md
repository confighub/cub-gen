# cub-gen Testing Strategy

This repo inherits cub-scout quality discipline with scope adjusted for `cub-gen`.

## Test tiers

### Tier 0: smoke

- Build CLI binary.

### Tier 1: unit/contract

- `go test ./...`
- Core detection/import/flow logic tests in `internal/...`.
- Cross-family contract-triple conformance fixtures in `internal/contracts/triple_conformance_fixtures_test.go` (Helm/Score/Spring/Backstage/Ably/Ops).
- Bridge bundle tests in `internal/publish/...`.
- Attestation model tests in `internal/attest/...`.

### Tier 2: parity/golden

- CLI output contract tests in `cmd/cub-gen/gitops_parity_test.go`.
- Generator inventory contract tests in `cmd/cub-gen/generators_parity_test.go` (full list + JSON/table filter contracts for kind/profile/capability, including comma-separated multi-value filters, combined filters, empty-match behavior, and strict unknown-filter error modes).
- Bridge symmetry matrix coverage gate in `cmd/cub-gen/examples_matrix_test.go` (fails if any registry generator kind is missing from `publish -> verify -> attest -> verify-attestation` family matrix).
- Bridge publish golden locks in `cmd/cub-gen/publish_parity_test.go` (import/direct paths for helm/score/spring/backstage/ably/ops).
- Verify command behavior tests in `cmd/cub-gen/verify_command_test.go`.
- Verify command golden locks in `cmd/cub-gen/verify_parity_test.go`.
- Attest command behavior tests in `cmd/cub-gen/attest_command_test.go`.
- Attest command golden locks in `cmd/cub-gen/attest_parity_test.go`.
- Verify-attestation behavior tests in `cmd/cub-gen/verify_attestation_command_test.go`.
- Verify-attestation golden locks in `cmd/cub-gen/verify_attestation_parity_test.go` (including linked helm/score/spring/backstage/ably/ops JSON contracts).
- Golden files under `cmd/cub-gen/testdata/parity/`.
- Includes command help/usage goldens to lock human-facing CLI contract.
- Includes generator inventory table/JSON/help goldens.
- Helm example is covered across discover/import/cleanup parity paths.
- Score, Spring Boot, Backstage IDP, Ably app-config, and Ops workflow discover/import JSON contracts are golden-locked.
- Score import JSON contract is golden-locked (`gitops-import-score.golden.json`).
- Spring Boot import JSON contract is golden-locked (`gitops-import-spring.golden.json`).
- Backstage import JSON contract is golden-locked (`gitops-import-backstage.golden.json`).
- Ably import JSON contract is golden-locked (`gitops-import-ably.golden.json`).
- Ops workflow import JSON contract is golden-locked (`gitops-import-ops.golden.json`).

### Tier 3: examples proof

- Run at least one command per example for changed behavior.
- For bridge changes, validate import -> publish pipeline output.
- Automated path-mode smoke for Helm/Score/Spring/Backstage/Ably/Ops in `cmd/cub-gen/examples_smoke_test.go`.
- Automated path-mode bridge smoke (`publish -> verify -> attest -> verify-attestation`) for Helm/Score/Spring/Backstage/Ably/Ops in `cmd/cub-gen/examples_bridge_smoke_test.go`.
- Publish/verify/attest command tests include Helm/Score/Spring/Backstage/Ably/Ops bundle flows.
- Verify-attestation command tests include Helm/Score/Spring/Backstage/Ably/Ops attestation flows (with and without linked bundle checks).

## Required commands

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v
go test ./cmd/cub-gen -run '^(TestBridgeSymmetryMatrix|TestExamplesPathModeBridgeFlow)$' -count=1 -v
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
```

## Updating goldens

```bash
make update-goldens
```

## Rules

1. Same input => same output.
2. Golden output is treated as contract surface.
3. Unsupported behavior must fail explicitly, never silently.
4. Every user-visible change needs either a new golden or an explicit reason why not.
