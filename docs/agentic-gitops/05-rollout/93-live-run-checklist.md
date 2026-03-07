# Live Run Checklist (Today)

Date: 2026-03-07

## T-15 minutes

1. Open these files:
   - `90-today-demo-plan.md`
   - `91-speaker-sheet-today.md`
   - `92-two-minute-modules.md`
2. Confirm opening line is visible:
   - `Flux/Argo reconcile. ConfigHub decides. Git records.`
3. Confirm baseline line is visible:
   - users already have Kubernetes + Flux/Argo + Git + OCI (Helm optional).
4. Decide module order for this audience:
   - A Score.dev
   - B Traefik/Helm
   - C Spring Boot
   - D CI Hub + Fleet wave

## T-5 minutes

1. Pick one likely objection per module:
   - A: "Why not just Score + Git?"
   - B: "Are you replacing Flux/Argo?"
   - C: "Why not just app YAML in repo?"
   - D: "Can CI do this without brittle scripts?"
2. Set hard stop timer for 45 minutes.
3. Keep one fallback sentence ready:
   - "This module is standalone; we can jump to any other module."

## Live timing card (45 minutes)

1. 0:00-3:00 framing
2. 3:00-11:00 module A (8 min)
3. 11:00-19:00 module B (8 min)
4. 19:00-27:00 module C (8 min)
5. 27:00-35:00 module D (8 min)
6. 35:00-41:00 deep-dive/Q&A
7. 41:00-45:00 close

## Per-module micro checklist

1. State setup line (20s).
2. Deliver 2-minute script.
3. Show one concrete artifact/flow from existing docs.
4. Land one value line.
5. Repeat boundary line once.

## Mandatory enforcement proof gate (before close)

A demo only qualifies as Agentic GitOps if all checks below are true:

1. Active reconciler proof exists (`WET -> LIVE` via Flux/Argo or equivalent).
2. Signed `GeneratorContract` is shown (or queried) with deterministic output hash.
3. `ProvenanceRecord` includes immutable `input_hash`, `toolchain_version`,
   `policy_version`, `run_id`, and artifacts.
4. `OwnershipMap` is referenced by inverse plan.
5. Out-of-scope inverse write attempt is shown as auto-`BLOCK`.
6. Replay mismatch path is shown as auto-`ESCALATE`.
7. Decision gate shown explicitly as `ALLOW | ESCALATE | BLOCK`.
8. `ALLOW` path shows attestation linkage (`who`, evidence, decision).
9. Protected DRY write-back is PR/MR-only in the flow.
10. Verification failure path is read-only evidence mode.
11. Mutation ledger append is shown for the executed change.

Naming rule:

1. If check 1 fails, present the flow as `governed config automation` (not
   Agentic GitOps).

## Close (say exactly)

1. Day 1: import + query + governed mutation API.
2. Day 2: deterministic write-back PR/MR + promotion.
3. Day 3: optional `cub-track` mutation ledger for human/CI/AI.

Final sentence:

"Pick any entry point. The operating boundary stays fixed, and adoption is incremental."
