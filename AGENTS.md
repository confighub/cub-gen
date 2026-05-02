# cub-gen

`cub-gen` makes Generators explicit.

A **Generator** is a deterministic function on config data. It reads files teams
already keep in Git, such as app code, Helm values, Score files, Spring config,
OpenChoreo CRs, environment settings, and secret references. It returns
deployable Kubernetes config.

The product model is:

```text
Component
  -> Variant
      -> Base Variant
      -> Deployment Variant
          -> Target
          -> Connections
          -> Change
          -> Proof
```

Keep this simple. Do not explain cub-gen first as DRY/WET, generator contracts,
or internal roadmap language. Start with the plain question:

```text
If this deployed field is wrong, where do I fix it?
```

## Build and run locally

```bash
go build ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform ./examples/scoredev-paas
./cub-gen change explain --space platform --owner app-team ./examples/springboot-paas
./cub-gen platform import --json ./testdata/platform-estate/platform.yaml
./cub-gen platform import --json ./testdata/variant-topology/platform.yaml
./cub-gen platform fanout --json ./testdata/variant-fanout/platform.yaml
./cub-gen enrich preview --space platform --json ./testdata/app-of-apps-standalone
./cub-gen normalize preview --space platform --patch ./examples/springboot-paas
```

## Non-negotiable principles

1. Deterministic behavior: same input means same output.
2. Parse, do not guess: derive classifications from explicit artifacts.
3. Local-first: no implicit deploys and no hidden control-plane side effects.
4. Parity-first: preserve the `cub gitops` command contract where declared in
   `PARITY.md`.
5. Graceful degradation: unsupported paths must return explicit diagnostics.
6. Proof-first: every feature needs tests, docs, and explainable output.
7. Plain English: avoid hype, vague AI language, and internal roadmap jargon in
   user-facing docs.

## Public language rules

Use these terms:

- **Generator**: a function on config data.
- **Component**: the reusable app/service/platform thing.
- **Variant**: a member of a Component family.
- **Base Variant**: a Variant that is not deployed and has no Target/live
  address.
- **Deployment Variant**: a Variant that is deployed to a Target and has live
  addressable config/resources.
- **Deployable Variant**: only use this as plain English for Deployment Variant,
  not as the whole ConfigHub Variant model.
- **Change**: a proposed edit to source or rendered config.
- **Proof**: field origin, owner, route, and decision evidence.

Prefer:

- "source config" or the exact file name over "intent".
- "rendered config" or "deployable Kubernetes config" over "WET" unless the
  existing file already uses DRY/WET.
- "several repos" or "multi-repo platform" over "estate".
- "reviewable cleanup patches" over "governed cleanups".

Do not write public docs as if cub-gen hides Kubernetes or replaces Helm, Argo,
Flux, Score, Spring, or OpenChoreo. It makes their Generator behavior explicit
so ConfigHub can reason about provenance, ownership, and edit routes.

## Pre-coding proof requirements

Every feature/bugfix issue must define before coding:

1. Deterministic success criteria (exact input -> exact output).
2. Proof matrix: unit, parity/golden, docs, and example proof as applicable.
3. Degradation behavior (missing metadata, unknown generator, unsupported flags).
4. Definition of done: tests + docs + explainable output.

## Definition of done

A change is complete only when:

1. Required tests pass locally.
2. Parity/golden outputs are intentionally updated and reviewed when behavior
   changes.
3. User-facing docs/examples are aligned.
4. Contract drift is either avoided or explicitly documented in `PARITY.md`.
5. New platform claims are backed by fixture proof and clear degradation paths.

## Current important surfaces

- `gitops discover/import`: repo-first detection and import with `cub gitops`
  command-shape parity.
- `platform import`: read several app/platform repos as Components, Variants,
  Deployment Variants, Targets, Generators, and diagnostics.
- `platform fanout`: emit one proof bundle per declared Deployment Variant.
- `enrich preview/write`: create sidecar provenance proof without manifest
  rewrites.
- `normalize preview`: propose reviewable cleanup patches without touching
  source or rendered YAML.
- `bridge link`: connect a GitHub PR and ConfigHub MR through one `change_id`.
- `generators`: list registry-backed supported Generator families.

The intended future product command is `cub gen`; today this repo still builds
`cub-gen`. Keep `docs/cub-gen-plugin.md` aligned when command-shape claims
change.

## Mandatory local validation

```bash
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand|TestGeneratorsGolden)' -count=1 -v
go test ./cmd/cub-gen -run '^(TestBridgeSymmetryMatrix|TestExamplesPathModeBridgeFlow)$' -count=1 -v
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
# or:
make ci
```

For docs-only edits, at minimum run:

```bash
git diff --check
```

Run `mkdocs build --strict` when navigation, docs links, or generated docs pages
change.
