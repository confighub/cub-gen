# AI Start Here: `springboot-paas`

Use this example when the question is not just "what rendered this field?" but
"should this Spring config be changed here, lifted upstream, or blocked?"

## What this proves

- Spring config provenance and ownership from `application*.yaml` plus platform policy
- a local `ALLOW` path and a local `BLOCKED` path for mutation routing
- direct embedded ConfigHub payload mutation for an app-owned field
- a deeper connected ConfigHub walkthrough
- a real standalone live-cluster app proof

## What you need installed

- Go 1.22+
- `jq`
- Java 21 + Maven if you want the app/runtime path
- `cub` only for connected mode
- kind, kubectl, and the helper scripts only for the live-cluster path

## What this reads and writes

| Step | Reads | Writes | Mutates repo/backend/live? |
|---|---|---|---|
| `go build -o ./cub-gen ./cmd/cub-gen` | local source | `./cub-gen` binary | local binary only |
| `./generator/render.sh --explain` | Spring config + platform policy | stdout only | no |
| `gitops import --json` | `application*.yaml`, `pom.xml`, `platform/`, `gitops/` | stdout JSON only | no |
| `demo-governed-routes.sh` | field-routes contract | `.tmp/springboot-governed-routes/<run>/...` | no repo/backend/live mutation |
| `demo-embedded-config-mutation.sh` | payload yaml + field routes | `.tmp/springboot-embedded-config/<run>/...` | scratch clone only |
| `run-connected-smoke.sh` | repo + ConfigHub auth/context | `.tmp/connected-smoke/<run>/...` | backend evidence only |
| `demo-connected.sh` | repo + ConfigHub auth/context | temporary connected lifecycle artifacts unless `OUTPUT_DIR` is set | backend evidence only |
| live-cluster proof | repo + cluster helpers | cluster objects, image build artifacts, worker install state | yes, live cluster |

## Safest first step

Build once, then use the read-only explanation and import paths:

```bash
go build -o ./cub-gen ./cmd/cub-gen
./examples/springboot-paas/generator/render.sh --explain
./cub-gen gitops import --space platform --json ./examples/springboot-paas \
  | jq '{
      profile: .discovered[0].generator_profile,
      dry_inputs,
      wet_manifest_targets,
      inverse_hint: .provenance[0].inverse_edit_pointers[0]
    }'
```

What it does:
- reads source config and platform policy only
- writes nothing except stdout
- explains the generator boundary before any mutation route proof

Success looks like:
- the explain output describes the Spring source/config boundary
- `generator_profile` is `springboot-paas`
- `dry_inputs` and `wet_manifest_targets` are non-empty

## Safe run order

1. Repo-only preview:
   `./examples/springboot-paas/generator/render.sh --explain`
2. Repo-only provenance:
   `./cub-gen gitops import --space platform --json ./examples/springboot-paas`
3. Local route proof:
   `./examples/springboot-paas/demo-governed-routes.sh`
4. Local apply-here proof:
   `./examples/springboot-paas/demo-embedded-config-mutation.sh`
5. Connected environment check:
   `cub auth login && ./examples/demo/run-connected-smoke.sh`
6. Deep connected walkthrough:
   `cub auth login && ./examples/springboot-paas/demo-connected.sh`
7. Live-cluster proof:
   `./bin/create-cluster && ./bin/build-image && ./bin/install-worker && ./verify-e2e.sh`

## What to verify after each major step

- Repo-only preview: the explain output and import JSON agree on the source/config boundary.
- Local route proof: `feature.inventory.reservationMode` is allowed and `spring.datasource.url` is blocked; artifacts land under `.tmp/springboot-governed-routes/...`.
- Local apply-here proof: the reservation mode changes in the embedded payload and the datasource mutation is rejected; artifacts land under `.tmp/springboot-embedded-config/...`.
- Connected smoke: ConfigHub auth/context is valid and the smoke summaries include a non-empty `change_id`.
- Deep connected walkthrough: a terminal connected decision state is returned for the same `change_id`.
- Live-cluster proof: `verify-e2e.sh` succeeds and `inventory-api` is reachable on the kind cluster.

## GUI checkpoints

- ConfigHub GUI: inspect the change referenced by `change_id` after smoke or deep connected mode.
- Spring payload helpers: compare and refresh-preview outputs should show the app-owned override preserved.
- Live runtime: inspect `inventory-api` health and cluster objects after `verify-e2e.sh`.

## Trust order when outputs disagree

1. Trust the generator explain/import output for ownership and field routing.
2. Trust `springboot validate-mutation` and embedded-config mutation results for local route enforcement.
3. Trust the connected backend decision output for connected decision state.
4. Trust live HTTP and kubectl checks for runtime truth.

## Cleanup

- Remove local proof artifacts with `rm -rf .tmp/springboot-governed-routes .tmp/springboot-embedded-config .tmp/connected-smoke`
- Tear down clusters and workers with the same helper scripts used for the live path
