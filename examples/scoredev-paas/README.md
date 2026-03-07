# scoredev-paas example (platform + app)

This fixture demonstrates app-first DRY intent (`score.yaml`) with platform contracts and GitOps transport.

## Narrative turns

1. Prompt-first app change
- Developer updates workload image and env vars in `score.yaml`.
- `cub-gen` maps each change to WET field lineage.

2. Platform guardrail check
- Platform policy checks required probes/resources from contract files.
- Decision path remains explicit (`ALLOW | ESCALATE | BLOCK`).

3. Runtime rollout
- Flux/Argo syncs WET manifests.
- ConfigHub keeps attestation + provenance continuity.

4. Promotion
- Reusable app conventions can be promoted to platform defaults.

## Ownership map

- App-owned DRY: `score.yaml`, `app/*`
- Platform-owned DRY: `platform/contracts/*`, `platform/policies/*`
- Runtime reconcile: Flux/Argo
