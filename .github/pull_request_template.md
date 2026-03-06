## Summary

- What user problem does this change solve?
- What behavior changed?

## Scope

- In scope:
- Out of scope:

## Deterministic Success Criteria (Required)

1. [ ] Criterion 1 (exact input -> exact expected output)
2. [ ] Criterion 2 (exact input -> exact expected output)
3. [ ] Criterion 3 (exact input -> exact expected output)

## Proof Matrix (Required Before Merge)

| Proof tier | Required? | Command(s) | Deterministic assertion |
|---|---|---|---|
| Unit | Yes | `go test ./...` | Core logic output is stable |
| Parity/Golden | If CLI/JSON/table output changed | `go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v` | Contract output is stable |
| Example proof | If user-visible flow changed | `go run ./cmd/cub-gen ...` | Docs/example output matches implementation |

## Commands Run

- [ ] `go build ./cmd/cub-gen`
- [ ] `go test ./...`
- [ ] `go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v`
- Additional commands:

## Graceful Degradation / Error Behavior

- Missing metadata behavior:
- Unsupported input behavior:
- How false confidence is avoided:

## Contract / Docs / Parity Updates

- [ ] `PARITY.md` updated (if contract changed)
- [ ] `README.md` and/or docs updated for user-visible changes
- [ ] `test/TEST-INVENTORY.md` updated if test surface changed
