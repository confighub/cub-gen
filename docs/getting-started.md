# Getting Started

Get from an existing GitOps repo to a first useful answer in under 10 minutes.

This guide is for teams that already have:

- a source repo in GitHub or Git,
- Helm, Score, Spring Boot, or workflow config as authoring input,
- Flux or Argo CD handling WET to LIVE reconciliation.

`cub-gen` adds source-side provenance. ConfigHub adds shared governance and
evidence. [`cub-scout`](https://github.com/confighub/cub-scout) adds
cluster-side inspection.

## Start from the setup you already have

| If you already have... | Start here | First value |
|----------|------------|-------------|
| Helm plus Flux/Argo | [`../examples/helm-paas`](../examples/helm-paas/) | Values ownership and rendered-field provenance |
| Spring Boot app repos | [`../examples/springboot-paas`](../examples/springboot-paas/) | App-vs-platform ownership in familiar config |
| A running cluster and GitOps controller | ConfigHub GitOps import + [`cub-scout`](https://github.com/confighub/cub-scout) + then `cub-gen` | Cluster/runtime view first, source provenance second |

## What cub-gen does

`cub-gen` is a deterministic generator importer. It reads your existing config
and tells you:

| Question | Answer |
|----------|--------|
| What type of project is this? | Generator detection (Helm, Score, Spring Boot, etc.) |
| Which files are human-editable intent? | DRY source classification |
| Which files are rendered output? | WET manifest classification |
| For any deployed field, where do I edit it? | Field-origin tracing + inverse-edit guidance |
| How do I prove what changed? | Verification, attestation, and evidence bundles |

## How the tools fit together

| Tool | Starts from | What it answers first |
|------|-------------|-----------------------|
| `cub-gen` | Source repo | Which DRY file/path produced this rendered field? |
| [`cub-scout`](https://github.com/confighub/cub-scout) | Cluster and controller reality | What is running and where is drift? |
| ConfigHub | Shared intended/evidence/governance state | What changed, what was approved, and what evidence exists? |

## The DRY → WET model

```
DRY source (what you author)     →  Generator  →  WET manifests (what gets deployed)
  values.yaml                         Helm           deployment.yaml
  score.yaml                          Score          service.yaml
  application.yaml                    Spring Boot    configmap.yaml
```

- **DRY**: The human-editable source of truth (e.g., `values.yaml`)
- **WET**: The expanded, hydrated output (e.g., rendered Kubernetes manifests)
- **Generator**: The tool that transforms DRY to WET (Helm, Score, etc.)

`cub-gen` traces this transformation so you always know which DRY file to edit
when you need to change a deployed value.

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- Git

## Build

```bash
git clone https://github.com/confighub/cub-gen.git
cd cub-gen
go build -o cub-gen ./cmd/cub-gen
```

## Use your own repo in 3 commands

```bash
REPO=/path/to/your/repo
./cub-gen gitops discover --space platform "$REPO"
./cub-gen gitops import --space platform --json "$REPO" "$REPO" \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
./cub-gen change preview --space platform "$REPO" "$REPO"
```

Connected mode for the same repo:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"
./cub-gen change run --mode connected --base-url "$BASE_URL" --token "$TOKEN" --space platform "$REPO" "$REPO"
```

## Your first import (Helm)

The three core commands mirror `cub gitops`:

### 1. Discover generator roots

```bash
./cub-gen gitops discover --space platform ./examples/helm-paas
```

This scans the repo and classifies it as a Helm generator (`helm-paas` profile).

### 2. Import with provenance

```bash
./cub-gen gitops import --space platform --json \
  ./examples/helm-paas ./examples/helm-paas | jq .
```

The import output includes:

- **`generator_profile`** — which generator family was detected
- **`dry_inputs`** — the human-editable source files (Chart.yaml, values.yaml)
- **`wet_manifest_targets`** — the rendered deployment artifacts
- **`provenance`** — field-origin map and inverse-edit pointers

### 3. Clean up discover state

```bash
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

## Inspect provenance

The key value of cub-gen is the provenance trail. Focus on the inverse-edit pointers:

```bash
./cub-gen gitops import --space platform --json \
  ./examples/helm-paas ./examples/helm-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      dry_inputs,
      wet_manifest_targets,
      provenance: .provenance[0] | {
        chart_path,
        values_paths,
        rendered_object_lineage
      }
    }'
```

This tells you: for any WET field, where is the DRY source to edit it safely.

## Why this matters right after import

Import should give you an immediate answer before any migration story:

1. what this repo renders,
2. which source file owns the deployed field you care about,
3. what evidence bundle you can verify next,
4. how to connect the repo-side answer to cluster-side inspection and ConfigHub.

That is why the first useful sequence is:

1. import the repo,
2. inspect provenance,
3. build or verify evidence,
4. compare that answer with runtime inspection.

## Try other generators

Each generator follows the same three-command flow:

=== "Score.dev"

    ```bash
    ./cub-gen gitops discover --space platform ./examples/scoredev-paas
    ./cub-gen gitops import --space platform --json \
      ./examples/scoredev-paas ./examples/scoredev-paas | jq .
    ./cub-gen gitops cleanup --space platform ./examples/scoredev-paas
    ```

=== "Spring Boot"

    ```bash
    ./cub-gen gitops discover --space platform ./examples/springboot-paas
    ./cub-gen gitops import --space platform --json \
      ./examples/springboot-paas ./examples/springboot-paas | jq .
    ./cub-gen gitops cleanup --space platform ./examples/springboot-paas
    ```

=== "Backstage IDP"

    ```bash
    ./cub-gen gitops discover --space platform ./examples/backstage-idp
    ./cub-gen gitops import --space platform --json \
      ./examples/backstage-idp ./examples/backstage-idp | jq .
    ./cub-gen gitops cleanup --space platform ./examples/backstage-idp
    ```

## Bridge artifacts (publish, verify, attest)

Generate a ConfigHub-ready change bundle with digest verification:

```bash
./cub-gen publish --space platform \
  ./examples/helm-paas ./examples/helm-paas \
  | jq '{schema_version, source, change_id, summary}'
```

Verify bundle integrity:

```bash
./cub-gen publish --space platform \
  ./examples/helm-paas ./examples/helm-paas \
  | ./cub-gen verify --in -
```

Emit an attestation record:

```bash
./cub-gen publish --space platform \
  ./examples/helm-paas ./examples/helm-paas \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{status, verifier, bundle_digest, attestation_digest}'
```

## If you are starting from a cluster, not a repo

Use ConfigHub GitOps import and [`cub-scout`](https://github.com/confighub/cub-scout)
first when your first question is "what is running?" rather than "what source
produced this?" Then come back to `cub-gen` when you want the source-side DRY to
WET answer for the field or workload you found.

## What happens after import? (Day-2)

Import is day 1. The real value shows on day 2:

| Day | What you do | What you gain |
|-----|-------------|---------------|
| **Day 1** | Import and explain | Field-origin tracing, ownership clarity, inverse-edit guidance |
| **Day 2** | Governed change | ALLOW/BLOCK decisions, policy enforcement via ConfigHub |
| **Day 3** | AI-assisted changes (optional) | Same governance for human and AI edits |

After import, your next steps are:

1. **Make a governed change**: Edit a DRY file, run `publish`, and see the decision
2. **Connect to ConfigHub**: Push the bundle to ConfigHub for cross-repo visibility
3. **Enable promotion**: Use ConfigHub to promote patterns to reusable base config

The governance model treats human and AI-assisted changes the same way:
verification, attestation, and policy enforcement are the safety boundary.

## Run demo modules

Five self-contained demo modules, each runnable independently:

```bash
./examples/demo/module-1-helm-import.sh
./examples/demo/module-2-score-field-map.sh
./examples/demo/module-3-spring-ownership.sh
./examples/demo/module-4-bridge-governance.sh
./examples/demo/module-5-no-config-platform.sh
```

Or all at once:

```bash
./examples/demo/run-all-modules.sh
```

## 10-minute adoption path (Helm/Flux/Argo)

What stays unchanged:

- Flux/Argo remains the reconciler for WET &rarr; LIVE
- Git/OCI remains the transport path
- Existing cluster/controller permissions and PR workflow stay in place

What you add:

- `cub-gen gitops discover` to classify generator roots
- `cub-gen gitops import` to emit DRY/WET contracts + provenance/inverse pointers
- `cub-gen gitops cleanup` to clear local discover state
- ConfigHub for shared evidence and governed decisions
- `cub-scout` for cluster-side inspection after reconciliation

Boundary language (aligned with [PARITY.md](parity.md)):

- **matched**: `gitops discover|import|cleanup` command shape and output contracts
- **matched**: bridge artifacts (`publish`, `verify`, `attest`, `verify-attestation`) symmetric across all 8 generators
- **partial**: local state/artifacts stand in for server-side units during this phase
- **partial**: bridge flow commands (`ingest`, `decision`, `promote`) produce correct contract shapes; [ConfigHub backend integration](platform.md) is the next step

## Terminology

| Term | Meaning in cub-gen |
|------|-------------------|
| DRY source | Human-editable app/platform intent (`values.yaml`, `score.yaml`, `application.yaml`) |
| WET rendered units | Explicit rendered deployment-facing units/manifests |
| Generator | Tool that transforms DRY to WET (Helm, Score, Spring Boot, etc.) |
| Provenance | Record of DRY inputs, rendered outputs, field-origin map, inverse-edit pointers |
| Inverse map | Guidance from changed WET field → where to edit DRY safely |
| Pre-sync | `cub-gen` stops before WET→LIVE; Flux/Argo own reconciliation |
| Verification | Cryptographic proof that a bundle is intact |
| Attestation | Record of who verified a bundle and when |
| Governance | Policy enforcement (ALLOW/ESCALATE/BLOCK) via ConfigHub decision engine |
| Mutation ledger | Audit trail of all config changes with evidence chain |

## Next steps

- [The ConfigHub Platform](platform.md) — how cub-gen connects to ConfigHub, bridge workers, and Flux/ArgoCD
- [CLI Reference](cli-reference.md) — full command and flag documentation
- [Architecture](agentic-gitops/02-design/00-agentic-gitops-design.md) — DRY/WET model, contract triples, governed execution
- [Worked Examples](agentic-gitops/03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md) — end-to-end Score.dev walkthrough
- [Adoption Path & FAQ](agentic-gitops/05-rollout/40-adoption-and-reference.md) — progressive adoption ladder
