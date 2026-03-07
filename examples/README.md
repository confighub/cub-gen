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

## v0.2 preview examples

4. `backstage-idp`
5. `ably-config`
6. `ops-workflow`

## Demo rule

All examples assume:
- Kubernetes runtime
- Git + OCI transport
- Flux or Argo reconciliation
- ConfigHub governance and API layer on top
