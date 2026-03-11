# Action Plan v0.1 Execution Checklist

Status date: 2026-03-11
Owner posture: successful OSS project owner (developer-first, low-friction adoption)

Goal: make `DRY -> WET -> LIVE` feel native for app/AI developers, with verification and attestation as default safety loops.

## Status snapshot

| Item | Status | Evidence |
|---|---|---|
| 1. Lock language and scope | Done | `change preview/run/explain` in [README](../../README.md), no public `fastpath` positioning |
| 2. Prioritize user segments | Done | ranked segment table in [docs/workflows/user-story-acceptance.md](../workflows/user-story-acceptance.md) |
| 3. Canonical prompt-as-DRY story | Done | [docs/workflows/prompt-as-dry.md](../workflows/prompt-as-dry.md), runnable scripts in `examples/demo/prompt-as-dry-*.sh` |
| 4. First-class CLI contract | Done | [docs/contracts/change-cli-v1.md](../contracts/change-cli-v1.md) |
| 5. First-class API contract | Done | [docs/contracts/change-api-v1.md](../contracts/change-api-v1.md) + schemas in `docs/contracts/schemas/` |
| 6. Thin MVP over existing pipeline | Done | first-class `change preview/run/explain` in `cmd/cub-gen/main.go` |
| 7. Prove in real invocation contexts | Partial | terminal + CI are wired; agent/tool-call path exists via adapter but lifecycle proof needs same `change_id` binding |
| 8. Gate AI-only rollout | Not started | policy/matrix + enforcement hooks not yet merged |

## Concrete issue titles (next filing set)

### P0 (close remaining adoption gaps)

1. `feat(cli): add change explain --change-id mode for stable lifecycle drilldown`
- Why: item 7 success requires one `change_id` lifecycle across terminal, CI, and agent paths.
- Done when: `change explain --change-id <id>` reads prior artifacts/state and returns explanation without minting a new lifecycle.

2. `examples(agent): add runnable tool-call path proving same change_id across preview/run/explain`
- Why: complete item 7 with explicit agent proof.
- Done when: one script under `examples/demo/` runs adapter request/response flow and emits a single summary showing shared `change_id`.

3. `policy(ai-only): publish allowed-scope matrix + mandatory rollback hooks`
- Why: item 8 direct deliverable.
- Done when: new policy doc defines in-scope/out-of-scope mutations, required approvals, rollback proposal path, and hard deny list.

4. `ci(policy): fail AI-only demo lanes on out-of-scope mutations`
- Why: ensure AI-only cannot bypass governance.
- Done when: CI check blocks when AI-only scripts violate allowed-scope matrix rules.

### P1 (hardening)

5. `docs(workflows): add AI-only guardrails section to prompt-as-dry and user-story acceptance`
- Why: keep docs synchronized with safety policy.
- Done when: both docs reference the same enforcement matrix and failure behavior.

## PR sequence (recommended)

1. PR-A: `feat(cli): add change explain --change-id mode`
2. PR-B: `examples(agent): add tool-call lifecycle proof script + summary artifact`
3. PR-C: `docs(policy): AI-only allowed-scope matrix and rollback hooks`
4. PR-D: `ci(policy): enforce AI-only scope gate in CI`
5. PR-E: `docs(sync): wire policy references into prompt-as-dry and user-story acceptance`

## Acceptance checks (project-level)

Run locally:

```bash
make ci-local
./examples/demo/app-ai-change-run.sh ./examples/scoredev-paas
./examples/demo/story-7-ci-api-flow-connected.sh
./examples/demo/<agent-tool-call-proof>.sh
```

Release gate for this action plan:

1. Item 7 is marked `Done` with one shared `change_id` proof artifact.
2. Item 8 is marked `Done` with policy + CI enforcement merged.
