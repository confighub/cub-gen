# Agentic GitOps Design and Execution Plan (ConfigHub-First)

**Status:** Combined reference  
**Date:** 2026-03-04  
**Audience:** Product, platform engineering, runtime engineering, and GTM

This document intentionally combines the numbered execution-plan variant and the
unnumbered design-spec variant so both entry points stay in sync while we keep
one canonical body of thinking.

---

## Part 1. Execution Plan

Source: former `docs/agentic-gitops/02-design/00-agentic-gitops-design.md`

### Agentic GitOps Plan (ConfigHub-First)

**Status:** Execution plan
**Date:** 2026-03-04
**Audience:** Platform engineering, product, runtime engineering

#### 1. Plan Summary

Build a ConfigHub-first Agentic GitOps control loop for teams running:

1. Kubernetes
2. GitOps (Flux/Argo)
3. Helm-heavy delivery
4. Internal platform abstractions (templates, app frameworks, Score-style, platform SDKs)

The system must let AI and humans propose changes quickly while preserving
explicit governance, provenance, and deterministic promotion into platform DRY.

#### 2. Primary User and Job-to-be-Done

Primary user:

1. Platform engineer using an AI coding/ops tool.

Core job:

1. Import real repos into ConfigHub as DRY/WET units.
2. Evaluate and approve changes in ConfigHub MR.
3. Execute governed rollout through existing reconciler flow.
4. Promote reusable app changes into platform base DRY without drift.

#### 3. Non-Negotiable Invariants

1. Nothing implicit ever deploys.
2. Nothing observed from live state silently overwrites DRY intent.
3. Merge approval and deploy approval remain separate controls.
4. One `change_id` links CH MR, Git PR, provenance, execution, and attestation.
5. Flux/Argo remain reconcilers; ConfigHub remains decision authority.
6. `Agentic GitOps` MUST include an active GitOps inner loop (`WET -> LIVE`
   reconciliation by Flux/Argo or equivalent); otherwise label the system
   `governed config automation`.

#### 4. Scope and Boundaries

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

#### 5. Target End-State Architecture

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

#### 6. Data Model Needed for Plan Execution

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

#### 7. Entry Paths (All Converge to CH MR)

1. `Git PR -> CH MR`
2. `CH MR -> Git PR`
3. `LIVE -> CH MR proposal`

Rule:

1. CH MR is the single governance object regardless of entry path.

#### 8. Execution Plan (Phased)

##### Phase 0 (2 weeks): Contract Freeze + Detection Baseline

Deliverables:

1. Freeze contract triple schemas (`generator`, `provenance`, `inverse`).
2. Implement generator detection output for Helm + Score + custom template repos.
3. Define machine-readable error envelope and exit codes for agent workflows.

Exit criteria:

1. Same repo analyzed twice yields deterministic detection output.
2. Unknown patterns reported explicitly, not silently ignored.

##### Phase 1 (3-4 weeks): Import and Render Lineage

Deliverables:

1. Import pipeline creates `DryUnit/WetUnit/GeneratorUnit`.
2. Provenance written for each render (`generator.version`, `inputs.digest`, artifact digests).
3. Initial field-origin map for common fields (replicas, image, resources, ports, env).

Exit criteria:

1. Imported app can answer "what produced this WET and when".
2. Stale-render detection works from digest mismatch.

##### Phase 2 (3-4 weeks): CH MR Governance Loop

Deliverables:

1. CH-first proposal workflow with optional paired Git PR.
2. Decision gate with `ALLOW|ESCALATE|BLOCK`.
3. Execution receipts + attestation links under one `change_id`.

Exit criteria:

1. Merge approval and deploy decision are visibly separate.
2. No execution without allow path.

##### Phase 3 (3 weeks): Inverse and LIVE Proposal Path

Deliverables:

1. `WET/live -> DRY` inverse proposal generation.
2. LIVE-origin drift creates proposal MR, never direct source overwrite.
3. Confidence scoring + mandatory review for low-confidence inversions.

Exit criteria:

1. Live drift can be converted into governed DRY proposal safely.
2. Zero silent live->DRY overwrites.

##### Phase 4 (2-3 weeks): Upstream Promotion Automation

Deliverables:

1. Reuse scoring and promotion suggestions.
2. Auto-open promotion PR/MR to platform base DRY when reusable.
3. Overlay cleanup suggestions after upstream merge.

Exit criteria:

1. Reusable app changes are converged upstream with auditable flow.
2. Overlay drift trend declines release-over-release.

#### 9. MVP Adapter Priorities

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

#### 10. Operational KPIs

1. Time-to-first-governed-import: under 30 minutes.
2. Explain latency (`change_id` to answer): under 2 seconds typical.
3. Origin-map coverage for critical fields: >= 90%.
4. Live proposal safety incidents: 0 silent overwrites.
5. Overlay convergence: measurable reduction in long-lived app overlays.

#### 11. Risks and Mitigations

1. Adapter inconsistency.
   Mitigation: schema conformance tests + golden fixtures per adapter.
2. Over-automation risk in promotion.
   Mitigation: promotion is suggestion/open-PR, never direct merge.
3. User confusion around CH vs Git authority.
   Mitigation: explicit UI wording and single `change_id` tracking.
4. Evidence data sensitivity.
   Mitigation: redaction defaults + retention policy + compact Git write-back.

#### 12. Immediate Next Actions (Two-Week Sprint)

1. Lock the three contract schemas as v1.
2. Build Helm-first import path with provenance digest.
3. Ship CH MR skeleton with decision stub (`PENDING/ALLOW/BLOCK`).
4. Produce one end-to-end demo repo:
   Helm chart + app overlay + promotion to platform base.

#### 13. Related Docs

1. `50-dual-approval-gitops-gh-pr-and-ch-mr.md`
2. `60-stored-in-git-vs-confighub.md`
3. `../03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md`
4. `../03-worked-examples/02-traefik-helm-dry-wet-unit-worked-example.md`
5. `../04-schemas/00-gitops-checkpoint-schemas.md`


---

## Part 2. Design Reference

Source: former `docs/agentic-gitops/agentic-gitops-design.md`

### Agentic GitOps Design (ConfigHub-First)

**Status:** Draft design spec  
**Date:** 2026-03-04  
**Audience:** Product, platform engineering, runtime engineering, and GTM

#### 1. One-Line Definition

Agentic GitOps is a ConfigHub-first control loop where AI-assisted mutations are
imported, evaluated, approved, executed, and attested as governed change
transactions across DRY and WET layers.

Qualification rule: this term applies only when an active GitOps reconciler
loop (`WET -> LIVE`) is present (Flux/Argo or equivalent). Without that loop,
use `governed config automation`.

#### 2. Design Center (Primary User)

Primary user is a platform engineer using an AI tool.

Design optimizes for:

1. AI-generated proposals at high volume.
2. Deterministic machine-readable outputs (`--json`, stable schemas, stable IDs).
3. Fast explainability across repo, runtime, and policy layers.
4. Clear human decision points at merge and deploy gates.

#### 3. Problem Statement

Classical GitOps gives good reconciliation, but weak governance context for AI-era
operations.

Teams need to answer:

1. What intent was proposed?
2. Which generator produced this WET artifact?
3. Which DRY field owns this deployed field?
4. Why was execution allowed or blocked?
5. What outcome and attestation were recorded?

#### 4. Non-Negotiable Invariants

1. Nothing implicit ever deploys.
2. Nothing observed in live silently overwrites DRY intent.
3. Merge approval and deploy approval are separate controls.
4. One canonical `change_id` joins all records across CH, Git, OCI, and runtime.
5. Flux/Argo remain reconcilers; ConfigHub remains governance authority.
6. Protected DRY write-back is PR/MR-only.

#### 5. What We Are Building

Build a generator-native import and governance engine in ConfigHub that can point
at app repos and convert them into managed units and contracts.

Input repositories can contain:

1. templated app configs,
2. Spring Boot app configs,
3. Score.dev workloads,
4. Helm charts and values,
5. mixed platform/app GitOps layouts.

Each imported system is normalized into:

1. DRY units (authoring intent),
2. WET units (deployment contracts),
3. generator contracts (`DRY -> WET`),
4. provenance records,
5. inverse transform plans (`WET/live -> DRY proposal`).

#### 6. Platform Boundary (Component Responsibilities)

1. `cub-track`: local Git-native mutation linkage and explain/search.
2. `cub-scout`: GitOps explorer and evidence normalizer.
3. `confighub-scan`: risk and policy signal engine.
4. `confighub`: decision and attestation authority; CH MR orchestration.
5. `confighub-actions`: tokened execution runtime after `ALLOW`.
6. Flux/Argo: reconciliation engines only.

#### 7. Core Control Loop

```text
import -> detect generator -> create DRY/WET units -> propose change
-> render/evaluate -> approve -> ALLOW|ESCALATE|BLOCK -> execute
-> verify/attest -> promote reusable DRY upstream
```

#### 8. Entry Paths (All Converge to CH MR)

1. `Git PR -> CH MR`
2. `CH MR -> Git PR`
3. `LIVE observation -> CH MR proposal`

All paths converge to one governed object: `ChangeMR` in ConfigHub.

#### 9. Contract Triple (Required Per Generator)

Every adapter/imported generator must provide three contracts.

##### 9.1 Generator Contract (`DRY -> WET`)

Defines:

1. input types and schema refs,
2. generator/adapter version,
3. deterministic render behavior and constraints,
4. WET output artifact shape and transport (`OCI` default, Git optional).
5. signed contract and deterministic output hash.

##### 9.2 Provenance Schema

Defines:

1. immutable input hash/digest,
2. source artifacts and revisions,
3. generator identity and version,
4. toolchain version, policy version, run ID,
5. output artifact digest,
6. controller target and merge links.

##### 9.3 Inverse Transformer Schema (`WET/live -> DRY`)

Defines:

1. allowed reverse mappings (WET path -> DRY path),
2. patch operations and safety constraints,
3. ownership (`app-team`, `platform-team`, `read-only`),
4. confidence and review requirements.
5. replay-check mismatch escalation (`ESCALATE`) and out-of-scope auto-`BLOCK`.

#### 10. ConfigHub Object Model (Logical)

Core entities:

1. `DryUnit`
2. `WetUnit`
3. `GeneratorUnit`
4. `ProvenanceRecord`
5. `FieldOriginMap`
6. `InversePatchPlan`
7. `ChangeMR`
8. `DecisionReceipt`
9. `ExecutionReceipt`
10. `OutcomeReceipt`
11. `AttestationRecord`

Relationship rules:

1. `GeneratorUnit` links one or more `DryUnit` revisions to one `WetUnit` revision.
2. `ProvenanceRecord` is mandatory for each rendered `WetUnit` revision.
3. `FieldOriginMap` references generator + dry/wet revisions.
4. `InversePatchPlan` references a concrete source (`WET` diff or `LIVE` observation).
5. `ChangeMR` links Git PRs, CH MR, receipts, and attestation by `change_id`.

#### 11. Storage and Transport Model

1. Git is DRY collaboration ingress and review surface.
2. OCI is default WET transport for Flux/Argo.
3. ConfigHub stores WET governance graph, policy traces, approvals, telemetry.
4. Git write-back contains compact DRY receipts and stable digests only.

#### 12. Agent-Optimized Command Profile (Proposed)

Note: names can map to existing `discover/import` commands during rollout.

##### 12.1 Import and detect

1. `cub gitops import --repo <url> --ref <sha|branch> --space <space> --json`
2. `cub gitops detect --repo <url> --ref <sha|branch> --json`

##### 12.2 Proposal lifecycle

1. `cub gitops propose --change-id <id> --from <dry|wet|live> --json`
2. `cub gitops evaluate --change-id <id> --json`
3. `cub gitops promote --change-id <id> --upstream <base-unit> --json`

##### 12.3 Explainability

1. `cub gitops explain --change-id <id> --json`
2. `cub gitops origin --wet-path <path> --change-id <id> --json`
3. `cub gitops inverse-plan --change-id <id> --json`

Command requirements:

1. idempotent execution for agent retries,
2. deterministic exit codes,
3. stable machine-readable error envelopes,
4. no interactive prompts unless explicitly enabled.

#### 13. API Surface (Minimum)

1. `POST /v1/imports`
2. `POST /v1/imports/{import_id}/analyze`
3. `POST /v1/changes/upsert`
4. `POST /v1/changes/{change_id}/evaluate`
5. `POST /v1/changes/{change_id}/decision`
6. `POST /v1/changes/{change_id}/execute`
7. `POST /v1/changes/{change_id}/promote`
8. `GET /v1/changes/{change_id}`
9. `GET /v1/changes/{change_id}/origin-map`
10. `GET /v1/changes/{change_id}/inverse-plan`

#### 14. Trust Tier and Governance

1. Tier 0: observe only.
2. Tier 1: low-risk apply domains.
3. Tier 2: medium-risk with human approval.
4. Tier 3: high-risk/prod with strongest attestation and dual approval.

Decision semantics:

1. `ALLOW` permits token issuance and execution.
2. `ESCALATE` requires explicit approver action.
3. `BLOCK` forbids execution until change is updated.
4. `ALLOW` requires attestation linkage (actor + evidence + decision).

#### 15. Promotion Model (App -> Platform)

Default promotion path:

1. app team changes bounded app DRY,
2. ConfigHub renders/evaluates and opens/updates CH MR (+ paired Git PR if enabled),
3. platform engineer approves app change in CH,
4. governed execution runs on allow path,
5. if reusable, ConfigHub opens promotion PR/MR to upstream platform DRY/base unit,
6. after upstream approvals, ConfigHub merges Git PR,
7. app overlay is reduced/removed to prevent long-term drift.

Guardrail:

1. never auto-write to platform main DRY without separate upstream review/merge.

#### 16. LIVE-Origin (Kargo-Style) Integration

1. live drift is ingested as evidence, not source-of-truth.
2. ConfigHub creates proposal MR from live evidence with explicit drift class.
3. accepted proposal is converted into DRY patch and follows normal governance path.
4. rejected proposal triggers revert/remediation path.

#### 17. Adapter Requirements (Must/Should)

Must:

1. emit generator contract metadata,
2. emit provenance tuple (`generator`, `version`, `input_digest`, `output_digest`),
3. emit field-origin coverage for critical runtime fields,
4. emit inverse-plan entries or explicit non-reversible markers.

Should:

1. classify risk hints by target environment,
2. include ownership hints (`app` vs `platform`) for mapped fields,
3. expose deterministic dry-run mode.

#### 18. MVP Scope (Phase Plan)

##### Phase 0: Contracts and read path

1. freeze contract triple schema set,
2. implement import detect/analyze path,
3. expose explain/origin read APIs.

##### Phase 1: Repo import and render linkage

1. support Helm, Score.dev, Spring Boot adapters,
2. create DRY/WET/Generator units on import,
3. persist provenance records for each render.

##### Phase 2: CH MR governance loop

1. converge all entry paths to `ChangeMR`,
2. support decision gates and receipt generation,
3. integrate tokened execution + attestation.

##### Phase 3: Inverse transform and live proposals

1. enable `WET/live -> DRY` proposal generation,
2. enforce explicit accept/reject workflows,
3. prevent silent write-backs from live evidence.

##### Phase 4: Upstream promotion automation

1. auto-open promotion PR/MR when reuse score is high,
2. require upstream approvals,
3. auto-suggest overlay cleanup.

#### 19. Success Metrics

1. Time to first governed import: under 30 minutes for a new repo.
2. Explainability SLA: `explain` returns in under 2 seconds for typical change.
3. Origin coverage: critical runtime fields mapped to DRY source at >= 90%.
4. Safe live intake: 0 silent live-to-DRY overwrites.
5. Promotion reuse: measurable reduction in long-lived app overlays.

#### 20. Risks and Mitigations

1. Adapter quality variance.
   Mitigation: contract conformance tests + required fields.
2. False confidence in inverse transforms.
   Mitigation: confidence scoring + mandatory review for low confidence.
3. Workflow confusion between CH and Git.
   Mitigation: single `change_id` and explicit authority boundary.
4. Sensitive data leakage in evidence.
   Mitigation: redaction, retention controls, and write-back minimization.

#### 21. Related Specs

1. `docs/agentic-gitops/02-design/50-dual-approval-gitops-gh-pr-and-ch-mr.md`
2. `docs/agentic-gitops/01-vision/02-next-gen-gitops-ai-era.md`
3. `docs/agentic-gitops/02-design/60-stored-in-git-vs-confighub.md`
4. `docs/agentic-gitops/gitops-checkpoint-prd.md`
5. `docs/agentic-gitops/04-schemas/00-gitops-checkpoint-schemas.md`
6. `docs/agentic-gitops/03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md`
7. `docs/agentic-gitops/03-worked-examples/02-traefik-helm-dry-wet-unit-worked-example.md`

