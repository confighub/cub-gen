# Backstage IDP — Governed Software Catalog

Your Backstage software catalog is the single source of truth for service
ownership, lifecycle, and discoverability. Every team registers their services
in `catalog-info.yaml`. The platform team enforces standards: valid owners,
approved lifecycle stages, consistent naming.

The problem: catalog changes happen via Git PRs, but there's no structured
governance over *what* changed. When 50 teams rename owners during a reorg,
you need traceability — not just commit history. ConfigHub makes every catalog
change traceable, auditable, and queryable across repos.

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding catalog governance | Jump to [Run from ConfigHub](#run-from-confighub-connected-mode) |
| **Existing Backstage user** adding ConfigHub | Jump to [Try it](#try-it) then connect later |

Both paths lead to the same outcome: governed catalog entities with field-origin tracing.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real app** | Backstage `catalog-info.yaml` Component entity |
| **Real inspection target** | Backstage Developer Portal catalog page |
| **Validation** | Platform catalog standards (owner, lifecycle, naming) |
| **Sync transport** | Backstage native Git sync (no Flux/Argo needed) |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "Who changed the service owner?" | Field-origin tracing to `catalog-info.yaml` | Owner transfers → governed audit |
| "Are all services on valid lifecycle?" | Platform standards enforcement | Invalid lifecycle → BLOCK |
| "Which services still use old team name?" | Cross-repo catalog queries | Reorg cleanup → queryable |

## Domain POV (Backstage/catalog admins)

Use this example if your catalog is operationally critical:

- `catalog-info.yaml` drives ownership, lifecycle, and service discoverability,
- high-volume metadata edits happen during reorgs or compliance updates,
- you need policy checks and provenance without changing Backstage workflows.

The first value is catalog trust: owner/lifecycle edits become explicit,
reviewable, and queryable across repos.

## What you get

- **Catalog entity governance**: ownership, lifecycle, and type changes are
  traced with full provenance
- **Platform standards enforcement**: required fields, valid lifecycle stages,
  naming conventions — enforced at publish time
- **Cross-repo catalog queries**: "which services are still owned by the old
  team?" — answerable in one query
- **Change audit trail**: who changed the owner, who approved it, and why

## How Backstage maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              BACKSTAGE (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ catalog-info.yaml   │          │ Component entity     │         │ Service catalog  │
│ app-config.yaml     │──import─▶│ Catalog metadata     │──sync──▶│ Portal pages     │
│ platform/catalog-   │          │ Ownership records    │         │ API docs         │
│   standards.yaml    │          │                      │         │                 │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  App team: catalog-info.          Structured entity data            What's visible
  Platform: standards.             with field provenance.            in Backstage.
```

**DRY** is what teams edit: `catalog-info.yaml` declares the service identity
(name, owner, type, lifecycle). `app-config.yaml` configures the portal
infrastructure. Platform standards define valid values.

**WET** is what cub-gen produces: structured catalog entity metadata with every
field traced back to its source. The platform's catalog standards validate
required fields and naming conventions.

**LIVE** is what's visible in Backstage. Backstage syncs from Git natively —
no Flux or ArgoCD needed for catalog entities.

| File | Owner | What it controls |
|------|-------|-----------------|
| `catalog-info.yaml` | App team | Service identity — name, owner, type, lifecycle |
| `app-config.yaml` | Platform | Portal config — base URLs, backend endpoints |
| `platform/catalog-standards.yaml` | Platform | Required fields, valid lifecycles, naming conventions |

## If you already run Backstage catalogs at scale

This example is for platform teams and service owners who already depend on
catalog metadata quality:

- Teams treat `catalog-info.yaml` as the source of service ownership and lifecycle.
- Reorgs create high-volume owner/lifecycle edits across many repos.
- Reviewers need fast answers about whether changes violate catalog standards.

cub-gen keeps the Backstage authoring model and adds explicit provenance plus
policy-aware ownership routing for catalog mutations.

## Why this maps cleanly to the cub-gen framework

| Existing Backstage model | cub-gen concept | Why it matters |
|------|------|------|
| `catalog-info.yaml` entity spec | DRY service metadata intent | Service teams keep editing canonical Backstage files. |
| Normalized entity data | WET records with provenance | Owner/lifecycle fields become auditable by source line. |
| Catalog standards | Governance policy layer | Invalid ownership or lifecycle transitions can be blocked. |
| Backstage sync from Git | LIVE catalog state | Catalog runtime behavior remains unchanged. |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect Backstage catalog entity
./cub-gen gitops discover --space platform --json ./examples/backstage-idp

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

cub-gen detects `catalog-info.yaml` with `backstage.io/v1alpha1` apiVersion
and classifies it as `backstage-idp`. The import traces every entity field
back to its source with ownership metadata.

## Real-world scenario: ownership transfer during team reorg

**Who**: A company with 200 microservices and a Backstage-powered developer
portal. The payments team is splitting into payments-core and payments-fraud.

### Scenario A — Valid owner transfer (ALLOW)

The app team updates the service owner to a valid team name:

```yaml
# catalog-info.yaml — ownership transfer
spec:
  owner: team-payments-core   # was team-payments (valid team)
  lifecycle: production       # unchanged
```

```bash
# cub-gen detects the ownership change
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp

# Evidence chain
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Platform validates: "team-payments-core" is in the approved team list → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by engineering-director --reason "Q2 team reorg"
```

After the reorg, ConfigHub can answer: "are there any services still owned by
the old team-payments?" — a cross-repo query that would otherwise require
grepping across 200 repositories.

### Scenario B — Invalid lifecycle or unknown owner (BLOCK)

The app team tries to set an invalid lifecycle or an unknown owner:

```yaml
# catalog-info.yaml — invalid change
spec:
  owner: random-person           # NOT in approved team list
  lifecycle: testing             # NOT a valid lifecycle stage
```

```bash
# cub-gen detects the invalid change
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp

# Evidence chain
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp > bundle.json
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Platform validates: owner not in team list, lifecycle not valid → BLOCK
./cub-gen bridge decision apply --decision decision.json --state BLOCK \
  --approved-by governance-bot \
  --reason "Owner 'random-person' not in approved team list. Lifecycle 'testing' not valid (use: experimental, production, deprecated)."
```

## How it works

cub-gen's `backstage-idp` generator detects `catalog-info.yaml` containing a
`backstage.io/v1alpha1` apiVersion with `kind: Component`. On import:

1. **Classifies inputs** — `catalog-info.yaml` (role: catalog-entity),
   `app-config.yaml` (role: portal-config)
2. **Maps field origins** — `spec.owner`, `spec.lifecycle`, `spec.type` all
   trace to `catalog-info.yaml` with ownership metadata
3. **Validates standards** — platform catalog standards check required fields,
   valid lifecycles (experimental, production, deprecated), and naming patterns
4. **Emits inverse guidance** — "to change the service owner, edit
   `catalog-info.yaml` spec.owner"

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `catalog-info.yaml` | App team | Backstage catalog entity |
| `app-config.yaml` | Platform | Portal infrastructure config |
| `platform/catalog-standards.yaml` | Platform | Required fields, valid lifecycles, naming rules |

## Next steps

- **App-only config**: [`just-apps-no-platform-config`](../just-apps-no-platform-config/) —
  simplest possible example with no platform layer
- **Helm + Backstage**: [`helm-paas`](../helm-paas/) — catalog entries often
  live alongside Helm charts in the same repo
- **Full platform story**: see the [platform architecture](../../docs/platform.md)

## Run from ConfigHub (connected mode)

If you already have ConfigHub, start here:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"

# Publish and ingest
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map (catalog fields → source)
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp \
  | jq '.provenance[0].field_origin_map'

# Catalog entity analysis
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp \
  | jq '.provenance[0].backstage_entity_analysis'

# Evidence bundle
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

## 7. Try one governed change

**ALLOW path**: App team transfers ownership to valid team:

```yaml
# catalog-info.yaml change
spec:
  owner: team-payments-core  # valid team in approved list
```

Result: Owner is in approved team list → **ALLOW**

**BLOCK path**: App team sets unknown owner or invalid lifecycle:

```yaml
# catalog-info.yaml change
spec:
  owner: john-doe        # NOT a team, just a person
  lifecycle: alpha       # NOT valid (use experimental/production/deprecated)
```

Result: Owner not in approved list, lifecycle not valid → **BLOCK**

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/backstage-idp/demo-local.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/backstage-idp/demo-connected.sh
```
