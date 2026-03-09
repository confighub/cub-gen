# Plan: c3agent Deepening (Go-canonical)

**Status**: Implemented  
**Implemented on**: 2026-03-09  
**Primary PR**: [#117](https://github.com/confighub/cub-gen/pull/117)

## Scope delivered

1. Deepened c3agent DRY model in `examples/c3agent/`.
2. Tightened detection to structural top-level YAML/JSON checks.
3. Expanded c3agent from 2 to 11 WET targets.
4. Expanded inverse model to 7 policy keys.
5. Expanded rendered lineage templates to 11 targets.
6. Updated c3agent parity goldens and regenerated triple-style docs.
7. CI proof gate green (`make ci`).

## Delivered files (high-signal)

- `examples/c3agent/c3agent.yaml`
- `examples/c3agent/c3agent-prod.yaml`
- `internal/detect/detect.go`
- `internal/importer/importer.go`
- `internal/registry/registry.go`
- `cmd/cub-gen/testdata/parity/gitops-discover-c3agent.golden.json`
- `cmd/cub-gen/testdata/parity/gitops-import-c3agent.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-c3agent.golden.json`
- `cmd/cub-gen/testdata/parity/publish-from-import-c3agent.golden.json`
- `docs/triple-styles/style-a-yaml/c3agent.yaml`
- `docs/triple-styles/style-b-markdown/c3agent.md`
- `docs/triple-styles/style-c-yaml-plus-docs/c3agent/triple.yaml`
- `docs/triple-styles/style-c-yaml-plus-docs/c3agent/triple.md`

## Explicit non-goals (still deferred)

1. `cub-gen render` / manifest deployment execution.
2. YAML-bundle runtime migration.
3. Docker Compose WET target.
4. Operation-composition formalization.
