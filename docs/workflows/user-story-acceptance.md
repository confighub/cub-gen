# User Story Acceptance Matrix

This matrix maps user-story IDs to runnable acceptance scripts in this repo.

## Scope

- Local-first examples prove generator contracts and provenance semantics.
- Connected scripts prove ConfigHub API coupling for ingest/query paths.
- LIVE scripts prove reconciler behavior in cluster runtime (Flux/Argo).

## Segment priority (v0.2)

| Rank | Segment | Definition | Primary adoption outcome |
|---|---|---|---|
| 1 | (b) Brownfield Spring Boot/Helm users | Teams with existing app-platform patterns already in GitOps repos. | Cut troubleshooting and ownership ambiguity without repo redesign. |
| 2 | (c) AI + developer collaborative workflows | Human + agent teams proposing and reviewing config/workflow changes together. | One governed change/evidence loop shared by human and agent actors. |
| 3 | (a) Greenfield platform builders | Teams designing a new internal platform contract from scratch. | Ship platform operations + constraints quickly with clear boundaries. |
| 4 | (d) AI-only guarded pilot | Limited-scope autonomous workflows under strict policy controls. | Prove safe autonomy with explicit allowed scope and rollback hooks. |

## Story acceptance map

| Story | Primary segment | Secondary segment | Current acceptance path | Primary script(s) | Proof artifact |
|---|---|---|---|---|---|
| 1 | (b) Brownfield Spring Boot/Helm users | (c) AI + developer collaborative workflows | Connected import + query by `change_id` | `examples/demo/story-1-existing-repo-connected.sh` | `.tmp/story-1/*/*/story-1-summary.json` |
| 2 | (b) Brownfield Spring Boot/Helm users | (a) Greenfield platform builders | Helm local lifecycle | `examples/demo/module-1-helm-import.sh` | `import` output + field-origin map |
| 3 | (b) Brownfield Spring Boot/Helm users | (c) AI + developer collaborative workflows | Score local lifecycle | `examples/demo/module-2-score-field-map.sh` | `import` output + inverse-edit map |
| 4 | (b) Brownfield Spring Boot/Helm users | (a) Greenfield platform builders | Spring ownership | `examples/demo/module-3-spring-ownership.sh` | ownership annotations in import bundle |
| 5 | (a) Greenfield platform builders | (b) Brownfield Spring Boot/Helm users | Bridge governance contract | `examples/demo/module-4-bridge-governance.sh` | decision and promotion state JSON |
| 6 | (b) Brownfield Spring Boot/Helm users | (a) Greenfield platform builders | No-config-platform app-config governance | `examples/demo/module-5-no-config-platform.sh` | import/publish/verify artifacts |
| 7 | (b) Brownfield Spring Boot/Helm users | (c) AI + developer collaborative workflows | CI + agent tool-call connected API flow | `examples/demo/story-7-ci-api-flow-connected.sh`, `examples/demo/story-7-agent-tool-call-connected.sh` | `.tmp/ci-connected/*/*/story-7-summary.json`, `.tmp/agent-tool-call/*/*/story-7-agent-summary.json` |
| 8 | (b) Brownfield Spring Boot/Helm users | (a) Greenfield platform builders | Label/taxonomy evolution | `examples/demo/story-8-label-evolution-connected.sh` | `.tmp/story-8/*/*/story-8-summary.json` (includes backend changeset/query evidence) |
| 9 | (b) Brownfield Spring Boot/Helm users | (c) AI + developer collaborative workflows | Governed multi-repo wave | `examples/demo/story-9-multi-repo-wave-connected.sh` | `.tmp/waves/*/wave-summary.json` |
| 10 | (b) Brownfield Spring Boot/Helm users | (c) AI + developer collaborative workflows | Signed commit + branch protection proof | `examples/demo/story-10-signed-writeback-proof-connected.sh` | `.tmp/story-10/*/*/story-10-summary.json` |
| 11 | (c) AI + developer collaborative workflows | (b) Brownfield Spring Boot/Helm users | LIVE break-glass accept/revert proposal | `examples/demo/story-11-live-breakglass-proposal-connected.sh` | `.tmp/story-11/*/*/story-11-summary.json` (includes backend proposal changesets + query hits) |
| 12 | (c) AI + developer collaborative workflows | (d) AI-only guarded pilot | Unified human/CI/AI mutation model | `examples/demo/story-12-unified-actor-evidence.sh` | `.tmp/story-12/*/*/story-12-summary.json` |
| 13 | (b) Brownfield Spring Boot/Helm users | (a) Greenfield platform builders | Live reconcile create/update/drift correction | `examples/demo/e2e-live-reconcile-flux.sh`, `examples/demo/e2e-live-reconcile-argo.sh` | script summary JSON |

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
export APP_PR_REPO=owner/app-repo
export APP_PR_NUMBER=123
export PROMOTION_PR_REPO=owner/promotion-repo
export PROMOTION_PR_NUMBER=456
./examples/demo/run-phase-4-connected-stories.sh

# live reconciler proofs
./examples/demo/e2e-live-reconcile-flux.sh
./examples/demo/e2e-live-reconcile-argo.sh
```

## Notes

- Connected scripts require `cub auth login` or `CONFIGHUB_TOKEN` in CI.
- Story 10 requires real GitHub PR coordinates (`APP_PR_REPO`, `APP_PR_NUMBER`, `PROMOTION_PR_REPO`, `PROMOTION_PR_NUMBER`) and `gh` auth (`GH_TOKEN`/`GITHUB_TOKEN` or `gh auth login`).
- If ingest endpoint is unavailable at the configured base URL, connected scripts fail fast with remediation text.
- AI-only pilot lanes are restricted by [ai-only-guardrails.md](ai-only-guardrails.md) and enforced in CI by `test/checks/check-ai-only-scope.sh`.
