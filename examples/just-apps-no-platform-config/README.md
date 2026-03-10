# Just Apps, No Platform Config (Ably)

**Pattern: app talks to an external service — no platform-side contracts, just provider config.**

## 1. What is this?

A checkout service needs real-time event channels from Ably (a managed pub/sub service). The app team declares what channels they need and where to find credentials. There are no platform templates, no Helm charts, no Kubernetes manifests to render — just configuration for an external provider.

This is the simplest cub-gen pattern: app-config-only. It exists to show that the DRY/WET governance model works for *any* configuration, not just Kubernetes workloads. Even a 10-line provider config gets field-origin tracing, inverse-edit guidance, and a verifiable change bundle.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **App team** | `ably.yaml` — channel names, app identity | Channel names, environment, credential refs |
| **Platform team** | (none today — future: channel policies, schema validation) | Nothing in this example |
| **GitOps reconciler** | N/A — Ably is a managed service, not a cluster resource | N/A |

This is intentionally the "no platform" end of the spectrum. The canonical pattern still applies: DRY intent → WET rendered config → governed change bundle. The platform directory is empty because there are no platform contracts *yet*. When the platform team later adds channel naming policies or credential rotation rules, they go in `platform/`.

## 3. What does cub-gen add?

Even without platform contracts, cub-gen provides:

- **Generator detection**: recognizes `ably.yaml` as an ably-config source (capabilities: `app-config-only`, `provider-config`, `inverse-provider-config-patch`)
- **Field-origin tracing**: every field in the rendered config maps back to the DRY source
- **Inverse-edit guidance**: "to change the outbound channel in production, edit `ably-prod.yaml`"
- **Change bundles**: `publish` + `verify` + `attest` produce the same evidence chain as any other generator

```bash
# Discover — detects ably generator
./cub-gen gitops discover --space platform --json ./examples/just-apps-no-platform-config

# Import — produces DRY/WET classification with provenance
./cub-gen gitops import --space platform --json ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, provenance: .provenance[0].field_origin_map}'
```

## 4. How do I run it?

```bash
# Build
go build -o ./cub-gen ./cmd/cub-gen

# Discover
./cub-gen gitops discover --space platform ./examples/just-apps-no-platform-config

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config

# Full bridge flow
./cub-gen publish --space platform ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config > /tmp/ably-bundle.json
./cub-gen verify --in /tmp/ably-bundle.json
./cub-gen attest --in /tmp/ably-bundle.json --verifier ci-bot > /tmp/ably-attestation.json
./cub-gen verify-attestation --in /tmp/ably-attestation.json --bundle /tmp/ably-bundle.json

# Cleanup
./cub-gen gitops cleanup --space platform ./examples/just-apps-no-platform-config
```

## 5. Real-world example using ConfigHub

A checkout team at an e-commerce company uses Ably for real-time order notifications.

**Day 1: Channel change request**

The app team needs a new event channel for order cancellations. They edit `ably.yaml`:

```yaml
channels:
  inbound: checkout.inbound
  outbound: checkout.outbound
  cancellations: checkout.cancellations   # new
```

**Day 2: Governed review**

```bash
# cub-gen detects the change and produces a bundle
./cub-gen publish --space platform ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json
```

The change bundle shows exactly what changed (new channel added), who authored it (app team), and what inverse-edit pointers apply (edit `ably.yaml` channels section).

**Day 3: ConfigHub ingests the bundle**

```bash
# Submit to ConfigHub
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json

# Decision engine evaluates — ALLOW (no platform policy violations)
./cub-gen bridge decision create --ingest ingest.json > decision.json
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by app-lead --reason "standard channel addition"
```

Even without platform contracts, the governed pipeline provides:
- Audit trail of every channel change
- Attestation linking the change to CI verification
- Decision record showing who approved and why

**Future: Platform adds channel policies**

When the platform team later adds `platform/policies/channel-naming.yaml` enforcing a naming convention, the same pipeline catches violations *before* they reach Ably — without changing the app team's workflow.

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `ably.yaml` | App team | DRY intent — channels, app identity, credential ref |
| `ably-prod.yaml` | App team | Production overlay — EU region, prod channel names |

## Why this pattern matters

Not every app has platform contracts. Not every service runs on Kubernetes. But every configuration change deserves provenance, and every change to production deserves a governed decision. The ably-config pattern proves the model scales down to the simplest case.
