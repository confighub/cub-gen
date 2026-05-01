# Generator Families

Total: 11

| Kind | Profile | Resource Kind | Resource Type | Capabilities |
| --- | --- | --- | --- | --- |
| `app-of-apps` | `app-of-apps` | `Application` | `argoproj.io/v1alpha1/Application` | root-application, child-application-catalog, observed-expansion, inverse-app-catalog-patch |
| `applicationset` | `applicationset` | `ApplicationSet` | `argoproj.io/v1alpha1/ApplicationSet` | observed-expansion, authoritative-list-expansion, authoritative-clusters-expansion, graceful-degradation |
| `backstage` | `backstage-idp` | `Component` | `backstage.io/v1alpha1/Component` | catalog-metadata, render-manifests, inverse-catalog-patch |
| `c3agent` | `c3agent` | `ConfigMap` | `v1/ConfigMap` | fleet-config, agent-orchestration, inverse-fleet-config-patch |
| `helm` | `helm-paas` | `HelmRelease` | `helm.toolkit.fluxcd.io/v2/HelmRelease` | render-manifests, values-overrides, inverse-values-patch |
| `no-config-platform` | `no-config-platform` | `ConfigMap` | `v1/ConfigMap` | app-config-only, provider-config, inverse-provider-config-patch |
| `openchoreo` | `openchoreo` | `Workload` | `openchoreo.dev/v1alpha1/Workload` | platform-crds, rendered-release, generated-resource-ownership, inverse-route-patch, adoption-report |
| `opsworkflow` | `ops-workflow` | `Workflow` | `argoproj.io/v1alpha1/Workflow` | workflow-plan, governed-execution-intent, inverse-workflow-patch |
| `score` | `scoredev-paas` | `Application` | `argoproj.io/v1alpha1/Application` | render-manifests, workload-spec, inverse-score-patch |
| `springboot` | `springboot-paas` | `Kustomization` | `kustomize.toolkit.fluxcd.io/v1/Kustomization` | render-app-config, profile-overrides, inverse-app-config-patch |
| `swamp` | `swamp` | `Workflow` | `swamp.dev/v1/Workflow` | workflow-automation, model-orchestration, inverse-workflow-patch |

## `app-of-apps`

- Profile: `app-of-apps`
- Resource: `Application` (`argoproj.io/v1alpha1/Application`)
- Capabilities: root-application, child-application-catalog, observed-expansion, inverse-app-catalog-patch
- Default input role: `argo-application`
- Default owner: `app-catalog-owner`
- Field-origin transform: `app-of-apps-catalog`
- Field-origin overlay transform: `app-of-apps-root-expansion`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `root-application` | root-application.yaml, root-application.yml, root-app.yaml, root-app.yml, app-of-apps.yaml, app-of-apps.yml | - | - | - |
| `child-application` | - | apps/, applications/ | - | .yaml, .yml |

### Role Owners
| Role | Owner |
| --- | --- |
| `child-application` | `app-catalog-owner` |
| `root-application` | `platform-engineer` |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `child_name` | `app-catalog-owner` | 0.92 | `false` |
| `root_path` | `platform-engineer` | 0.88 | `true` |
| `source_path` | `app-catalog-owner` | 0.90 | `true` |
| `source_repo` | `app-catalog-owner` | 0.90 | `true` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `child_name` | `app-catalog-owner` | 0.92 |
| `root_path` | `platform-engineer` | 0.88 |
| `source_path` | `app-catalog-owner` | 0.90 |
| `source_repo` | `app-catalog-owner` | 0.90 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `child_name` | 0.92 |
| `root_path` | 0.88 |
| `source_path` | 0.90 |
| `source_repo` | 0.90 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `child_catalog_path` | `apps` |
| `root_application_path` | `root-application.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `child_name` | Child Application identity is owned by the child app catalog. |
| `root_path` | The root Application controls which catalog path is expanded. |
| `source_path` | Child Application source path is selected by the child app catalog. |
| `source_repo` | Child Application repo URL is selected by the child app catalog. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `child_name` | Route: apply-here. Edit metadata.name in {{child_application_path}}. |
| `root_path` | Route: lift-upstream. Edit spec.source.path in {{root_application_path}} to change the child app catalog. |
| `source_path` | Route: apply-here. Edit spec.source.path in {{child_application_path}}. |
| `source_repo` | Route: apply-here. Edit spec.source.repoURL in {{child_application_path}}. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Application` | `{{name}}` | `platform-runtime` | `argocd` | `spec.source.path` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Application` | `{{name}}` | `argocd` | `root_application_path` | `-` | `false` | `spec.source.path` | `false` |

## `applicationset`

- Profile: `applicationset`
- Resource: `ApplicationSet` (`argoproj.io/v1alpha1/ApplicationSet`)
- Capabilities: observed-expansion, authoritative-list-expansion, authoritative-clusters-expansion, graceful-degradation
- Default input role: `applicationset-input`
- Default owner: `platform-engineer`
- Field-origin transform: `applicationset-template`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `application-set` | applicationset.yaml, applicationset.yml | - | - | - |
| `cluster-inventory` | - | clusters/, inventory/clusters/, platform/clusters/ | - | .yaml, .yml, .json |

### Role Owners
| Role | Owner |
| --- | --- |
| `application-set` | `platform-engineer` |
| `cluster-inventory` | `platform-engineer` |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `child_name` | `platform-engineer` | 0.89 | `false` |
| `source_path` | `platform-engineer` | 0.86 | `true` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `child_name` | `platform-engineer` | 0.89 |
| `source_path` | `platform-engineer` | 0.86 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `child_name` | 0.89 |
| `source_path` | 0.86 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `application_set_path` | `applicationset.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `child_name` | Child Application identity is generated from the parent ApplicationSet template. |
| `source_path` | Child Application source path is generated from the parent ApplicationSet template. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `child_name` | Edit spec.template.metadata.name in {{application_set_path}}. |
| `source_path` | Edit spec.template.spec.source.path in {{application_set_path}}. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `ApplicationSet` | `{{name}}` | `platform-runtime` | `argocd` | `-` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ApplicationSet` | `{{name}}` | `argocd` | `application_set_path` | `-` | `false` | `spec` | `false` |

## `backstage`

- Profile: `backstage-idp`
- Resource: `Component` (`backstage.io/v1alpha1/Component`)
- Capabilities: catalog-metadata, render-manifests, inverse-catalog-patch
- Default input role: `backstage-input`
- Default owner: `platform-engineer`
- Field-origin transform: `backstage-component-to-application`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `catalog-spec` | catalog-info.yaml, catalog-info.yml | - | - | - |
| `app-config` | app-config.yaml, app-config.yml | - | - | - |

### Role Owners
| Role | Owner |
| --- | --- |
| `app-config` | `app-team` |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `identity` | `platform-engineer` | 0.87 | `false` |
| `lifecycle` | `platform-engineer` | 0.82 | `true` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `lifecycle` | `platform-engineer` | 0.82 |
| `name` | `platform-engineer` | 0.90 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `identity` | 0.90 |
| `lifecycle` | 0.82 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `catalog_path` | `catalog-info.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `identity` | Backstage component identity is sourced from {{catalog_path}}. |
| `lifecycle` | Lifecycle changes impact platform ownership and support policy. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `lifecycle` | Edit spec.lifecycle in {{catalog_path}} and coordinate rollout policy. |
| `name` | Edit metadata.name in {{catalog_path}}. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Application` | `{{name}}` | `platform-runtime` | `apps` | `metadata.name` |
| `ConfigMap` | `{{name}}-catalog` | `platform-runtime` | `apps` | `spec.lifecycle` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Application` | `{{name}}` | `apps` | `catalog_path` | `-` | `false` | `metadata.name` | `false` |
| `ConfigMap` | `{{name}}-catalog` | `apps` | `catalog_path` | `-` | `false` | `spec.lifecycle` | `false` |

## `c3agent`

- Profile: `c3agent`
- Resource: `ConfigMap` (`v1/ConfigMap`)
- Capabilities: fleet-config, agent-orchestration, inverse-fleet-config-patch
- Default input role: `fleet-config`
- Default owner: `app-team`
- Field-origin transform: `c3agent-config-to-runtime`
- Field-origin overlay transform: `c3agent-overlay-merge`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `fleet-config-base` | c3agent.yaml, c3agent.yml, c3agent.json | - | - | - |
| `fleet-config-overlay` | - | - | c3agent- | .yaml, .yml, .json |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `agent_runtime` | `platform-engineer` | 0.88 | `true` |
| `component_ports` | `platform-engineer` | 0.84 | `true` |
| `credentials` | `platform-engineer` | 0.86 | `true` |
| `fleet_config` | `app-team` | 0.91 | `false` |
| `rbac` | `platform-engineer` | 0.82 | `true` |
| `replicas` | `app-team` | 0.89 | `false` |
| `storage` | `platform-engineer` | 0.85 | `true` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `agent_runtime` | `platform-engineer` | 0.88 |
| `component_ports` | `platform-engineer` | 0.84 |
| `credentials` | `platform-engineer` | 0.86 |
| `fleet_config` | `app-team` | 0.91 |
| `rbac` | `platform-engineer` | 0.82 |
| `replicas` | `app-team` | 0.89 |
| `storage` | `platform-engineer` | 0.85 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `agent_runtime` | 0.88 |
| `component_ports_base` | 0.84 |
| `component_ports_overlay` | 0.80 |
| `credentials` | 0.86 |
| `fleet_config` | 0.91 |
| `rbac` | 0.82 |
| `replicas_base` | 0.89 |
| `replicas_overlay` | 0.85 |
| `storage` | 0.85 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `base_config_path` | `c3agent.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `agent_runtime` | Agent runtime image and budget settings affect platform execution behavior. |
| `component_ports` | Component port changes affect platform networking and service mesh. |
| `credentials` | Credential references impact platform secret management. |
| `fleet_config` | Fleet configuration (model, concurrency) is sourced from {{base_config_path}}. |
| `rbac` | RBAC resources are platform-governed and must align with security policy. |
| `replicas` | Replica tuning affects fleet concurrency and runtime cost. |
| `storage` | Storage sizing and binding affect persistent runtime state. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `agent_runtime` | Edit agent_runtime.image or agent_runtime.max_budget_usd in {{base_config_path}}. |
| `component_ports_base` | Edit components.controlplane.grpc_port or components.gateway.grpc_port in {{base_config_path}}. |
| `component_ports_overlay` | Edit component ports in {{overlay_config_path}} for environment-specific values; use {{base_config_path}} for defaults. |
| `credentials` | Edit credentials section in {{base_config_path}} and coordinate with platform secret management. |
| `fleet_config` | Edit fleet.agent_model or fleet.max_concurrent_tasks in {{base_config_path}}. |
| `rbac` | Edit service identity in {{base_config_path}} and coordinate with platform security owners. |
| `replicas_base` | Edit components.controlplane.replicas or components.gateway.replicas in {{base_config_path}}. |
| `replicas_overlay` | Edit component replica counts in {{overlay_config_path}} for environment-specific values; use {{base_config_path}} for defaults. |
| `storage` | Edit storage.task_pvc or storage.task_pvc_size in {{base_config_path}}. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `ConfigMap` | `{{name}}-fleet-config` | `platform-runtime` | `apps` | `fleet.agent_model` |
| `Secret` | `{{name}}-fleet-credentials` | `platform-runtime` | `apps` | `credentials.anthropic_key_ref` |
| `Deployment` | `{{name}}-controlplane` | `platform-runtime` | `apps` | `components.controlplane.replicas` |
| `Deployment` | `{{name}}-gateway` | `platform-runtime` | `apps` | `components.gateway.replicas` |
| `Service` | `{{name}}-controlplane` | `platform-runtime` | `apps` | `components.controlplane.grpc_port` |
| `Service` | `{{name}}-gateway` | `platform-runtime` | `apps` | `components.gateway.grpc_port` |
| `ServiceAccount` | `{{name}}-agent` | `platform-runtime` | `apps` | `service` |
| `ClusterRole` | `{{name}}-job-runner` | `platform-runtime` | `-` | `service` |
| `ClusterRoleBinding` | `{{name}}-job-runner` | `platform-runtime` | `-` | `service` |
| `PersistentVolumeClaim` | `{{name}}-taskdata` | `platform-runtime` | `apps` | `storage.task_pvc_size` |
| `ConfigMap` | `{{name}}-job-template` | `platform-runtime` | `apps` | `agent_runtime.image` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ConfigMap` | `{{name}}-fleet-config` | `apps` | `base_config_path` | `-` | `false` | `fleet.agent_model` | `false` |
| `Secret` | `{{name}}-fleet-credentials` | `apps` | `base_config_path` | `-` | `false` | `credentials.anthropic_key_ref` | `false` |
| `Deployment` | `{{name}}-controlplane` | `apps` | `base_config_path` | `-` | `false` | `components.controlplane.replicas` | `false` |
| `Deployment` | `{{name}}-gateway` | `apps` | `base_config_path` | `-` | `false` | `components.gateway.replicas` | `false` |
| `Service` | `{{name}}-controlplane` | `apps` | `base_config_path` | `-` | `false` | `components.controlplane.grpc_port` | `false` |
| `Service` | `{{name}}-gateway` | `apps` | `base_config_path` | `-` | `false` | `components.gateway.grpc_port` | `false` |
| `ServiceAccount` | `{{name}}-agent` | `apps` | `base_config_path` | `-` | `false` | `service` | `false` |
| `ClusterRole` | `{{name}}-job-runner` | `-` | `base_config_path` | `-` | `false` | `service` | `false` |
| `ClusterRoleBinding` | `{{name}}-job-runner` | `-` | `base_config_path` | `-` | `false` | `service` | `false` |
| `PersistentVolumeClaim` | `{{name}}-taskdata` | `apps` | `base_config_path` | `-` | `false` | `storage.task_pvc_size` | `false` |
| `ConfigMap` | `{{name}}-job-template` | `apps` | `base_config_path` | `-` | `false` | `agent_runtime.image` | `false` |

## `helm`

- Profile: `helm-paas`
- Resource: `HelmRelease` (`helm.toolkit.fluxcd.io/v2/HelmRelease`)
- Capabilities: render-manifests, values-overrides, inverse-values-patch
- Default input role: `helm-input`
- Default owner: `platform-engineer`
- Field-origin transform: `helm-template`
- Field-origin overlay transform: `helm-values-overlay`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `chart` | chart.yaml | - | - | - |
| `application-set` | applicationset.yaml, applicationset.yml | - | - | - |
| `cluster-inventory` | - | platform/clusters/ | - | .yaml, .yml, .json |
| `managed-service-catalog` | - | platform/catalogs/managed-service-catalog/ | - | .yaml, .yml, .json |
| `customer-service-catalog` | - | platform/catalogs/customer-service-catalog/ | - | .yaml, .yml, .json |
| `values` | - | - | values | .yaml, .yml |

### Role Owners
| Role | Owner |
| --- | --- |
| `customer-service-catalog` | `app-team` |
| `values` | `app-team` |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `image_tag` | `app-team` | 0.86 | `false` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `image_tag` | `app-team` | 0.86 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `image_tag_base` | 0.86 |
| `image_tag_overlay` | 0.90 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `chart_path` | `Chart.yaml` |
| `chart_role` | `chart` |
| `primary_values_path` | `values.yaml` |
| `values_role` | `values` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `image_tag` | Container image tag maps cleanly to helm values. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `image_tag_base` | Edit values.image.tag in {{base_values_path}}. |
| `image_tag_overlay` | Edit values.image.tag in {{overlay_values_path}} for environment-specific overrides; use {{base_values_path}} for defaults. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `HelmRelease` | `{{name}}` | `platform-runtime` | `apps` | `-` |
| `Deployment` | `{{name}}` | `platform-runtime` | `apps` | `values.image.tag` |
| `Service` | `{{name}}` | `platform-runtime` | `apps` | `values.service.port` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `HelmRelease` | `{{name}}` | `apps` | `chart_path` | `-` | `false` | `Chart.yaml` | `false` |
| `Deployment` | `{{name}}` | `apps` | `values_paths` | `chart_path` | `true` | `values.image.tag` | `false` |
| `Service` | `{{name}}` | `apps` | `values_paths` | `chart_path` | `true` | `values.service.port` | `false` |

## `no-config-platform`

- Profile: `no-config-platform`
- Resource: `ConfigMap` (`v1/ConfigMap`)
- Capabilities: app-config-only, provider-config, inverse-provider-config-patch
- Default input role: `provider-config`
- Default owner: `app-team`
- Field-origin transform: `no-config-platform-to-runtime`
- Field-origin overlay transform: `no-config-platform-overlay-merge`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `provider-config-base` | no-config-platform.yaml, no-config-platform.yml, no-config-platform.json | - | - | - |
| `provider-config-overlay` | - | - | no-config-platform- | .yaml, .yml, .json |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `channels` | `app-team` | 0.88 | `false` |
| `environment` | `app-team` | 0.90 | `false` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `channels` | `app-team` | 0.88 |
| `environment` | `app-team` | 0.90 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `channels_base` | 0.88 |
| `channels_overlay` | 0.84 |
| `environment` | 0.90 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `base_config_path` | `no-config-platform.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `channels` | Channel mapping is app-level runtime behavior. |
| `environment` | Environment is sourced from {{base_config_path}}. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `channels_base` | Edit channels.inbound in {{base_config_path}}. |
| `channels_overlay` | Edit channels.inbound in {{overlay_config_path}} for environment-specific behavior; use {{base_config_path}} for defaults. |
| `environment` | Edit app.environment in {{base_config_path}}. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `ConfigMap` | `{{name}}-provider-config` | `platform-runtime` | `apps` | `app.environment` |
| `Secret` | `{{name}}-provider-credentials` | `platform-runtime` | `apps` | `credentials.api_key_ref` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ConfigMap` | `{{name}}-provider-config` | `apps` | `base_config_path` | `-` | `false` | `app.environment` | `false` |
| `Secret` | `{{name}}-provider-credentials` | `apps` | `base_config_path` | `-` | `false` | `credentials.api_key_ref` | `false` |
| `ConfigMap` | `{{name}}-provider-config` | `apps` | `overlay_config_path` | `-` | `false` | `channels.inbound` | `true` |

## `openchoreo`

- Profile: `openchoreo`
- Resource: `Workload` (`openchoreo.dev/v1alpha1/Workload`)
- Capabilities: platform-crds, rendered-release, generated-resource-ownership, inverse-route-patch, adoption-report
- Default input role: `openchoreo-input`
- Default owner: `platform-engineer`
- Field-origin transform: `openchoreo-release-render`
- Field-origin overlay transform: `openchoreo-environment-binding`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `component-type` | - | - | component-type- | .yaml, .yml |
| `component-type` | component-type.yaml, component-type.yml | - | - | - |
| `component` | - | - | component- | .yaml, .yml |
| `component` | component.yaml, component.yml | - | - | - |
| `workload` | - | - | workload- | .yaml, .yml |
| `workload` | workload.yaml, workload.yml | - | - | - |
| `release-binding` | - | - | release-binding- | .yaml, .yml |
| `release-binding` | release-binding.yaml, release-binding.yml | - | - | - |
| `secret-reference` | - | - | secret-reference- | .yaml, .yml |
| `secret-reference` | secret-reference.yaml, secret-reference.yml | - | - | - |
| `rendered-release` | - | - | rendered-release- | .yaml, .yml |
| `rendered-release` | rendered-release.yaml, rendered-release.yml | - | - | - |
| `rendered-manifest` | - | rendered/ | - | .yaml, .yml |

### Role Owners
| Role | Owner |
| --- | --- |
| `component` | `app-team` |
| `component-type` | `platform-engineer` |
| `release-binding` | `environment-owner` |
| `rendered-manifest` | `platform-runtime` |
| `rendered-release` | `platform-runtime` |
| `secret-reference` | `security-team` |
| `workload` | `app-team` |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `env_var` | `environment-owner` | 0.84 | `false` |
| `image` | `app-team` | 0.88 | `false` |
| `mounted_file` | `app-team` | 0.83 | `false` |
| `platform_default` | `platform-engineer` | 0.78 | `true` |
| `resource_limit` | `platform-engineer` | 0.80 | `true` |
| `secret_ref` | `security-team` | 0.86 | `true` |
| `service_port` | `platform-engineer` | 0.82 | `true` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `env_var` | `environment-owner` | 0.84 |
| `image` | `app-team` | 0.88 |
| `mounted_file` | `app-team` | 0.83 |
| `platform_default` | `platform-engineer` | 0.78 |
| `resource_limit` | `platform-engineer` | 0.80 |
| `secret_ref` | `security-team` | 0.86 |
| `service_port` | `platform-engineer` | 0.82 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `env_var` | 0.84 |
| `image` | 0.88 |
| `mounted_file` | 0.83 |
| `platform_default` | 0.78 |
| `resource_limit` | 0.80 |
| `secret_ref` | 0.86 |
| `service_port` | 0.82 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `component_type_path` | `component-type.yaml` |
| `release_binding_path` | `release-binding.yaml` |
| `rendered_manifest_path` | `rendered/deployment.yaml` |
| `rendered_release_path` | `rendered-release.yaml` |
| `secret_reference_path` | `secret-reference.yaml` |
| `workload_path` | `workload.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `env_var` | Environment values flow through the environment/release binding. |
| `image` | Container image is app-owned Workload intent. |
| `mounted_file` | Mounted files are app-owned Workload intent unless platform policy says otherwise. |
| `platform_default` | Platform defaults are owned by the ComponentType or platform policy, not generated Deployment YAML. |
| `resource_limit` | Resource limits are environment/platform-owned policy defaults. |
| `secret_ref` | Secret references are security-owned bindings and should not be edited on generated resources. |
| `service_port` | Service port is constrained by the ComponentType platform contract. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `env_var` | Route: apply-here. Edit environment binding data in {{release_binding_path}}. |
| `image` | Route: lift-upstream. Edit spec.containers.main.image in {{workload_path}}. |
| `mounted_file` | Route: lift-upstream. Edit mounted file data in {{workload_path}}. |
| `platform_default` | Route: block/escalate. Edit the platform default in {{component_type_path}} or platform policy, not the generated Deployment. |
| `resource_limit` | Route: overlay. Keep this as an environment/platform overlay in {{release_binding_path}} or policy. |
| `secret_ref` | Route: block/escalate. Edit {{secret_reference_path}} through the security-owned secret flow. |
| `service_port` | Route: lift-upstream. Edit the service port contract in {{component_type_path}}. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `RenderedRelease` | `{{name}}-prod` | `platform-runtime` | `apps` | `spec` |
| `Deployment` | `{{name}}` | `platform-runtime` | `apps` | `spec.containers.main.image` |
| `Service` | `{{name}}` | `platform-runtime` | `apps` | `spec.service.port` |
| `ConfigMap` | `{{name}}-files` | `platform-runtime` | `apps` | `spec.containers.main.files.LOG_FORMAT.value` |
| `Secret` | `{{name}}-secret-ref` | `security-team` | `apps` | `spec.secretRef` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `RenderedRelease` | `{{name}}-prod` | `apps` | `rendered_release_path` | `-` | `false` | `spec` | `false` |
| `Deployment` | `{{name}}` | `apps` | `workload_path` | `-` | `false` | `spec.containers.main.image` | `false` |
| `Deployment` | `{{name}}` | `apps` | `release_binding_path` | `-` | `false` | `spec.environment.env.LOG_LEVEL` | `false` |
| `Deployment` | `{{name}}` | `apps` | `secret_reference_path` | `-` | `false` | `spec.secretRef` | `false` |
| `Service` | `{{name}}` | `apps` | `component_type_path` | `-` | `false` | `spec.service.port` | `false` |
| `ConfigMap` | `{{name}}-files` | `apps` | `workload_path` | `-` | `false` | `spec.containers.main.files.LOG_FORMAT.value` | `false` |

## `opsworkflow`

- Profile: `ops-workflow`
- Resource: `Workflow` (`argoproj.io/v1alpha1/Workflow`)
- Capabilities: workflow-plan, governed-execution-intent, inverse-workflow-patch
- Default input role: `operations-input`
- Default owner: `platform-engineer`
- Field-origin transform: `ops-workflow-to-argo-workflow`
- Field-origin overlay transform: `ops-workflow-overlay-merge`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `operations-base` | operations.yaml, operations.yml, workflow.yaml, workflow.yml | - | - | - |
| `operations-overlay` | - | - | operations-, workflow- | .yaml, .yml |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `image_tag` | `platform-engineer` | 0.87 | `true` |
| `schedule` | `platform-engineer` | 0.84 | `true` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `image_tag` | `platform-engineer` | 0.87 |
| `schedule` | `platform-engineer` | 0.84 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `image_tag` | 0.87 |
| `schedule_base` | 0.84 |
| `schedule_overlay` | 0.80 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `base_spec_path` | `operations.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `image_tag` | Deployment action image tag is sourced from {{base_spec_path}}. |
| `schedule` | Schedule changes affect operational execution timing. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `image_tag` | Edit actions.deploy.image_tag in {{base_spec_path}}. |
| `schedule_base` | Edit triggers.schedule in {{base_spec_path}}. |
| `schedule_overlay` | Edit triggers.schedule in {{overlay_spec_path}} for environment-specific cadence; use {{base_spec_path}} for defaults. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `platform-runtime` | `ops` | `actions.deploy.image_tag` |
| `Job` | `{{name}}-dry-run` | `platform-runtime` | `ops` | `triggers.schedule` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `ops` | `base_spec_path` | `-` | `false` | `actions.deploy.image_tag` | `false` |
| `Job` | `{{name}}-dry-run` | `ops` | `base_spec_path` | `-` | `false` | `triggers.schedule` | `false` |
| `Workflow` | `{{name}}-workflow` | `ops` | `overlay_spec_path` | `-` | `false` | `triggers.schedule` | `true` |

## `score`

- Profile: `scoredev-paas`
- Resource: `Application` (`argoproj.io/v1alpha1/Application`)
- Capabilities: render-manifests, workload-spec, inverse-score-patch
- Default input role: `score-input`
- Default owner: `app-team`
- Field-origin transform: `score-to-k8s`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `score-spec` | score.yaml, score.yml | - | - | - |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `env_var` | `app-team` | 0.90 | `false` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `env_var` | `app-team` | 0.90 |
| `image` | `app-team` | 0.94 |
| `port` | `app-team` | 0.91 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `env_var` | 0.90 |
| `image` | 0.94 |
| `port` | 0.91 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `container_name` | `main` |
| `service_port_name` | `web` |
| `source_path` | `score.yaml` |
| `variable_name` | `LOG_LEVEL` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `env_var` | Score variable maps to a single Kubernetes env var. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `env_var` | Edit {{variable_name}} under containers.{{container_name}}.variables in {{source_path}}. |
| `image` | Edit the Score container image in {{source_path}}. |
| `port` | Edit {{service_port_name}} service port in {{source_path}}. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Application` | `{{name}}` | `platform-runtime` | `apps` | `-` |
| `Deployment` | `{{name}}` | `platform-runtime` | `apps` | `containers.{{container}}.image` |
| `Service` | `{{name}}` | `platform-runtime` | `apps` | `service.ports.{{service_port}}.port` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Application` | `{{name}}` | `apps` | `source_path` | `-` | `false` | `metadata.name` | `false` |
| `Deployment` | `{{name}}` | `apps` | `source_path` | `-` | `false` | `containers.{{container_name}}.image` | `false` |
| `Service` | `{{name}}` | `apps` | `source_path` | `-` | `false` | `service.ports.{{service_port_name}}.port` | `false` |

## `springboot`

- Profile: `springboot-paas`
- Resource: `Kustomization` (`kustomize.toolkit.fluxcd.io/v1/Kustomization`)
- Capabilities: render-app-config, profile-overrides, inverse-app-config-patch
- Default input role: `spring-input`
- Default owner: `platform-engineer`
- Field-origin transform: `spring-config-to-manifest`
- Field-origin overlay transform: `spring-profile-overlay`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `build-config` | pom.xml, build.gradle, build.gradle.kts | - | - | - |
| `app-config-base` | application.yaml, application.yml | - | - | - |
| `app-config-profile` | - | - | application- | .yaml, .yml |

### Role Owners
| Role | Owner |
| --- | --- |
| `app-config-base` | `app-team` |
| `app-config-profile` | `app-team` |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `app_name` | `app-team` | 0.88 | `false` |
| `datasource_url` | `platform-engineer` | 0.78 | `true` |
| `server_port` | `app-team` | 0.91 | `false` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `app_name` | `app-team` | 0.89 |
| `datasource_url` | `platform-engineer` | 0.78 |
| `server_port` | `app-team` | 0.91 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `app_name` | 0.89 |
| `datasource_url` | 0.78 |
| `server_port_base` | 0.92 |
| `server_port_overlay` | 0.88 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `base_config_path` | `src/main/resources/application.yaml` |
| `build_config_path` | `pom.xml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `app_name` | Application identity should be app-editable without platform escalation. |
| `datasource_url` | Database connectivity impacts shared runtime dependencies. |
| `server_port` | Application listener port is an app-level configuration concern. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `app_name` | Edit spring.application.name in {{base_config_path}}. |
| `datasource_url` | Edit spring.datasource.url in {{base_config_path}} and coordinate with platform ownership rules. |
| `server_port_base` | Edit server.port in {{base_config_path}}. |
| `server_port_overlay` | Edit server.port in {{profile_config_path}} for environment overrides; use {{base_config_path}} for the default. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Kustomization` | `{{name}}` | `platform-runtime` | `apps` | `-` |
| `Deployment` | `{{name}}` | `platform-runtime` | `apps` | `server.port` |
| `ConfigMap` | `{{name}}-config` | `platform-runtime` | `apps` | `spring.datasource.url` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Kustomization` | `{{name}}` | `apps` | `build_config_path` | `-` | `false` | `build` | `false` |
| `Deployment` | `{{name}}` | `apps` | `base_config_path` | `-` | `false` | `spring.application.name` | `false` |
| `ConfigMap` | `{{name}}-config` | `apps` | `base_config_path` | `-` | `false` | `spring.datasource.url` | `false` |
| `Deployment` | `{{name}}` | `apps` | `profile_config_path` | `base_config_path` | `false` | `server.port` | `false` |

## `swamp`

- Profile: `swamp`
- Resource: `Workflow` (`swamp.dev/v1/Workflow`)
- Capabilities: workflow-automation, model-orchestration, inverse-workflow-patch
- Default input role: `swamp-input`
- Default owner: `app-team`
- Field-origin transform: `swamp-workflow-to-execution`
- Field-origin overlay transform: `swamp-overlay-merge`

### Input Role Rules
| Role | Exact basenames | Path prefixes | Prefixes | Extensions |
| --- | --- | --- | --- | --- |
| `swamp-config-base` | .swamp.yaml, .swamp.yml | - | - | - |
| `swamp-workflow` | - | - | workflow- | .yaml, .yml |

### Inverse Patch Templates
| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `model_binding` | `app-team` | 0.88 | `false` |
| `vault_config` | `platform-engineer` | 0.85 | `true` |
| `workflow_definition` | `app-team` | 0.90 | `false` |

### Inverse Pointer Templates
| Key | Owner | Confidence |
| --- | --- | --- |
| `model_binding` | `app-team` | 0.88 |
| `vault_config` | `platform-engineer` | 0.85 |
| `workflow_definition` | `app-team` | 0.90 |

### Field Origin Confidences
| Key | Confidence |
| --- | --- |
| `model_binding_base` | 0.88 |
| `model_binding_workflow` | 0.84 |
| `vault_config` | 0.85 |
| `workflow_definition` | 0.90 |

### Hint Defaults
| Key | Value |
| --- | --- |
| `base_config_path` | `.swamp.yaml` |

### Inverse Patch Reasons
| Key | Reason |
| --- | --- |
| `model_binding` | Model bindings define which automation models execute in workflows. |
| `vault_config` | Vault configuration impacts platform credential infrastructure. |
| `workflow_definition` | Workflow job and step definitions are app-level automation intent. |

### Inverse Edit Hints
| Key | Hint |
| --- | --- |
| `model_binding_base` | Edit model references in {{base_config_path}}. |
| `model_binding_workflow` | Edit model method bindings in {{workflow_path}} for task-specific overrides. |
| `vault_config` | Edit vaults section in {{base_config_path}} and coordinate with platform secret infrastructure. |
| `workflow_definition` | Edit jobs and steps in the workflow YAML file. |

### WET Targets
| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `platform-runtime` | `apps` | `jobs[].steps[].task` |
| `ConfigMap` | `{{name}}-swamp-config` | `platform-runtime` | `apps` | `swamp.version` |

### Rendered Lineage Templates
| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `apps` | `base_config_path` | `-` | `false` | `vaults.default.type` | `false` |
| `ConfigMap` | `{{name}}-swamp-config` | `apps` | `base_config_path` | `-` | `false` | `swamp.version` | `false` |
| `Workflow` | `{{name}}-workflow` | `apps` | `workflow_path` | `-` | `false` | `jobs[].steps[].task` | `true` |
