# cub-gen Execution Board: Sprint 1 and Sprint 2

Status date: 2026-03-06

## Objective

Close v0.2 as a stable standalone baseline, then run a contract-triple hardening sprint and stop at a formal Go/No-Go gate before any ConfigHub bridge coupling.

## Milestones

1. Milestone #2: Sprint 1: v0.2 Closeout (2026-03-09 to 2026-03-20)
2. Milestone #3: Sprint 2: Contract Triple Gate (2026-03-23 to 2026-04-03)

## Sprint 1 Board (Milestone #2)

Ordered execution:

1. [#71](https://github.com/confighub/cub-gen/issues/71) `[S1-01] chore(v0.2): freeze parity contract baseline + drift checklist`
2. [#72](https://github.com/confighub/cub-gen/issues/72) `[S1-02] refactor(v0.2): move rendered lineage templates into registry`
3. [#73](https://github.com/confighub/cub-gen/issues/73) `[S1-03] refactor(v0.2): move Helm provenance source-path semantics into registry`
4. [#74](https://github.com/confighub/cub-gen/issues/74) `[S1-04] test(v0.2): add generator metadata conformance suite`
5. [#75](https://github.com/confighub/cub-gen/issues/75) `[S1-05] docs(v0.2): publish 10-minute Flux/Argo/Helm adoption path`
6. [#76](https://github.com/confighub/cub-gen/issues/76) `[S1-06] release(v0.2): cut v0.2-preview.1 + release notes`

Dependency logic:

1. `#71` must land before any final release work.
2. `#72` and `#73` can run in parallel after `#71`.
3. `#74` depends on `#72` and `#73`.
4. `#75` can run in parallel with `#74`, but must reflect landed command surface.
5. `#76` depends on `#71` through `#75`.

## Sprint 2 Board (Milestone #3)

Ordered execution:

1. [#77](https://github.com/confighub/cub-gen/issues/77) `[S2-01] feat(contracts): add schema validation gate for contract triple`
2. [#78](https://github.com/confighub/cub-gen/issues/78) `[S2-02] feat(import): enforce no-triple-no-governed-import gate`
3. [#79](https://github.com/confighub/cub-gen/issues/79) `[S2-03] test(contracts): add cross-family triple conformance fixtures`
4. [#80](https://github.com/confighub/cub-gen/issues/80) `[S2-04] ci(bridge): enforce publish/verify/attest symmetry matrix`
5. [#81](https://github.com/confighub/cub-gen/issues/81) `[S2-05] docs(contracts): publish canonical triple + storage boundary docs`
6. [#82](https://github.com/confighub/cub-gen/issues/82) `[S2-06] gate: Go/No-Go decision record before ConfigHub bridge coupling`

Dependency logic:

1. `#77` is foundational.
2. `#78` depends on `#77`.
3. `#79` depends on `#77` and `#78`.
4. `#80` can start after `#79` scaffolding is in place.
5. `#81` depends on landed behavior from `#77` through `#80`.
6. `#82` is the mandatory stop gate and depends on `#77` through `#81`.

## Logical Break Point (Mandatory)

Stop at completion of `#82`.

Do not start any ConfigHub bridge-coupled implementation until `#82` records one explicit outcome:

1. `GO`: contract and adoption bar met, proceed to bridge backlog.
2. `NO-GO`: continue standalone hardening with follow-on issues.

Go/No-Go checklist in `#82` must include:

1. Determinism and parity lock integrity.
2. Contract-triple conformance across all supported families.
3. Bridge artifact symmetry coverage.
4. Adoption clarity for Flux/Argo/Helm users.
5. Git (DRY linkage) vs ConfigHub (WET governance) boundary clarity.

Recorded outcome:

1. `GO` documented in `docs/decisions/2026-03-06-sprint-2-go-no-go.md`.

## Post-S2 Bridge Coupling (Completed)

Post-gate bridge backlog from `#82` is now complete:

1. `#95` closed via PR `#99`: change-bundle ingest mapping into governed WET artifacts with idempotency and duplicate-safe handling.
2. `#96` closed via PR `#100`: governed decision + attestation state contract, explicit `ALLOW|ESCALATE|BLOCK` transition gates, and query-by-`change_id`.
3. `#97` closed via PR `#101`: Git PR <-> ConfigHub MR linkage model and upstream DRY promotion flow with explicit separate-review guardrail.

## Proof Commands (Required for Done)

1. `go test ./...`
2. `make ci`

## Notes

1. This board is sequencing-focused and does not replace detailed issue acceptance criteria.
2. Scope creep into runtime reconciliation or server coupling is out of plan for these two sprints.
