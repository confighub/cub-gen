# cub-gen

Working on this repo as the next maintainer or AI? Start with [AI-HANDOVER.md](AI-HANDOVER.md).

See where every deployed config value came from, starting from the repo you
already have.

`cub-gen` is a source-side provenance and governed-change companion for GitOps
teams. It is for people who already run GitHub, Helm, Score, Spring Boot, or
workflow config, and already rely on Flux or Argo CD to reconcile what reaches
their clusters.

**gen = generator.** A generator is a function that maps DRY source (your `values.yaml`, `score.yaml`, etc.) to WET rendered output (the manifests that reach your cluster). `cub-gen` detects which generators your repo uses, runs the mapping, and records provenance — so every deployed field traces back to a source file, line, and owner.

If you already run app/config in Git, OCI artifacts, and Flux/Argo reconciliation, `cub-gen` answers:

- Which source file/path controls this live field?
- What rendered manifests did this repo actually produce?
- Did the right team edit the right thing?
- Can we block unsafe edits before they hit cluster?

It does this by classifying your existing repo and mapping rendered fields back
to source file, line, and owner.

## Start with the path you already have

| If you already run... | Start here | First useful answer |
|-----------|-----------|---------------------|
| Helm plus Flux/Argo platform repos | [helm-paas](examples/helm-paas/) | Which values file owns this rendered field? |
| Spring Boot services in GitOps | [springboot-paas](examples/springboot-paas/) | Which `application.yaml` setting or platform file should I edit? |
| Cluster-first GitOps operations | ConfigHub GitOps import + [cub-scout](https://github.com/confighub/cub-scout) + then `cub-gen` | What is running, and what source produced it? |

## What it is not

- Not a Kubernetes reconciler
- Not a Flux/Argo replacement
- Not an OCI replacement

Flux/Argo still reconcile to LIVE. `cub-gen` adds governance before deploy and traceability after deploy.

## Why import?

Import should answer something useful right away:

- what rendered manifests this repo produces,
- which DRY file controls a deployed field,
- what evidence bundle or governed change to inspect next,
- how the repo-side answer lines up with cluster-side inspection in `cub-scout`
  and ConfigHub.

## Two import paths, two jobs

There are two related import flows in the ConfigHub world:

- `cub gitops import` in ConfigHub imports existing Argo/Flux applications from a cluster or worker target.
- `cub-gen gitops import` reads source repos such as Helm, Score.dev, Spring Boot, or workflow config and emits provenance, inverse-edit guidance, and evidence.

They complement each other:

- use ConfigHub GitOps import for brownfield cluster/app onboarding,
- use `cub-gen` when you want source-to-runtime traceability and governed changes from DRY config.

## How the tools fit together

| Tool | Starts from | Best first question |
|------|-------------|---------------------|
| `cub-gen` | Source repo and generator inputs | Which DRY file/path produced this rendered field? |
| [`cub-scout`](https://github.com/confighub/cub-scout) | Cluster, reconciler, and live runtime state | What is running, who owns it, and where is drift? |
| [ConfigHub](https://github.com/confighubai/confighub) | Shared intended state, evidence, and governance state | What changed, what was approved, and what evidence exists across repos and clusters? |

## What it looks like

```bash
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{origin: .provenance[0].field_origin_map[0], inverse: .provenance[0].inverse_edit_pointers[0]}'
```

```json
{
  "origin": {
    "dry_path": "values.image.tag",
    "wet_path": "Deployment/spec/template/spec/containers[0]/image",
    "source_path": "values.yaml",
    "transform": "helm-template",
    "confidence": 0.86
  },
  "inverse": {
    "wet_path": "Deployment/spec/template/spec/containers[0]/image",
    "dry_path": "values.image.tag",
    "owner": "app-team",
    "edit_hint": "Edit chart values file and keep chart template unchanged.",
    "confidence": 0.86
  }
}
```

You see exactly which source field produced each deployed value, who owns it, and where to edit it safely.

## What you get

- **Generator detection** — recognizes 8 config styles: Helm, Score.dev, Spring Boot, Backstage, No-Config-Platform, ops-workflow, c3agent, Swamp
- **DRY/WET classification** — separates human-authored intent from rendered deployment artifacts
- **Field-origin tracing** — maps every deployed field back to its source file, line, and owner
- **Inverse-edit guidance** — tells you where to edit DRY source to change a deployed value
- **Change CLI** — `change preview`, `change run`, `change explain` for day-to-day workflows

## First runs (local, no login)

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Platform-first: existing Helm + Flux/Argo team
./examples/helm-paas/demo-local.sh

# App-first: existing Spring Boot team
./examples/springboot-paas/demo-local.sh
```

## Use Your Repo in 3 Commands

```bash
REPO=/path/to/your/repo
./cub-gen change preview --space platform "$REPO" "$REPO"
./cub-gen gitops import --space platform --json "$REPO" "$REPO" \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
./cub-gen change explain --space platform --owner app-team "$REPO" "$REPO"
```

## Next step (connected, ConfigHub)

```bash
cub auth login

# Platform-first connected path
./examples/helm-paas/demo-connected.sh

# App-first connected path
./examples/springboot-paas/demo-connected.sh
```

If you are starting from a cluster or controller rather than a repo, use
ConfigHub GitOps import and [`cub-scout`](https://github.com/confighub/cub-scout)
for the cluster-side path first, then use `cub-gen` to trace a chosen field back
to DRY source.

## Confidence scores

Use confidence to decide routing speed:

- `>= 0.90`: proceed with normal app/team edit flow
- `0.75 - 0.89`: run `change preview` and `change explain` before merge
- `< 0.75`: escalate for platform review

Full guide: [docs/workflows/confidence-scores.md](docs/workflows/confidence-scores.md)

## Supported generators

| Generator | Detects | Example |
|-----------|---------|---------|
| Helm | `Chart.yaml` + `values.yaml` | [helm-paas](examples/helm-paas/) |
| Score.dev | `score.yaml` | [scoredev-paas](examples/scoredev-paas/) |
| Spring Boot | `application.yaml` | [springboot-paas](examples/springboot-paas/) |
| Backstage | `catalog-info.yaml` | [backstage-idp](examples/backstage-idp/) |
| No-Config-Platform | `app.yaml` provider config | [just-apps-no-platform-config](examples/just-apps-no-platform-config/) |
| Ops Workflow | `ops-workflow.yaml` | [ops-workflow](examples/ops-workflow/) |
| C3 Agent | `c3agent.yaml` | [c3agent](examples/c3agent/) |
| Swamp | `.swamp.yaml` + `workflow-*.yaml` | [swamp-automation](examples/swamp-automation/) |

Per-generator recipes and bridge flow examples: [CLI Reference](docs/cli-reference.md)

## Part of the ConfigHub platform

`cub-gen` is the local-first on-ramp. It runs standalone with no backend required.

When you need cross-repo queries, policy enforcement, and governed decisions, connect to [ConfigHub](https://github.com/confighubai/confighub). The flow: DRY files in Git &rarr; `cub-gen` classifies and traces &rarr; `publish` produces change bundles &rarr; ConfigHub ingests, enforces policy (ALLOW/ESCALATE/BLOCK) &rarr; bridge workers connect to clusters &rarr; Flux/Argo reconciles as before.

If you already use ConfigHub's GitOps Import wizard, think of it this way:

- ConfigHub imports cluster-discovered Argo/Flux applications.
- `cub-gen` imports source-side generators before they ever become opaque cluster objects.
- `cub-scout` inspects the cluster-side and controller-side reality after those objects reach runtime.

See the [platform docs](docs/platform.md) for the full story.

## Who this is for

Platform engineers, SREs, and app developers who want to know exactly what changed, who owns it, and where to fix it — without changing their existing deployment workflow.

## What cub-gen does vs what requires ConfigHub

| Capability | cub-gen (local) | ConfigHub (connected) |
|-----------|----------------|----------------------|
| Generator detection + DRY/WET classification | Yes | Yes |
| Field-origin tracing + inverse-edit guidance | Yes | Yes |
| Change CLI (preview, run, explain) | Yes | Yes |
| Evidence bundles (publish, verify, attest) | Yes | Yes |
| Cross-repo queries + policy enforcement | -- | Yes |
| Governed decisions (ALLOW/ESCALATE/BLOCK) | -- | Yes |
| Bridge workers + cluster integration | -- | Yes |

## Documentation

Docs currently live in this repo:

- [Docs index](docs/index.md) — overview and navigation
- [Getting Started](docs/getting-started.md) — 10-minute quickstart
- [CLI Reference](docs/cli-reference.md) — all commands, flags, and generator recipes
- [Demo Guide](docs/demo-guide.md) — runnable demo scripts and scenarios
- [Examples](examples/README.md) — complete runnable scenarios for every generator
- [Platform](docs/platform.md) — how cub-gen connects to ConfigHub
- [Persona 5-minute runbooks](docs/workflows/persona-5-minute-runbooks.md) — stack-specific entry paths
- [Change CLI contract](docs/contracts/change-cli-v1.md) — change preview/run/explain specification
- [Operation registry for real apps](docs/workflows/operation-registry-real-apps.md) — registry-backed platform governance

## Live reconciler proofs

```bash
./examples/demo/e2e-live-reconcile-flux.sh     # Flux on kind cluster
./examples/demo/e2e-live-reconcile-argo.sh     # Argo CD on kind cluster
```

Both prove create, update, and drift-correction on a real cluster.

## Execution status

| Status | What is true today |
|---|---|
| Strong now | Dual-mode example entrypoints exist across the main catalog; connected story scripts exist for stories 1-13; Flux and Argo live reconciler proofs run on real clusters |
| In progress | Flagship examples are still being hardened against the universal example contract: real-cluster outcome, two-audience path, visible ConfigHub value, and governed `ALLOW` plus `ESCALATE`/`BLOCK` proof |
| Actively tracked | Example reset execution is being driven through issues `#173`, `#177`, `#178`, `#179`, `#180`, `#182`, `#183`, `#185`, and `#187` |

For exact per-example counts and classifications, use the generated [Example Truth Matrix](docs/testing/example-truth-matrix.md). It is derived from the runnable catalog, source-side tests, connected runners, and real live-proof harnesses.

We have runnable paths for the full PRD story surface, but we are not treating every
story as fully acceptance-complete until the flagship examples and release gates
prove the new standard end to end.

## Test and contribute

```bash
make ci                # build + all tests (local)
make ci-connected      # connected mode tests (requires cub auth login)
go test ./...          # unit + golden + parity tests
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development rules, test requirements, and how to add a new generator.
