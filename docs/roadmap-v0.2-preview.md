# cub-gen v0.2 Preview Roadmap

Status date: 2026-03-06

## Goal

Move from v0.1 "first 3 examples" hardening to a broader generator family surface while keeping deterministic local-first contracts.

## v0.2 preview status (in progress)

Implemented in this preview slice:

1. Added `backstage-idp` generator family (detect + import).
2. Added example repo fixture at `examples/backstage-idp/`.
3. Extended command behavior tests (publish/verify/attest/verify-attestation) to include Backstage alongside Helm/Score/Spring.
4. Added gitops parity goldens for Backstage discover/import:
   - `gitops-discover-backstage.golden.json`
   - `gitops-import-backstage.golden.json`
5. Extended path-mode smoke and bridge smoke tests to include Backstage.

## v0.2 preview invariants

1. Deterministic output remains mandatory.
2. No implicit deploy/reconcile path in `cub-gen`.
3. Flux/Argo continue as runtime reconciliation engines.
4. Contract changes require golden lock updates plus explicit docs/test inventory updates.

## Next slices toward v0.2

1. Add one app-config-only generator family (e.g. config-provider style DRY source).
2. Add one ops/workflow-style generator family.
3. Define a stable extension surface for adding generator families without large switch edits.
4. Add parity tests for generator-family capability metadata.
5. Keep bridge artifacts (`publish/verify/attest`) symmetric across all supported families.
