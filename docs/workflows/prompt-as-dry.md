# Prompt as DRY (Worked Example)

This is the canonical answer to: "What does prompt-as-DRY look like in a real workflow?"

## The idea

A human or agent writes high-level intent first.

Example intent:

```text
Deploy checkout-service to staging.
Then run a validation step.
Only use approved workflow models.
```

`cub-gen` does not generate this intent for you. It governs the handoff from the intent artifact (DRY) to governed outputs (WET evidence + decisions) and then to LIVE reconciliation via Flux/Argo.

## The DRY artifact (compiled from prompt)

In this repo, the compiled DRY artifact is:

- `examples/swamp-automation/workflow-deploy.yaml`

And the platform guardrails are:

- `examples/swamp-automation/platform/swamp-constraints.yaml`

Together, these represent "prompt as DRY": intent captured as a short workflow spec plus policy.

## Local mode (no login): fast proof

Run one command:

```bash
./examples/demo/prompt-as-dry-local.sh
```

This executes:

1. `gitops import`
2. `publish`
3. `verify`
4. `attest`
5. mutation-card projection

You get a mutation card with exactly what a developer/operator needs:

```json
{
  "change": {
    "change_id": "chg_...",
    "bundle_digest": "sha256:...",
    "attestation_digest": "sha256:..."
  },
  "edit_recommendation": {
    "owner": "app-team",
    "wet_path": "Workflow/spec/jobs",
    "dry_path": "jobs[].steps[].task",
    "edit_hint": "Edit jobs and steps in the workflow YAML file.",
    "confidence": 0.9
  }
}
```

Artifact output path:

- `.tmp/app-ai-change-run/<repo>-<timestamp>/mutation-card.json`

## Connected mode (ConfigHub): governed run

All connected flows start with auth:

```bash
cub auth login
```

Then run a connected lifecycle for the same DRY artifact:

```bash
./examples/demo/prompt-as-dry-connected.sh
```

This path performs real backend ingest/query and writes per-phase summaries:

- `.tmp/connected-lifecycle/prompt-dry-swamp/create/summary.json`
- `.tmp/connected-lifecycle/prompt-dry-swamp/update/summary.json`

Each summary includes the governed decision state for the same `change_id` lifecycle.

## Where "preview / run / explain" fits today

Until first-class `cub-gen change ...` commands land, use this mapping:

- `change preview` -> `cub-gen gitops import --json ...` (look at provenance + inverse pointers)
- `change run` -> `app-ai-change-run.sh` (local) or `run-confighub-lifecycle-connected.sh` (connected)
- `change explain` -> inspect `inverse_edit_pointers` / `mutation-card.json` edit recommendation

## Why this is compelling

It gives one closed loop for humans and agents:

1. intent (prompt) is compiled into a DRY workflow file,
2. governed artifacts are produced and verified,
3. ConfigHub decision state is queryable in connected mode,
4. inverse edit guidance tells you exactly what to change next.

That is DRY -> WET -> LIVE readiness with verification/attestation as the safety loop.

## Brownfield mapping (same pattern)

The same loop works for existing stacks:

- Helm: `./examples/helm-paas/demo-local.sh` or `./examples/helm-paas/demo-connected.sh`
- Spring Boot: `./examples/springboot-paas/demo-local.sh` or `./examples/springboot-paas/demo-connected.sh`

No workflow migration is required; the governance loop is additive.
