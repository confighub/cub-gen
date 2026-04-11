# Example Truth Matrix

Generated from repo structure, source-side tests, the connected smoke lane, and live-proof harness scripts. Do not edit by hand; regenerate with `go run ./tools/example-truth-matrix --format markdown`.

## Summary

- Featured examples: `12`
- Generator fixtures: `8`
- Source-chain verified: `8`
- Connected mode present: `12`
- Connected smoke gated: `2`
- Real live proof: `none=9`, `paired-harness=1`, `standalone=2`
- AI-first surface: `none=3`, `partial=2`, `explicit=7`

## Matrix

| Example | Generator fixture | Source chain verified | Connected mode | Connected smoke lane | Real live proof | AI-first surface | Tracking issues |
|---|---|---|---|---|---|---|---|
| `ai-ops-paas` | no | no | yes | no | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `backstage-idp` | yes | yes | yes | no | `none` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `c3agent` | yes | yes | yes | no | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183), [#216](https://github.com/confighub/cub-gen/issues/216) |
| `confighub-actions` | no | no | yes | no | `none` | `partial` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `helm-paas` | yes | yes | yes | yes | `paired-harness` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#177](https://github.com/confighub/cub-gen/issues/177), [#183](https://github.com/confighub/cub-gen/issues/183), [#187](https://github.com/confighub/cub-gen/issues/187) |
| `just-apps-no-platform-config` | yes | yes | yes | no | `none` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `live-reconcile` | no | no | yes | no | `standalone` | `none` | [#173](https://github.com/confighub/cub-gen/issues/173), [#181](https://github.com/confighub/cub-gen/issues/181), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `ops-workflow` | yes | yes | yes | no | `none` | `partial` | [#173](https://github.com/confighub/cub-gen/issues/173), [#180](https://github.com/confighub/cub-gen/issues/180), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `scoredev-paas` | yes | yes | yes | no | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#178](https://github.com/confighub/cub-gen/issues/178), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `springboot-paas` | yes | yes | yes | yes | `standalone` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#179](https://github.com/confighub/cub-gen/issues/179), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `swamp-automation` | yes | yes | yes | no | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#180](https://github.com/confighub/cub-gen/issues/180), [#183](https://github.com/confighub/cub-gen/issues/183) |
| `swamp-project` | no | no | yes | no | `none` | `explicit` | [#173](https://github.com/confighub/cub-gen/issues/173), [#180](https://github.com/confighub/cub-gen/issues/180), [#183](https://github.com/confighub/cub-gen/issues/183) |

## Proof References

### `ai-ops-paas`

- Source chain: --
- Connected mode: `./examples/ai-ops-paas/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: `examples/README.md#ai--automation-patterns`

### `backstage-idp`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/backstage-idp/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: --

### `c3agent`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/c3agent/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: `examples/README.md#ai--automation-patterns`

### `confighub-actions`

- Source chain: --
- Connected mode: `./examples/confighub-actions/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: `examples/demo/README.md#ai-work-platform-track`

### `helm-paas`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/helm-paas/demo-connected.sh`
- Connected smoke lane: `make ci-connected`, `./examples/demo/run-connected-smoke.sh`
- Real live: `./examples/helm-paas/demo-runtime.sh`, `./examples/demo/e2e-connected-governed-reconcile-helm.sh`, `./examples/live-reconcile/demo-local.sh`
- AI-first: `./examples/helm-paas/AI_START_HERE.md`, `./examples/helm-paas/prompts.md`, `./examples/helm-paas/contracts.md`
- Notes: Real LIVE proof is exposed through an example-owned helm-paas wrapper, but still uses the shared live-reconcile harness under the hood.

### `just-apps-no-platform-config`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/just-apps-no-platform-config/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: --

### `live-reconcile`

- Source chain: --
- Connected mode: `./examples/live-reconcile/demo-connected.sh`
- Connected smoke lane: --
- Real live: `./examples/demo/e2e-live-reconcile-flux.sh`, `./examples/demo/e2e-live-reconcile-argo.sh`, `./examples/demo/e2e-connected-governed-reconcile-helm.sh`
- AI-first: --
- Notes: Runtime harness for WET->LIVE proof; source-side generator proof lives in paired examples.

### `ops-workflow`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/ops-workflow/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: `examples/demo/README.md#ai-work-platform-track`

### `scoredev-paas`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/scoredev-paas/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: `./examples/scoredev-paas/AI_START_HERE.md`, `./examples/scoredev-paas/prompts.md`, `./examples/scoredev-paas/contracts.md`
- Notes: Score now has example-owned local governed proof via ./examples/scoredev-paas/demo-governed-workload.sh for ALLOW versus ESCALATE, even though standalone live-cluster proof is still open.

### `springboot-paas`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/springboot-paas/demo-connected.sh`
- Connected smoke lane: `make ci-connected`, `./examples/demo/run-connected-smoke.sh`
- Real live: `./examples/springboot-paas/verify-e2e.sh`, `./examples/springboot-paas/confighub-verify.sh`, `./examples/springboot-paas/bin/create-cluster`, `./examples/springboot-paas/bin/build-image`
- AI-first: `./examples/springboot-paas/AI_START_HERE.md`, `./examples/springboot-paas/prompts.md`, `./examples/springboot-paas/contracts.md`
- Notes: Standalone real-cluster proof: Kind cluster + ConfigHub worker + inventory-api HTTP verification.

### `swamp-automation`

- Source chain: `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v`
- Connected mode: `./examples/swamp-automation/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: `./examples/swamp-automation/AI_START_HERE.md`, `./examples/swamp-automation/prompts.md`, `./examples/swamp-automation/contracts.md`

### `swamp-project`

- Source chain: --
- Connected mode: `./examples/swamp-project/demo-connected.sh`
- Connected smoke lane: --
- Real live: --
- AI-first: `examples/README.md#ai--automation-patterns`

