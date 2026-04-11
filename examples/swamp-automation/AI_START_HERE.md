# AI Start Here: `swamp-automation`

Use this example when the first operator may be an AI assistant and the real
authoring surface is prompt/context plus a workflow spec.

## What this proves

- structural workflow governance for agent-written workflow YAML
- prompt-as-DRY as a first-class governed path
- a local `ALLOW` path and a local `BLOCK` path for model/method and required-step policy
- a deeper connected ConfigHub walkthrough
- honest current limit: structural governance is stronger than live runtime proof here

## What you need installed

- Go 1.22+
- `jq`
- `cub` only for connected mode

## What this reads and writes

| Step | Reads | Writes | Mutates repo/backend/live? |
|---|---|---|---|
| `go build -o ./cub-gen ./cmd/cub-gen` | local source | `./cub-gen` binary | local binary only |
| `gitops import --json` | `workflow-deploy.yaml`, `.swamp.yaml`, `platform/registry.yaml`, `platform/swamp-constraints.yaml` | stdout JSON only | no |
| `prompt-as-dry-local.sh` | workflow files + AI-only guardrails | `.tmp/app-ai-change-run/<repo>-<timestamp>/...` | no repo/backend/live mutation |
| `demo-governed-structure.sh` | scratch clone + workflow files + Swamp constraints | `.tmp/swamp-governed-structure/<run>/...` | scratch clone only |
| `demo-local.sh` | example repo | temporary lifecycle simulation artifacts | no live/backend mutation |
| `run-connected-smoke.sh` | repo + ConfigHub auth/context | `.tmp/connected-smoke/<run>/...` | backend evidence only |
| `prompt-as-dry-connected.sh` | workflow files + ConfigHub auth/context | temporary connected lifecycle artifacts unless `OUTPUT_DIR` is set | backend evidence only |
| `demo-connected.sh` | repo + ConfigHub auth/context | temporary connected lifecycle artifacts unless `OUTPUT_DIR` is set | backend evidence only |

## Safest first step

Build once, then start with the read-only provenance preview:

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops import --space platform --json ./examples/swamp-automation \
  | jq '{
      profile: .discovered[0].generator_profile,
      dry_inputs,
      wet_manifest_targets,
      inverse_hint: .provenance[0].inverse_edit_pointers[0]
    }'
```

What it does:
- reads workflow files and platform constraints only
- writes nothing except stdout
- shows the workflow profile, governed outputs, and one inverse-edit hint

Success looks like:
- `generator_profile` is `swamp`
- `dry_inputs` is non-empty
- `wet_manifest_targets` is non-empty
- `inverse_hint.owner` is populated

The safest AI-first mutation-free next step is:

```bash
./examples/demo/prompt-as-dry-local.sh ./examples/swamp-automation
```

That path still avoids repo, backend, and live mutation; it writes only local
`.tmp/app-ai-change-run/...` artifacts after enforcing the AI-only guardrails.

## Safe run order

1. Repo-only provenance:
   `./cub-gen gitops import --space platform --json ./examples/swamp-automation`
2. AI-safe local loop:
   `./examples/demo/prompt-as-dry-local.sh ./examples/swamp-automation`
3. Local governed ALLOW/BLOCK proof:
   `./examples/swamp-automation/demo-governed-structure.sh`
4. Local structural walkthrough:
   `./examples/swamp-automation/demo-local.sh`
5. Connected environment check:
   `cub auth login && ./examples/demo/run-connected-smoke.sh`
6. AI-safe connected loop:
   `cub auth login && ./examples/demo/prompt-as-dry-connected.sh ./examples/swamp-automation`
7. Deep connected example walkthrough:
   `cub auth login && ./examples/swamp-automation/demo-connected.sh`

## What to verify after each major step

- Repo-only provenance: the import JSON includes workflow provenance and inverse-edit hints.
- AI-safe local loop: `.tmp/app-ai-change-run/.../mutation-card.json` exists and verification is true.
- Local governed proof: `.tmp/swamp-governed-structure/.../allow-summary.json` reports `ALLOW` and `block-summary.json` reports `BLOCK`.
- Local structural walkthrough: the create/update lifecycle completes with valid attestation outputs.
- Connected smoke: ConfigHub auth/context is valid and the smoke summaries include a non-empty `change_id`.
- AI-safe connected loop: the same prompt-as-DRY path reaches a connected decision state.
- Deep connected example walkthrough: a terminal decision state is returned for the workflow change.

## GUI checkpoints

- ConfigHub GUI: inspect the change referenced by `change_id` after connected prompt-as-DRY or deep connected mode.
- Audit/evidence view: compare the local mutation card with the connected decision state for the same workflow change.

## Trust order when outputs disagree

1. Trust the local mutation card and import JSON for source-to-governed evidence.
2. Trust the AI-only guardrail script for whether the prompt-as-DRY lane is even allowed to run.
3. Trust the connected backend decision output for final connected decision state.
4. Treat runtime claims as partial unless you have an external Swamp runtime proof.

## Cleanup

- Remove local proof artifacts with `rm -rf .tmp/app-ai-change-run .tmp/connected-smoke`
- If you set `OUTPUT_DIR` for connected runs, remove that directory when finished
