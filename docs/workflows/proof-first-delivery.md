# Proof-First Delivery (cub-gen)

This is the inherited delivery model from cub-scout, trimmed for cub-gen scope.

## Mandatory sequence (per task)

1. Define task boundary and non-goals.
2. Define contract impact (CLI/JSON/parity).
3. Define proof matrix before implementation.
4. Define example/fixture impact.
5. Implement smallest vertical slice.
6. Run proofs immediately (run #1).
7. Patch only failures.
8. Re-run proofs until stable (>=2 consecutive passes).
9. Update docs/parity/examples.
10. Mark done only after final pass post-doc updates.

If the task changes a frozen contract surface, run the additional checklist:

1. `docs/testing/contract-drift-checklist.md`

## Proof matrix template

| Proof tier | Required? | Command | Assertion |
|---|---|---|---|
| Unit | Yes | `go test ./...` | deterministic logic/output |
| Parity/Golden | For CLI behavior | `go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v` | stable command contract |
| Example proof | For user-visible changes | `go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v` | docs/example matches behavior |

## Completion rule

No task is complete on one pass. Require:

1. at least 2 consecutive passing proof runs,
2. one final pass after docs/example updates.
