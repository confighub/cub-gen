# Generator Families

Total: 8

| Kind | Profile | Resource Kind | Resource Type | Capabilities |
| --- | --- | --- | --- | --- |
| `ably` | `ably-config` | `ConfigMap` | `v1/ConfigMap` | app-config-only, provider-config, inverse-provider-config-patch |
| `backstage` | `backstage-idp` | `Component` | `backstage.io/v1alpha1/Component` | catalog-metadata, render-manifests, inverse-catalog-patch |
| `c3agent` | `c3agent` | `ConfigMap` | `v1/ConfigMap` | fleet-config, agent-orchestration, inverse-fleet-config-patch |
| `helm` | `helm-paas` | `HelmRelease` | `helm.toolkit.fluxcd.io/v2/HelmRelease` | render-manifests, values-overrides, inverse-values-patch |
| `opsworkflow` | `ops-workflow` | `Workflow` | `argoproj.io/v1alpha1/Workflow` | workflow-plan, governed-execution-intent, inverse-workflow-patch |
| `score` | `scoredev-paas` | `Application` | `argoproj.io/v1alpha1/Application` | render-manifests, workload-spec, inverse-score-patch |
| `springboot` | `springboot-paas` | `Kustomization` | `kustomize.toolkit.fluxcd.io/v1/Kustomization` | render-app-config, profile-overrides, inverse-app-config-patch |
| `swamp` | `swamp` | `Workflow` | `swamp.dev/v1/Workflow` | workflow-automation, model-orchestration, inverse-workflow-patch |
