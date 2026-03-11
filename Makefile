.PHONY: build test test-parity test-contracts test-bridge-symmetry test-examples test-connected-entrypoints test-connected-lifecycles test-phase-3-stories test-phase-4-stories test-connected-governed-reconcile-helm test-live-reconcile-flux test-live-reconcile-argo lint-dual-mode check-story-status check-story-evidence update-goldens sync-triple-styles ci ci-local ci-connected ci-connected-troubleshoot docs docs-serve

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

test-phase-4-stories:
	./examples/demo/run-phase-4-connected-stories.sh

test-connected-governed-reconcile-helm:
	RECONCILER=both ./examples/demo/e2e-connected-governed-reconcile-helm.sh

test-live-reconcile-flux:
	./examples/demo/e2e-live-reconcile-flux.sh

test-live-reconcile-argo:
	./examples/demo/e2e-live-reconcile-argo.sh

lint-dual-mode:
	./test/checks/check-example-dual-mode.sh

check-story-status:
	./test/checks/check-story-status.sh

check-story-evidence:
	./test/checks/check-story-evidence.sh

update-goldens:
	UPDATE_GOLDEN=1 go test ./cmd/cub-gen -run 'Golden' -count=1 -v

sync-triple-styles:
	go run ./cmd/cub-gen-style-sync

ci-local: build test test-contracts test-bridge-symmetry test-examples lint-dual-mode check-story-status

ci-connected: build test-connected-entrypoints test-connected-lifecycles test-phase-3-stories test-phase-4-stories test-connected-governed-reconcile-helm test-live-reconcile-flux test-live-reconcile-argo check-story-evidence

ci-connected-troubleshoot:
	CONNECTED_FALLBACK_MODE=changeset ALLOW_FALLBACK_INGEST=1 ALLOW_STORY_10_SKIP=1 $(MAKE) ci-connected

ci: ci-local

docs:
	mkdocs build --strict

docs-serve:
	mkdocs serve
