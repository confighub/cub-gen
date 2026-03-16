# Prompt as DRY (Worked Example)

This is the canonical answer to: "What does prompt-as-DRY look like in a real workflow?"

## The conceptual model

Human and AI-assisted changes can run through the same governed path:

| Concept | How it maps |
|---------|-------------|
| **Prompt + context** | DRY input (what the team or agent authors) |
| **LLM/agent layer** | Non-deterministic generator (produces WET output) |
| **Verification + attestation** | Safety boundary (makes non-determinism governable) |
| **Mutation ledger** | Compliance and forensics proof |

The key insight: an LLM behaves like a non-deterministic generator. The same
verification, attestation, and governance that makes deterministic generators
safe also makes AI-assisted changes safe.

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

## AI-only guardrails (pilot)

Prompt-first AI-only lanes are intentionally narrow during rollout:

- allowed repos/examples: `swamp-automation`, `ops-workflow`
- hard deny patterns: `cluster-admin`, `system:masters`, destructive namespace deletion patterns
- mandatory rollback/revert hook in workflow YAML

Guardrails are enforced by demo scripts before import/publish:

- `examples/demo/prompt-as-dry-local.sh`
- `examples/demo/prompt-as-dry-connected.sh`

Policy details:

- [AI-only guardrails](ai-only-guardrails.md)

## Where "preview / run / explain" fits

Use the first-class commands directly:

- `change preview` -> compact mutation card and evidence summary
- `change run` -> local or connected governed execution in one command
- `change explain` -> direct inverse-edit answer for a selected field/owner filter

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
