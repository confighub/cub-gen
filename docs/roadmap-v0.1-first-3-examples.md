# cub-gen v0.1 Roadmap: First 3 Examples

Status date: 2026-03-06

## Purpose

Deliver a deterministic, local-first generator import prototype for teams using Kubernetes + GitOps + Helm + platform abstractions.

`cub-gen` is intentionally pre-sync in v0.1:

1. It discovers and imports DRY generator inputs.
2. It emits deterministic lineage/provenance + inverse-edit hints.
3. It does **not** reconcile to live clusters.

WET -> LIVE sync remains the job of Flux/Argo using Git/OCI transport.

## Product boundary (v0.1)

1. `cub-gen`: generator import + provenance contract.
2. Flux/Argo: runtime reconciliation.
3. ConfigHub (future bridge): governed WET store + decision/attestation path.

## Why this matters

Teams are moving away from exposing raw Argo/Flux internals to app developers.
They want opinionated app kits (DRY) while still preserving auditable runtime intent (WET).

`cub-gen` provides the missing bridge:

1. DRY source remains readable/editable by app/platform teams and AI tools.
2. Generated WET artifacts are explicit and traceable.
3. Each generated field has provenance and inverse mapping back to DRY edit points.

## User stories to prove in v0.1

1. Platform engineer with AI tooling can point `cub-gen` at a repo and get deterministic discover/import outputs without backend setup.
2. App team can stay mostly in app-level config (or prompts) and still get clear guidance on which DRY files to edit.
3. GitOps team can keep existing Flux/Argo pipelines unchanged while adopting DRY->WET provenance.

## Work items

1. [x] [#6](https://github.com/confighub/cub-gen/issues/6) MVP-00: lock CLI parity against `cub gitops`.
2. [x] [#2](https://github.com/confighub/cub-gen/issues/2) MVP-01: score.dev import with provenance + inverse map.
3. [x] [#3](https://github.com/confighub/cub-gen/issues/3) MVP-02: Helm PaaS import parity.
4. [x] [#4](https://github.com/confighub/cub-gen/issues/4) MVP-03: Spring Boot PaaS import path.
5. [x] [#5](https://github.com/confighub/cub-gen/issues/5) MVP-04: adoption-focused docs/UX.

## Current status snapshot (implemented)

1. `gitops discover/import/cleanup` parity contracts are golden-locked.
2. First three examples (Helm, Score, Spring Boot) are golden-locked for discover/import JSON outputs.
3. Top-level and subcommand help/usage output is golden-locked.
4. Path-mode smoke tests prove direct `./examples/...` usage without alias config.
5. Local bridge path is available and tested:
   - `publish` (bundle generation)
   - `verify` (bundle integrity)
   - `attest` (attestation record)
   - `verify-attestation` (attestation integrity + optional bundle linkage)

Milestone: [v0.1 - first 3 examples](https://github.com/confighub/cub-gen/milestone/1)

## Example narratives (what users should see)

### 1) score.dev

Input DRY:

1. `score.yaml` app contract.
2. Optional environment-specific app config.

Expected output:

1. Deterministic discover/import records.
2. Field-origin map tying generated manifest fields to score source.
3. Inverse mapping that says where to edit DRY when a WET field changes.

### 2) Helm PaaS

Input DRY:

1. `Chart.yaml`, templates, values files.

Expected output:

1. Deterministic classification of chart + values sources.
2. Explicit DRY/WET distinction in output payloads.
3. Lineage that survives repeated imports.

### 3) Spring Boot PaaS

Input DRY:

1. `pom.xml` + `application.yaml` style app config.

Expected output:

1. App-first import shape (not raw cluster-centric UX).
2. Provenance and inverse-edit hints for app and platform owners.
3. Deterministic command parity behavior.

## Proof-first delivery gate (mandatory)

For each issue:

1. Define proof matrix before coding.
2. Implement small vertical slice.
3. Pass proofs twice consecutively.
4. Update docs/examples.
5. Run one final proof pass.

Required commands:

1. `go test ./...`
2. `go test ./cmd/cub-gen -run '^TestGitOpsParity' -count=1 -v`
3. Example command for changed path (`discover` or `import`).

## Non-goals in v0.1

1. No direct cluster deployment/reconciliation.
2. No ConfigHub API dependency in core commands.
3. No non-deterministic AI rendering in core generator function.

## Handoff to next milestone

After v0.1 proves deterministic import/provenance:

1. Add optional bridge to ConfigHub as WET store and governance authority.
2. Keep Git/OCI + Flux/Argo reconciliation model intact.
3. Add additional generator families (IDP catalogs, app-config-only providers, ops workflow definitions).
