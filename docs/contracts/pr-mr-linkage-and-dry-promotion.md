# PR<->MR Linkage and Upstream DRY Promotion Contract

This contract defines the bidirectional handoff between Git PR workflow and ConfigHub MR workflow for governed promotion.

## Goal

Use one shared `change_id` to correlate:

1. Git PR (code/config change in app overlay DRY)
2. ConfigHub MR (governed WET decision flow)
3. Upstream platform DRY promotion PR/MR (reusable default promotion)

## Correlation model

Canonical review-link record (`cub.confighub.io/pr-mr-promotion-flow/v1`) includes:

1. `change_id`
2. `git_pr` (`repo`, `number`, `url`, optional `commit_sha`)
3. `confighub_mr` (`id`, `url`, optional status)
4. `status` (`OPEN|MERGED`)
5. `updated_at`

This enables PR<->MR cross-navigation and deterministic audit joins.

## Promotion flow (recommended path)

1. App team ships feature + config change in app DRY (bounded app overlay) and opens Git PR.
2. ConfigHub renders/evaluates, posts evidence, and opens/updates MR in ConfigHub.
3. Platform engineer merge-approves the app change in ConfigHub MR.
4. ConfigHub records governed decision (`ALLOW | ESCALATE | BLOCK`).
5. On `ALLOW` + successful rollout, ConfigHub opens promotion PR/MR to upstream platform DRY/base when reusable.
6. Platform maintainers perform separate platform review and merge upstream promotion PR in Git.
7. App overlay is reduced/removed to avoid long-lived drift.

## Guardrails (anti-pattern blockers)

Blocked by contract:

1. Promotion merge without prior `ALLOW`.
2. Promotion merge without separate platform review approval.
3. Direct auto-write into protected upstream platform DRY without a reviewable PR/MR.

## Implementation anchors

1. Correlation + promotion state machine: `internal/bridge/workflow.go`
2. Gate tests and happy-path proof: `internal/bridge/workflow_test.go`
