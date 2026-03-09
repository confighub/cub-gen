.PHONY: build test test-parity test-contracts test-bridge-symmetry test-examples update-goldens sync-triple-styles ci docs docs-serve

PARITY_TEST_PATTERN := ^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand|TestGeneratorsGolden)
BRIDGE_SYMMETRY_PATTERN := ^(TestBridgeSymmetryMatrix|TestExamplesPathModeBridgeFlow)$

build:
	go build ./cmd/cub-gen

test:
	go test ./...

test-contracts:
	go test ./cmd/cub-gen -run '$(PARITY_TEST_PATTERN)' -count=1 -v

test-parity: test-contracts

test-bridge-symmetry:
	go test ./cmd/cub-gen -run '$(BRIDGE_SYMMETRY_PATTERN)' -count=1 -v

test-examples:
	go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$$' -count=1 -v

update-goldens:
	UPDATE_GOLDEN=1 go test ./cmd/cub-gen -run 'Golden' -count=1 -v

sync-triple-styles:
	go run ./cmd/cub-gen-style-sync

ci: build test test-contracts test-bridge-symmetry test-examples

docs:
	mkdocs build --strict

docs-serve:
	mkdocs serve
