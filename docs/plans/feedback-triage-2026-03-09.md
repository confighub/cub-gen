# Feedback Triage: 2026-03-09

**Source**: user validation feedback  
**Status**: active triage  
**Owner**: Product + DX + Platform Engineering

## Validated strengths

1. `go test ./...` green.
2. Demo aggregators and Flux live e2e run successfully.
3. Top-level onboarding clarity improved.
4. Qualification caveat and user-story matrix are explicit.
5. Go-canonical generator contract decision is clear and implemented.
6. c3agent deepening is implemented and documented.

## Findings to action

| Finding | Impact | Action | Priority | Status |
|---|---|---|---|---|
| Docs fragmentation and off-repo references in long docs | New-user confusion | Add "current source of truth" map + mark archive docs clearly + remove absolute/local path refs from active docs | P1 | Open |
| Example catalog stale vs actual lifecycle scope | Trust erosion | Keep examples catalog synced to lifecycle script fixture list | P1 | In progress |
| Lifecycle flow mostly simulated governance today | Expectation mismatch | Add explicit simulation caveats in README/demo docs + add backend-connected roadmap milestones | P0 | In progress |
| Live e2e is Flux-only + static fixtures | Coverage gap | Add Argo live e2e entrypoint milestone + add per-example render-to-live roadmap item | P1 | Open |
| Detection still uses string heuristics for several families | Mis-detection risk | Replace heuristic detection with structural checks per generator family + add false-positive tests | P0 | Open |
| c3agent catalog metadata (`ResourceKind`) can mislead | Product positioning confusion | Clarify catalog semantics and/or add composite kind for multi-target generators | P1 | Open |

## Immediate changes shipped in response

1. Updated `examples/README.md` to match current 10-fixture lifecycle matrix.
2. Added explicit simulation caveats in `README.md` lifecycle sections.
3. Added explicit live e2e scope caveats in `examples/live-reconcile/README.md`.

## Next issue set to file

1. `[DX] Remove absolute/local file references from active docs and mark archive docs clearly`
2. `[E2E] Add Argo live reconciliation end-to-end entrypoint`
3. `[E2E] Support per-example cub-gen render -> live reconcile proof path`
4. `[Detect] Convert heuristic detector branches to structural detection + confidence tests`
5. `[Catalog] Clarify/adjust multi-target generator resource-kind semantics (c3agent)`
