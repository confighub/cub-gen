# 2026-03-08 Generator Triple Visibility Options

## Purpose

Capture the three implementation approaches for making generator triples visible and editable by platform teams.

A generator triple here means:

1. generator contract
2. provenance schema + field origin map
3. inverse transform plan + ownership/review policy

## Current state

As of `main` today, triple definitions are primarily embedded in Go:

- `internal/registry/registry.go`
- consumed by importer/detect/publish/parity flows

This is strongly typed and testable, but not easily discoverable for non-Go platform users.

## Option A: YAML-first registry

### What changes

- Move per-generator family definitions into `generators/*.yaml` files.
- Keep Go structs, but load via parser at startup from embedded files.
- Keep detection code in Go initially.

### Benefits

- Platform engineers can read/edit/copy generator definitions directly.
- New generator proposals can start as YAML changes.
- Cleaner review diffs for ownership/field-map policy changes.

### Risks

- Need robust schema validation and migration checks.
- Loader errors become a runtime startup risk if not validated in CI.
- Detection logic remains code-first unless separately externalized.

## Option B: Generated docs from Go source

### What changes

- Keep source of truth in Go registry.
- Add doc generator to produce per-generator markdown + diagrams.
- Add CI gate: generated docs must match source.

### Benefits

- Minimal runtime risk, additive architecture.
- Fastest path to visibility without refactoring registry storage.
- Strong traceability from source code to rendered docs.

### Risks

- Editing still requires Go changes.
- Docs are read artifacts, not editable source.

## Option C: YAML source + generated docs

### What changes

- YAML is source of truth.
- Docs generated from YAML.
- Go loads validated YAML and enforces typed constraints.

### Benefits

- Best authoring UX for platform teams.
- Best discoverability via generated docs.
- Clear separation: authoring (YAML) vs runtime enforcement (Go).

### Risks

- Highest implementation scope.
- Requires migration tooling + fallback strategy.
- Needs careful compatibility strategy for existing tests/goldens.

## Comparison matrix

| Dimension | Option A YAML-first | Option B Docs-only | Option C YAML+Docs |
|---|---|---|---|
| Platform editability | High | Low | High |
| Runtime change risk | Medium | Low | Medium |
| Delivery speed | Medium | High | Low-Medium |
| Long-term scalability | High | Medium | High |
| Refactor effort | Medium | Low | High |

## Recommended rollout sequence

1. **Start with Option B now** for immediate visibility and low risk.
2. **Pilot Option A for one family** (e.g., `c3agent`) behind strict validation.
3. If pilot quality is high, **move toward Option C** as target architecture.

## Validation requirements (all options)

1. keep `make ci` green
2. enforce parity/golden stability for existing command surfaces
3. enforce deterministic triple generation
4. enforce schema validation before merge
5. enforce backward-compatible output shape unless version bump is explicit

## Related references

- `HANDOVER.md`
- `docs/decisions/2026-03-08-swamp-c3agent-corrections.md`
- local planning note from prior session: `/Users/alexis/.claude/plans/snappy-frolicking-blossom.md`
