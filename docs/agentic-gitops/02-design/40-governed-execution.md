# Governed Execution: Evidence, Trust, and Write-Back Semantics

**Part of:** [AI and GitOps v7 Document Set](../00-index/00-gitops7-index.md)
**Status:** Planning doc (v7)
**Date:** 2026-02-28
**Audience:** Security, compliance, SREs, platform teams
**Purpose:** Two-loop model, evidence and drift, write-back semantics, failure modes

---

## Table of Contents

1. [The Two-Loop Model](#1-the-two-loop-model)
2. [Evidence and the Drift Loop](#2-evidence-and-the-drift-loop)
3. [Write-Back Semantics](#3-write-back-semantics)
4. [Failure Modes](#4-failure-modes)
5. [Cross-References](#5-cross-references)

---

## 1. The Two-Loop Model

### From Classical to Governed GitOps

Classical GitOps covers commit-to-cluster convergence. Governed GitOps extends
every phase with intent capture, policy gates, and evidence:

| Phase | Classical GitOps | Governed (Agentic) GitOps | What's Added |
|-------|-----------------|--------------------------|--------------|
| **Author** | Human writes YAML, commits to Git | Human or agent expresses intent (DRY) — directly or through ConfigHub's editing surface; generator renders explicit config (WET) | Provenance: generator version, input digest, field-origin map |
| **Review** | PR review (human eyeballs) | PR review + policy evaluation (`ALLOW | ESCALATE | BLOCK`) | Automated constraint validation; trust tier gating |
| **Publish** | Merge to main; reconciler watches repo | ConfigHub stores Unit, publishes OCI artifact or Git source | Immutable intended state with provenance; not just "latest commit on main" |
| **Reconcile** | Flux/Argo pull and apply | Flux/Argo pull and apply *(unchanged)* | — (reconcilers are not replaced) |
| **Observe** | `flux get all` / Argo health check | cub-scout structured evidence: field-level diff + provenance link | Drift is typed, classified, and linked back to the operation that set the expected value |
| **Respond** | Manual fix or re-sync | Policy-driven: alert, propose-revert, propose-accept, require-approval | Governed reverse flow — no silent overwrite in either direction |
| **Record** | Git log + Flux/Argo events | [cub-track](../05-rollout/10-cub-track.md) ChangeInteractionCard: intent + decision + execution + outcome | Mutation ledger answers "why was this allowed?" not just "what changed" |
| **Attest** | *(not modeled)* | Signed attestation: actor, intent revision, artifact digest, observed result | Audit-grade proof that authority, action, and outcome are linked |

The left two columns are what Flux/Argo users already have. The right two columns
are what this model adds. Nothing in the left column is removed or replaced.

### Why Governed GitOps Is Inherently Bidirectional

Classical GitOps is one-directional by design: Git wins, always. But break-glass
fixes, runtime-discovered state (autoscaler adjustments, cert rotations), and AI
agents acting on live systems all produce legitimate changes that originate in the
cluster, not in Git. Without a governed reverse path, those changes are either
silently reverted or silently tolerated — both bad.

This model adds the reverse direction explicitly: observe, produce evidence,
propose back to intended state, review, accept or reject. Neither direction is
automatic.

### Outer Loop (Authority and Governance)

```
ask -> specify -> plan -> decide -> authorize
```

Outputs:
1. Intended state artifact(s)
2. Policy decision (`ALLOW | ESCALATE | BLOCK`)
3. Verification verdict + attestation (`pass | fail | abstain`)
4. Scoped execution authority (short-lived token)

This loop is where **authoritative generators operate**. DRY intent is transformed
into explicit WET configuration, validated against platform constraints, and
authorized for publication. In-cluster templaters (ApplicationSet, App-of-Apps) run
in the inner loop instead; their outputs are observed by cub-scout but not governed
at this stage.

### Inner Loop (Runtime Reconciliation)

```
publish -> reconcile -> observe -> verify -> attest
```

Flux/Argo remain in this loop today. ConfigHub publishes OCI artifacts or Git source
updates. Flux/Argo detect changes and reconcile clusters. cub-scout observes the
result.

### Combined Flow

```
Developer/Agent Intent (DRY)
        |
        v                          ConfigHub Editing Surface
    Generator (deterministic,  <-- (resolves field-origin map,
     versioned)                     writes back to DRY source)
        |                                    ^
        v                                    |
ConfigHub (WET, system of record  ----------+
     for intended state)           (user views, compares,
        |                          traces, edits here)
        v
    Publish (OCI artifact or Git source update)
        |
        v
    GitOps Reconcile (Flux/Argo, inner loop)
        |
        v
    Runtime (what actually exists)
        |
        v
    cub-scout Observes
        |
        v
    Evidence Bundle (structured diff + provenance)
        |
        +---> Export (Slack, Jira, S3, ConfigHub history)
        |
        +---> Decision (human review, policy engine, or workflow)
```

### Platform Engineering Flow

```
Platform team provides generator + constraints
        |
App team writes app code + app config (DRY intent)
        |     ^
        |     |--- ConfigHub editing surface (resolves field-origin
        |          map, commits change to DRY source in Git)
        v
Generator turns DRY into deployment manifests (WET)
        |
ConfigHub stores WET as Units with provenance + field-origin maps
        |
Workers publish artifacts (OCI default, or Git source)
        |
Flux/Argo reconcile cluster from published artifacts
        |
cub-scout observes runtime, produces evidence
```

### Mandatory enforcement controls (normative)

The governed path is valid only when all controls below pass:

1. `GeneratorContract` is signed and includes deterministic output hash.
2. `ProvenanceRecord` captures immutable `input_hash`, `toolchain_version`,
   `policy_version`, `run_id`, and artifact digests.
3. Inverse write is bounded to `OwnershipMap` declared paths.
4. Out-of-scope write is auto-`BLOCK`.
5. Replay verification is mandatory before protected write-back.
6. Replay mismatch is auto-`ESCALATE`.
7. Policy decision is explicit: `ALLOW | ESCALATE | BLOCK`.
8. `ALLOW` requires attestation linkage (`who`, `evidence_bundle_id`,
   `decision_id`).
9. Protected DRY updates are PR/MR-only; no silent direct writes.
10. Verification failure downgrades flow to read-only evidence mode.
11. Every decision and mutation appends to mutation ledger.

Qualification rule:

1. If the inner loop (`WET -> LIVE` reconciliation by Flux/Argo or equivalent)
   is absent, this remains governed config automation, not Agentic GitOps.

---

## 2. Evidence and the Drift Loop

### Three Sources of Truth

| Source | Contains | Authority For |
|--------|----------|---------------|
| **ConfigHub** | Intended WET | "What should exist" |
| **Cluster** | Running state | "What does exist" |
| **cub-scout** | Observed evidence | "What we can prove" |

Evidence is the structured diff between intended and observed state, captured at a
point in time. Evidence is observational — creating or exporting it never modifies
intended or runtime state.

### Scope of cub-scout Observation

cub-scout compares cluster reality against the reconciler's intended state (what
Flux/Argo is trying to apply). It does not compare stored WET against what the
generator *would* produce from current DRY inputs — that is a staleness check,
owned by ConfigHub's publishing pipeline via `inputs.digest` comparison.

cub-scout's evidence feeds into the broader system: [cub-track](../05-rollout/10-cub-track.md)
can enrich mutation records with field-origin data, and ConfigHub can correlate
evidence with provenance to classify drift causes.

### Evidence Bundle Schema (v2 Proposed)

> **Migration note:** The current codebase uses `BundleSummary` (v1). The schema
> below is the **proposed v2** — it adds `observation.differences` and `provenance`
> fields. v1 remains valid; v2 is additive, behind a schema version flag.
>
> | v1 (current) | v2 (proposed) | Notes |
> |--------------|---------------|-------|
> | `BundleSummary` | `EvidenceBundle` | Rename + richer structure |
> | `summary` field | `summary.title` + `summary.severity` | Split for export routing |
> | N/A | `observation.differences[]` | Structured diff per resource/field |
> | N/A | `provenance.intended_operations` | Links to generator operations |

```yaml
apiVersion: confighub.io/v1
kind: EvidenceBundle

metadata:
  id: string
  created_at: string        # ISO 8601
  type: drift | verification | snapshot

subject:
  unit: string              # Unit name
  app: string               # App name
  deployment: string        # Deployment name
  cluster: string           # Target cluster observed

observation:
  intended: object          # What ConfigHub says should exist
  observed: object          # What cub-scout found
  differences:
    - resource: {apiVersion, kind, namespace, name}
      field: string         # JSONPath
      expected: any
      observed: any
      classification:
        type: added | removed | modified
        likely_cause: manual_edit | controller | unknown

provenance:
  intended_operations: []string
  intended_at: string
  intended_commit: string

summary:
  title: string
  severity: info | warning | critical
```

### Drift Response Policies

| Policy | Behavior |
|--------|----------|
| `alert` | Notify, don't change |
| `propose-revert` | Create PR to restore intended state |
| `propose-accept` | Create PR to accept observed state as new intent |
| `require-approval` | Create PR for human decision |

Policies can vary by label (`variant=dev` -> accept; `variant=prod` -> revert).

**ConfigHub is not a controller.** It does not reconcile. cub-scout observes,
ConfigHub records, and humans or policies decide. Evidence is the interface between
observation and decision.

---

## 3. Write-Back Semantics

Observed runtime changes **do not** overwrite intent. They produce explicit proposals:

1. cub-scout detects drift and produces evidence bundle
2. Evidence is exported (Slack, Jira, ConfigHub)
3. If policy says `propose-accept`: a merge request is created to update intended state
4. If policy says `propose-revert`: a merge request is created to restore intended state
5. Human or automation reviews and accepts/rejects

Write-backs from agents follow the same pattern. An agent changes an operational
overlay value, proposes it as a merge request, and the mutation is logged via
[cub-track](../05-rollout/10-cub-track.md) in both ConfigHub and Git (depending on scope).

### Overlay Edits: Transitional, Not Steady-State

Variant overlay edits on WET output do **not** require immediate write-back to
generator intent. But they are a **transitional escape hatch**, not a permanent
workflow.

The preferred path for per-variant changes is the generator's input surface —
per-environment values files, variant-specific context, or extended generator
input schemas. When a user needs a change that the generator doesn't support,
the first response should be to extend the generator input, not to overlay the
output.

If an overlay is applied to WET:

1. The system records it as a governed mutation ([cub-track](../05-rollout/10-cub-track.md) ChangeInteractionCard)
2. `cub-track suggest` flags the field's DRY origin, if one exists
3. The overlay is classified as "overlay drift from DRY source" — distinct from
   runtime drift
4. Promote the overlay to DRY intent/generator when the change becomes reusable,
   default-worthy, or long-lived

The system should create friction — evidence, staleness detection,
[cub-track](../05-rollout/10-cub-track.md) redirection — that encourages promotion back to DRY
rather than normalizing WET-space editing for generator-backed config.

---

## 4. Failure Modes

In each case, the failure is explicit, surfaced early, and non-destructive.

| Failure | How It Surfaces | Resolution |
|---------|----------------|------------|
| **Drift** | cub-scout evidence bundle | Policy-driven: alert, revert, accept, or require-approval |
| **Conflicting platform rules** | Deployment creation fails with explicit error | Platform teams coordinate; system surfaces conflicts, does not resolve them |
| **Generator bug** | Output reproducible via input digest; diff available | Fix generator, re-render; change is explicit and diffable |
| **Invalid operation** | Validation rejects before rendering | Fix inputs or obtain platform exception |
| **Manual runtime change** | Detected as drift on next observation | Depends on drift policy; never silently accepted |
| **Overlay drift from DRY** | WET field changed without `inputs.digest` change; [cub-track](../05-rollout/10-cub-track.md) classifies as overlay | Promote to DRY input or expire; `cub-track suggest` redirects to DRY source |
| **Stale render** | `inputs.digest` changed but WET not re-rendered; detected by ConfigHub publishing pipeline | Re-render from current DRY inputs; diff shows what changed |
| **Generator version mismatch** | Provenance shows old version; version pinning prevents silent upgrade | Explicit re-render; diff shows changes |

---

## 5. Cross-References

This document is part of the AI and GitOps v7 Document Set. Related documents:

| Document | Covers |
|----------|--------|
| [01 — Introducing Agentic GitOps](../01-vision/01-introducing-agentic-gitops.md) | Why agentic GitOps exists, foundational invariants, classical GitOps gaps |
| [02 — Generators PRD](../02-design/10-generators-prd.md) | Generator model, maturity levels, authoring landscape |
| [03 — Field-Origin Maps and Editing](../02-design/20-field-origin-maps-and-editing.md) | Field-origin maps, editing model, provenance tracking |
| [04 — App Model and Contracts](../02-design/30-app-model-and-contracts.md) | Entity definitions, operating boundary, constraints, staging model, operations |
| [05 — cub-track](../05-rollout/10-cub-track.md) | Git-native mutation ledger, ChangeInteractionCard, trust tiers, attestation, adoption stages |
| **06 — Governed Execution** (this document) | Two-loop model, evidence, write-back semantics, failure modes |
| [07 — User Experience](../05-rollout/30-user-experience.md) | Four surfaces, two personas, import flow, generator UX, AI tooling, skill definition |
| [08 — Adoption and Reference](../05-rollout/40-adoption-and-reference.md) | Adoption path, value analysis, pricing boundary, worked examples, FAQ |
