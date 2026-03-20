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
