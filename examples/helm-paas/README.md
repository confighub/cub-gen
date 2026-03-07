# helm-paas example (platform + app)

This fixture models a realistic platform + app repo where a platform team owns the chart contract and runtime guardrails, while an app team owns selected values.

## Narrative turns

1. App feature change
- App team bumps `image.tag` and toggles an app flag in `values.yaml`.
- `cub-gen gitops import` shows this as app-editable DRY (`values.*`) with inverse pointers.

2. Environment rollout
- Ops applies override in `values-prod.yaml` for replicas/resources.
- Import output keeps the env override lineage explicit.

3. Governed decision
- Bundle + attestation move through `bridge decision` state.
- Platform approver issues explicit `ALLOW | ESCALATE | BLOCK`.

4. Promotion upstream
- After successful rollout, promotion path opens a platform DRY PR for reusable defaults.

## Ownership map

- Platform-owned DRY: `Chart.yaml`, `templates/*`, `platform/*`
- App-owned DRY: `values.yaml`, selected keys in `values-prod.yaml`
- Runtime reconcile: Flux/Argo consume WET artifacts via Git/OCI

## Key files

- Chart: `Chart.yaml`
- App defaults: `values.yaml`
- Prod overrides: `values-prod.yaml`
- Flux transport sample: `gitops/flux/helmrelease.yaml`
- Argo transport sample: `gitops/argo/application.yaml`
