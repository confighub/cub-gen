# Agentic GitOps Design (ConfigHub-First)

**Status:** Draft design spec  
**Date:** 2026-03-04  
**Audience:** Product, platform engineering, runtime engineering, and GTM

## 1. One-Line Definition

Agentic GitOps is a ConfigHub-first control loop where AI-assisted mutations are
imported, evaluated, approved, executed, and attested as governed change
transactions across DRY and WET layers.

Qualification rule: this term applies only when an active GitOps reconciler
loop (`WET -> LIVE`) is present (Flux/Argo or equivalent). Without that loop,
use `governed config automation`.

## 2. Design Center (Primary User)

Primary user is a platform engineer using an AI tool.

Design optimizes for:

1. AI-generated proposals at high volume.
2. Deterministic machine-readable outputs (`--json`, stable schemas, stable IDs).
3. Fast explainability across repo, runtime, and policy layers.
4. Clear human decision points at merge and deploy gates.

## 3. Problem Statement

Classical GitOps gives good reconciliation, but weak governance context for AI-era
operations.

Teams need to answer:

1. What intent was proposed?
2. Which generator produced this WET artifact?
3. Which DRY field owns this deployed field?
4. Why was execution allowed or blocked?
5. What outcome and attestation were recorded?

## 4. Non-Negotiable Invariants

1. Nothing implicit ever deploys.
2. Nothing observed in live silently overwrites DRY intent.
3. Merge approval and deploy approval are separate controls.
4. One canonical `change_id` joins all records across CH, Git, OCI, and runtime.
5. Flux/Argo remain reconcilers; ConfigHub remains governance authority.
6. Protected DRY write-back is PR/MR-only.

## 5. What We Are Building

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

## 6. Platform Boundary (Component Responsibilities)

1. `cub-track`: local Git-native mutation linkage and explain/search.
2. `cub-scout`: GitOps explorer and evidence normalizer.
3. `confighub-scan`: risk and policy signal engine.
4. `confighub`: decision and attestation authority; CH MR orchestration.
5. `confighub-actions`: tokened execution runtime after `ALLOW`.
6. Flux/Argo: reconciliation engines only.

## 7. Core Control Loop

```text
import -> detect generator -> create DRY/WET units -> propose change
-> render/evaluate -> approve -> ALLOW|ESCALATE|BLOCK -> execute
-> verify/attest -> promote reusable DRY upstream
```

## 8. Entry Paths (All Converge to CH MR)

1. `Git PR -> CH MR`
2. `CH MR -> Git PR`
3. `LIVE observation -> CH MR proposal`

All paths converge to one governed object: `ChangeMR` in ConfigHub.

## 9. Contract Triple (Required Per Generator)

Every adapter/imported generator must provide three contracts.

### 9.1 Generator Contract (`DRY -> WET`)

Defines:

1. input types and schema refs,
2. generator/adapter version,
3. deterministic render behavior and constraints,
4. WET output artifact shape and transport (`OCI` default, Git optional).
5. signed contract and deterministic output hash.

### 9.2 Provenance Schema

Defines:

1. immutable input hash/digest,
2. source artifacts and revisions,
3. generator identity and version,
4. toolchain version, policy version, run ID,
5. output artifact digest,
6. controller target and merge links.

### 9.3 Inverse Transformer Schema (`WET/live -> DRY`)

Defines:

1. allowed reverse mappings (WET path -> DRY path),
2. patch operations and safety constraints,
3. ownership (`app-team`, `platform-team`, `read-only`),
4. confidence and review requirements.
5. replay-check mismatch escalation (`ESCALATE`) and out-of-scope auto-`BLOCK`.

## 10. ConfigHub Object Model (Logical)

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

## 11. Storage and Transport Model

1. Git is DRY collaboration ingress and review surface.
2. OCI is default WET transport for Flux/Argo.
3. ConfigHub stores WET governance graph, policy traces, approvals, telemetry.
4. Git write-back contains compact DRY receipts and stable digests only.

## 12. Agent-Optimized Command Profile (Proposed)

Note: names can map to existing `discover/import` commands during rollout.

### 12.1 Import and detect

1. `cub gitops import --repo <url> --ref <sha|branch> --space <space> --json`
2. `cub gitops detect --repo <url> --ref <sha|branch> --json`

### 12.2 Proposal lifecycle

1. `cub gitops propose --change-id <id> --from <dry|wet|live> --json`
2. `cub gitops evaluate --change-id <id> --json`
3. `cub gitops promote --change-id <id> --upstream <base-unit> --json`

### 12.3 Explainability

1. `cub gitops explain --change-id <id> --json`
2. `cub gitops origin --wet-path <path> --change-id <id> --json`
3. `cub gitops inverse-plan --change-id <id> --json`

Command requirements:

1. idempotent execution for agent retries,
2. deterministic exit codes,
3. stable machine-readable error envelopes,
4. no interactive prompts unless explicitly enabled.

## 13. API Surface (Minimum)

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

## 14. Trust Tier and Governance

1. Tier 0: observe only.
2. Tier 1: low-risk apply domains.
3. Tier 2: medium-risk with human approval.
4. Tier 3: high-risk/prod with strongest attestation and dual approval.

Decision semantics:

1. `ALLOW` permits token issuance and execution.
2. `ESCALATE` requires explicit approver action.
3. `BLOCK` forbids execution until change is updated.
4. `ALLOW` requires attestation linkage (actor + evidence + decision).

## 15. Promotion Model (App -> Platform)

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

## 16. LIVE-Origin (Kargo-Style) Integration

1. live drift is ingested as evidence, not source-of-truth.
2. ConfigHub creates proposal MR from live evidence with explicit drift class.
3. accepted proposal is converted into DRY patch and follows normal governance path.
4. rejected proposal triggers revert/remediation path.

## 17. Adapter Requirements (Must/Should)

Must:

1. emit generator contract metadata,
2. emit provenance tuple (`generator`, `version`, `input_digest`, `output_digest`),
3. emit field-origin coverage for critical runtime fields,
4. emit inverse-plan entries or explicit non-reversible markers.

Should:

1. classify risk hints by target environment,
2. include ownership hints (`app` vs `platform`) for mapped fields,
3. expose deterministic dry-run mode.

## 18. MVP Scope (Phase Plan)

### Phase 0: Contracts and read path

1. freeze contract triple schema set,
2. implement import detect/analyze path,
3. expose explain/origin read APIs.

### Phase 1: Repo import and render linkage

1. support Helm, Score.dev, Spring Boot adapters,
2. create DRY/WET/Generator units on import,
3. persist provenance records for each render.

### Phase 2: CH MR governance loop

1. converge all entry paths to `ChangeMR`,
2. support decision gates and receipt generation,
3. integrate tokened execution + attestation.

### Phase 3: Inverse transform and live proposals

1. enable `WET/live -> DRY` proposal generation,
2. enforce explicit accept/reject workflows,
3. prevent silent write-backs from live evidence.

### Phase 4: Upstream promotion automation

1. auto-open promotion PR/MR when reuse score is high,
2. require upstream approvals,
3. auto-suggest overlay cleanup.

## 19. Success Metrics

1. Time to first governed import: under 30 minutes for a new repo.
2. Explainability SLA: `explain` returns in under 2 seconds for typical change.
3. Origin coverage: critical runtime fields mapped to DRY source at >= 90%.
4. Safe live intake: 0 silent live-to-DRY overwrites.
5. Promotion reuse: measurable reduction in long-lived app overlays.

## 20. Risks and Mitigations

1. Adapter quality variance.
   Mitigation: contract conformance tests + required fields.
2. False confidence in inverse transforms.
   Mitigation: confidence scoring + mandatory review for low confidence.
3. Workflow confusion between CH and Git.
   Mitigation: single `change_id` and explicit authority boundary.
4. Sensitive data leakage in evidence.
   Mitigation: redaction, retention controls, and write-back minimization.

## 21. Related Specs

1. `docs/reference/dual-approval-gitops-gh-pr-and-ch-mr.md`
2. `docs/reference/next-gen-gitops-ai-era.md`
3. `docs/reference/stored-in-git-vs-confighub.md`
4. `docs/reference/gitops-checkpoint-prd.md`
5. `docs/reference/gitops-checkpoint-schemas.md`
6. `docs/reference/scoredev-dry-wet-unit-worked-example.md`
7. `docs/reference/traefik-helm-dry-wet-unit-worked-example.md`
