# App/Deployment Concepts

This document defines the canonical app/deployment model used in `cub-gen` examples.
Based on Jesper's promotion-demo-data model.

## Core conceptual model

The model uses an **App-Deployment-Target** relationship pattern:

| Concept | Definition | Example |
|---------|------------|---------|
| **App** | Software product belonging to a department | `eshop`, `aichat` |
| **Target** | Kubernetes cluster deployment destination | `us-prod-1`, `eu-dev-1` |
| **Deployment** | Junction: one app + one target with config | `us-prod-1-eshop` |

A deployment space contains units (api, frontend, postgres, redis, worker, etc.).

## ConfigHub primitive mapping

Since ConfigHub uses spaces/units/labels (not native App/Deployment types):

| Concept | Primitive | Example |
|---------|-----------|---------|
| App | Labels on spaces/units | `App=eshop` |
| Target | Target object in infrastructure space | `us-prod-1` cluster |
| Deployment | Space containing units | `us-prod-1-eshop` space |

## Standard labels

### Target labels

| Label | Values | Purpose |
|-------|--------|---------|
| `TargetRole` | Dev, QA, Staging, Prod | Environment tier |
| `TargetRegion` | US, EU, APAC | Geographic region |

### Deployment/unit labels

| Label | Values | Purpose |
|-------|--------|---------|
| `App` | Application identifier | Which app this belongs to |
| `AppOwner` | Marketing, Product, Platform | Department ownership |
| `TargetRole` | Inherited from target | Environment context |
| `TargetRegion` | Inherited from target | Region context |

## Query patterns

These labels enable cross-dimensional queries:

```bash
# All eshop deployments
cub query --label App=eshop

# All production deployments in EU
cub query --label TargetRole=Prod --label TargetRegion=EU

# All apps owned by Marketing team
cub query --label AppOwner=Marketing
```

## Example structure

A typical deployment space:

```
us-prod-1-eshop/                    # Deployment space
  labels:
    App: eshop
    AppOwner: Marketing
    TargetRole: Prod
    TargetRegion: US
  units:
    - api                           # Backend service
    - frontend                      # Web UI
    - postgres                      # Database
    - redis                         # Cache
    - worker                        # Background jobs
```

## Promotion flow

Promotion operates on labeled groupings:

1. **Source**: `us-dev-1-eshop` (Dev environment)
2. **Target**: `us-prod-1-eshop` (Prod environment)
3. **Filter**: By `App` label, not by repo path
4. **Gate**: Separate approval for each `TargetRole` transition

This enables:

- Promote "all eshop configs" from Dev to Staging
- Promote "all Marketing apps" to Prod (batch)
- Query which deployments are out of sync across environments

## cub-gen alignment

When `cub-gen` imports a repo, it should:

1. **Detect app identity** from labels, Chart.yaml, or score.yaml metadata
2. **Classify targets** from directory structure or kustomize overlays
3. **Emit provenance** that includes App/Target context
4. **Support queries** by standard labels in connected mode

Example import output alignment:

```json
{
  "generator_profile": "helm-paas",
  "app_context": {
    "app": "eshop",
    "app_owner": "Marketing"
  },
  "target_context": {
    "target_role": "Prod",
    "target_region": "US"
  },
  "provenance": {
    "dry_inputs": ["values.yaml", "values-prod.yaml"],
    "wet_targets": ["deployment.yaml", "service.yaml"]
  }
}
```

## Example validation

Every example should demonstrate:

| Check | How to validate |
|-------|-----------------|
| App identity is extractable | Import shows `app_context` or equivalent |
| Target context is captured | Import shows `target_context` or equivalent |
| Labels are standard | Uses TargetRole/TargetRegion/App/AppOwner where applicable |
| Query works | Connected mode query by labels returns expected results |

## Related documentation

- [Ilya/CoreWeave Checklist](ilya-coreweave-acceptance-checklist.md) — acceptance criteria
- [Universal Example Contract](../workflows/example-checklist.md) — example requirements
- [Promotion Demo Data](https://github.com/confighub/examples/tree/main/promotion-demo-data) — source model
