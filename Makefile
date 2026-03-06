.PHONY: build test test-parity test-contracts test-examples update-goldens ci

PARITY_TEST_PATTERN := ^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)

build:
	go build ./cmd/cub-gen

test:
	go test ./...

test-contracts:
	go test ./cmd/cub-gen -run '$(PARITY_TEST_PATTERN)' -count=1 -v

test-parity: test-contracts

test-examples:
	go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$$' -count=1 -v

update-goldens:
	UPDATE_GOLDEN=1 go test ./cmd/cub-gen -run 'Golden' -count=1 -v

ci: build test test-contracts test-examples
