# cub-gen

`cub-gen` answers one operational question:

`a deployed field changed; which file/path should I edit, who owns it, and what evidence proves this change was safe?`

Point it at config you already have (Helm, Score, Spring Boot, Backstage, ops workflows, c3agent, provider config), and it emits:

- provenance (`what generated this`)
- field-origin maps (`which source field controls this deployed field`)
- inverse-edit guidance (`edit this file/path`)
- optional governed change bundles for ConfigHub (`publish -> verify -> attest`)

It is for two teams:

- teams with existing platform/app patterns (Helm, Score, Spring Boot, workflows) that need governance and traceability,
- teams rolling out a new internal platform quickly with clear ownership boundaries.

## What It Is

- A deterministic CLI for `discover -> import -> publish -> verify -> attest`.
- A DRY -> WET analysis/import layer that keeps source ownership explicit.
- A dual-mode workflow:
  - `Local mode`: no login; analyze and generate evidence in place.
  - `Connected mode`: send the same artifacts to ConfigHub decision APIs.

## What It Is Not

- Not a Kubernetes reconciler.
- Not a Flux/Argo replacement.
- Not a standalone policy runtime by itself.

Flux/Argo still reconcile to LIVE. `cub-gen` adds governance before deploy and traceability after deploy.

## Core Value in 10 Seconds

Take a deployed field and trace it to the exact source edit path:

```json
{
  "wet_path": "Deployment/spec/template/spec/containers[name=main]/image",
  "dry_path": "containers.main.image",
  "source_file": "score.yaml",
  "owner": "app-team",
  "edit_hint": "Edit the Score container image in score.yaml.",
  "confidence": 0.94
}
```

That is the day-to-day workflow this tool optimizes.

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

## Fast Path: App/AI Programmer (One Command)

If you want one command that returns the edit recommendation plus evidence artifacts:

```bash
./examples/demo/app-ai-fastpath.sh ./examples/scoredev-paas
```

Output:

- `change_id`, `bundle_digest`, `attestation_digest`
- detected profile(s) and target counts
- highest-confidence edit recommendation (`owner`, `wet_path`, `dry_path`, `edit_hint`)
- artifact paths in `.tmp/app-ai-fastpath/...`

## Quickstart: Connected Mode (ConfigHub)

All connected paths start with login.

```bash
# 1) Authenticate
cub auth login
TOKEN="$(cub auth get-token)"
cub context get --json | jq -r '.coordinate.user'

# 2) Run connected lifecycle coverage
./examples/demo/run-all-connected-lifecycles.sh
./examples/demo/run-all-connected-entrypoints.sh

# 3) Optional: run one connected example directly
./examples/helm-paas/demo-connected.sh
```

If ingest returns `404`, your configured base URL does not expose the governed bridge endpoint used by this demo flow. Set `CONFIGHUB_BASE_URL` to a backend endpoint that supports ingest/query.
If your backend uses non-default paths, set:

- `BRIDGE_INGEST_ENDPOINT` (for `bridge ingest`)
- `BRIDGE_DECISION_ENDPOINT` (for `bridge decision query`)

## Live Reconciler Proofs

We ship live reconcile E2E scripts for both controllers:

- Flux: `./examples/demo/e2e-live-reconcile-flux.sh`
- Argo CD: `./examples/demo/e2e-live-reconcile-argo.sh`
- Connected governed + reconcile (helm-paas): `./examples/demo/e2e-connected-governed-reconcile-helm.sh`

Both prove create, update, and drift-correction on a real `kind` cluster.

## Example Entry Points

Every example has both wrappers:

- Local: `./examples/<example>/demo-local.sh`
- Connected: `cub auth login` then `./examples/<example>/demo-connected.sh`

Start with the catalog:

- [examples/README.md](/Users/alexis/Public/github-repos/cub-gen/examples/README.md)
- Includes a persona-first quick navigator (`Choose your starting view`) for Helm, Spring Boot, Score, AI, and Ops users.
- 5-minute stack-specific entry paths: [persona-5-minute-runbooks.md](/Users/alexis/Public/github-repos/cub-gen/docs/workflows/persona-5-minute-runbooks.md)

### Workflow-First Start (Ops + Swamp)

If your platform is workflow-heavy (operations workflows or agent-authored workflows), start here:

```bash
# Ops workflow structural governance
./examples/ops-workflow/demo-local.sh
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '.provenance[0].ops_workflow_analysis'

# Swamp workflow-graph governance
./examples/swamp-automation/demo-local.sh
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '.provenance[0].swamp_workflow_analysis'
```

This shows structural change governance first: actions/schedules/approval gates for Ops, and models/methods/required steps for Swamp.

## Which Story Should You Read First?

- New to cub-gen: [Build your own Heroku in a weekend](/Users/alexis/Public/github-repos/cub-gen/docs/workflows/build-your-own-heroku-in-a-weekend.md)
- Fastest persona-based starts: [Persona 5-minute runbooks](/Users/alexis/Public/github-repos/cub-gen/docs/workflows/persona-5-minute-runbooks.md)
- Demo scripts index: [examples/demo/README.md](/Users/alexis/Public/github-repos/cub-gen/examples/demo/README.md)
- User-story acceptance matrix: [user-story-acceptance.md](/Users/alexis/Public/github-repos/cub-gen/docs/workflows/user-story-acceptance.md)
- Generator contract boundary: [canonical-triple-and-storage-boundary.md](/Users/alexis/Public/github-repos/cub-gen/docs/contracts/canonical-triple-and-storage-boundary.md)

## CI Targets

```bash
make ci-local       # build + tests + parity + docs/coverage gates
make ci-connected   # connected entrypoints + lifecycle + phase-3/4 stories + connected full-loop helm e2e + flux/argo live reconcile gates
make ci             # alias of ci-local
```

Story-specific connected scripts (Phase 3):

- `./examples/demo/story-1-existing-repo-connected.sh`
- `./examples/demo/story-7-ci-api-flow-connected.sh`
- `./examples/demo/story-9-multi-repo-wave-connected.sh`
- `./examples/demo/story-12-unified-actor-evidence.sh`
- `./examples/demo/run-phase-3-connected-stories.sh`

Story-specific connected scripts (Phase 4):

- `./examples/demo/story-8-label-evolution-connected.sh`
- `./examples/demo/story-10-signed-writeback-proof-connected.sh`
- `./examples/demo/story-11-live-breakglass-proposal-connected.sh`
- `./examples/demo/run-phase-4-connected-stories.sh`

Story 10 (signed write-back proof) requires real GitHub PR coordinates:
`APP_PR_REPO`, `APP_PR_NUMBER`, `PROMOTION_PR_REPO`, `PROMOTION_PR_NUMBER`
and `gh` auth (`GH_TOKEN`/`GITHUB_TOKEN` or `gh auth login`).

Workflow template for non-interactive CI auth:

- [connected-story7.yml](/Users/alexis/Public/github-repos/cub-gen/.github/workflows/connected-story7.yml)

## User-Story Coverage Snapshot

| Status | User stories | Notes |
|---|---|---|
| Met/strong in current demos | 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13 | Connected lifecycle is backend-authoritative for decision state, Story 8/11 source signals come from connected lifecycle artifacts, and Story 10 captures real GitHub signed-commit + branch-protection evidence. |
| Partial (simulated/local-first, not full backend/runtime integration) | None | Remaining connected fallback mode is available only as explicit troubleshooting path. |
| Deferred | None | Deferred stories now have connected acceptance scripts and evidence outputs. |

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
