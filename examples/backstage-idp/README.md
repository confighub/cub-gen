# Backstage IDP — Governed Software Catalog

Your Backstage software catalog is the single source of truth for service
ownership, lifecycle, and discoverability. Every team registers their services
in `catalog-info.yaml`. The platform team enforces standards: valid owners,
approved lifecycle stages, consistent naming.

The problem: catalog changes happen via Git PRs, but there's no structured
governance over *what* changed. When 50 teams rename owners during a reorg,
you need traceability — not just commit history. ConfigHub makes every catalog
change traceable, auditable, and queryable across repos.

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

### The change — 15 services need new owners

```yaml
# catalog-info.yaml — before
spec:
  owner: team-payments

# catalog-info.yaml — after
spec:
  owner: team-payments-core
```

### Governed pipeline

```bash
# cub-gen detects the ownership change
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp

# Evidence chain
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Platform validates: "team-payments-core" is in the approved team list → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by engineering-director --reason "Q2 team reorg"
```

After the reorg, ConfigHub can answer: "are there any services still owned by
the old team-payments?" — a cross-repo query that would otherwise require
grepping across 200 repositories.

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

## Local and Connected Entrypoints

From repo root:

```bash
echo "local/offline"
./examples/backstage-idp/demo-local.sh

echo "connected (requires ConfigHub auth)"
cub auth login
./examples/backstage-idp/demo-connected.sh
```
