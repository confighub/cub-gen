# Backstage IDP (Developer Portal)

**Pattern: developer portal catalog registration — Backstage catalog entities as governed configuration with platform-enforced ownership and lifecycle standards.**

## 1. What is this?

A payments team registers their service in the company's Backstage developer portal. The catalog entry (`catalog-info.yaml`) declares the service identity, ownership, and lifecycle stage. The app config (`app-config.yaml`) sets up the portal URLs. The platform team enforces standards: every component must have an owner, a lifecycle stage, and conform to the company's catalog schema.

This is configuration governance for developer experience: the portal catalog is the single source of truth for service ownership, and cub-gen ensures every change to that catalog is traceable, governed, and auditable.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **App team** | `catalog-info.yaml` — component identity, type, owner | Service name, description, lifecycle stage |
| **Platform team** | `app-config.yaml` — portal infrastructure config | Base URLs, backend endpoints |
| **Platform team** | `platform/` — catalog standards (future) | Required fields, naming conventions, ownership rules |
| **GitOps reconciler** | N/A — Backstage syncs from Git natively | N/A |

## 3. What does cub-gen add?

cub-gen treats Backstage catalog entities as a generator source:

- **Generator detection**: recognizes `catalog-info.yaml` with `backstage.io/v1alpha1` apiVersion (capabilities: `catalog-metadata`, `render-manifests`, `inverse-catalog-patch`)
- **DRY/WET classification**: catalog entity is app DRY (service identity), app config is platform DRY (portal infra)
- **Field-origin tracing**: owner, lifecycle, and component type all trace back to the DRY catalog file
- **Inverse-edit guidance**: "to change the service owner, edit `catalog-info.yaml` spec.owner"

```bash
# Discover
./cub-gen gitops discover --space platform --json ./examples/backstage-idp

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

## 4. How do I run it?

```bash
# Build
go build -o ./cub-gen ./cmd/cub-gen

# Discover
./cub-gen gitops discover --space platform ./examples/backstage-idp

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp

# Full bridge flow
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp > /tmp/backstage-bundle.json
./cub-gen verify --in /tmp/backstage-bundle.json
./cub-gen attest --in /tmp/backstage-bundle.json --verifier ci-bot > /tmp/backstage-attestation.json
./cub-gen verify-attestation --in /tmp/backstage-attestation.json --bundle /tmp/backstage-bundle.json

# Cleanup
./cub-gen gitops cleanup --space platform ./examples/backstage-idp
```

## 5. Real-world example using ConfigHub

A company with 200 microservices uses Backstage for service discovery and ownership tracking. Every service has a `catalog-info.yaml` in its repo.

**Scenario: Ownership transfer during team reorganization**

The payments team is splitting into payments-core and payments-fraud. Services need to update their catalog ownership:

```yaml
# Before
spec:
  owner: team-payments

# After
spec:
  owner: team-payments-core
```

**Governed pipeline:**

```bash
# 1. cub-gen detects the ownership change
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp
# Field-origin: spec.owner changed in catalog-info.yaml (app-team owned)

# 2. Produce evidence chain
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 3. ConfigHub ingests and evaluates
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example
# Decision engine checks: is "team-payments-core" a valid team? → ALLOW
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by engineering-director --reason "Q2 team reorg"
```

**What ConfigHub provides:**
- **Ownership audit**: "which services changed ownership in Q2?" — answerable from decision history
- **Completeness check**: "are there any services still owned by the old team-payments?" — cross-repo query
- **Standard enforcement**: platform policies can require valid team names, lifecycle stages, and component types
- **Change provenance**: every catalog change traces back to who made it, who approved it, and why

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `catalog-info.yaml` | App team | Backstage catalog entity — service identity, owner, lifecycle |
| `app-config.yaml` | Platform team | Backstage portal config — base URLs, backend endpoints |

## Related examples

- [`just-apps-no-platform-config`](../just-apps-no-platform-config/) — Another "just config, no manifests" pattern. Shows that governance works for external service configuration.
- [`helm-paas`](../helm-paas/) — The Kubernetes workload end of the spectrum. The Backstage catalog entry often lives alongside a Helm chart in the same repo.
