# Decision: Keep Go Registry as Canonical Generator Triple Source

Date: 2026-03-08
Status: accepted

## Decision

Keep `internal/registry/registry.go` as the canonical source of truth for generator triples.

## Rationale

1. Runtime behavior is already contract-tested against the Go registry.
2. Existing import/discover/generator parity suites are stable and green.
3. We can still provide platform-readable projections (YAML + Markdown) without introducing dual-write drift risk.

## Consequence

1. `docs/triple-styles/style-a-yaml/*.yaml` are read-only projections, not runtime-loaded specs.
2. `docs/triple-styles/style-b-markdown/*.md` and style-C pairs are documentation projections.
3. New/changed generator behavior is implemented in Go first, then projected via:
   - `go run ./cmd/cub-gen-style-sync`
   - `make sync-triple-styles`

## Guardrails

1. `cmd/cub-gen-style-sync/main_test.go` enforces checked-in projection sync.
2. CI fails on projection drift.

## Future Review Trigger

Revisit YAML-as-runtime only if platform-owner authoring in Go becomes a sustained adoption blocker.
