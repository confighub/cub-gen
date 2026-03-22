# AI Handover

Start here if you are the next AI picking up work in this repo.

Use this file for current priorities and safe first steps. Use `CLAUDE.md` for
stable repo-wide coding and validation rules.

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
- current synced head: `3bc35a3` `docs: strengthen flagship example adoption paths (#204)`
