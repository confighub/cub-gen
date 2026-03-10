.PHONY: build test test-parity test-contracts test-bridge-symmetry test-examples test-connected-entrypoints test-connected-lifecycles test-phase-3-stories test-live-reconcile-flux test-live-reconcile-argo lint-dual-mode check-story-status update-goldens sync-triple-styles ci ci-local ci-connected docs docs-serve

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

test-connected-entrypoints:
	./examples/demo/run-all-connected-entrypoints.sh

test-connected-lifecycles:
	./examples/demo/run-all-connected-lifecycles.sh

test-phase-3-stories:
	./examples/demo/run-phase-3-connected-stories.sh

test-live-reconcile-flux:
	./examples/demo/e2e-live-reconcile-flux.sh

test-live-reconcile-argo:
	./examples/demo/e2e-live-reconcile-argo.sh

lint-dual-mode:
	./test/checks/check-example-dual-mode.sh

check-story-status:
	./test/checks/check-story-status.sh

update-goldens:
	UPDATE_GOLDEN=1 go test ./cmd/cub-gen -run 'Golden' -count=1 -v

sync-triple-styles:
	go run ./cmd/cub-gen-style-sync

ci-local: build test test-contracts test-bridge-symmetry test-examples lint-dual-mode check-story-status

ci-connected: build test-connected-entrypoints test-connected-lifecycles test-phase-3-stories test-live-reconcile-flux test-live-reconcile-argo check-story-status

ci: ci-local

docs:
	mkdocs build --strict

docs-serve:
	mkdocs serve
