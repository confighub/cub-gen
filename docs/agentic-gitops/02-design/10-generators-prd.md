# ConfigHub Generators PRD

**Status:** Draft for review
**Date:** 2026-03-04
**Owner:** ConfigHub Product + Platform Engineering
**Audience:** Platform teams and broader Git users (app teams, ops teams, IDP teams, AI-assisted developers) authoring deploy/runtime config through Git-centric workflows

## 1. Executive Summary

ConfigHub Generators turn heterogeneous app/platform source repos into governed,
traceable deployment contracts.

The product must let teams:

1. import existing Helm, Score.dev, Spring Boot, and custom platform patterns,
2. convert them into DRY/WET + generator contracts,
3. run AI-assisted changes through ConfigHub MR governance,
4. promote reusable app changes into platform base DRY safely.

This is an adoption bridge, not a reconciler replacement. Flux/Argo stay in place.

Tooling boundary for this PRD:

1. Primary command surface in this project is `cub-gen ...`.
2. Future promotion targets are `cub gen ...` and/or `cub gitops ...`.
3. Command parity is required when promotion happens (`cub-gen import` maps to promoted surfaces).

## 2. Problem

Teams can already render manifests. They cannot reliably answer:

1. what generator version produced this WET,
2. which DRY field controls this deployed field,
3. whether live drift should be accepted or reverted,
4. how to promote app-local change into shared platform defaults without drift.

AI increases mutation rate and multiplies this problem.

## 3. Product Thesis

Every imported generator must provide a mandatory contract triple:

1. `GeneratorContract` (`DRY -> WET` behavior)
2. `ProvenanceRecord` (lineage and digest evidence)
3. `InverseTransformPlan` (`WET/live -> DRY` proposal plan)

No contract triple, no governed import.

## 4. Target Users and Jobs

### Primary user

1. Platform engineer using AI assistants.

### Secondary users

1. App engineers consuming platform defaults and overlays.
2. SRE/on-call teams debugging rollout and drift.
3. Git-centric teams using templates/framework configs/app-config repos.
4. IDP/portal teams providing self-service over Git-authored intent.

### Core jobs to be done

1. "Import this repo and make it governable."
2. "Explain what generated this deployment."
3. "Convert this live drift into a safe proposal."
4. "Promote this reusable app change to platform base DRY."
5. "Get first value without changing existing Git + Flux/Argo flow."

### User stories (adoption-first)

1. As a Helm/Kustomize Git user, I can run `cub-gen detect/import` on an existing repo and immediately query provenance without refactoring my repo.
2. As a Spring Boot team, I can keep framework config as DRY input and get explicit WET manifests with field-origin mapping for critical ops fields.
3. As a Score.dev user, I can preserve my workload abstraction and still get governed dry/wet lineage in ConfigHub.
4. As an app-config platform user (for example feature-flag/runtime config repos), I can import literal WET config into the same governance model.
5. As an ops engineer, I can trace a production config field to source intent quickly enough for incident response.
6. As a platform lead, I can onboard mixed Git teams with a single cognitive-simple path: detect -> import -> explain -> evaluate.
7. As a CI-centric platform team, I can call ConfigHub mutation/query APIs from existing pipelines and avoid repo/file-path scripting.
8. As a platform owner, I can evolve labels and taxonomy (app/service/target) without repo surgery and without breaking operational queries.
9. As an SRE, I can run a governed multi-repo change wave (for example CVE patch or image policy update) with per-target decisions and evidence.
10. As a security lead, I can prove that branch protections, required approvals, and signed-commit controls were preserved during ConfigHub write-back.
11. As an on-call engineer, I can convert a LIVE break-glass change into an explicit proposal (`accept` or `revert`) instead of silent overwrite.
12. As a compliance reviewer, I can see human, CI bot, and AI-agent mutations in one decision/attestation model keyed by one `change_id`.
13. As a regulated platform team, I can run the same model on-prem/air-gapped with exportable evidence and no SaaS dependency.

## 5. Invariants

1. Nothing implicit ever deploys.
2. Nothing observed from live silently overwrites DRY.
3. Merge approval and deploy decision are separate.
4. One `change_id` links CH/Git/OCI/runtime artifacts.
5. Flux/Argo remain reconcilers; ConfigHub is decision authority.
6. `Agentic GitOps` **MUST** include an active inner reconciliation loop (`WET -> LIVE`) by Flux/Argo (or equivalent). If that loop is absent, classify the system as `governed config automation`.

### Enforcement model (mandatory)

The following are hard requirements, not guidance:

1. `GeneratorContract` must be signed and include deterministic output hash.
2. `ProvenanceRecord` must include immutable `input_hash`, `toolchain_version`, `policy_version`, `run_id`, and artifact digests.
3. `InverseTransformPlan` may only write fields/paths declared in `OwnershipMap`.
4. Any out-of-scope write is auto-`BLOCK`.
5. Replay check is required; hash mismatch is auto-`ESCALATE`.
6. Policy gate is required before write-back: `ALLOW | ESCALATE | BLOCK`.
7. `ALLOW` requires attestation (actor, evidence bundle, decision record).
8. Protected DRY write-back is PR/MR-only; no silent direct mutation.
9. Verification failure downgrades to read-only evidence mode.
10. Every decision and mutation appends to the mutation ledger.

## 6. Scope

### In scope (MVP)

Baseline environment assumption for demos/examples in this PRD: users already have
Kubernetes, a GitOps reconciler (Flux or Argo), Git, and OCI transport; Helm is
optional.

1. Helm-first import and governance.
2. Score.dev and Spring Boot adapters.
3. Custom adapter SDK for internal platform generators.
4. CH MR-centric flow with optional paired Git PR.
5. Provenance + origin maps + inverse proposal plans.
6. LIVE -> CH MR proposal flow.
7. Promotion workflow from app overlay to platform base DRY.
8. Adoption-first UX: first value in one session without workflow migration.
9. Broad Git ingress: detection taxonomy covers templates, framework generators, workload abstractions, app-config imports, ops/action manifests, and AI-authored changes (with deterministic adapters implemented per MVP scope).
10. Qualification gate in demos/docs: if no active GitOps reconciler loop exists, present the flow as `governed config automation`, not Agentic GitOps.

### Out of scope (MVP)

1. Replacing Flux/Argo.
2. Storing full high-volume runtime telemetry in Git.
3. Auto-merging platform base DRY writes.
4. Full enterprise reporting suite before workflow hardening.
5. Non-deterministic AI generator execution model (explicitly deferred gap).

## 7. Product Flows

### Flow A: Repo import

1. User points ConfigHub/cub at repo + ref.
2. System detects generator patterns.
3. System creates/updates DRY/WET/Generator units.
4. System records provenance and initial origin map coverage.

### Flow B: CH-first change

1. User/agent proposes DRY change in CH MR.
2. System renders/evaluates and posts evidence.
3. Human approves merge in CH (and paired Git flow if enabled).
4. Decision gate enforces `ALLOW|ESCALATE|BLOCK`.
5. On allow path, execution runs and attestation is linked.

### Flow C: LIVE-origin proposal (kargo-style)

1. Observer detects live drift.
2. ConfigHub creates proposal MR with drift class + inverse patch plan.
3. Reviewer accepts/rejects; no silent overwrite.
4. Accepted proposal follows standard CH flow.

### Flow D: Upstream promotion

1. Reusable app change detected.
2. ConfigHub opens promotion PR/MR to platform base DRY.
3. Upstream approvals complete.
4. ConfigHub merges Git PR and suggests overlay cleanup.

## 8. Functional Requirements

### FR1: Deterministic detection

System must emit deterministic generator detection results for the same repo/ref.

Acceptance:

1. Stable output ordering and IDs.
2. Explicit `unknown` classification when undecidable.

### FR2: Multi-generator import

System must support repos containing multiple generator styles.

Acceptance:

1. Each detected generator has separate contract metadata.
2. Unit linkage preserves per-app and per-env boundaries.

### FR3: Mandatory provenance fields

Per render, these fields are required:

1. `generator.name`
2. `generator.version`
3. `inputs.digest`
4. `rendered.at`
5. output artifact digest

Acceptance:

1. Missing required field blocks governed import/update.

### FR4: Field-origin maps

System must map critical WET paths to DRY source paths + ownership.

Acceptance:

1. MVP critical fields: replicas, image, resources, ports, probes, env.
2. Ownership tag required: `app-team|platform-team|read-only`.

### FR5: Inverse transform plans

System must generate proposal-only DRY patch plans from WET/live deltas.

Acceptance:

1. Every patch has confidence score and review flag.
2. Low-confidence patches require explicit review.

### FR6: CH MR as governance object

All entry paths must converge to CH MR.

Acceptance:

1. `Git PR -> CH MR`, `CH MR -> Git PR`, `LIVE -> CH MR` all supported.
2. Same `change_id` across linked records.

### FR7: Decision gate enforcement

Execution must be blocked unless decision path permits it.

Acceptance:

1. `ALLOW`: execute.
2. `ESCALATE`: wait for approval.
3. `BLOCK`: no execution.

### FR8: Promotion workflow

Reusable app change must be promotable to base DRY via PR/MR.

Acceptance:

1. Promotion link references origin `change_id`.
2. No direct auto-write to platform main DRY.

### FR9: Drift-classed live proposals

Live drift proposals must include drift class metadata.

Acceptance:

1. At minimum: `stale-render`, `overlay-drift`, `manual-live-mutation`, `unknown`.

### FR10: Explainability API/CLI

System must explain generator provenance and origin mapping per change.

Acceptance:

1. Query by `change_id` and by WET path.
2. Deterministic machine-readable output.

### FR11: Adapter SDK contract

Custom platform adapters must emit the triple contract.

Acceptance:

1. Adapter registration rejected unless schemas validate.

### FR12: Git/OCI/CH boundary enforcement

System must respect DRY/WET storage boundary.

Acceptance:

1. Git write-back limited to compact receipts/IDs.
2. OCI used as default WET transport.
3. Policy traces and runtime telemetry stored in ConfigHub.

### FR13: Signed deterministic generation contract

Every governed render must be backed by a signed deterministic generator contract.

Acceptance:

1. Contract signature is present and verifiable.
2. Deterministic output hash is present for each render outcome.
3. Missing signature/hash blocks execution path.

### FR14: OwnershipMap-enforced inverse write

Inverse write paths must be bounded by ownership declarations.

Acceptance:

1. Every inverse patch references ownership scope.
2. Out-of-scope patch attempt is auto-`BLOCK`.
3. Block reason includes offending path(s).

### FR15: Replay and escalation gate

Write-back must pass replay consistency against declared generator contract.

Acceptance:

1. Replay run is mandatory before protected write-back.
2. Replay mismatch triggers mandatory `ESCALATE`.
3. `ALLOW` path requires linked attestation record.

### FR16: Protected DRY write-back mode

Protected DRY branches can only be changed through PR/MR flow.

Acceptance:

1. Write-back mode recorded as `pull-request` or `merge-request`.
2. Direct protected-branch write mode is rejected.
3. Verification failure downgrades to read-only evidence mode.

## 9. Non-Functional Requirements

1. Explain latency: p95 < 2s for normal change scope.
2. Import reliability: idempotent operations under retry.
3. Agent compatibility: stable JSON outputs and exit codes.
4. Security: redaction, retention controls, token hygiene.
5. Auditability: attestation linkage for governed applies.
6. Cognitive simplicity: platform engineer can complete first governed import path in <= 10 minutes.
7. Migration friction: existing Flux/Argo reconcile path remains unchanged during MVP adoption.
8. Enforcement correctness: 100% of governed writes satisfy FR13-FR16 gates.

## 10. Data Model (Minimum)

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

## 11. Adapter Matrix (MVP)

| Adapter | Input | Output | MVP | Notes |
|---|---|---|---|---|
| Helm | chart + values + overlays | rendered manifests / OCI | Yes | Priority 1 |
| Score.dev | score.yaml + env context | rendered manifests | Yes | strong field mapping |
| Spring Boot | application config + deploy intent | rendered manifests | Yes | framework-driven defaults |
| Custom Platform | DSL/SDK/config model | generated config | Yes via SDK | must emit contract triple |

Determinism rule for MVP:

1. All adapters in the MVP matrix must be deterministic by contract.
2. AI non-deterministic authoring is tracked as a separate follow-on profile with extra governance requirements.

## 12. API and CLI Profile (MVP)

Command mapping policy:

1. Current project docs and demos use `cub-gen ...`.
2. Promotion targets (`cub gen ...` or `cub gitops ...`) are deferred decisions.
3. Subcommands must remain one-to-one between prototype and promoted surfaces.

### CLI

Prototype binary commands:

1. `cub-gen detect --repo --ref --json`
2. `cub-gen import --repo --ref --space --json`
3. `cub-gen evaluate --change-id --json`
4. `cub-gen origin --change-id --wet-path --json`
5. `cub-gen inverse-plan --change-id --json`
6. `cub-gen promote --change-id --upstream --json`

Future promoted equivalents (post-prototype):

1. `cub gitops detect --repo --ref --json`
2. `cub gitops import --repo --ref --space --json`
3. `cub gitops evaluate --change-id --json`
4. `cub gitops origin --change-id --wet-path --json`
5. `cub gitops inverse-plan --change-id --json`
6. `cub gitops promote --change-id --upstream --json`

### API

1. `POST /v1/imports`
2. `POST /v1/imports/{id}/analyze`
3. `POST /v1/changes/upsert`
4. `POST /v1/changes/{change_id}/evaluate`
5. `POST /v1/changes/{change_id}/decision`
6. `POST /v1/changes/{change_id}/execute`
7. `POST /v1/changes/{change_id}/promote`
8. `GET /v1/changes/{change_id}`
9. `GET /v1/changes/{change_id}/origin-map`
10. `GET /v1/changes/{change_id}/inverse-plan`

## 13. Rollout Plan

### M1: Helm-first import + provenance

1. Deterministic detection/import for Helm repos.
2. Required provenance captured.
3. Basic origin map coverage.

### M2: CH MR governance loop

1. Decision gate + receipts + attestation linkage.
2. Optional Git mirror path.
3. Publish first-class Spring Boot dry/wet worked example aligned to Flux/Argo operating boundary.

### M3: Inverse + LIVE proposals

1. Live drift to proposal workflow.
2. Confidence-based inverse plans.

### M4: Upstream promotion automation

1. Reuse scoring + promotion PR/MR opening.
2. Overlay cleanup guidance.

## 14. Metrics

1. Time to first governed import: < 30 minutes.
2. Origin map critical field coverage: >= 90%.
3. Silent live overwrite incidents: 0.
4. Promotion success rate (reusable to base DRY): tracked per quarter.
5. Mean time to explain change lineage: < 2 minutes end-user workflow.
6. Time to first user-visible value (`detect -> import -> explain`): <= 10 minutes in docs demo path.

## 15. Risks and Mitigations

1. Adapter inconsistency.
   Mitigation: schema conformance tests and golden fixtures.
2. Inverse mapping errors.
   Mitigation: confidence thresholds + manual review gates.
3. Governance fatigue from noisy proposals.
   Mitigation: drift deduplication and proposal prioritization.
4. CH/Git authority confusion.
   Mitigation: single `change_id`, clear ownership labels in UI/docs.

## 16. Open Questions

1. Default promotion behavior: suggest-only or auto-open PR/MR?
2. Minimum mandatory Spring Boot field coverage for MVP?
3. Adapter certification process for internal platform teams?
4. Confidence threshold policy by trust tier?

## 17. Related Specs

1. `00-agentic-gitops-design.md`
2. `50-dual-approval-gitops-gh-pr-and-ch-mr.md`
3. `60-stored-in-git-vs-confighub.md`
4. `../04-schemas/generator-contract.v1.schema.json`
5. `../04-schemas/provenance-record.v1.schema.json`
6. `../04-schemas/inverse-transform-plan.v1.schema.json`
7. `../03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md`
8. `../03-worked-examples/02-traefik-helm-dry-wet-unit-worked-example.md`
9. `../03-worked-examples/03-spring-boot-dry-wet-unit-worked-example.md`
10. `80-agentic-gitops-enforcement-matrix.md`
