# cub-gen

See where every deployed config value came from.

`cub-gen` is a governance + traceability sidecar for GitOps.

**gen = generator.** A generator is a function that maps DRY source (your `values.yaml`, `score.yaml`, etc.) to WET rendered output (the manifests that reach your cluster). `cub-gen` detects which generators your repo uses, runs the mapping, and records provenance — so every deployed field traces back to a source file, line, and owner.

If you already run app/config in Git, OCI artifacts, and Flux/Argo reconciliation, `cub-gen` answers:

- Which source file/path controls this live field?
- Did the right team edit the right thing?
- Can we block unsafe edits before they hit cluster?

It does this by classifying your existing repo and mapping rendered fields back
to source file, line, and owner.

## What it is not

- Not a Kubernetes reconciler
- Not a Flux/Argo replacement
- Not an OCI replacement

Flux/Argo still reconcile to LIVE. `cub-gen` adds governance before deploy and traceability after deploy.

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

## Quickstart (local, no login)

```bash
go build -o ./cub-gen ./cmd/cub-gen

# One-command change preview
./cub-gen change preview --space platform ./examples/scoredev-paas ./examples/scoredev-paas

# Or the core flow
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
```

## Use Your Repo in 3 Commands

```bash
REPO=/path/to/your/repo
./cub-gen change preview --space platform "$REPO" "$REPO"
./cub-gen change run --mode local --space platform "$REPO" "$REPO"
./cub-gen change explain --space platform --owner app-team "$REPO" "$REPO"
```

## Quickstart (connected, ConfigHub)

```bash
cub auth login
TOKEN="$(cub auth get-token)"
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen change run --mode connected --base-url "$BASE_URL" --token "$TOKEN" \
  --space platform ./examples/helm-paas ./examples/helm-paas
```

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

Per-generator recipes and bridge flow examples: [CLI Reference](https://confighub.github.io/cub-gen/cli-reference/)

## Part of the ConfigHub platform

`cub-gen` is the local-first on-ramp. It runs standalone with no backend required.

When you need cross-repo queries, policy enforcement, and governed decisions, connect to [ConfigHub](https://github.com/confighubai/confighub). The flow: DRY files in Git &rarr; `cub-gen` classifies and traces &rarr; `publish` produces change bundles &rarr; ConfigHub ingests, enforces policy (ALLOW/ESCALATE/BLOCK) &rarr; bridge workers connect to clusters &rarr; Flux/Argo reconciles as before.

See the [platform docs](https://confighub.github.io/cub-gen/platform/) for the full story.

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

Full docs: **https://confighub.github.io/cub-gen/**

- [Getting Started](https://confighub.github.io/cub-gen/getting-started/) — 10-minute quickstart
- [CLI Reference](https://confighub.github.io/cub-gen/cli-reference/) — all commands, flags, and generator recipes
- [Demo Guide](https://confighub.github.io/cub-gen/demo-guide/) — runnable demo scripts and scenarios
- [Examples](examples/README.md) — complete runnable scenarios for every generator
- [Platform](https://confighub.github.io/cub-gen/platform/) — how cub-gen connects to ConfigHub
- [Persona 5-minute runbooks](docs/workflows/persona-5-minute-runbooks.md) — stack-specific entry paths
- [Change CLI contract](docs/contracts/change-cli-v1.md) — change preview/run/explain specification
- [Operation registry for real apps](docs/workflows/operation-registry-real-apps.md) — registry-backed platform governance

## Live reconciler proofs

```bash
./examples/demo/e2e-live-reconcile-flux.sh     # Flux on kind cluster
./examples/demo/e2e-live-reconcile-argo.sh     # Argo CD on kind cluster
```

Both prove create, update, and drift-correction on a real cluster.

## User-story coverage

| Status | User stories |
|---|---|
| Met/strong in current demos | 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13 |
| Partial (simulated/local-first, not full backend/runtime integration) | None |
| Deferred | None |

## Test and contribute

```bash
make ci                # build + all tests (local)
make ci-connected      # connected mode tests (requires cub auth login)
go test ./...          # unit + golden + parity tests
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development rules, test requirements, and how to add a new generator.
