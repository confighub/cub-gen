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
| 7. Prove in real invocation contexts | Done | terminal + CI + agent tool-call proof with shared `change_id` (`story-7-ci-api-flow-connected.sh`, `story-7-agent-tool-call-connected.sh`) |
| 8. Gate AI-only rollout | Done | `docs/workflows/ai-only-guardrails.md` + guardrail enforcement in prompt-as-dry scripts + `test/checks/check-ai-only-scope.sh` CI gate |

## Concrete issue titles (next filing set)

### P0 (close remaining adoption gaps)

1. `feat(cli): add change explain --change-id mode for stable lifecycle drilldown`
- Why: item 7 success requires one `change_id` lifecycle across terminal, CI, and agent paths.
- Status: done via `cmd/cub-gen/main.go` and tests in `cmd/cub-gen/change_command_test.go`.

2. `examples(agent): add runnable tool-call path proving same change_id across preview/run/explain`
- Why: complete item 7 with explicit agent proof.
- Status: done via `examples/demo/story-7-agent-tool-call-connected.sh`.

3. `policy(ai-only): publish allowed-scope matrix + mandatory rollback hooks`
- Why: item 8 direct deliverable.
- Status: done via `docs/workflows/ai-only-guardrails.md` and `examples/demo/lib/ai-only-guardrails.sh`.

4. `ci(policy): fail AI-only demo lanes on out-of-scope mutations`
- Why: ensure AI-only cannot bypass governance.
- Status: done via `test/checks/check-ai-only-scope.sh` wired into `make ci-local`.

### P1 (hardening)

5. `docs(workflows): add AI-only guardrails section to prompt-as-dry and user-story acceptance`
- Why: keep docs synchronized with safety policy.
- Status: done; enforced by `test/checks/check-ai-only-scope.sh` (requires both docs to reference `ai-only-guardrails.md`).

## PR sequence (recommended)

Completed:

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
./examples/demo/story-7-agent-tool-call-connected.sh
```

Release gate for this action plan:

1. Item 7 is marked `Done` with one shared `change_id` proof artifact.
2. Item 8 is marked `Done` with policy + CI enforcement merged.
