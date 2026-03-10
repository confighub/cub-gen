# ConfigHub Actions (Lifecycle Governance)

**Pattern: ConfigHub's own release lifecycle — plan → verify → deploy — expressed as governed operations configuration.**

## 1. What is this?

A platform team uses ConfigHub Actions to govern their own release lifecycle. When a commit lands, three actions execute in sequence: a dry-run plan, a policy check, and a governed apply. In production, the verify step requires two approvals and the deploy step is restricted to business hours.

This is the same operations model as [`ops-workflow`](../ops-workflow/), but applied to ConfigHub's own lifecycle. It demonstrates that the governance pattern is recursive — ConfigHub governs its own changes the same way it governs application changes.

## 2. Who does what?

| Role | Owns | Edits |
|------|------|-------|
| **Platform team** | `operations.yaml` — action types, trigger config | Action sequence, trigger events |
| **Platform team** | `operations-prod.yaml` — production overrides | Approval requirements, deploy windows |
| **Platform owner** | `platform/` — lifecycle policies | Required action types, approval thresholds |
| **GitOps reconciler** | N/A — actions execute within ConfigHub's decision engine | N/A |

## 3. What does cub-gen add?

The `ops-workflow` generator detects ConfigHub Actions the same way it detects any operations workflow:

- **Generator detection**: recognizes `operations.yaml` with `workflow:` + `actions:` structure
- **DRY/WET classification**: action definitions are DRY, rendered execution plan is WET
- **Field-origin tracing**: action types, approval requirements, and deploy windows all trace to DRY sources
- **Inverse-edit guidance**: "to change approval requirements in production, edit `operations-prod.yaml` actions.verify section"

```bash
# Discover
./cub-gen gitops discover --space platform --json ./examples/confighub-actions

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/confighub-actions ./examples/confighub-actions \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs}'
```

## 4. How do I run it?

```bash
# Build
go build -o ./cub-gen ./cmd/cub-gen

# Discover
./cub-gen gitops discover --space platform ./examples/confighub-actions

# Import with provenance
./cub-gen gitops import --space platform --json ./examples/confighub-actions ./examples/confighub-actions

# Full bridge flow
./cub-gen publish --space platform ./examples/confighub-actions ./examples/confighub-actions > /tmp/actions-bundle.json
./cub-gen verify --in /tmp/actions-bundle.json
./cub-gen attest --in /tmp/actions-bundle.json --verifier ci-bot > /tmp/actions-attestation.json
./cub-gen verify-attestation --in /tmp/actions-attestation.json --bundle /tmp/actions-bundle.json

# Cleanup
./cub-gen gitops cleanup --space platform ./examples/confighub-actions
```

## 5. Real-world example using ConfigHub

A platform team manages 30 microservices through ConfigHub. Every release follows the same lifecycle: plan → verify → deploy. The lifecycle itself is governed configuration.

**Scenario: Tightening production approval requirements**

The security team requires three approvals for production deploys (up from two). The platform team edits `operations-prod.yaml`:

```yaml
actions:
  verify:
    approvals:
      required: 3   # was 2
  deploy:
    window: business-hours
```

**Governed pipeline:**

```bash
# 1. cub-gen detects the policy change
./cub-gen gitops import --space platform --json ./examples/confighub-actions ./examples/confighub-actions
# Field-origin: actions.verify.approvals.required changed in operations-prod.yaml

# 2. Produce evidence chain
./cub-gen publish --space platform ./examples/confighub-actions ./examples/confighub-actions > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 3. ConfigHub ingests — governance governs itself
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by security-lead --reason "SOC2 compliance: 3 approvals for prod"
```

**The recursive governance loop:**
1. ConfigHub Actions define the lifecycle (plan → verify → deploy)
2. Changes to the lifecycle definition are *themselves* governed through ConfigHub
3. The decision engine evaluates the lifecycle change using the current lifecycle rules
4. Every change to governance policy has the same provenance and attestation as every app change

This is not circular — it's the same governance model applied at every level. The lifecycle definition is configuration; configuration gets governed.

## The action lifecycle

```
Commit lands
    │
    ▼
plan (dry-run)
    │  What would change? Show the diff.
    ▼
verify (policy-check)
    │  Do the changes pass platform policies?
    │  In prod: requires N approvals.
    ▼
deploy (apply)
    │  Apply the change with attestation.
    │  In prod: restricted to business hours.
    ▼
Decision recorded in ConfigHub
```

## Key files

| File | Owner | Purpose |
|------|-------|---------|
| `operations.yaml` | Platform team | DRY intent — action types (dry-run, policy-check, apply), commit trigger |
| `operations-prod.yaml` | Platform team | Production overlay — approval count, deploy window |

## Related examples

- [`ops-workflow`](../ops-workflow/) — Scheduled maintenance workflows. Same operations model, different trigger pattern (cron vs commit).
- [`swamp-automation`](../swamp-automation/) — Swamp AI workflow automation. Same governance, different execution engine.
