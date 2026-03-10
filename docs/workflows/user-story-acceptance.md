# User Story Acceptance Matrix

This matrix maps user-story IDs to runnable acceptance scripts in this repo.

## Scope

- Local-first examples prove generator contracts and provenance semantics.
- Connected scripts prove ConfigHub API coupling for ingest/query paths.
- LIVE scripts prove reconciler behavior in cluster runtime (Flux/Argo).

## Story acceptance map

| Story | Current acceptance path | Primary script(s) | Evidence artifact |
|---|---|---|---|
| 1 | Connected import + query by `change_id` | `examples/demo/story-1-existing-repo-connected.sh` | `.tmp/story-1/*/*/story-1-summary.json` |
| 2 | Helm local lifecycle | `examples/demo/module-1-helm-import.sh` | `import` output + field-origin map |
| 3 | Score local lifecycle | `examples/demo/module-2-score-field-map.sh` | `import` output + inverse-edit map |
| 4 | Spring ownership | `examples/demo/module-3-spring-ownership.sh` | ownership annotations in import bundle |
| 5 | Bridge governance contract | `examples/demo/module-4-bridge-governance.sh` | decision and promotion state JSON |
| 6 | No-config-platform app-config governance | `examples/demo/module-5-no-config-platform.sh` | import/publish/verify artifacts |
| 7 | CI-centric connected API flow | `examples/demo/story-7-ci-api-flow-connected.sh` | `.tmp/ci-connected/*/*/story-7-summary.json` |
| 8 | Label/taxonomy evolution | `examples/demo/story-8-label-evolution-connected.sh` | `.tmp/story-8/*/*/story-8-summary.json` |
| 9 | Governed multi-repo wave | `examples/demo/story-9-multi-repo-wave-connected.sh` | `.tmp/waves/*/wave-summary.json` |
| 10 | Signed commit + branch protection proof | `examples/demo/story-10-signed-writeback-proof-connected.sh` | `.tmp/story-10/*/*/story-10-summary.json` |
| 11 | LIVE break-glass accept/revert proposal | `examples/demo/story-11-live-breakglass-proposal-connected.sh` | `.tmp/story-11/*/*/story-11-summary.json` |
| 12 | Unified human/CI/AI mutation model | `examples/demo/story-12-unified-actor-evidence.sh` | `.tmp/story-12/*/*/story-12-summary.json` |
| 13 | Live reconcile create/update/drift correction | `examples/demo/e2e-live-reconcile-flux.sh`, `examples/demo/e2e-live-reconcile-argo.sh` | script summary JSON |

## Runner groups

```bash
# local-first base proofs
./examples/demo/run-all-modules.sh
./examples/demo/run-all-confighub-lifecycles.sh

# connected lifecycle matrix
cub auth login
./examples/demo/run-all-connected-lifecycles.sh

# connected phase-3 stories (1,7,9,12)
cub auth login
./examples/demo/run-phase-3-connected-stories.sh

# connected phase-4 stories (8,10,11)
cub auth login
./examples/demo/run-phase-4-connected-stories.sh

# live reconciler proofs
./examples/demo/e2e-live-reconcile-flux.sh
./examples/demo/e2e-live-reconcile-argo.sh
```

## Notes

- Connected scripts require `cub auth login` or `CONFIGHUB_TOKEN` in CI.
- If ingest endpoint is unavailable at the configured base URL, connected scripts fail fast with remediation text.
