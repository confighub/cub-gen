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

## How to add a new generator

Adding a new generator requires changes in four areas: detection, registry,
example, and tests.

### 1. Add detection logic

Create a detector in `internal/detect/detect.go`. Your detector must:

- Walk the input directory for signature files (e.g., `my-tool.yaml`)
- Validate file content to confirm the match (e.g., contains `apiVersion: mytool/v1`)
- Return a `DetectionResult` with kind, confidence (0.0-1.0), inputs list, and root path
- Be deterministic — same input directory always produces the same result

```go
// Example: detect my-tool.yaml with "apiVersion: mytool/v1"
func detectMyTool(root string) ([]DetectionResult, error) {
    // Walk for signature file, validate content, return results
}
```

### 2. Register the generator family

Add a `FamilySpec` in `internal/registry/registry.go`. The spec must declare:

- **Profile name** (e.g., `"my-tool-paas"`) — user-facing identifier
- **Kind** (e.g., `"mytool"`) — internal kind key
- **ResourceKind** and **ResourceType** — the primary Kubernetes resource
- **Capabilities** — what the generator can do (e.g., `render-manifests`)
- **HintDefaults** — default file paths for detection
- **InversePatchTemplates** — editable fields with ownership and confidence scores
- **FieldOriginConfidences** — confidence levels for field-origin tracing
- **RenderedLineageTemplates** — expected WET output targets

### 3. Create an example directory

Create `examples/my-tool/` with:

- DRY source files (e.g., `my-tool.yaml`, `my-tool-prod.yaml`)
- `platform/` directory with illustrative policies
- `docs/user-stories.md` with 3-4 user stories
- `README.md` following the template in `examples/README.md`

Add the enforcement disclaimer to any platform policy files:

```yaml
# NOTE: This policy is illustrative. ConfigHub's decision engine evaluates
# these constraints server-side. cub-gen reads them for field-origin tracing
# and documentation but does not enforce them at import time.
```

### 4. Add tests

- **Detection test**: verify your detector finds the example with correct confidence
- **Import test**: verify DRY/WET classification and field-origin mapping
- **Golden test**: generate golden files with `UPDATE_GOLDEN=1` and commit them
- **Parity test**: ensure `discover` → `import` → `publish` → `verify` flow works

```bash
# Generate golden files for a new generator
UPDATE_GOLDEN=1 go test ./cmd/cub-gen -run 'TestGitOpsParityGolden.*MyTool' -count=1 -v

# Verify all tests pass
make ci
```

### Generator checklist

- [ ] Detector added with content validation (not just filename matching)
- [ ] FamilySpec registered with all required fields
- [ ] Confidence scores calibrated (0.85-0.98 typical range)
- [ ] Example directory created with README, platform policies, user stories
- [ ] Golden test files committed
- [ ] `make ci` passes
- [ ] Example listed in `examples/README.md`

## Mandatory commands before PR

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
```
