# cub-track: Git-Native Mutation Ledger

> cub-track records **why** configuration changes happen — not just what changed,
> but who proposed it, what evidence existed, what was decided, and what the
> outcome was. It stores this as structured records linked to Git commits,
> producing an audit trail that classical GitOps cannot.

**Part of:** [AI and GitOps v7 Document Set](/docs/agentic-gitops/00-index/00-gitops7-index.md)
**Status:** Planning doc (v7)
**Date:** 2026-02-28

---

Qualification rule:

Use `Agentic GitOps` only when an active inner reconciliation loop
(`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that
loop, classify the flow as `governed config automation`.

---

## Table of Contents

1. [Summary](#summary)
2. [Motivation](#motivation)
3. [What cub-track Is](#what-cub-track-is)
4. [The Core Primitive: Change Interaction Card](#the-core-primitive-change-interaction-card)
5. [Commands](#commands)
6. [Git Storage Model](#git-storage-model)
7. [Trust Tiers](#trust-tiers)
8. [Attestation Contract](#attestation-contract)
9. [Examples](#examples)
10. [Adoption Stages](#adoption-stages)
11. [FAQ](#faq)
12. [Cross-References](#cross-references)

---

## Summary

**cub-track** is an open-source, Git-native mutation ledger for AI-assisted
GitOps changes. It links every configuration mutation to a structured record
that captures intent, evidence, decision, execution, and outcome — making the
full governance chain inspectable from a single Git commit.

cub-track works with any Git-based GitOps workflow. It is compatible with Flux,
ArgoCD, and Helm. It runs locally with no backend dependency (Stage 0), and
optionally connects to ConfigHub for cross-repo search, policy evaluation, and
governed execution (Stages 1-3).

In governed mode, verification evidence and attestation are required before and after execution; they are core to the model, not add-ons.

---

## Motivation

### The problem

AI agents are starting to author configuration changes. An agent might scale a
Deployment based on a traffic forecast, bump a chart version after a security
advisory, toggle a feature flag in response to an error budget, or restructure
a Kustomize overlay to reduce duplication. This is useful. It is also
ungoverned.

Classical GitOps captures what changed (the diff) and whether reconciliation
happened (Flux sync status, Argo health check). It does not capture:

- **Why was this change proposed?** What intent or evidence triggered it?
- **What was decided?** Did a policy evaluate it? Was a human consulted?
- **What happened afterward?** Did the rollout succeed? Did drift occur?

When a human makes one change a week, the answers live in PR descriptions,
Slack threads, and tribal knowledge. When an AI agent makes forty-seven changes
overnight, those answers need to be structured, searchable, and provable.

### The three questions

cub-track exists to answer three questions for any configuration mutation:

| Question | What answers it |
|----------|----------------|
| **Why was this change proposed?** | The intent record: what the actor wanted to achieve |
| **What was decided?** | The decision record: which policy evaluated, ALLOW / ESCALATE / BLOCK |
| **What happened?** | The outcome record: execution result + post-execution observation |

### A concrete scenario

An AI agent made 47 configuration changes overnight — scaling adjustments,
image updates, feature flag toggles, certificate rotations. At 9am, the
compliance team asks: *"Can you prove that each change was authorized and
validated?"*

**Without cub-track:** The evidence is scattered across Git commits (some with
useful messages, some not), Flux sync events, Argo health checks, and a chat
transcript in the AI tool. Reconstructing which change was authorized by which
policy requires manual archaeology across systems. The audit takes days.

**With cub-track:** Each of the 47 changes has a ChangeInteractionCard linked
to the Git commit via a trailer. Each card has structured fields: intent (what
the agent proposed), decision (which policy evaluated it, ALLOW/ESCALATE/BLOCK),
execution (what actually ran), and outcome (what cub-scout observed afterward).
The mutation ledger *is* the compliance export. The audit takes minutes.

---

## What cub-track Is

- **Open source.** MIT-licensed, part of the cub-scout project.
- **Git-native.** Metadata lives in Git: commit trailers, compact receipts on a
  metadata branch. No external database required.
- **Local-first.** Works on a single repo with no backend dependency. Run
  `cub-track enable` and start recording.
- **Compatible.** Works with Flux, ArgoCD, Helm, Kustomize, or any Git-based
  workflow. Does not replace any controller or reconciler.
- **Incremental.** Stage 0 is free and standalone. Each subsequent stage adds
  capability without requiring migration from the previous stage.

### What cub-track is NOT

- **Not a controller.** It does not reconcile or modify cluster state.
- **Not a reconciler.** Flux and Argo remain the runtime reconcilers.
- **Not a replacement for Git.** Git remains a primary collaboration and ingress surface for many teams. cub-track adds structured metadata to Git commits.
- **Not a replacement for CI/CD.** cub-track records governance; it does not
  run pipelines.
- **Not a chat logger.** The unit of governance is a structured mutation record,
  not a conversation transcript.

---

## The Core Primitive: Change Interaction Card

The unit of governance in cub-track is the **ChangeInteractionCard** — a
structured record that captures the full lifecycle of a configuration mutation.

The card is not a log entry. It is a governed mutation record with five
sections, each answering a specific question:

```
intent + evidence + decision + execution + outcome
```

### Schema

```json
{
  "schema_version": "change-interaction-card.v1",
  "card_id": "cic_7f8e9d0c1b2a3f44",

  "identity": {
    "repo": "github.com/acme/platform",
    "commit_sha": "9f29a5c...",
    "trailers": {
      "Cub-Mutation": "cic_7f8e9d0c1b2a3f44",
      "Cub-Agent": "codex"
    }
  },

  "intent": {
    "summary": "Scale payments-api to 5 replicas in prod",
    "domain": "app",
    "targets": [
      {
        "kind": "Deployment",
        "namespace": "payments",
        "name": "payments-api"
      }
    ]
  },

  "decision": {
    "result": "ALLOW",
    "verification_result": "pass",
    "reason": "Policy checks passed: trust tier 1, replicas within platform constraint range",
    "policy_refs": ["policy.gitops.tier1.app-scaling"]
  },

  "execution": {
    "status": "succeeded",
    "runtime": "confighub-actions"
  },

  "outcome": {
    "result": "applied",
    "message": "Deployment healthy after rollout: 5/5 replicas ready"
  }
}
```

### What each section answers

| Section | Question | Key fields |
|---------|----------|------------|
| **identity** | Which commit? Which repo? | `commit_sha`, `repo`, `trailers` |
| **intent** | What did the actor want to achieve? | `summary`, `domain`, `targets` |
| **decision** | Was it allowed? By what policy? | `result` (ALLOW/ESCALATE/BLOCK), `policy_refs` |
| **execution** | What actually ran? | `status` (succeeded/failed/partial), `runtime` |
| **outcome** | What happened afterward? | `result` (applied/failed/rolled-back), observation |

The card is linked to its Git commit via the `Cub-Mutation` trailer. Given a
commit SHA, you can retrieve the card. Given a card, you can retrieve the
commit. The linkage is bidirectional and immutable.

---

## Commands

### MVP Commands

These commands are the initial cub-track surface — available at Stage 0 (OSS
Local) with no backend dependency.

#### `cub-track enable`

Install Git hooks and initialize the metadata branch.

```bash
$ cub-track enable

✓ Pre-commit hook installed
✓ Metadata branch created: cub/mutations/v1
✓ cub-track is ready. Commits will now record mutation context.
```

This creates:
- A pre-commit hook that prompts for mutation context on config changes
- A metadata branch (`cub/mutations/v1`) for append-only card storage
- A deprecated compatibility read alias (`cub/checkpoints/v1`) for legacy repos

#### `cub-track explain --commit <sha>`

Explain a commit in intent / decision / outcome terms.

```bash
$ cub-track explain --commit 9f29a5c

Commit:  9f29a5c "Scale payments-api replicas to 5"
Card:    cic_7f8e9d0c1b2a3f44

Intent:  Scale payments-api to 5 replicas in prod
         Domain: app
         Targets: Deployment/payments/payments-api

Decision: ALLOW
         Policy: policy.gitops.tier1.app-scaling
         Reason: Trust tier 1, replicas within platform constraint range (2-10)

Execution: succeeded
         Runtime: confighub-actions

Outcome: applied
         Deployment healthy: 5/5 replicas ready
```

Every field is structured and queryable. No guessing from commit messages.

#### `cub-track search`

Search mutation history by text, file, agent, or decision.

```bash
# Find all changes by the traffic-forecaster agent
$ cub-track search --agent traffic-forecaster

  cic_7f8e9d0c  9f29a5c  ALLOW   "Scale payments-api to 5 replicas"
  cic_3a4b5c6d  e1f2a3b  ALLOW   "Scale orders-api to 3 replicas"
  cic_8d9e0f1a  b2c3d4e  ESCALATE "Scale billing-api to 10 replicas"  ← required human approval

# Find all escalated changes
$ cub-track search --decision ESCALATE

  cic_8d9e0f1a  b2c3d4e  ESCALATE "Scale billing-api to 10 replicas"
  cic_f1a2b3c4  d5e6f7a  ESCALATE "Update prod TLS certificate"

# Find changes to a specific file
$ cub-track search --file "payments/deployment.yaml"

  cic_7f8e9d0c  9f29a5c  ALLOW   "Scale payments-api to 5 replicas"
  cic_b3c4d5e6  a7b8c9d  ALLOW   "Bump payments-api image to 1.2.3"
```

### Planned Commands (Post-MVP)

These commands extend cub-track's role as a **redirection layer** — guiding
users from WET edits back to DRY sources when field-origin maps are available.

#### `cub-track explain --commit <sha> --fields`

Enrich the explanation with DRY origin information from field-origin maps.

```bash
$ cub-track explain --commit 9f29a5c --fields

Commit:  9f29a5c "Scale payments-api replicas to 5"
Card:    cic_7f8e9d0c1b2a3f44

Changed fields:
  spec.replicas: 2 → 5
    DRY source: values-prod.yaml:14 in acme/platform-config
    Editable by: app-team

  (no other fields affected)

Intent:  Scale payments-api to 5 replicas in prod
Decision: ALLOW (policy.gitops.tier1.app-scaling)
Outcome: applied, healthy
```

The user sees not just what changed, but *where the authoritative source lives*.
Next time they know to edit `values-prod.yaml:14` directly.

#### `cub-track suggest`

Before committing a WET change, check provenance. If the changed field has a
DRY origin, surface it.

```bash
$ git add payments/deployment.yaml
$ cub-track suggest

⚠  You are editing generated output.

  Changed field: spec.replicas (2 → 5)
  DRY source:    values-prod.yaml:14 in acme/platform-config
  Generator:     helm (traefik/traefik@33.2.1)

  → Edit the DRY source instead?
    $ cub edit payments-api --field spec.replicas --variant prod

  → Proceed with WET edit anyway? (will be classified as overlay drift)
    [y/n]
```

This creates **productive friction**: the user is not blocked, but they are
informed. If they proceed, the change is recorded as overlay drift — a distinct
category from runtime drift — and the system will flag it for future promotion
back to DRY.

---

## Git Storage Model

cub-track follows a strict **DRY/WET type boundary** with a dual-store pattern: compact, reviewable linkage artifacts in Git; control-plane dry+wet Units and governance state in ConfigHub.

### What Git stores (primary collaboration ingress)

Git stores the linkage artifacts — small, immutable, and reviewable in PRs:

| Artifact | Format | Purpose |
|----------|--------|---------|
| **Commit trailers** | `Cub-Mutation: <card_id>` | Link commit to its card |
| | `Cub-Checkpoint: <card_id>` (legacy read) | Backward compatibility with older repos |
| | `Cub-Intent: <summary>` | Human-readable intent in the commit |
| | `Cub-Agent: <agent_id>` | Which agent authored the change |
| **Linkage receipts** | `linkage-receipt.v1` | Trailer → card ID mapping |
| **Governance receipts** (Stage 2+) | `decision-receipt.v1` | Policy decision summary |
| | `execution-receipt.v1` | Execution result summary |
| | `outcome-receipt.v1` | Post-execution observation summary |
| **Metadata branch** | `cub/mutations/v1` | Append-only card storage |

> **Compatibility note:** `Cub-Mutation` is the write default.
> `Cub-Checkpoint` is a deprecated read-only alias for existing history.
> Readers accept both; writers emit `Cub-Mutation` only.

### What ConfigHub stores (dry units, wet units, and governance state)

When connected (Stage 1+), ConfigHub stores control-plane state, including dry units, wet units, and governance metadata:

- Full policy input/output graphs and rule traces
- Approval chain metadata
- Token issuance metadata (scoped execution authority)
- Full execution records with timing and error detail
- Pre/post scan findings and evidence links
- Cross-repo correlation and search indexes

### Anti-patterns

| Don't | Why |
|-------|-----|
| Store full transcripts in Git | Bloats repo, not reviewable in PRs |
| Store tokens or auth material in Git | Security risk |
| Use Git as a telemetry store | Wrong tool; use a time-series database |
| Maintain mutable indexes in Git | Git is append-only by design; mutable state belongs in a database |

---

## Trust Tiers

cub-track enforces a tiered trust model for mutation rights. Higher tiers
require stronger evidence and approval before execution is authorized.

| Tier | Capability | Approval Required | Example |
|------|------------|-------------------|---------|
| **0** | Observe only | None | cub-scout read access; no apply rights |
| **1** | Low-risk apply | Automated policy check | Scale within platform constraints; image update within approved registry |
| **2** | Medium-risk apply | Human approval | Cross-environment promotion; new service registration |
| **3** | High-risk / prod | Dual approval + strong attestation | Production infrastructure changes; security policy modifications |

Execution rights are tier-bound. The policy engine evaluates the trust tier
before issuing a scoped execution token. An agent at Tier 1 cannot perform
Tier 2 actions without escalation.

---

## Attestation Contract

Every governed execution (Tier 1+) must produce an attestation that includes,
at minimum, six fields. These are the minimum set required to answer *"who did
what, why were they allowed, and what happened?"* after the fact.

| Field | What it records | Example |
|-------|----------------|---------|
| **actor** | Who or what initiated the mutation | `traffic-forecaster` (agent), `alexis` (user), `ci/github-actions` (CI) |
| **approved_intent_revision** | The exact revision of intended state that was authorized | `sha256:9f29a5c...` |
| **rendered_artifact_digest** | SHA-256 of the published OCI artifact or Git tree | `sha256:a1b2c3d...` |
| **target_scope** | Cluster, namespace, and resource(s) the execution may touch | `prod-cluster/payments/Deployment/payments-api` |
| **observed_result** | Post-execution observation | `applied` / `partial` / `failed` / `rolled-back` |
| **policy_checks** | Which policies were evaluated and their outcomes | `tier1.app-scaling: ALLOW`, `image-registry: ALLOW` |

Higher tiers may add fields (e.g., dual approval chain at Tier 3, scan findings
at Tier 2+), but these six are always present.

---

## Examples

### Example 1: Explaining a commit after an incident

A Deployment is crashing in production at 3am. The on-call engineer finds the
last config commit and asks cub-track what happened:

```bash
$ cub-track explain --commit e4f5a6b

Commit:  e4f5a6b "Bump traefik chart to 33.2.1"
Card:    cic_a1b2c3d4e5f6

Intent:  Upgrade traefik Helm chart from 32.1.0 to 33.2.1 across all envs
         Domain: platform
         Targets: Deployment/traefik (dev, staging, prod)

Decision: ALLOW
         Policy: policy.gitops.tier2.chart-upgrade
         Reason: Chart from approved repo, pre-publish validation passed

Execution: succeeded
         Runtime: confighub-actions
         Duration: 4m 12s

Outcome: partial
         dev: healthy (IngressRoute v3 API)
         staging: healthy (IngressRoute v3 API)
         prod: UNHEALTHY — IngressRoute v3 API not supported by
               traefik 32.x data plane (Helm CLI 3.13.2 on prod)

⚠  Generator version mismatch detected:
   dev/staging: Helm CLI 3.14.0
   prod: Helm CLI 3.13.2
   The chart rendered differently on prod due to the older CLI.
```

The engineer knows the root cause in under 2 minutes: the chart upgrade
produced different output on prod because prod runs an older Helm CLI version.
Without cub-track, this would require correlating Git logs, Helm release
history, and Flux events across three clusters.

### Example 2: AI agent governance flow

An AI agent proposes a scaling change. cub-track evaluates it against the
trust tier and escalates when needed:

```
AI Agent (traffic-forecaster):
  "Scale payments-api to 10 replicas in prod based on traffic forecast"

cub-track: Recording mutation intent...
  Intent: scale spec.replicas 5 → 10 (variant=prod)
  Agent: traffic-forecaster
  Trust tier: 2 (medium-risk, requires human approval)

  → Escalating to human approval.
  → Created PR: acme/payments-api#47 "Scale payments-api to 10 replicas"
  → Notified: @alexis (on-call), #payments-ops (Slack)

  Waiting for approval...

Alexis (in ConfigHub GUI or Slack): [Approve]

cub-track: Decision: ALLOW (approved by alexis)
  → Committing to values-prod.yaml
  → Re-rendering via helm template...
  → Publishing OCI artifact...
  → Flux reconciling...
  → cub-scout observing...
  → ✓ 10/10 replicas healthy

Attestation recorded:
  actor: traffic-forecaster
  approved_by: alexis
  intent_revision: sha256:a1b2c3d
  artifact_digest: sha256:e4f5a6b
  target_scope: prod-cluster/payments/Deployment/payments-api
  observed_result: applied, 10/10 replicas healthy
  policy_checks: tier2.scaling: ESCALATE → ALLOW (human approved)
```

The full chain — intent, escalation, human approval, execution, observation,
attestation — is recorded as a single ChangeInteractionCard. Months later, an
auditor can retrieve this card from the commit trailer and see the entire
governance chain.

### Example 3: Searching mutation history

```bash
# "Show me everything that changed in payments-api this week"
$ cub-track search --text payments-api --since 7d

  cic_7f8e9d0c  9f29a5c  ALLOW     "Scale to 5 replicas"         (traffic-forecaster)
  cic_b3c4d5e6  a7b8c9d  ALLOW     "Bump image to 1.2.3"         (ci/github-actions)
  cic_d5e6f7a8  c9d0e1f  ALLOW     "Update env var DB_POOL=20"   (alexis)
  cic_f7a8b9c0  e1f2a3b  ESCALATE  "Change resource limits"      (cost-optimizer) ← human approved

# "Show me all changes the cost-optimizer agent has ever made"
$ cub-track search --agent cost-optimizer

  cic_f7a8b9c0  e1f2a3b  ESCALATE  "Change resource limits"      payments-api/prod
  cic_1a2b3c4d  f5a6b7c  ALLOW     "Reduce replicas to 3"        orders-api/staging
  cic_3c4d5e6f  a7b8c9d  BLOCK     "Remove resource limits"      billing-api/prod ← policy blocked
```

### Example 4: Redirecting a WET edit to DRY

A developer edits a rendered manifest directly. `cub-track suggest` catches
this before commit:

```bash
$ vim payments/deployment.yaml     # manually changes replicas: 2 → 5
$ git add payments/deployment.yaml
$ cub-track suggest

⚠  You are editing generated output.

  Changed field: spec.replicas (2 → 5)
  DRY source:    values-prod.yaml:14 in acme/platform-config
  Generator:     helm (traefik/traefik@33.2.1)
  Last rendered:  2026-02-27T14:30:00Z

  This field is generated by Helm from values-prod.yaml. If you edit the
  rendered output directly, the change will be overwritten on the next render.

  Recommended:
    $ cub edit payments-api --field spec.replicas --variant prod --value 5

  This will:
    1. Update values-prod.yaml:14 in acme/platform-config
    2. Re-render via helm template
    3. Show the diff (only spec.replicas changes)
    4. Commit to the correct repo

  → Use the recommended path? [y/n]
  → Proceed with WET edit anyway? (recorded as overlay drift) [y/n]
```

If the developer proceeds with the WET edit, cub-track:
1. Records it as a ChangeInteractionCard with `domain: overlay`
2. Classifies the change as **overlay drift from DRY source** — a distinct
   category from runtime drift (cluster vs. intended state)
3. Flags it for future promotion back to DRY when the change stabilizes

---

## Adoption Stages

cub-track adoption is incremental. Each stage adds capability without requiring
migration from the previous stage.

| Stage | You install | You get | Git writes (portable linkage) | ConfigHub writes (dry+wet control-plane state) |
|-------|------------|---------|-------------------------------|-----------------------------------------------|
| **0. OSS Local** | `cub-track` only | Commit-linked mutation history, explain, search | Trailers + local card + linkage receipts | None required |
| **1. Connected** | + ConfigHub credentials | Cross-repo search, centralized provenance | Same compact linkage artifacts | Ingested card index, evidence catalog, dry/wet unit linkage |
| **2. Governed** | + policy/runtime services | Policy-gated mutation: ALLOW / ESCALATE / BLOCK | + governance receipts (decision, execution, outcome) | Full policy traces, attestation, dry+wet governance joins |
| **3. Enterprise** | + org controls | Audit-grade reporting, retention, compliance exports | Same compact governance receipts | Full retention, RBAC, analytics across dry+wet estate |

### What you get at each stage

**Stage 0 (OSS Local)** — no backend, no account, no cost. Run `cub-track
enable` on any Git repo. Every config commit gets a trailer linking to a
ChangeInteractionCard. Use `explain` to understand commits and `search` to find
changes by agent, decision, or text. This is useful on its own — you get
structured mutation history that Git logs alone cannot provide.

> **Stage 0 note:** cub-track writes linkage receipts (trailer → card ID) but
> not governance receipts (decision, execution, outcome). Governance receipts
> require Stage 2+ where `confighub-scan` provides policy decisions.

**Stage 1 (Connected)** — add ConfigHub credentials. Cards are ingested into
ConfigHub's search index. You can now search across repos: *"show me every
change the cost-optimizer agent made across all 40 team repos this week."*
Evidence from cub-scout is correlated with mutation records.

**Stage 2 (Governed)** — add policy and runtime services. Mutations are now
evaluated against trust tiers before execution. The decision (ALLOW / ESCALATE
/ BLOCK) is recorded in the card and as a compact receipt in Git. This is where
cub-track becomes a governance layer, not just a record-keeping tool.

**Stage 3 (Enterprise)** — add org controls. Retention policies, RBAC,
compliance exports, fleet-wide analytics. The cards and attestations become the
audit artifact for regulatory reporting.

---

## FAQ

### Can I use cub-track without ConfigHub?

Yes. Stage 0 requires only `cub-track` and a Git repository. You get
commit-linked mutation history, explain, and search with no backend dependency.
cub-scout also works standalone against any kubectl context. ConfigHub adds
centralized storage, cross-repo search, policy evaluation, and governed
execution — but the local tools are useful on their own.

### What about secrets?

cub-track never stores secret values. ChangeInteractionCards may reference
secret names or paths (e.g., "updated the DB_PASSWORD secret reference in the
payments ConfigMap"), but the actual secret values are never recorded in cards,
receipts, or Git trailers. Integration with external secret operators (External
Secrets Operator, Sealed Secrets, Vault) is via reference, not by value.

### How does cub-track interact with Flux/Argo?

cub-track is complementary. Flux and Argo are reconcilers — they make the
cluster match a declared source. cub-track records the governance context
around the changes that Flux/Argo reconcile. They are different layers:

| Layer | Tool | Concern |
|-------|------|---------|
| **Reconciliation** | Flux / Argo | Make runtime match intended state |
| **Observation** | cub-scout | Detect when runtime diverges from intended state |
| **Mutation governance** | cub-track | Record why changes were proposed, decided, and executed |

cub-track does not interfere with Flux/Argo reconciliation. It adds metadata
to the Git commits that Flux/Argo consume, making the governance chain visible
without changing the reconciliation flow.

### What's the overhead?

Minimal. cub-track adds:
- **Git trailers** on config commits (~100 bytes per commit)
- **Compact receipts** on a metadata branch (~500 bytes per mutation)
- **A pre-commit hook** that prompts for mutation context (skippable for
  non-config changes)

The metadata branch (`cub/mutations/v1`) is append-only and separate from the
main branch. It does not affect your main branch history, CI pipelines, or
repo size in any meaningful way.

New writes always go to `cub/mutations/v1`; `cub/checkpoints/v1` remains a temporary read alias during migration.

### How does cub-track relate to field-origin maps?

Field-origin maps are produced by generators (see [02-generators-prd.md](/docs/agentic-gitops/02-design/10-generators-prd.md)).
They map each WET output field to its DRY input source (file, path, line,
editable_by). cub-track uses field-origin maps in two commands:

- **`explain --fields`** enriches commit explanations with DRY source info
- **`suggest`** detects when a user edits a generated field and redirects them
  to the DRY source

Without field-origin maps, cub-track still works — it records mutations and
governance. With field-origin maps, it also becomes a redirection layer that
guides users from WET back to DRY.

### What's the difference between overlay drift and runtime drift?

Two distinct categories:

| Type | What happened | Detected by |
|------|---------------|-------------|
| **Runtime drift** | Cluster state differs from reconciler's intended state | cub-scout (cluster vs. intended) |
| **Overlay drift** | WET field changed without `inputs.digest` change — someone edited the generator output, not the generator input | cub-track (WET vs. DRY provenance) |

Runtime drift means the cluster diverged from what Flux/Argo is trying to
apply. Overlay drift means the stored WET diverged from what the generator
would produce. They can coexist, and they require different responses.

---

## Cross-References

| Document | Relationship |
|----------|-------------|
| [01 — Introducing Agentic GitOps](/docs/agentic-gitops/01-vision/01-introducing-agentic-gitops.md) | The "why": classical gaps and invariants that cub-track enforces |
| [02 — Generators PRD](/docs/agentic-gitops/02-design/10-generators-prd.md) | Generator model and field-origin maps that cub-track uses for redirection |
| [03 — Field-Origin Maps and Editing](/docs/agentic-gitops/02-design/20-field-origin-maps-and-editing.md) | The editing model and adoption ladder that cub-track supports |
| [04 — App Model and Contracts](/docs/agentic-gitops/02-design/30-app-model-and-contracts.md) | Operating boundary: cub-track's role vs. ConfigHub, cub-scout, Flux/Argo |
| [06 — Governed Execution](/docs/agentic-gitops/02-design/40-governed-execution.md) | The two-loop model, evidence bundles, and write-back semantics that cub-track feeds into |
| [07 — User Experience](/docs/agentic-gitops/05-rollout/30-user-experience.md) | cub-track CLI examples, AI skill integration, Day 2 scenarios |
| [08 — Adoption and Reference](/docs/agentic-gitops/05-rollout/40-adoption-and-reference.md) | Adoption path showing where cub-track fits in the overall journey |
