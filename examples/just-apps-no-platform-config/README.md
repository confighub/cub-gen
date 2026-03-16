# Just Apps, No Platform Config — Governed Provider Config

Not every service runs on Kubernetes. Not every config needs a platform layer.
Sometimes you just have a provider config file — realtime channels, feature
flags, provider routes — and you want the same governance you'd get for a Helm
chart.

This is the simplest cub-gen example: app-only configuration with no platform
contracts. It proves the governance model works for *any* configuration, not
just Kubernetes workloads.

## 1. Who this is for

| If you are... | Start here |
|---------------|------------|
| **Existing ConfigHub user** adding provider config governance | Jump to [Run from ConfigHub](#run-from-confighub-connected-mode) |
| **App team without a platform layer** | Jump to [Try it](#try-it) — simplest path |

Both paths lead to the same outcome: governed provider config with field-origin tracing.

## 2. What runs

| Component | What it is |
|-----------|------------|
| **Real app** | Provider config file (channels, credentials, settings) |
| **Real inspection target** | Rendered ConfigMap with field provenance |
| **Platform layer** | Empty — no contracts yet (that's the point) |
| **Sync transport** | Provider-native sync (not Flux/Argo) |

## 3. Why ConfigHub + cub-gen helps here

| Pain | Answer | Governed change win |
|------|--------|---------------------|
| "What changed in prod config?" | Field-origin tracing to source file | Channel adds → auditable |
| "Can we add governance later?" | Same pipeline, just add policies | Future-proof |
| "No K8s, can we still use this?" | Provider-agnostic governance | Works for any config |

## Domain POV (app teams without a formal platform)

This example is for teams that manage provider config directly today:

- no full platform contract yet,
- app teams still need safe change flow and traceability,
- platform policy may be added later without breaking existing authoring.

The first value is immediate visibility on plain config files, before any major
platform rollout.

## What you get

- **Field-origin tracing**: every channel, credential ref, and setting maps
  back to its source file and line
- **Production overlay tracking**: `no-config-platform-prod.yaml` overrides are traced
  separately from base `no-config-platform.yaml`
- **Change bundles**: the same publish → verify → attest → bridge flow works
  here as it does for Helm or Spring Boot
- **Future-proof**: when the platform team later adds channel naming policies
  or credential rotation rules, the same pipeline enforces them

## How provider config maps to DRY / WET / LIVE

```
  YOU EDIT (DRY)                    cub-gen TRACES (WET)              PROVIDER (LIVE)
┌─────────────────────┐          ┌──────────────────────┐         ┌─────────────────┐
│ no-config-platform.yaml        │          │ ConfigMap            │         │ Provider channels│
│ no-config-platform-prod.yaml   │──import─▶│ Provider config      │──API───▶│ Live messaging   │
│ platform/ (empty)   │          │ with provenance      │         │ Prod settings    │
└─────────────────────┘          └──────────────────────┘         └─────────────────┘
  App team: provider config.       Rendered config with              What's active
  No platform layer yet.           field-origin tracing.             in the provider.
```

**DRY** is what the app team edits: `no-config-platform.yaml` declares channels, app identity,
and credential references. `no-config-platform-prod.yaml` overrides for production.

**WET** is what cub-gen produces: rendered provider config as a ConfigMap with
every field traced back to its DRY source.

**LIVE** is what's active in the external service. There's no Flux/ArgoCD here —
the provider has its own sync mechanism.

This is intentionally the "no platform" end of the spectrum. The `platform/`
directory is empty because there are no platform contracts *yet*. When the
platform team later adds policies, they go in `platform/` and the same pipeline
enforces them.

| File | Owner | What it controls |
|------|-------|-----------------|
| `no-config-platform.yaml` | App team | Base config — channels, app identity, credential refs |
| `no-config-platform-prod.yaml` | App team | Production overlay — prod channels, region settings |
| `platform/.gitkeep` | — | Placeholder for future platform policies |

## If you already manage provider config directly

This example is for teams that run app-level provider config without a strong
platform abstraction yet:

- Application teams own channel/topic/credential config directly.
- There is little or no platform policy at first.
- Incidents still require tracing which config line changed live behavior.

cub-gen gives you governance visibility before you build a full platform layer:
field origins, ownership boundaries, and evidence artifacts over plain app config.

## Why this maps cleanly to the cub-gen framework

| Existing provider-config model | cub-gen concept | Why it matters |
|------|------|------|
| `no-config-platform*.yaml` | DRY app intent | Teams keep editing familiar provider config files. |
| Rendered provider payloads | WET targets with provenance | Every live-impacting field can be traced back to source. |
| Optional future `platform/` rules | Governance layer | You can add policy later without replacing authoring workflow. |
| Provider sync/runtime | LIVE state | Runtime remains external; cub-gen focuses on safe config change flow. |

## Try it

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Detect provider config
./cub-gen gitops discover --space platform --json ./examples/just-apps-no-platform-config

# Import with field-origin tracing
./cub-gen gitops import --space platform --json \
  ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, provenance: .provenance[0].field_origin_map}'
```

cub-gen detects `no-config-platform.yaml` as an `no-config-platform` provider source. Even without
platform policies, the import traces every field and computes inverse-edit
guidance.

## Real-world scenario: adding a new event channel

**Who**: A checkout team at an e-commerce company using a realtime provider for
order notifications.

### Scenario A — Standard channel addition (ALLOW)

The app team adds a new cancellation channel:

```yaml
# no-config-platform.yaml
channels:
  inbound: checkout.inbound
  outbound: checkout.outbound
  cancellations: checkout.cancellations   # new
```

```bash
# Produce evidence bundle
./cub-gen publish --space platform \
  ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# Bridge to ConfigHub
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# No platform policies → ALLOW (app-team scope, no violations)
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by app-lead --reason "standard channel addition"
```

Even without platform contracts, the governed pipeline provides:
- Audit trail of every channel change
- Attestation linking the change to CI verification
- Decision record showing who approved and why

### Scenario B — Future policy violation (BLOCK)

When the platform team adds `platform/policies/channel-naming.yaml` enforcing
a `{team}.{purpose}` naming convention:

```yaml
# no-config-platform.yaml — violates future naming policy
channels:
  inbound: checkout.inbound
  outbound: checkout.outbound
  bad_channel: LOUD_CHANNEL_NAME   # violates {team}.{purpose} pattern
```

```bash
# After platform adds channel-naming.yaml policy
./cub-gen publish --space platform \
  ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config > bundle.json
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
./cub-gen bridge ingest --in bundle.json --base-url "$BASE_URL" > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json

# Platform policy now enforces naming → BLOCK
./cub-gen bridge decision apply --decision decision.json --state BLOCK \
  --approved-by governance-bot \
  --reason "Channel 'LOUD_CHANNEL_NAME' violates naming policy: must use {team}.{purpose} format."
```

The same pipeline catches violations *before* they reach the provider —
without changing the app team's workflow.

## How it works

cub-gen's `no-config-platform` generator detects `no-config-platform.yaml` containing a service
identifier matching the provider-config pattern. On import:

1. **Classifies inputs** — `no-config-platform.yaml` (role: provider-config-base),
   `no-config-platform-prod.yaml` (role: provider-config-overlay)
2. **Maps field origins** — channels, credential refs, and app identity all
   trace to their source file with ownership metadata
3. **Handles empty platform** — the platform directory is recognized as empty;
   no contract validation occurs, but the governance pipeline still works
4. **Emits inverse guidance** — "to change the outbound channel in production,
   edit `no-config-platform-prod.yaml` channels section"

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `no-config-platform.yaml` | App team | Provider config — channels, identity, credentials |
| `no-config-platform-prod.yaml` | App team | Production overlay |
| `platform/.gitkeep` | — | Future platform policies |

## Why this pattern matters

The governance model scales down. A 10-line provider config deserves the same
provenance and decision trail as a 200-line Helm chart. When you later add
platform policies, the pipeline already exists.

## Next steps

- **Backstage catalog**: [`backstage-idp`](../backstage-idp/) — another
  non-K8s governance pattern
- **Full platform example**: [`helm-paas`](../helm-paas/) — the Kubernetes
  workload end of the spectrum
- **E2E demo**: `../demo/module-5-no-config-platform.sh`

## Run from ConfigHub (connected mode)

If you already have ConfigHub, start here:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"

# Publish and ingest
./cub-gen publish --space platform \
  ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN"
```

## 6. Inspect the result

After running discover/import, inspect:

```bash
# Field-origin map (provider fields → source)
./cub-gen gitops import --space platform --json \
  ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config \
  | jq '.provenance[0].field_origin_map'

# Provider config analysis
./cub-gen gitops import --space platform --json \
  ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config \
  | jq '.provenance[0].provider_config_analysis'

# Evidence bundle
./cub-gen publish --space platform \
  ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config \
  | jq '{change_id, bundle_digest: .bundle.digest}'
```

## 7. Try one governed change

**ALLOW path**: App team adds standard channel:

```yaml
# no-config-platform.yaml change
channels:
  cancellations: checkout.cancellations  # follows {team}.{purpose}
```

Result: No policy violations (or no policies yet) → **ALLOW**

**BLOCK path**: After platform adds naming policy:

```yaml
# no-config-platform.yaml change
channels:
  bad: ALLCAPS_CHANNEL   # violates naming convention
```

Result: Policy violation detected → **BLOCK**

## Local and Connected Entrypoints

From repo root:

```bash
# Local/offline
./examples/just-apps-no-platform-config/demo-local.sh

# Connected (requires ConfigHub auth)
cub auth login
./examples/just-apps-no-platform-config/demo-connected.sh
```
