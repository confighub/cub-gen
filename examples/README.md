# cub-gen Examples

These examples are for source-first questions:

- which file or path produced this rendered field?
- what should I edit safely?
- what provenance bundle can I verify or publish?

If your question starts in a live cluster instead of a source repo, start with
[`cub-scout`](https://github.com/confighub/cub-scout) or `cub gitops import`
first.

## Start With One Flagship Path

Do not start by reading the whole catalog. Start with the example that matches
the stack you already run.

| If you already run... | Start here | First command | What success looks like |
|---|---|---|---|
| Helm plus Argo/Flux platform repos | [`helm-paas`](./helm-paas/) | `./examples/demo/start-platform-first.sh` | you can trace one rendered field back to chart or values ownership |
| Spring Boot app repos | [`springboot-paas`](./springboot-paas/) | `./examples/demo/start-app-first.sh` | you can tell which config changes are app-owned, lift-upstream, or blocked |
| Score.dev workloads | [`scoredev-paas`](./scoredev-paas/) | `./examples/demo/start-score-first.sh` | you can trace `score.yaml` intent to rendered runtime fields |
| Real GitOps runtime proof | [`live-reconcile`](./live-reconcile/) | `RECONCILER=both ./examples/live-reconcile/demo-local.sh` | you see WET output survive Flux or Argo reconciliation on kind |

If you are unsure, start with `helm-paas` for the platform view or
`springboot-paas` for the app-team view.

## What Is Strongest Today

- [`helm-paas`](./helm-paas/) is the strongest platform-first path today. It
  covers source-side provenance, governed change, layered Helm/ApplicationSet
  proof, and a paired live runtime wrapper.
- [`springboot-paas`](./springboot-paas/) is the strongest standalone
  end-to-end app example today.
- [`scoredev-paas`](./scoredev-paas/) is the strongest Score end-to-end path
  today.
- [`live-reconcile`](./live-reconcile/) is the runtime harness. Pair it with
  `helm-paas` when you need WET-to-LIVE proof instead of source-side ownership.

For exact proof status, use the generated
[Example Truth Matrix](../docs/testing/example-truth-matrix.md).

## Run Order That Repeats Well

From the repo root:

```bash
go build -o ./cub-gen ./cmd/cub-gen
```

Then use this order:

1. Run one local wrapper first: `./examples/<example>/demo-local.sh`
2. If the local proof makes sense, log in: `cub auth login`
3. Run the connected smoke check: `./examples/demo/run-connected-smoke.sh`
4. Then run the connected wrapper: `./examples/<example>/demo-connected.sh`

The wrappers are the safest first-run path because they encode the current happy
path for each flagship example.

## Example Families

### Flagship platform and app paths

- [`helm-paas`](./helm-paas/) — Helm, overlays, ApplicationSet layering, live
  reconciler pairing
- [`springboot-paas`](./springboot-paas/) — Spring Boot app/team vs platform
  ownership
- [`scoredev-paas`](./scoredev-paas/) — Score intent to rendered manifests
- [`live-reconcile`](./live-reconcile/) — Flux and Argo runtime proof harness

### Workflow and automation

- [`ops-workflow`](./ops-workflow/) — governed SRE workflow config
- [`swamp-automation`](./swamp-automation/) — governed AI-written workflow
  structure
- [`confighub-actions`](./confighub-actions/) — ConfigHub lifecycle as governed
  config

### AI, fleet, and platform governance

- [`c3agent`](./c3agent/) — governed AI fleet config
- [`ai-ops-paas`](./ai-ops-paas/) — fuller platform version of the AI fleet
  story
- [`swamp-project`](./swamp-project/) — governed Helm/runtime side of Swamp

## AI + automation patterns

This section stays intentionally small because the truth-matrix tooling still
uses it as a repo-level AI marker.

| Example | Best starting question |
|---|---|
| [`c3agent`](./c3agent/) | how do we govern model, budget, and credential changes for an AI fleet? |
| [`ai-ops-paas`](./ai-ops-paas/) | what does the fuller platform version of that AI fleet story look like? |
| [`swamp-automation`](./swamp-automation/) | how do we govern agent-written workflow graph changes? |
| [`swamp-project`](./swamp-project/) | how do we govern the Helm/runtime side of an AI platform? |

### Other stacks and companion examples

- [`backstage-idp`](./backstage-idp/) — Backstage catalog governance
- [`just-apps-no-platform-config`](./just-apps-no-platform-config/) — smallest
  app-only governance path
- [`demo`](./demo/) — shared wrappers and connected smoke runners
- [`incubator`](./incubator/) — experimental or companion material

## Source-First vs Cluster-First

Use these `cub-gen` examples when you start in a source repo and want source to
rendered provenance.

Use cluster-first tools when the urgent question is "what is running right now?":

- [`cub-scout`](https://github.com/confighub/cub-scout) for read-only cluster
  observation, ownership, and troubleshooting
- `cub gitops import` for ConfigHub import starting from a live Argo or Flux
  target

Those are complementary flows, not competing ones.

## When You Need More Than The Index

- each example README carries its own honest "what this proves today" section
- the [Example Truth Matrix](../docs/testing/example-truth-matrix.md) is the
  repo-wide proof source
- the [Domain POV Matrix](../docs/workflows/domain-pov-matrix.md) helps if you
  want to pick by team or operating model instead of by technology

If the first operator is an AI assistant, the flagship examples also ship
example-local `AI_START_HERE.md` files and prompt packs. Use those inside the
selected example, not as a replacement for choosing the right example first.
