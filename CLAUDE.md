# cub-gen

Deterministic Git-native generator importer with command-shape parity to `cub gitops`.

## Build and run

```bash
go build ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform ./examples/scoredev-paas ./examples/scoredev-paas
./cub-gen gitops cleanup --space platform ./examples/springboot-paas
```

## Non-negotiable principles

1. Deterministic behavior: same input => same output.
2. Parse, don't guess: derive classifications from explicit artifacts.
3. Local-first: no implicit deploys, no hidden control-plane side effects.
4. Parity-first: preserve `cub gitops` command contract shape where declared in `PARITY.md`.
5. Graceful degradation: unsupported/unknown paths must return explicit errors.
6. Test-every-change: `go test ./...` and parity tests must pass.

## Pre-coding proof requirements

Every feature/bugfix issue must define before coding:

1. Deterministic success criteria (exact input -> exact output).
2. Proof matrix (unit + parity/golden + example proof as applicable).
3. Degradation behavior (missing metadata, unknown generator, unsupported flags).
4. Definition of done: tests + docs + explainable output.

## Definition of done

A change is complete only when:

1. Required tests pass locally.
2. Parity/golden outputs are intentionally updated and reviewed.
3. User-facing docs/examples are aligned.
4. Contract drift is either avoided or explicitly documented in `PARITY.md`.

## Mandatory local validation

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^TestGitOpsParity' -count=1 -v
```
