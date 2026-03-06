# Contract Drift Checklist (v0.2 Freeze)

Use this checklist whenever a PR intentionally changes a frozen CLI/help/JSON/table contract.

## Preconditions

1. Link the change to an issue that explicitly justifies why contract drift is required.
2. Confirm the drift is intentional (not incidental refactor fallout).

## Required update sequence

1. Update behavior in code/tests.
2. Regenerate goldens intentionally:

```bash
UPDATE_GOLDEN=1 go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand|TestGeneratorsGolden)' -count=1 -v
```

3. Verify full proof suite:

```bash
go test ./...
make ci
```

4. Update parity and docs in the same PR:
   - `PARITY.md` (contract lock references and proof artifacts)
   - User-facing docs (`README.md` and/or roadmap docs) if command UX changed
5. Include a concise before/after contract summary in the PR body.

## Merge gate

Do not merge a contract-drift PR unless all are true:

1. Goldens updated intentionally with `UPDATE_GOLDEN=1`.
2. `go test ./...` passes.
3. `make ci` passes.
4. `PARITY.md` and relevant docs are updated.
