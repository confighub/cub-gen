# opsworkflow Triple

- Profile: `ops-workflow`
- Resource: `Workflow` (`argoproj.io/v1alpha1/Workflow`)
- Capabilities: workflow-plan, governed-execution-intent, inverse-workflow-patch

```mermaid
flowchart LR
  subgraph DRY["DRY Inputs"]
    d1["operations-base: operations.yaml, operations.yml, workflow.yaml, workflow.yml<br/>owner: platform-engineer"]
    d2["operations-overlay: operations-*.yaml | operations-*.yml | workflow-*.yaml | workflow-*.yml<br/>owner: platform-engineer"]
  end
  gen["opsworkflow (ops-workflow)<br/>capabilities: workflow-plan, governed-execution-intent, inverse-workflow-patch"]
  subgraph WET["WET Targets"]
    w1["Workflow {{name}}-workflow<br/>owner: platform-runtime<br/>namespace: ops<br/>source: actions.deploy.image_tag"]
    w2["Job {{name}}-dry-run<br/>owner: platform-runtime<br/>namespace: ops<br/>source: triggers.schedule"]
  end
  d1 --> gen
  d2 --> gen
  gen --> w1
  gen --> w2
```

## Contract

- Default input role: `operations-input`
- Default owner: `platform-engineer`

### Input role rules

| Role | Exact basenames | Prefixes | Extensions |
| --- | --- | --- | --- |
| `operations-base` | operations.yaml, operations.yml, workflow.yaml, workflow.yml | - | - |
| `operations-overlay` | - | operations-, workflow- | .yaml, .yml |

### Role owners

| Role | Owner |
| --- | --- |

### Role schema refs

| Role | Schema ref |
| --- | --- |
| `operations-base` | `https://schema.confighub.dev/generators/ops-workflow-v1` |
| `operations-overlay` | `https://schema.confighub.dev/generators/ops-workflow-v1` |

### WET targets

| Kind | Name template | Owner | Namespace | Source DRY path template |
| --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `platform-runtime` | `ops` | `actions.deploy.image_tag` |
| `Job` | `{{name}}-dry-run` | `platform-runtime` | `ops` | `triggers.schedule` |

## Provenance

- Field-origin transform: `ops-workflow-to-argo-workflow`
- Field-origin overlay transform: `ops-workflow-overlay-merge`

### Field-origin confidences

| Key | Confidence |
| --- | --- |
| `image_tag` | 0.87 |
| `schedule_base` | 0.84 |
| `schedule_overlay` | 0.80 |

### Rendered lineage templates

| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Workflow` | `{{name}}-workflow` | `ops` | `base_spec_path` | `` | `false` | `actions.deploy.image_tag` | `false` |
| `Job` | `{{name}}-dry-run` | `ops` | `base_spec_path` | `` | `false` | `triggers.schedule` | `false` |
| `Workflow` | `{{name}}-workflow` | `ops` | `overlay_spec_path` | `` | `false` | `triggers.schedule` | `true` |

## Inverse

### Inverse patch templates

| Key | Editable by | Confidence | Requires review |
| --- | --- | --- | --- |
| `image_tag` | `platform-engineer` | 0.87 | `true` |
| `schedule` | `platform-engineer` | 0.84 | `true` |

### Inverse pointer templates

| Key | Owner | Confidence |
| --- | --- | --- |
| `image_tag` | `platform-engineer` | 0.87 |
| `schedule` | `platform-engineer` | 0.84 |

### Inverse patch reasons

| Key | Reason |
| --- | --- |
| `image_tag` | Deployment action image tag is sourced from {{base_spec_path}}. |
| `schedule` | Schedule changes affect operational execution timing. |

### Inverse edit hints

| Key | Hint |
| --- | --- |
| `image_tag` | Edit actions.deploy.image_tag in {{base_spec_path}}. |
| `schedule_base` | Edit triggers.schedule in {{base_spec_path}}. |
| `schedule_overlay` | Edit triggers.schedule in {{overlay_spec_path}} for environment-specific cadence; use {{base_spec_path}} for defaults. |

### Hint defaults

| Key | Value |
| --- | --- |
| `base_spec_path` | `operations.yaml` |
