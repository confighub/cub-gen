# Example repos for demo narratives

These fixtures are intentionally small but realistic enough to demonstrate platform + app collaboration and governance turns.

## Core examples (MVP demos)

1. `helm-paas`
- Platform-owned chart contract + app-owned values overlays.
- Shows DRY value edits, env rollout, and guarded promotion.

2. `scoredev-paas`
- App-first workload intent with platform contracts/policies.
- Shows field-origin and inverse-edit mapping.

3. `springboot-paas`
- Application code + app config + platform runtime policy split.
- Shows app/team ownership boundaries in inverse pointers.

4. `ably-config`
- App-config style platform integration with explicit inverse edit guidance.
- Included in the core PaaS demo path alongside Spring.

## AI work platform scenarios

5. `jesper-ai-cloud`
6. `swamp-project`
7. `confighub-actions`
8. `ops-workflow`

Run the AI track scripts under:
- `examples/demo/ai-work-platform/`

## Additional preview examples

9. `backstage-idp`

## Demo rule

All examples assume:
- Kubernetes runtime
- Git + OCI transport
- Flux or Argo reconciliation
- ConfigHub governance and API layer on top
