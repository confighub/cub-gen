# cub-gen

`cub-gen` is the Git-side application layer for ConfigHub: it reads app/platform config, maps it to governed platform output, and tells you exactly what to edit when something changes.

It is for two teams:

- teams with existing platform/app patterns (Helm, Score, Spring Boot, workflows) that need governance and traceability,
- teams rolling out a new internal platform quickly with clear ownership boundaries.

## What It Is

- A deterministic CLI for `discover -> import -> publish -> verify -> attest`.
- A generator framework: each generator turns app config into platform config plus governance and attestation metadata.
- A dual-mode workflow:
  - `Local mode`: no login, fast onboarding.
  - `Connected mode`: authenticated calls to ConfigHub.

## What It Is Not

- Not a Kubernetes reconciler.
- Not a Flux/Argo replacement.
- Not an app runtime by itself.

Flux/Argo still reconcile to LIVE. `cub-gen` adds governance before deploy and traceability after deploy.

## Core Value in 10 Seconds

You can point at a deployed field and get the source-of-truth edit path.

```json
{
  "wet_field": "Deployment/spec/template/spec/containers/0/image",
  "source_file": "values.yaml",
  "source_path": "image.tag",
  "owner": "app-team",
  "confidence": 0.91,
  "edit_hint": "Edit values.yaml:image.tag (or env overlay for prod-only changes)."
}
```

That is the day-to-day problem cub-gen solves.

## ConfigHub Is Real (Today)

Connected mode targets a live ConfigHub backend. Backend OSS repo:

- [confighubai/confighub](https://github.com/confighubai/confighub)

## Quickstart: Local Mode (No Login)

```bash
go build -o ./cub-gen ./cmd/cub-gen
./examples/demo/run-all-modules.sh
./examples/demo/run-all-confighub-lifecycles.sh
```

This gives immediate value with local evidence and lifecycle outputs.

## Quickstart: Connected Mode (ConfigHub)

All connected paths start with login.

```bash
# 1) Authenticate
cub auth login
TOKEN="$(cub auth get-token)"
cub context get --json | jq -r '.coordinate.user'

# 2) Run connected lifecycle coverage
./examples/demo/run-all-connected-lifecycles.sh

# 3) Optional: run one connected example directly
./examples/helm-paas/demo-connected.sh
```

If ingest returns `404`, your configured base URL does not expose the governed bridge endpoint used by this demo flow. Set `CONFIGHUB_BASE_URL` to a backend endpoint that supports ingest/query.

## Live Reconciler Proofs

We ship live reconcile E2E scripts for both controllers:

- Flux: `./examples/demo/e2e-live-reconcile-flux.sh`
- Argo CD: `./examples/demo/e2e-live-reconcile-argo.sh`

Both prove create, update, and drift-correction on a real `kind` cluster.

## Example Entry Points

Every example has both wrappers:

- Local: `./examples/<example>/demo-local.sh`
- Connected: `cub auth login` then `./examples/<example>/demo-connected.sh`

Start with the catalog:

- [examples/README.md](/Users/alexis/Public/github-repos/cub-gen/examples/README.md)

## Which Story Should You Read First?

- New to cub-gen: [Build your own Heroku in a weekend](/Users/alexis/Public/github-repos/cub-gen/docs/workflows/build-your-own-heroku-in-a-weekend.md)
- Demo scripts index: [examples/demo/README.md](/Users/alexis/Public/github-repos/cub-gen/examples/demo/README.md)
- User-story acceptance matrix: [user-story-acceptance.md](/Users/alexis/Public/github-repos/cub-gen/docs/workflows/user-story-acceptance.md)
- Generator contract boundary: [canonical-triple-and-storage-boundary.md](/Users/alexis/Public/github-repos/cub-gen/docs/contracts/canonical-triple-and-storage-boundary.md)

## CI Targets

```bash
make ci-local       # build + tests + parity + docs/coverage gates
make ci-connected   # connected lifecycle + flux/argo live reconcile gates
make ci             # alias of ci-local
```

Story-specific connected scripts (Phase 3):

- `./examples/demo/story-1-existing-repo-connected.sh`
- `./examples/demo/story-7-ci-api-flow-connected.sh`
- `./examples/demo/story-9-multi-repo-wave-connected.sh`
- `./examples/demo/story-12-unified-actor-evidence.sh`
- `./examples/demo/run-phase-3-connected-stories.sh`

Workflow template for non-interactive CI auth:

- [connected-story7.yml](/Users/alexis/Public/github-repos/cub-gen/.github/workflows/connected-story7.yml)

## User-Story Coverage Snapshot

| Status | User stories | Notes |
|---|---|---|
| Met/strong in current demos | 2, 3, 4, 5, 6, 13 | Proven by current local-first examples and lifecycle flows. |
| Partial (simulated/local-first, not full backend/runtime integration) | 1, 7, 9, 12 | Command shape and evidence model are present; backend/runtime coupling is still partial. |
| Deferred | 8, 10, 11 | Requires additional platform/runtime features and connected workflows. |

## Repo Map

- CLI code: `cmd/cub-gen`, `internal/*`
- Example suites: `examples/*`
- Demo runners: `examples/demo/*`
- Contracts and decisions: `docs/contracts`, `docs/decisions`
- Workflow docs: `docs/workflows`

## Development

```bash
go test ./...
make ci-local
```

For contribution details, see:

- [CONTRIBUTING.md](/Users/alexis/Public/github-repos/cub-gen/CONTRIBUTING.md)
