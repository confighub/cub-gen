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
6. Added generator capability contract tests across all supported families.
7. Added `ably-config` app-config-only generator family (detect + import).
8. Added example repo fixture at `examples/ably-config/`.
9. Added gitops parity goldens for Ably discover/import:
   - `gitops-discover-ably.golden.json`
   - `gitops-import-ably.golden.json`
10. Extended publish parity goldens to all supported families (Helm/Score/Spring/Backstage/Ably) in both from-import and direct modes.
11. Extended verify-attestation linked parity goldens to include Backstage and Ably.
12. Extended path-mode smoke and bridge smoke tests to include Ably.
13. Added `ops-workflow` generator family (detect + import).
14. Added example repo fixture at `examples/ops-workflow/`.
15. Added gitops parity goldens for Ops workflow discover/import:
   - `gitops-discover-ops.golden.json`
   - `gitops-import-ops.golden.json`
16. Extended publish parity goldens and verify-attestation linked goldens to include Ops workflow.
17. Extended path-mode smoke and bridge smoke tests to include Ops workflow.
18. Added a shared generator family registry (`internal/registry`) for cross-cutting metadata (profile/resource mapping/capabilities) to reduce multi-file switch edits.
19. Extended the family registry to own DRY input role classification and role-owner mapping, removing importer-local switch logic for these semantics.
20. Extended the family registry to own family-aware input schema resolution, removing importer-local schema switch logic.
21. Updated `gitops` help to derive supported resource kinds from the family registry, avoiding manual help-text edits when families are added.
22. Extended the family registry to own wet-manifest target templates, removing importer-local wet target switch logic.
23. Added `cub-gen generators` command (table + JSON + help golden contracts) to expose registry-backed supported family inventory.
24. Extended `cub-gen generators` with deterministic filters (`--kind`, `--profile`, `--capability`) and locked filtered output contracts.

## v0.2 preview invariants

1. Deterministic output remains mandatory.
2. No implicit deploy/reconcile path in `cub-gen`.
3. Flux/Argo continue as runtime reconciliation engines.
4. Contract changes require golden lock updates plus explicit docs/test inventory updates.

## Next slices toward v0.2

1. Extend the family registry pattern further to cover additional family-level semantics (for example hint strategies and provenance-field templates).
2. Keep generator-family capability metadata contract tests updated as families are added.
3. Keep bridge artifacts (`publish/verify/attest`) symmetric across all supported families.
