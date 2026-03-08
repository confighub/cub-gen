# swamp Triple

- Profile: `swamp`
- Resource: `Workflow` (`swamp.dev/v1/Workflow`)
- Capabilities: workflow-automation, model-orchestration, inverse-workflow-patch

```mermaid
flowchart LR
  dry["DRY Inputs"] --> gen["Generator"] --> wet["WET Targets"]
```

## Contract

- Default input role: `swamp-input`
- Default owner: `app-team`

### Input role rules

| Role | Exact basenames | Prefixes | Extensions |
| --- | --- | --- | --- |
| `swamp-config-base` | .swamp.yaml, .swamp.yml | - | - |
| `swamp-workflow` | - | workflow- | .yaml, .yml |

### Role owners

| Role | Owner |
| --- | --- |

### Role schema refs

| Role | Schema ref |
| --- | --- |
| `swamp-config-base` | `https://schema.confighub.dev/generators/swamp-v1` |
| `swamp-workflow` | `https://schema.confighub.dev/generators/swamp-workflow-v1` |

### WET targets

| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `platform-runtime` | `apps` | `jobs[].steps[].task` |
| `ConfigMap` | `{{name}}-swamp-config` | `platform-runtime` | `apps` | `swamp.version` |

## Provenance

- Field-origin transform: `swamp-workflow-to-execution`
- Field-origin overlay transform: `swamp-overlay-merge`

### Field-origin confidences

| Key | Confidence |
| --- | --- |
| `model_binding_base` | 0.88 |
| `model_binding_workflow` | 0.84 |
| `vault_config` | 0.85 |
| `workflow_definition` | 0.90 |

### Rendered lineage templates

| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `apps` | `base_config_path` | `` | `false` | `vaults.default.type` | `false` |
| `ConfigMap` | `{{name}}-swamp-config` | `apps` | `base_config_path` | `` | `false` | `swamp.version` | `false` |
| `Workflow` | `{{name}}-workflow` | `apps` | `workflow_path` | `` | `false` | `jobs[].steps[].task` | `true` |

## Inverse

### Inverse patch templates

| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `model_binding` | `app-team` | 0.88 | `false` |
| `vault_config` | `platform-engineer` | 0.85 | `true` |
| `workflow_definition` | `app-team` | 0.90 | `false` |

### Inverse pointer templates

| Key | Owner | Confidence |
| --- | --- | --- |
| `model_binding` | `app-team` | 0.88 |
| `vault_config` | `platform-engineer` | 0.85 |
| `workflow_definition` | `app-team` | 0.90 |

### Inverse patch reasons

| Key | Reason |
| --- | --- |
| `model_binding` | Model bindings define which automation models execute in workflows. |
| `vault_config` | Vault configuration impacts platform credential infrastructure. |
| `workflow_definition` | Workflow job and step definitions are app-level automation intent. |

### Inverse edit hints

| Key | Hint |
| --- | --- |
| `model_binding_base` | Edit model references in {{base_config_path}}. |
| `model_binding_workflow` | Edit model method bindings in {{workflow_path}} for task-specific overrides. |
| `vault_config` | Edit vaults section in {{base_config_path}} and coordinate with platform secret infrastructure. |
| `workflow_definition` | Edit jobs and steps in the workflow YAML file. |

### Hint defaults

| Key | Value |
| --- | --- |
| `base_config_path` | `.swamp.yaml` |
