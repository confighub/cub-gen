# Plan: Complete All User Stories

**Status**: Proposed  
**Date**: 2026-03-09  
**Owner**: Product + Platform Engineering

## Goal

Move user stories `1..13` to fully met with reproducible proof, not narrative-only claims.

Current baseline:

- Met/strong: `2, 3, 4, 5, 6, 13`
- Partial: `1, 7, 9, 12`
- Deferred: `8, 10, 11`

## Workstream A: Close partial stories

### Story 1

As a Helm/Kustomize Git user, import an existing repo and query provenance without refactoring.

Deliverables:

1. Deterministic import from repo/ref path (not fixture-only assumptions).
2. Query surface showing provenance by `change_id`.
3. Example script showing "existing repo -> immediate explainability."

Exit proof:

1. New integration test proving import/query on existing-style repo.
2. Demo command transcript stored as docs artifact.

### Story 7

As a CI-centric team, use ConfigHub mutation/query APIs from pipelines without path scripting.

Deliverables:

1. CI-first bridge workflow with tokened ingest/decision/query.
2. GitHub Actions example for non-interactive pipeline usage.
3. API payload examples aligned with current CLI output contracts.

Exit proof:

1. CI workflow passes in repo.
2. Docs example runs end-to-end with deterministic outputs.

### Story 12

As compliance reviewer, see human/CI/AI mutations in one decision+attestation model.

Deliverables:

1. Unified actor model (`human`, `ci-bot`, `ai-agent`) in decision/attestation records.
2. Single `change_id` linkage across bundle, decision, attestation, and promotion artifacts.
3. Query/readout view that groups evidence by `change_id`.

Exit proof:

1. Golden tests for mixed-actor records.
2. One example showing all actor types on same change chain.

## Workstream B: Close deferred stories

### Story 8

As platform owner, evolve labels/taxonomy without repo surgery or query breakage.

Deliverables:

1. Versioned label schema with migration layer.
2. Backward-compatible query behavior for prior label versions.
3. Docs playbook for label evolution in production.

Exit proof:

1. Compatibility tests across old/new label schemas.

### Story 10

As security lead, prove branch protections/approvals/signed-commit controls were preserved.

Deliverables:

1. Evidence model for VCS protection metadata.
2. Decision gate requires protection evidence for write-back flows.
3. Audit view shows preserved controls per change.

Exit proof:

1. Failing test when protection evidence is missing.
2. Passing test with complete protection evidence.

### Story 11

As on-call engineer, convert LIVE break-glass change into explicit `accept` or `revert` proposal.

Deliverables:

1. Drift proposal command producing explicit `accept/revert` plan.
2. No silent overwrite path from LIVE to DRY.
3. Drift class attached (`stale-render`, `overlay-drift`, `manual-live-mutation`, `unknown`).

Exit proof:

1. End-to-end test for break-glass -> proposal -> decision chain.
2. Example demo script in `examples/demo`.

## Delivery sequence

1. Sprint 1: `1, 7, 12`
2. Sprint 2: `9, 11`
3. Sprint 3: `8, 10`
4. Sprint 4: hardening, docs, and matrix update to all-met

## Definition of done

1. Every story has a deterministic acceptance test.
2. Every story has an example or demo entry command.
3. Story matrix in README and demo docs updated from partial/deferred to met.
4. No claim remains without proof artifact.
