# AI Handover

Start here if you are the next AI picking up work in this repo.

Use this file for current priorities and safe first steps. Use `CLAUDE.md` for
stable repo-wide coding and validation rules.

## 2026-03-25 Handover Update

Short handover for the current local worktree:

- `#215` first slice is landed locally: generated example truth artifacts now live in `docs/testing/example-truth-matrix.md` and `docs/testing/example-truth-matrix.json`, derived by `internal/exampletruth/` and `tools/example-truth-matrix/`.
- `#183` first slice is landed locally: `Makefile` now gates matrix freshness, connected release-gate coverage, and Flow A / Flow B presence via `test/checks/check-example-truth-matrix.sh`, `test/checks/check-connected-release-gate.sh`, and `test/checks/check-flow-evidence.sh`.
- Docs now point status/readiness claims back to the generated matrix (`README.md`, `examples/README.md`, `examples/demo/README.md`, `docs/index.md`, `docs/testing/README.md`, `mkdocs.yml`).

Current local proof:

- `make ci-local` passes.
- `mkdocs build --strict` was not run successfully here because `mkdocs` is not installed in this shell.
- A separate credentialed environment is currently attempting `make ci-connected`; treat that result as the next real decision point.

Current derived truth from the generated matrix:

- 12 featured examples
- 8 first-class generator fixtures
- 8 source-chain verified fixtures
- 12 connected-release-gated examples
- live proof split: 10 `none`, 1 `paired-harness`, 1 `standalone`
- AI-first split: 6 `none`, 2 `partial`, 4 `explicit`

Next plan if the connected run passes:

1. Record the `ci-connected` result in `#183`, `#188`, and `#215`.
2. Commit this local slice on a branch instead of leaving it on dirty `main`.
3. Decide whether the next work is fixing connected regressions or tightening per-example universal-contract checks.

Next plan if the connected run fails:

1. Fix the first failing connected target or evidence check.
2. Re-run only the failing connected proof lane until stable.
3. Update the issue comments with the exact blocker instead of claiming the gate is complete.

## Repo truth

`cub-gen` is now much clearer on its product story than it was a week ago, but
it is not done.

What is solid now:

- the top-level docs lead with existing GitOps users instead of generator taxonomy,
- the flagship entrypoints are more concrete and tied to real app/platform questions,
- the repo explains how `cub-gen`, ConfigHub, and `cub-scout` fit together,
- the project no longer over-claims complete proof where the gates do not yet exist.

What is not done:

- the cluster-first companion path still needs a concrete, primary walkthrough,
- the connected acceptance gate still needs to enforce the flagship contract,
- the strongest examples still need end-to-end proof against the new standard.
- actual field-level `block/escalate` enforcement still depends on product work,
- direct mutation of embedded config payloads still has product gaps.

## Canonical journeys

These are the main user paths the repo now optimizes for:

1. platform-first: existing Helm plus Flux/Argo team
2. app-first: existing Spring Boot team
3. cluster-first companion: ConfigHub GitOps import plus `cub-scout`, then back to `cub-gen`

The current docs and examples should reinforce those journeys, not compete with
them.

## What merged recently

Recent merged work to preserve:

- `b204036` `docs: add root AI handover (#201)`
- `3059747` `docs: refocus entrypoints on existing GitOps adoption (#203)`
- `3bc35a3` `docs: strengthen flagship example adoption paths (#204)`

Those changes intentionally shifted the repo toward:

- existing-app and existing-GitOps relevance,
- two opinionated starter paths instead of a flat menu,
- explicit continuity with ConfigHub and `cub-scout`.

## Read these first

- [README.md](README.md)
- [examples/README.md](examples/README.md)
- [examples/demo/README.md](examples/demo/README.md)
- [examples/helm-paas/README.md](examples/helm-paas/README.md)
- [examples/live-reconcile/README.md](examples/live-reconcile/README.md)
- [examples/springboot-paas/README.md](examples/springboot-paas/README.md)
- [docs/plans/2026-03-20-existing-gitops-adoption-plan.md](docs/plans/2026-03-20-existing-gitops-adoption-plan.md)
- [docs/plans/2026-03-16-example-reset-execution-plan.md](docs/plans/2026-03-16-example-reset-execution-plan.md)
- [docs/testing/ilya-coreweave-acceptance-checklist.md](docs/testing/ilya-coreweave-acceptance-checklist.md)

## Safe cold-start

Start read-only and machine-readable first.

```bash
go run ./cmd/cub-gen generators --json
go run ./cmd/cub-gen gitops discover --space platform ./examples/helm-paas
go run ./cmd/cub-gen change preview --space platform --json ./examples/springboot-paas ./examples/springboot-paas
```

What they tell you:

- `generators --json`: supported generator families and capabilities
- `gitops discover`: how a repo is classified without pushing to ConfigHub or a cluster
- `change preview --json`: the governed change shape without executing it

Treat those as the default first move.

Heavier paths that may mutate state or depend on live systems:

- `make ci`
- `make ci-connected`
- `change run --mode connected`
- demo or e2e scripts that talk to ConfigHub, kind, Flux, or Argo

## What still matters most

The examples are the product surface.

Every flagship example should prove:

- real cluster
- real app, runtime, or workflow
- two audience paths:
  - existing ConfigHub user adding the platform tool
  - existing platform-tool user adding `cub-gen` + ConfigHub
- visible ConfigHub value
- one governed `ALLOW` path
- one governed `ESCALATE` or `BLOCK` path
- one live thing the user can inspect

For layered Helm/Argo/Kubara-like stacks, also prove:

- the generation chain
- the ownership boundary
- why downstream edits should route upstream instead

## Current reference implementation

The best current reference for the AI-first, proof-tiered example style now
lives in the public `examples` repo under
`incubator/springboot-platform-app`.

What exists there now:

- merged: real Spring Boot app plus HTTP tests,
- merged: ConfigHub-only proof for `inventory-api-dev`, `stage`, and `prod`,
- open stacked PR: Noop target proof for apply and re-apply without a real
  cluster ([examples#84](https://github.com/confighub/examples/pull/84)),
- open stacked PR: read-only `lift upstream` Redis bundle
  ([examples#87](https://github.com/confighub/examples/pull/87)),
- open stacked PR: read-only `block/escalate` boundary bundle
  ([examples#89](https://github.com/confighub/examples/pull/89)).

That stack is the clearest current example of:

- explicit proof levels,
- AI-first docs plus contracts,
- read-only preview surfaces before mutation,
- one mutation system with three routed outcomes,
- honest classification of what is still not proven.

Product gaps made concrete by that example are now tracked in:

- `#208`: direct mutation of dotted ConfigMap keys and embedded YAML-backed
  fields,
- `#207`: field-level `block/escalate` enforcement for generator-owned
  mutation routes.

## How to frame proof tasks

When proving GitOps or example behavior, strong task framing matters.

Use this pattern:

- start read-only,
- use the documented repo flow, not an improvised sequence,
- call out contaminated or hybrid state explicitly,
- say what does not count as proof,
- list the artifacts or states that must be shown,
- end with a result classification.

For GitOps proof in particular:

- do not count pre-existing running resources as proof of delivery,
- do not count direct `kubectl apply` of app manifests as proof of GitOps delivery,
- if the flow is hybrid, say exactly what it proves and what it does not.

Preferred result labels:

- `full GitOps proof`
- `partial/hybrid GitOps proof`
- `not proven`

## What not to claim

Do not say:

- "all PRD stories are fully complete"
- "all examples are fully hardened"
- "every flagship proof is acceptance-complete"
- "docs are published at https://confighub.github.io/cub-gen/"

Keep the wording conservative and evidence-based until the gates really prove it.

## Best next work

If you need the most useful next sequence, do this:

1. make the cluster-first companion path concrete from ConfigHub and `cub-scout` back into `cub-gen`
2. strengthen `#183` so the release gate proves the flagship contract
3. keep tightening `#177`, `#187`, and `#179` against real proof, not only better docs
4. only after the journeys are stable, package the AI-first companion docs more fully

## Validation

Before claiming progress, run:

```bash
./test/checks/check-docs-entrypoints.sh
./test/checks/check-story-status.sh
```

For broader local validation:

```bash
make ci
```

If you are intentionally working on connected flows and the environment is ready:

```bash
make ci-connected
```

## Last known synced state

At the time of this update:

- `main` matched `origin/main`
- current synced head: `8aa89ec` `docs: refresh handover and spring platform model (#205)`
