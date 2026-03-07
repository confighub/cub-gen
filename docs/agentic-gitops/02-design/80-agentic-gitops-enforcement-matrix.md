# Agentic GitOps Enforcement Matrix

**Status:** Normative mapping
**Date:** 2026-03-07
**Purpose:** Map each hard requirement to schema fields and demo proof gates.

## 1. Scope

This matrix is the canonical reference for enforcement across:

1. PRD requirements (`10-generators-prd.md`)
2. Governance model (`40-governed-execution.md`)
3. Schemas (`04-schemas/*.schema.json`)
4. Demo proof checklist (`05-rollout/93-live-run-checklist.md`)

## 2. Qualification Rule

A flow is `Agentic GitOps` only when an active GitOps reconciler loop (`WET -> LIVE`) is present (Flux/Argo or equivalent).

If that inner loop is absent, classify the flow as `governed config automation`.

## 3. Enforcement Mapping

| Rule ID | Hard Requirement (MUST) | Schema Field(s) | Demo Proof Check | Failure Action |
|---|---|---|---|---|
| R1 | Generator contract is signed and deterministic output hash exists | `generator-contract.v1`: `signature.*`, `deterministic=true`, `output_hash` | Show signed contract + hash | Block governed path |
| R2 | Provenance is immutable and complete | `provenance-record.v1`: `input_hash`, `toolchain_version`, `policy_version`, `run_id`, `artifacts[]`, `outputs[]` | Show record with all required fields | Block governed path |
| R3 | Inverse write is bounded to ownership scope | `inverse-transform-plan.v1`: `ownership_map_ref`, `patches[].ownership_scope` | Show inverse plan ownership mapping | Escalate for review |
| R4 | Out-of-scope write auto-blocks | `inverse-transform-plan.v1`: `enforcement.on_out_of_scope_write=BLOCK` | Attempt out-of-scope patch and show block | `BLOCK` |
| R5 | Replay check is mandatory and mismatch escalates | `inverse-transform-plan.v1`: `replay_check.status`, `replay_check.mismatch_action=ESCALATE`, `replay_check.replay_digest` | Show replay mismatch path | `ESCALATE` |
| R6 | Decision gate is explicit | `inverse-transform-plan.v1`: `decision_gate.result` enum (`ALLOW`,`ESCALATE`,`BLOCK`) | Show decision object in output | No execution on missing decision |
| R7 | `ALLOW` requires attestation linkage | `inverse-transform-plan.v1`: `attestation_required_on_allow=true`; provenance attestation digest linkage | Show actor + evidence bundle + decision linkage | Downgrade to `ESCALATE` or `BLOCK` |
| R8 | Protected DRY write-back is PR/MR only | `inverse-transform-plan.v1`: `writeback.mode` (PR/MR only), `writeback.protected_branch_only=true` | Show write-back mode in flow | Reject direct write |
| R9 | Verification failure becomes read-only evidence mode | `inverse-transform-plan.v1`: `verification.on_failure=read-only-evidence` | Force failing verification and show downgrade | Read-only evidence mode |
| R10 | Mutation and decision append to ledger | `inverse-transform-plan.v1`: `ledger.append_required=true` | Show ledger append event/record | Fail run closure |
| R11 | Agentic naming requires active reconciler loop | Docs rule + runtime architecture evidence (`Flux/Argo` reconcile path) | Show `WET -> LIVE` reconciler proof | Reclassify as governed config automation |

## 4. Release Gate

A demo, tutorial, or release can claim `Agentic GitOps` only if:

1. `R1-R11` all pass.
2. Evidence of pass/fail is retained in the run artifacts.
3. Any failed rule is visible to reviewer with explicit reason.

## 5. Minimal Validation Sequence

1. Validate `generator-contract.v1` object.
2. Validate `provenance-record.v1` object.
3. Validate `inverse-transform-plan.v1` object.
4. Execute replay check.
5. Evaluate decision gate.
6. Run verification checks.
7. Require attestation on `ALLOW`.
8. Execute PR/MR write-back only.
9. Append ledger records.
10. Confirm active reconciler proof (`WET -> LIVE`).

## 6. Cross-References

1. `02-design/10-generators-prd.md`
2. `02-design/40-governed-execution.md`
3. `04-schemas/generator-contract.v1.schema.json`
4. `04-schemas/provenance-record.v1.schema.json`
5. `04-schemas/inverse-transform-plan.v1.schema.json`
6. `05-rollout/93-live-run-checklist.md`
