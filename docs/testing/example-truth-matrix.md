# Example Truth Matrix

Generated from repo structure, source-side tests, connected runners, and live-proof harness scripts. Do not edit by hand; regenerate with `go run ./tools/example-truth-matrix --format markdown`.

## Summary

- Featured examples: `12`
- Generator fixtures: `8`
- Source-chain verified: `8`
- Connected mode present: `12`
- Connected release gated: `12`
- Real live proof: `none=9`, `paired-harness=1`, `standalone=2`
- AI-first surface: `none=6`, `partial=2`, `explicit=4`

## Matrix

| Example | Generator fixture | Source chain verified | Connected mode | Connected release gate | Real live proof | AI-first surface | Tracking issues |
|---|---|---|---|---|---|---|---|
| `ai-ops-paas` | no | no | yes | yes | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183), [#202](https://github.com/confighub/cub-gen/issues/202) |
| `backstage-idp` | yes | yes | yes | yes | `none` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `c3agent` | yes | yes | yes | yes | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183), [#202](https://github.com/confighub/cub-gen/issues/202), [#216](https://github.com/confighub/cub-gen/issues/216) |
| `confighub-actions` | no | no | yes | yes | `none` | `partial` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183), [#202](https://github.com/confighub/cub-gen/issues/202) |
| `helm-paas` | yes | yes | yes | yes | `paired-harness` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#177](https://github.com/confighub/cub-gen/issues/177), [#183](https://github.com/confighub/cub-gen/issues/183), [#187](https://github.com/confighub/cub-gen/issues/187) |
| `just-apps-no-platform-config` | yes | yes | yes | yes | `none` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `live-reconcile` | no | no | yes | yes | `standalone` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `ops-workflow` | yes | yes | yes | yes | `none` | `partial` | [#173](https://github.com/confighub/cub-gen/issues/173), [#180](https://github.com/confighub/cub-gen/issues/180), [#183](https://github.com/confighub/cub-gen/issues/183), [#202](https://github.com/confighub/cub-gen/issues/202) |
| `scoredev-paas` | yes | yes | yes | yes | `none` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#178](https://github.com/confighub/cub-gen/issues/178), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `springboot-paas` | yes | yes | yes | yes | `standalone` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#179](https://github.com/confighub/cub-gen/issues/179), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `swamp-automation` | yes | yes | yes | yes | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#180](https://github.com/confighub/cub-gen/issues/180), [#183](https://github.com/confighub/cub-gen/issues/183), [#202](https://github.com/confighub/cub-gen/issues/202) |
| `swamp-project` | no | no | yes | yes | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#180](https://github.com/confighub/cub-gen/issues/180), [#183](https://github.com/confighub/cub-gen/issues/183), [#202](https://github.com/confighub/cub-gen/issues/202) |

## Proof References

### `ai-ops-paas`

- Source chain: --
- Connected mode: `./examples/ai-ops-paas/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: `examples/README.md#ai--automation-patterns`

### `backstage-idp`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/backstage-idp/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: --

### `c3agent`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/c3agent/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: `examples/README.md#ai--automation-patterns`

### `confighub-actions`

- Source chain: --
- Connected mode: `./examples/confighub-actions/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: `examples/demo/README.md#ai-work-platform-track`

### `helm-paas`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/helm-paas/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: `./examples/demo/e2e-connected-governed-reconcile-helm.sh`, `./examples/live-reconcile/demo-local.sh`
- AI-first: --
- Notes: Real LIVE proof is paired through the live-reconcile harness, not standalone in helm-paas.

### `just-apps-no-platform-config`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/just-apps-no-platform-config/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: --

### `live-reconcile`

- Source chain: --
- Connected mode: `./examples/live-reconcile/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/e2e-live-reconcile-flux.sh`, `./examples/demo/e2e-live-reconcile-argo.sh`, `./test/checks/check-story-evidence.sh`
- Real live: `./examples/demo/e2e-live-reconcile-flux.sh`, `./examples/demo/e2e-live-reconcile-argo.sh`, `./examples/demo/e2e-connected-governed-reconcile-helm.sh`
- AI-first: --
- Notes: Runtime harness for WET->LIVE proof; source-side generator proof lives in paired examples.

### `ops-workflow`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/ops-workflow/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: `examples/demo/README.md#ai-work-platform-track`

### `scoredev-paas`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/scoredev-paas/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: --

### `springboot-paas`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/springboot-paas/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: `./examples/springboot-paas/verify-e2e.sh`, `./examples/springboot-paas/confighub-verify.sh`, `./examples/springboot-paas/bin/create-cluster`, `./examples/springboot-paas/bin/build-image`
- AI-first: --
- Notes: Standalone real-cluster proof: Kind cluster + ConfigHub worker + inventory-api HTTP verification.

### `swamp-automation`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/swamp-automation/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: `examples/README.md#ai--automation-patterns`

### `swamp-project`

- Source chain: --
- Connected mode: `./examples/swamp-project/demo-connected.sh`
- Connected release gate: `make ci-connected`, `./examples/demo/run-all-connected-lifecycles.sh`
- Real live: --
- AI-first: `examples/README.md#ai--automation-patterns`

