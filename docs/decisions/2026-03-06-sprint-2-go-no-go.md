# Sprint 2 Go/No-Go Decision (Pre-Bridge Coupling)

Date: 2026-03-06
Status: Final
Decision owner: cub-gen maintainers

## Scope

This decision is the mandatory Sprint 2 gate before any ConfigHub bridge-coupled implementation.

Gate target (from execution board):

1. Determinism and parity lock integrity.
2. Contract-triple conformance across all supported families.
3. Bridge artifact symmetry coverage.
4. Adoption clarity for Flux/Argo/Helm users.
5. Git (DRY linkage) vs ConfigHub (WET governance) boundary clarity.

## Evidence (landed Sprint 2 issues)

1. `#77` closed via PR `#90`: schema validation gate for contract triple.
2. `#78` closed via PR `#91`: no-triple-no-governed-import enforcement + deterministic blocker errors.
3. `#79` closed via PR `#92`: cross-family triple conformance fixtures.
4. `#80` closed via PR `#93`: explicit bridge symmetry matrix CI gate.
5. `#81` closed via PR `#94`: canonical triple + storage boundary docs.

## Gate checklist

### 1) Determinism and parity lock integrity

Result: PASS

Evidence:

1. `PARITY.md` lock remains active.
2. Required CI gates pass (`Build + Unit`, `CLI Parity`).
3. `make ci` passes with deterministic bridge symmetry matrix gate.

### 2) Contract-triple conformance across all supported families

Result: PASS

Evidence:

1. Embedded schemas + runtime validator in `internal/contracts/`.
2. Governed import block on missing/invalid triple.
3. Cross-family conformance fixtures for Helm/Score/Spring/Backstage/No Config Platform/Ops in `internal/contracts/triple_conformance_fixtures_test.go`.

### 3) Bridge artifact symmetry coverage

Result: PASS

Evidence:

1. Explicit matrix source in `cmd/cub-gen/examples_matrix_test.go`.
2. Symmetry flow coverage (`publish -> verify -> attest -> verify-attestation`) in `TestExamplesPathModeBridgeFlow`.
3. CI parity job runs `make test-bridge-symmetry`.

### 4) Adoption clarity for Flux/Argo/Helm users

Result: PASS

Evidence:

1. 10-minute adoption section in `README.md`.
2. Clear boundary wording (`matched|partial|deferred`) aligned with `PARITY.md`.
3. Copy/paste flow validated against `examples/helm-paas`.

### 5) Git DRY linkage vs ConfigHub WET governance boundary

Result: PASS

Evidence:

1. Canonical boundary doc: `docs/contracts/canonical-triple-and-storage-boundary.md`.
2. Boundary remains explicit:
   - Git: DRY source intent + linkage receipts.
   - ConfigHub: WET governance state + policy/attestation authority.
   - Flux/Argo: WET -> LIVE reconciliation.

## Decision

Outcome: GO

Rationale:

1. All Sprint 2 gate checklist items are met with merged code/docs and passing CI.
2. Contract triple and bridge symmetry controls are explicit and test-enforced.
3. Storage boundary is documented and aligned to current architecture.

## Guardrails for post-gate bridge work

1. Do not introduce implicit deploy/reconcile behavior in `cub-gen`.
2. Keep triple schema changes behind explicit contract tests and docs updates.
3. Preserve deterministic output constraints for import and bridge artifacts.
4. Keep Flux/Argo as runtime reconciler; bridge work targets governance integration only.

## Follow-on backlog (opened)

1. `#95` [Post-S2-01] bridge ingest: change-bundle -> ConfigHub governed WET artifact.
2. `#96` [Post-S2-02] bridge decision state: triple artifacts -> `ALLOW | ESCALATE | BLOCK` + attestation linkage.
3. `#97` [Post-S2-03] bridge workflow: Git PR <-> ConfigHub MR linkage + upstream DRY promotion flow.
