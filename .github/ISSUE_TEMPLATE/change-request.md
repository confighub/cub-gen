---
name: Change Request
description: Propose a feature or behavior change with proof-first acceptance criteria
title: "feat: "
labels: ["enhancement"]
assignees: []
---

## Problem

- What user problem are we solving?
- Why now?

## Scope

- In scope:
- Out of scope:

## Deterministic Success Criteria (Required Before Coding)

1. Criterion 1 (exact input -> exact expected output)
2. Criterion 2 (exact input -> exact expected output)
3. Criterion 3 (exact input -> exact expected output)

## Proof Matrix (Required Before Coding)

| Proof tier | Required? | Planned command(s) | Assertion |
|---|---|---|---|
| Unit | Yes | `go test ./...` | deterministic logic result |
| Parity/Golden | If output contract changes | `go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v` | stable CLI contract |
| Example proof | If user-visible behavior changes | `go run ./cmd/cub-gen ...` | docs/example aligned |

## Graceful Degradation

- Missing metadata behavior:
- Unknown/unsupported input behavior:
- Safety fallback behavior:

## Implementation Notes

- Key files/packages expected to change:
- Contract impact (`PARITY.md`):
- Risks:

## Definition of Done

- [ ] Required tests implemented and passing
- [ ] Proof matrix evidence added to PR
- [ ] Docs/examples updated
- [ ] Contract changes documented
