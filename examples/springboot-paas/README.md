# springboot-paas example (platform + app)

This fixture models a typical internal Java service repo where app teams edit business config and platform teams own runtime and shared dependency policy.

## Narrative turns

1. Feature rollout
- App team changes `server.port` and feature flags in `application-prod.yaml`.
- `cub-gen` reports those edits as app-team DRY with inverse pointers.

2. Platform safety check
- Platform verifies datasource/runtime boundaries and policy evidence.
- Governance decision is explicit before apply.

3. Runtime reconciliation
- Flux/Argo reconciles rendered WET manifests from Git/OCI.

4. Upstream promotion
- Reusable app-level defaults can be promoted to platform base after successful rollout.

## Ownership map

- App-owned DRY: `src/main/resources/application*.yaml`, `src/main/java/*`
- Platform-owned DRY: `platform/*`
- Runtime reconcile: Flux/Argo
