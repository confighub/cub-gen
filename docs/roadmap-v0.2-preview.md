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
7. Added `no-config-platform` app-config-only generator family (detect + import).
8. Added example repo fixture at `examples/just-apps-no-platform-config/`.
9. Added gitops parity goldens for No Config Platform discover/import:
   - `gitops-discover-no-config-platform.golden.json`
   - `gitops-import-no-config-platform.golden.json`
10. Extended publish parity goldens to all supported families (Helm/Score/Spring/Backstage/No Config Platform) in both from-import and direct modes.
11. Extended verify-attestation linked parity goldens to include Backstage and No Config Platform.
12. Extended path-mode smoke and bridge smoke tests to include No Config Platform.
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
25. Refactored input schema inference to be registry-driven (`RoleSchemaRefs`) instead of importer-local switch logic, preserving existing schema outputs while reducing per-family branching.
26. Expanded `generators` parity contracts to lock profile-filter, combined-filter, and empty-match JSON outputs.
27. Refactored generator hint default paths/labels into registry-driven `HintDefaults`, removing hardcoded importer defaults while preserving output behavior.
28. Refactored provenance `field_origin_map.transform` labels to be registry-driven (`FieldOriginTransform` / `FieldOriginOverlayTransform`) instead of importer string literals.
29. Expanded `generators` table-mode parity contracts to lock deterministic filtered and empty-match outputs.
30. Refactored inverse patch reason strings into registry-driven templates (`InversePatchReasons`) with placeholder rendering for family path hints.
31. Refactored inverse edit hint strings into registry-driven templates (`InverseEditHints`) with placeholder rendering for family path hints and overlay-specific messaging.
32. Improved `generators --help` to render supported kind/profile/capability values directly from the registry for self-discoverable filter usage.
33. Added comma-separated multi-value filtering support for `generators` (`--kind`, `--profile`, `--capability`) with parity contracts for multi-match outputs.
34. Added table-mode multi-filter parity contracts for `generators` (kind and capability comma-list flows) to keep human-readable output locked alongside JSON.
35. Added optional strict filter validation (`--strict-filters`) for `generators` to fail fast on unknown kind/profile/capability values.
36. Updated `README.md` generator inventory examples to include multi-value and strict-filter command flows for faster adoption.
37. Refactored provenance `field_origin_map.confidence` values into registry-driven templates (`FieldOriginConfidences`) to remove importer-local confidence literals while preserving output behavior.
38. Refactored inverse edit pointer ownership/confidence defaults into registry-driven templates (`InversePointerTemplates`) to remove importer-local policy literals while preserving output behavior.
39. Added `cub-gen generators --json --details` to expose registry-backed policy/provenance templates (`inverse_patch_templates`, `inverse_pointer_templates`, `field_origin_confidences`) for transparent family introspection.
40. Refactored rendered object lineage definitions into registry-driven templates (`RenderedLineageTemplates`), removing importer-local per-family lineage literals while preserving output behavior.
41. Refactored Helm provenance source-path semantics to use registry-driven role/hint metadata (`chart_role`, `values_role`, `primary_values_path`) with deterministic values ordering and preserved parity outputs.
42. Added generator metadata conformance tests to enforce cross-surface consistency between registry specs, `generators --json`, and `generators --json --details` outputs.
43. Added a single 10-minute Flux/Argo/Helm adoption path in `README.md`, with copy/paste commands and explicit parity boundary (`matched|partial|deferred`) language.
44. Cut release `v0.2-preview.1` from green `main` with release notes covering parity lock, supported families, boundaries, known limits, and adoption references.
45. Added strict schema validation gates for `GeneratorContract`, `ProvenanceRecord`, and `InverseTransformPlan` with embedded JSON schemas and importer runtime enforcement.
46. Enforced `no-triple-no-governed-import` gate with deterministic blocker errors for missing/cardiinality-mismatched contract triples and locked importer error output tests.
47. Added cross-family triple conformance fixtures for Helm/Score/Spring/Backstage/No Config Platform/Ops with deterministic re-run checks and registry-kind coverage enforcement.
48. Added explicit bridge symmetry matrix gate in CI (`TestBridgeSymmetryMatrix` + `make test-bridge-symmetry`) to enforce `publish -> verify -> attest -> verify-attestation` family coverage across all registry kinds.
49. Published canonical triple + storage boundary docs mapping schemas, runtime gates, and Git (DRY linkage) vs ConfigHub (WET governance) responsibilities.
50. Completed Sprint 2 Go/No-Go gate with documented `GO` outcome and opened post-gate bridge backlog issues (`#95`-`#97`).
51. Added first bridge ingest path (`internal/bridge`) to map `change-bundle` artifacts into ConfigHub governed WET ingest payloads with idempotency (`change_id + bundle_digest`) and duplicate-safe HTTP `409` handling.
52. Added governed decision-state bridge contract and runtime transitions (`INGESTED -> ATTESTED -> ALLOW|ESCALATE|BLOCK`) with explicit authority gates, attestation linkage validation, and query-by-`change_id` support in `internal/bridge`.
53. Added PR<->MR linkage and upstream DRY promotion guardrail state machine with explicit separate-review enforcement before protected platform-DRY merge.
54. Expanded `cub-gen generators --json --details` with full policy/provenance template fields (`hint_defaults`, inverse reasons/hints, input role rules/defaults, role ownership defaults, wet target templates, rendered lineage templates, and field-origin transform labels) and updated parity golden contracts.
55. Added `cub-gen generators --markdown` (+ `--details`) so platform owners can inspect family policy/provenance templates in GitHub-friendly Markdown without `jq` pipelines.

## v0.2 preview invariants

1. Deterministic output remains mandatory.
2. No implicit deploy/reconcile path in `cub-gen`.
3. Flux/Argo continue as runtime reconciliation engines.
4. Contract changes require golden lock updates plus explicit docs/test inventory updates.

## Next slices toward v0.2

1. Extend the family registry pattern further to cover additional family-level semantics (for example hint strategies and provenance-field templates).
2. Keep generator-family capability metadata contract tests updated as families are added.
3. Keep bridge artifacts (`publish/verify/attest`) symmetric across all supported families.

Execution board reference:

1. See `docs/execution-board-sprint-1-and-2.md` for sequenced Sprint 1 and Sprint 2 issue plan and the mandatory Go/No-Go break gate.
