# AI Start Here: `scoredev-paas`

Use this example when developers keep `score.yaml` as the app-team contract and
you need to explain how that intent becomes governed runtime config.

## What this proves

- Score field-origin tracing from `score.yaml` to rendered runtime fields
- a local `ALLOW` path and a local `ESCALATE` path for workload contracts
- a standalone live `checkout-api` runtime on kind from the merged Score inputs
- a deeper connected ConfigHub walkthrough

## What you need installed

- Go 1.22+
- Docker
- kind
- kubectl
- `jq`
- `cub` only for connected mode

## What this reads and writes

| Step | Reads | Writes | Mutates repo/backend/live? |
|---|---|---|---|
| `go build -o ./cub-gen ./cmd/cub-gen` | local source | `./cub-gen` binary | local binary only |
| `gitops discover/import` | `score.yaml`, `platform/contracts`, `platform/policies`, `gitops/` | stdout JSON only | no |
| `score validate-workload` | `score.yaml`, workload class contract | stdout only | no |
| `demo-governed-workload.sh` | example repo + contract | `.tmp/scoredev-governed-workload/<run>/...` | scratch clone only |
| `demo-runtime.sh` | example repo + Docker + kind | `var/runtime-manifests.yaml`, kind cluster state | live cluster only |
| `run-connected-smoke.sh` | repo + ConfigHub auth/context | `.tmp/connected-smoke/<run>/...` | backend evidence only |
| `demo-connected.sh` | repo + ConfigHub auth/context | temporary connected lifecycle artifacts unless `OUTPUT_DIR` is set | backend evidence only |

## Safest first step

Build once, then run a read-only preview plus a read-only contract check:

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops import --space platform --json ./examples/scoredev-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      dry_inputs,
      wet_manifest_targets,
      inverse_hint: .provenance[0].inverse_edit_pointers[0]
    }'
./cub-gen score validate-workload \
  --score ./examples/scoredev-paas/score.yaml \
  --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml
```

What it does:
- reads the Score example only
- writes nothing except stdout
- shows field provenance and whether the current workload stays inside contract

Success looks like:
- `generator_profile` is `scoredev-paas`
- `dry_inputs` includes `score.yaml`
- `wet_manifest_targets` is non-empty
- the workload contract returns an allowed result for the current fixture

## Safe run order

1. Repo-only preview:
   `./cub-gen gitops import --space platform --json ./examples/scoredev-paas`
2. Read-only contract check:
   `./cub-gen score validate-workload --score ./examples/scoredev-paas/score.yaml --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml`
3. Local governed proof:
   `./examples/scoredev-paas/demo-governed-workload.sh`
4. Standalone runtime proof:
   `./examples/scoredev-paas/demo-runtime.sh`
5. Connected environment check:
   `cub auth login && ./examples/demo/run-connected-smoke.sh`
6. Deep connected walkthrough:
   `cub auth login && ./examples/scoredev-paas/demo-connected.sh`

## What to verify after each major step

- Repo-only preview: the import JSON includes `score.yaml` lineage and inverse-edit hints.
- Read-only contract check: the current example returns an allowed result without mutating anything.
- Local governed proof: the image update stays allowed and the unapproved resource type escalates; artifacts land under `.tmp/scoredev-governed-workload/...`.
- Standalone runtime proof: the kind cluster runs `checkout-api`, `/healthz` returns `ok`, `/` returns `service=checkout-api`, and `logLevel=warn` from the prod overlay.
- Connected smoke: ConfigHub auth/context is valid and the smoke summaries include a non-empty `change_id`.
- Deep connected walkthrough: a terminal connected decision state is returned for the same `change_id`.

## GUI checkpoints

- ConfigHub GUI: inspect the change referenced by `change_id` after smoke or deep connected mode.
- Runtime note: the standalone live Score proof is local cluster evidence, not a ConfigHub-connected apply path.

## Trust order when outputs disagree

1. Trust local `gitops import --json` for source-to-render mapping.
2. Trust `score validate-workload` for current workload-class contract evaluation.
3. Trust the connected backend decision output for final connected decision state.

## Cleanup

- Remove local proof artifacts with `rm -rf .tmp/scoredev-governed-workload .tmp/connected-smoke`
- Remove the local kind cluster with `kind delete cluster --name scoredev-runtime`
