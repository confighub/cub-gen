# AI Start Here: `helm-paas`

Use this example when the repo already uses Helm plus Flux or Argo and the
first question is: "which values file or chart layer controls this field?"

## What this proves

- Helm field-origin tracing from chart/value inputs to rendered Kubernetes fields
- a local `ALLOW` path and a local `BLOCK` path for ownership boundaries
- a deeper connected ConfigHub walkthrough
- optional live Flux/Argo runtime proof

## What you need installed

- Go 1.22+
- `jq`
- `cub` only for connected mode
- `kubectl` and kind only for the live runtime path

## What this reads and writes

| Step | Reads | Writes | Mutates repo/backend/live? |
|---|---|---|---|
| `go build -o ./cub-gen ./cmd/cub-gen` | local source | `./cub-gen` binary | local binary only |
| `gitops discover/import` | `Chart.yaml`, `values*.yaml`, `templates/`, `platform/`, `gitops/` | stdout JSON only | no |
| `demo-governed-change.sh` | example repo + ownership gate scripts | `.tmp/helm-paas-governed-change/<run>/...` | scratch clone only |
| `run-connected-smoke.sh` | repo + ConfigHub auth/context | `.tmp/connected-smoke/<run>/...` | backend evidence only |
| `demo-connected.sh` | repo + ConfigHub auth/context | temporary connected lifecycle artifacts unless `OUTPUT_DIR` is set | backend evidence only |
| `demo-runtime.sh` | repo + ConfigHub auth/context + clusters | `.tmp/helm-paas-runtime/...` plus live namespace objects | yes, live cluster |

## Safest first step

Build once, then start with a read-only preview:

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      dry_inputs,
      wet_manifest_targets,
      inverse_hint: .provenance[0].inverse_edit_pointers[0]
    }'
```

What it does:
- reads the Helm example only
- writes nothing except stdout
- shows the generator profile, DRY inputs, rendered targets, and one edit hint

Success looks like:
- `generator_profile` is `helm-paas`
- `dry_inputs` includes values/chart inputs
- `wet_manifest_targets` is non-empty
- `inverse_hint.owner` is populated
- `inverse_hint.edit_hint` points at `values-prod.yaml` for the prod override and mentions `values.yaml` as the default

## Safe run order

1. Repo-only preview:
   `./cub-gen gitops import --space platform --json ./examples/helm-paas`
2. Local governed proof:
   `./examples/helm-paas/demo-governed-change.sh`
3. Connected environment check:
   `cub auth login && ./examples/demo/run-connected-smoke.sh`
4. Deep connected walkthrough:
   `cub auth login && ./examples/helm-paas/demo-connected.sh`
5. Live runtime proof:
   `cub auth login && RECONCILER=both ./examples/helm-paas/demo-runtime.sh`

## What to verify after each major step

- Repo-only preview: `dry_inputs`, `wet_manifest_targets`, and one inverse-edit pointer are present.
- Local governed proof: the allowed path passes and the template edit is rejected; artifacts land under `.tmp/helm-paas-governed-change/...`.
- Connected smoke: ConfigHub auth/context is valid and the flagship smoke summaries include a non-empty `change_id`.
- Deep connected walkthrough: a terminal decision state is returned for the same `change_id`.
- Live runtime: Flux and/or Argo report a healthy rollout and `kubectl get deploy,pods,svc` shows live objects.

## GUI checkpoints

- ConfigHub GUI: inspect the change referenced by `change_id` after smoke or deep connected mode.
- Flux/Argo UI or kubectl: inspect the live `payments-api` deployment after `demo-runtime.sh`.

## Trust order when outputs disagree

1. Trust local `gitops import --json` for source-to-render mapping.
2. Trust the connected backend decision output for `ALLOW/ESCALATE/BLOCK`.
3. Trust live Flux/Argo and `kubectl` for runtime truth.

## Cleanup

- Remove local proof artifacts with `rm -rf .tmp/helm-paas-governed-change .tmp/connected-smoke .tmp/helm-paas-runtime`
- Remove live namespaces/clusters with the same scripts you normally use for the live reconcile harness
