# AI Handover

Start here if you are the next AI picking up work in this repo.

This file is intentionally short, practical, and current.

## Repo truth

`cub-gen` is in a stronger place on messaging and example structure than it is on
final proof. Do not treat the project as "fully complete".

The honest current state is:

- the example catalog is broad and recognizable,
- the main examples have local and connected entrypoints,
- the README now explains the product in plain English,
- the repo no longer claims that every PRD story is fully complete,
- the remaining work is to make the flagship examples prove the new bar end to end.

## Current priority

The active execution stack is:

1. `#183` connected acceptance suite and release gate
2. `#177` Helm / Argo / Kubara-like flagship example
3. `#187` Kubara-like layered provenance capability
4. `#178` Score flagship example
5. `#179` Spring Boot flagship example
6. `#180` workflow platform examples
7. `#182` example/demo entrypoint cleanup
8. `#185` custom-generator onboarding path
9. `#200` AI best practices from public examples
10. `#173` umbrella tracking issue

Open issues:

- [#200](https://github.com/confighub/cub-gen/issues/200) follow AI best practices as developed in the public examples
- [#187](https://github.com/confighub/cub-gen/issues/187) layered provenance for Kubara-like platforms
- [#185](https://github.com/confighub/cub-gen/issues/185) custom-generator onboarding
- [#183](https://github.com/confighub/cub-gen/issues/183) connected acceptance gate
- [#182](https://github.com/confighub/cub-gen/issues/182) example/demo entrypoint redesign
- [#180](https://github.com/confighub/cub-gen/issues/180) ops-workflow + Swamp upgrade
- [#179](https://github.com/confighub/cub-gen/issues/179) Spring flagship
- [#178](https://github.com/confighub/cub-gen/issues/178) Score flagship
- [#177](https://github.com/confighub/cub-gen/issues/177) Helm/Kubara flagship
- [#173](https://github.com/confighub/cub-gen/issues/173) umbrella tracking issue

## The standard we are aiming for

The examples are the product surface.

Every flagship example should prove:

- real cluster
- real app/runtime/workflow
- two audience paths:
  - existing ConfigHub user adding the platform tool
  - existing platform-tool user adding `cub-gen` + ConfigHub
- visible ConfigHub value
- one governed `ALLOW` path
- one governed `ESCALATE` or `BLOCK` path
- a live thing the user can inspect

For layered platforms like Helm/Argo/Kubara-like stacks, also prove:

- the generation chain
- the ownership boundary
- why a downstream edit should be routed upstream instead

## Files that matter most

- [README.md](README.md)
- [examples/README.md](examples/README.md)
- [examples/demo/README.md](examples/demo/README.md)
- [docs/plans/2026-03-16-example-reset-execution-plan.md](docs/plans/2026-03-16-example-reset-execution-plan.md)
- [docs/plans/2026-03-20-existing-gitops-adoption-plan.md](docs/plans/2026-03-20-existing-gitops-adoption-plan.md)
- [docs/testing/ilya-coreweave-acceptance-checklist.md](docs/testing/ilya-coreweave-acceptance-checklist.md)
- [test/checks/check-story-status.sh](test/checks/check-story-status.sh)

## Important recent fixes

Do not accidentally revert these:

- The README no longer points users at a dead docs site.
- The top-level docs now use an honest "execution status" snapshot instead of an
  "all stories fully met" claim.
- `test/checks/check-story-status.sh` was updated to enforce the new execution
  status model and to use portable `grep` instead of assuming `rg` exists in CI.
- Several example-reset issues were intentionally reopened because the project
  structure improved faster than the real flagship proof did.

## Important things not to claim

Do not say:

- "all PRD stories are fully complete"
- "all examples are fully hardened"
- "docs are published at https://confighub.github.io/cub-gen/"

Unless those things become demonstrably true again, keep the wording conservative
and evidence-based.

## Known local workspace state

These are currently untracked locally and should not be swept up casually:

- `.tmp/`
- `examples/incubator/`

Treat them as user/local work unless explicitly asked to modify them.

## Safe cold-start

If you are the next AI, start read-only and machine-readable first.

These are the safest first commands:

```bash
go run ./cmd/cub-gen generators --json
go run ./cmd/cub-gen gitops discover --space platform ./examples/helm-paas
go run ./cmd/cub-gen change preview --space platform --json ./examples/springboot-paas ./examples/springboot-paas
```

What they do:

- `generators --json`: shows the supported generator families and capabilities
- `gitops discover`: classifies a repo without pushing anything to ConfigHub or a cluster
- `change preview --json`: shows the governed change shape without executing it

Treat these as read-only first steps.

Only move to connected or state-changing paths after you understand which
example or workflow you are touching.

Examples of heavier paths:

- `make ci`: local validation suite
- `make ci-connected`: connected validation suite, only if the environment is ready
- `change run --mode connected`: uses ConfigHub APIs
- demo scripts that ingest/query ConfigHub or apply to Flux/Argo/kind

## How to frame proof tasks

When a user asks you to prove a GitOps, ConfigHub, or example workflow, strong
task framing helps more than generic "run the demo" wording.

Good proof requests should say:

- start read-only,
- use the documented path from the repo, not an improvised sequence,
- call out contaminated or hybrid state explicitly,
- state what does *not* count as proof,
- list the exact artifacts or states that must be shown,
- end with a clear result classification.

For GitOps proof in particular:

- do not count pre-existing running resources as proof of delivery,
- do not count direct `kubectl apply` of app manifests as proof of GitOps delivery,
- if the documented flow is hybrid, say exactly what it proves and what it does not.

Preferred end states:

- `full GitOps proof`
- `partial/hybrid GitOps proof`
- `not proven`

## Validation commands

Use these before claiming progress:

```bash
make ci
```

If you are working on connected flows and the environment is ready:

```bash
make ci-connected
```

Useful spot checks:

```bash
./test/checks/check-story-status.sh
./test/checks/check-example-dual-mode.sh
./test/checks/check-docs-entrypoints.sh
```

## Good next move

If you need to choose one thing:

1. strengthen `#183` so the release gate actually proves the example contract
2. finish `#177` + `#187` together as the Helm/Kubara flagship
3. then do `#178` as the cleanest two-audience example

## Last known synced state

At the time this handover was written:

- `main` was synced with `origin/main`
- recent merged fix: `41406b1` `docs: align project status with active execution (#199)`

Update this section if you make material progress.
