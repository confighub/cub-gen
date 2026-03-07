# Agentic GitOps Plan (ConfigHub-First)

**Status:** Execution plan
**Date:** 2026-03-04
**Audience:** Platform engineering, product, runtime engineering

## 1. Plan Summary

Build a ConfigHub-first Agentic GitOps control loop for teams running:

1. Kubernetes
2. GitOps (Flux/Argo)
3. Helm-heavy delivery
4. Internal platform abstractions (templates, app frameworks, Score-style, platform SDKs)

The system must let AI and humans propose changes quickly while preserving
explicit governance, provenance, and deterministic promotion into platform DRY.

## 2. Primary User and Job-to-be-Done

Primary user:

1. Platform engineer using an AI coding/ops tool.

Core job:

1. Import real repos into ConfigHub as DRY/WET units.
2. Evaluate and approve changes in ConfigHub MR.
3. Execute governed rollout through existing reconciler flow.
4. Promote reusable app changes into platform base DRY without drift.

## 3. Non-Negotiable Invariants

1. Nothing implicit ever deploys.
2. Nothing observed from live state silently overwrites DRY intent.
3. Merge approval and deploy approval remain separate controls.
4. One `change_id` links CH MR, Git PR, provenance, execution, and attestation.
5. Flux/Argo remain reconcilers; ConfigHub remains decision authority.
6. `Agentic GitOps` MUST include an active GitOps inner loop (`WET -> LIVE`
   reconciliation by Flux/Argo or equivalent); otherwise label the system
   `governed config automation`.

## 4. Scope and Boundaries

In scope:

1. Repo import to DRY/WET + generator contracts.
2. CH-first change workflow with optional Git mirror.
3. Helm-first generator support plus adapter model for custom platforms.
4. LIVE->proposal path (`LIVE -> CH MR`) with explicit accept/reject.
5. Upstream promotion from app overlay to platform base DRY.

Out of scope (this plan horizon):

1. Replacing Flux/Argo controllers.
2. Storing bulky runtime telemetry in Git.
3. Unreviewed automatic writes to platform main DRY.
4. Building a hosted analytics product before core workflow is stable.

Qualification note:

1. This plan describes Agentic GitOps only when an active reconciler loop is
   present.
2. The same governance model can run without Flux/Argo, but that mode is named
   `governed config automation` and is not presented as Agentic GitOps.

## 5. Target End-State Architecture

Control loop:

```text
Repo/LIVE input
-> import + detect generator
-> create/update DRY/WET units
-> CH MR proposal
-> render + policy evaluation
-> ALLOW|ESCALATE|BLOCK
-> tokened execution
-> verification + attestation
-> promotion to upstream platform DRY (when reusable)
```

Component responsibilities:

1. `cub-track`: local Git mutation linkage and explain/search.
2. `cub-scout`: evidence normalization and live/repo discovery.
3. `confighub-scan`: risk/policy signals.
4. `confighub`: CH MR, decisions, provenance graph, attestation authority.
5. `confighub-actions`: execution runtime with scoped token after allow.
6. Flux/Argo: unchanged reconciler role.

## 6. Data Model Needed for Plan Execution

Required objects:

1. `DryUnit`
2. `WetUnit`
3. `GeneratorUnit`
4. `ProvenanceRecord`
5. `FieldOriginMap`
6. `InverseTransformPlan`
7. `ChangeMR`
8. `DecisionReceipt`
9. `ExecutionReceipt`
10. `OutcomeReceipt`
11. `AttestationRecord`

Storage split:

1. Git: DRY collaboration and compact receipts.
2. OCI: WET transport to Flux/Argo.
3. ConfigHub: policy graph, approvals, telemetry, attestation, provenance joins.

## 7. Entry Paths (All Converge to CH MR)

1. `Git PR -> CH MR`
2. `CH MR -> Git PR`
3. `LIVE -> CH MR proposal`

Rule:

1. CH MR is the single governance object regardless of entry path.

## 8. Execution Plan (Phased)

### Phase 0 (2 weeks): Contract Freeze + Detection Baseline

Deliverables:

1. Freeze contract triple schemas (`generator`, `provenance`, `inverse`).
2. Implement generator detection output for Helm + Score + custom template repos.
3. Define machine-readable error envelope and exit codes for agent workflows.

Exit criteria:

1. Same repo analyzed twice yields deterministic detection output.
2. Unknown patterns reported explicitly, not silently ignored.

### Phase 1 (3-4 weeks): Import and Render Lineage

Deliverables:

1. Import pipeline creates `DryUnit/WetUnit/GeneratorUnit`.
2. Provenance written for each render (`generator.version`, `inputs.digest`, artifact digests).
3. Initial field-origin map for common fields (replicas, image, resources, ports, env).

Exit criteria:

1. Imported app can answer "what produced this WET and when".
2. Stale-render detection works from digest mismatch.

### Phase 2 (3-4 weeks): CH MR Governance Loop

Deliverables:

1. CH-first proposal workflow with optional paired Git PR.
2. Decision gate with `ALLOW|ESCALATE|BLOCK`.
3. Execution receipts + attestation links under one `change_id`.

Exit criteria:

1. Merge approval and deploy decision are visibly separate.
2. No execution without allow path.

### Phase 3 (3 weeks): Inverse and LIVE Proposal Path

Deliverables:

1. `WET/live -> DRY` inverse proposal generation.
2. LIVE-origin drift creates proposal MR, never direct source overwrite.
3. Confidence scoring + mandatory review for low-confidence inversions.

Exit criteria:

1. Live drift can be converted into governed DRY proposal safely.
2. Zero silent live->DRY overwrites.

### Phase 4 (2-3 weeks): Upstream Promotion Automation

Deliverables:

1. Reuse scoring and promotion suggestions.
2. Auto-open promotion PR/MR to platform base DRY when reusable.
3. Overlay cleanup suggestions after upstream merge.

Exit criteria:

1. Reusable app changes are converged upstream with auditable flow.
2. Overlay drift trend declines release-over-release.

## 9. MVP Adapter Priorities

Priority order:

1. Helm (must-have)
2. Score.dev (must-have)
3. Spring Boot config adapter (must-have for platform teams with Java estates)
4. Generic template/custom platform adapter SDK (must-have)

Definition of done for any adapter:

1. Emits generator contract.
2. Emits provenance record.
3. Emits field-origin coverage for common operational fields.
4. Emits inverse mapping entries or explicit non-reversible markers.

## 10. Operational KPIs

1. Time-to-first-governed-import: under 30 minutes.
2. Explain latency (`change_id` to answer): under 2 seconds typical.
3. Origin-map coverage for critical fields: >= 90%.
4. Live proposal safety incidents: 0 silent overwrites.
5. Overlay convergence: measurable reduction in long-lived app overlays.

## 11. Risks and Mitigations

1. Adapter inconsistency.
   Mitigation: schema conformance tests + golden fixtures per adapter.
2. Over-automation risk in promotion.
   Mitigation: promotion is suggestion/open-PR, never direct merge.
3. User confusion around CH vs Git authority.
   Mitigation: explicit UI wording and single `change_id` tracking.
4. Evidence data sensitivity.
   Mitigation: redaction defaults + retention policy + compact Git write-back.

## 12. Immediate Next Actions (Two-Week Sprint)

1. Lock the three contract schemas as v1.
2. Build Helm-first import path with provenance digest.
3. Ship CH MR skeleton with decision stub (`PENDING/ALLOW/BLOCK`).
4. Produce one end-to-end demo repo:
   Helm chart + app overlay + promotion to platform base.

## 13. Related Docs

1. `50-dual-approval-gitops-gh-pr-and-ch-mr.md`
2. `60-stored-in-git-vs-confighub.md`
3. `../03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md`
4. `../03-worked-examples/02-traefik-helm-dry-wet-unit-worked-example.md`
5. `../04-schemas/00-gitops-checkpoint-schemas.md`
