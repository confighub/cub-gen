# c3agent Triple

- Profile: `c3agent`
- Resource: `ConfigMap` (`v1/ConfigMap`)
- Capabilities: fleet-config, agent-orchestration, inverse-fleet-config-patch

```mermaid
flowchart LR
  subgraph DRY["DRY Inputs"]
    d1["fleet-config-base: c3agent.yaml, c3agent.yml, c3agent.json<br/>owner: app-team"]
    d2["fleet-config-overlay: c3agent-*.yaml | c3agent-*.yml | c3agent-*.json<br/>owner: app-team"]
  end
  gen["c3agent (c3agent)<br/>capabilities: fleet-config, agent-orchestration, inverse-fleet-config-patch"]
  subgraph WET["WET Targets"]
    w1["ConfigMap {{name}}-fleet-config<br/>owner: platform-runtime<br/>namespace: apps<br/>source: fleet.agent_model"]
    w2["Secret {{name}}-fleet-credentials<br/>owner: platform-runtime<br/>namespace: apps<br/>source: credentials.anthropic_key_ref"]
    w3["Deployment {{name}}-controlplane<br/>owner: platform-runtime<br/>namespace: apps<br/>source: components.controlplane.replicas"]
    w4["Deployment {{name}}-gateway<br/>owner: platform-runtime<br/>namespace: apps<br/>source: components.gateway.replicas"]
    w5["Service {{name}}-controlplane<br/>owner: platform-runtime<br/>namespace: apps<br/>source: components.controlplane.grpc_port"]
    w6["Service {{name}}-gateway<br/>owner: platform-runtime<br/>namespace: apps<br/>source: components.gateway.grpc_port"]
    w7["ServiceAccount {{name}}-agent<br/>owner: platform-runtime<br/>namespace: apps<br/>source: service"]
    w8["ClusterRole {{name}}-job-runner<br/>owner: platform-runtime<br/>source: service"]
    w9["ClusterRoleBinding {{name}}-job-runner<br/>owner: platform-runtime<br/>source: service"]
    w10["PersistentVolumeClaim {{name}}-taskdata<br/>owner: platform-runtime<br/>namespace: apps<br/>source: storage.task_pvc_size"]
    w11["ConfigMap {{name}}-job-template<br/>owner: platform-runtime<br/>namespace: apps<br/>source: agent_runtime.image"]
  end
  d1 --> gen
  d2 --> gen
  gen --> w1
  gen --> w2
  gen --> w3
  gen --> w4
  gen --> w5
  gen --> w6
  gen --> w7
  gen --> w8
  gen --> w9
  gen --> w10
  gen --> w11
```

## Contract

- Default input role: `fleet-config`
- Default owner: `app-team`

### Input role rules

| Role | Exact basenames | Prefixes | Extensions |
| --- | --- | --- | --- |
| `fleet-config-base` | c3agent.yaml, c3agent.yml, c3agent.json | - | - |
| `fleet-config-overlay` | - | c3agent- | .yaml, .yml, .json |

### Role owners

| Role | Owner |
| --- | --- |

### Role schema refs

| Role | Schema ref |
| --- | --- |
| `fleet-config-base` | `https://schema.confighub.dev/generators/c3agent-v1` |
| `fleet-config-overlay` | `https://schema.confighub.dev/generators/c3agent-v1` |

### WET targets

| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `ConfigMap` | `{{name}}-fleet-config` | `platform-runtime` | `apps` | `fleet.agent_model` |
| `Secret` | `{{name}}-fleet-credentials` | `platform-runtime` | `apps` | `credentials.anthropic_key_ref` |
| `Deployment` | `{{name}}-controlplane` | `platform-runtime` | `apps` | `components.controlplane.replicas` |
| `Deployment` | `{{name}}-gateway` | `platform-runtime` | `apps` | `components.gateway.replicas` |
| `Service` | `{{name}}-controlplane` | `platform-runtime` | `apps` | `components.controlplane.grpc_port` |
| `Service` | `{{name}}-gateway` | `platform-runtime` | `apps` | `components.gateway.grpc_port` |
| `ServiceAccount` | `{{name}}-agent` | `platform-runtime` | `apps` | `service` |
| `ClusterRole` | `{{name}}-job-runner` | `platform-runtime` | `` | `service` |
| `ClusterRoleBinding` | `{{name}}-job-runner` | `platform-runtime` | `` | `service` |
| `PersistentVolumeClaim` | `{{name}}-taskdata` | `platform-runtime` | `apps` | `storage.task_pvc_size` |
| `ConfigMap` | `{{name}}-job-template` | `platform-runtime` | `apps` | `agent_runtime.image` |

## Provenance

- Field-origin transform: `c3agent-config-to-runtime`
- Field-origin overlay transform: `c3agent-overlay-merge`

### Field-origin confidences

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

### Rendered lineage templates

| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ConfigMap` | `{{name}}-fleet-config` | `apps` | `base_config_path` | `` | `false` | `fleet.agent_model` | `false` |
| `Secret` | `{{name}}-fleet-credentials` | `apps` | `base_config_path` | `` | `false` | `credentials.anthropic_key_ref` | `false` |
| `Deployment` | `{{name}}-controlplane` | `apps` | `base_config_path` | `` | `false` | `components.controlplane.replicas` | `false` |
| `Deployment` | `{{name}}-gateway` | `apps` | `base_config_path` | `` | `false` | `components.gateway.replicas` | `false` |
| `Service` | `{{name}}-controlplane` | `apps` | `base_config_path` | `` | `false` | `components.controlplane.grpc_port` | `false` |
| `Service` | `{{name}}-gateway` | `apps` | `base_config_path` | `` | `false` | `components.gateway.grpc_port` | `false` |
| `ServiceAccount` | `{{name}}-agent` | `apps` | `base_config_path` | `` | `false` | `service` | `false` |
| `ClusterRole` | `{{name}}-job-runner` | `` | `base_config_path` | `` | `false` | `service` | `false` |
| `ClusterRoleBinding` | `{{name}}-job-runner` | `` | `base_config_path` | `` | `false` | `service` | `false` |
| `PersistentVolumeClaim` | `{{name}}-taskdata` | `apps` | `base_config_path` | `` | `false` | `storage.task_pvc_size` | `false` |
| `ConfigMap` | `{{name}}-job-template` | `apps` | `base_config_path` | `` | `false` | `agent_runtime.image` | `false` |

## Inverse

### Inverse patch templates

| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `agent_runtime` | `platform-engineer` | 0.88 | `true` |
| `component_ports` | `platform-engineer` | 0.84 | `true` |
| `credentials` | `platform-engineer` | 0.86 | `true` |
| `fleet_config` | `app-team` | 0.91 | `false` |
| `rbac` | `platform-engineer` | 0.82 | `true` |
| `replicas` | `app-team` | 0.89 | `false` |
| `storage` | `platform-engineer` | 0.85 | `true` |

### Inverse pointer templates

| Key | Owner | Confidence |
| --- | --- | --- |
| `agent_runtime` | `platform-engineer` | 0.88 |
| `component_ports` | `platform-engineer` | 0.84 |
| `credentials` | `platform-engineer` | 0.86 |
| `fleet_config` | `app-team` | 0.91 |
| `rbac` | `platform-engineer` | 0.82 |
| `replicas` | `app-team` | 0.89 |
| `storage` | `platform-engineer` | 0.85 |

### Inverse patch reasons

| Key | Reason |
| --- | --- |
| `agent_runtime` | Agent runtime image and budget settings affect platform execution behavior. |
| `component_ports` | Component port changes affect platform networking and service mesh. |
| `credentials` | Credential references impact platform secret management. |
| `fleet_config` | Fleet configuration (model, concurrency) is sourced from {{base_config_path}}. |
| `rbac` | RBAC resources are platform-governed and must align with security policy. |
| `replicas` | Replica tuning affects fleet concurrency and runtime cost. |
| `storage` | Storage sizing and binding affect persistent runtime state. |

### Inverse edit hints

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

### Hint defaults

| Key | Value |
| --- | --- |
| `base_config_path` | `c3agent.yaml` |
