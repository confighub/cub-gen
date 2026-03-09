# Issue Seeds: User Stories + New-User Challenges

Use these as ready-to-file GitHub issues.

## A. User-story completion issues

1. `[US-01] Existing repo import + provenance query without refactor`
- Labels: `enhancement`, `story:1`, `area:cli`, `area:docs`
- Checklist:
  - [ ] repo/ref import flow deterministic
  - [ ] provenance query by `change_id`
  - [ ] example script/docs updated
  - [ ] integration test added

2. `[US-07] CI-first mutation/query integration without path scripting`
- Labels: `enhancement`, `story:7`, `area:bridge`, `area:demo`
- Checklist:
  - [ ] tokened CI workflow example
  - [ ] non-interactive bridge path validated
  - [ ] docs for CI usage updated
  - [ ] pipeline proof recorded

3. `[US-12] Unified human/CI/AI decision + attestation model`
- Labels: `enhancement`, `story:12`, `area:bridge`, `area:contracts`
- Checklist:
  - [ ] actor model normalized
  - [ ] single `change_id` linkage enforced
  - [ ] query/readout example added
  - [ ] golden coverage updated

4. `[US-09] Governed multi-repo change waves with per-target decisions`
- Labels: `enhancement`, `story:9`, `area:bridge`, `area:ops`
- Checklist:
  - [ ] wave manifest format defined
  - [ ] per-target `ALLOW|ESCALATE|BLOCK`
  - [ ] evidence aggregation implemented
  - [ ] end-to-end demo added

5. `[US-11] LIVE break-glass -> explicit accept/revert proposal`
- Labels: `enhancement`, `story:11`, `area:bridge`, `area:ops`
- Checklist:
  - [ ] explicit proposal command/path
  - [ ] drift classes included
  - [ ] no silent overwrite guard tested
  - [ ] demo proof added

6. `[US-08] Label taxonomy evolution without query breakage`
- Labels: `enhancement`, `story:8`, `area:contracts`, `area:cli`
- Checklist:
  - [ ] versioned label schema
  - [ ] backward-compatible query behavior
  - [ ] migration docs/playbook
  - [ ] compatibility tests

7. `[US-10] Preserve/prove branch protections and signed controls`
- Labels: `enhancement`, `story:10`, `area:security`, `area:bridge`
- Checklist:
  - [ ] protection evidence model
  - [ ] gate enforcement integrated
  - [ ] audit readout path
  - [ ] pass/fail tests for evidence presence

## B. New-user challenge operations issues

1. `[DX] Add new-user challenge issue template + labels`
- Labels: `enhancement`, `area:docs`, `area:workflow`
- Checklist:
  - [ ] template published
  - [ ] label taxonomy documented
  - [ ] triage owner assigned

2. `[DX] Weekly confused-user red-team run (Claude)`
- Labels: `enhancement`, `area:docs`, `area:demo`
- Checklist:
  - [ ] runbook linked
  - [ ] weekly cadence owner set
  - [ ] summary issue format enforced
  - [ ] trend metrics tracked

3. `[DX] Time-to-first-value dashboard from challenge issues`
- Labels: `enhancement`, `area:metrics`
- Checklist:
  - [ ] metric definitions frozen
  - [ ] baseline measured
  - [ ] weekly report cadence set

## C. Feedback-triage follow-up issues (2026-03-09)

1. `[DX] Remove absolute/local path references from active docs and mark archive docs`
- Labels: `enhancement`, `area:docs`, `challenge:new-user`
- Checklist:
  - [ ] absolute/local links removed from active docs
  - [ ] archive docs labeled as non-entrypoint
  - [ ] top-level docs map points to current sources of truth

2. `[E2E] Add Argo live reconciliation end-to-end entrypoint`
- Labels: `enhancement`, `area:demo`, `area:gitops`
- Checklist:
  - [ ] Argo live e2e script added
  - [ ] create/update/drift-correction proof included
  - [ ] docs and caveat matrix updated

3. `[E2E] Add per-example cub-gen render -> live reconcile proof path`
- Labels: `enhancement`, `area:demo`, `story:9`
- Checklist:
  - [ ] at least one example rendered by cub-gen in live e2e
  - [ ] evidence links change_id -> rendered artifacts -> live state
  - [ ] docs updated with command path and caveats

4. `[Detect] Replace heuristic strings.Contains branches with structural detection`
- Labels: `enhancement`, `area:adapter`, `story:1`
- Checklist:
  - [ ] each family has structural detection rules
  - [ ] false-positive regression tests added
  - [ ] confidence outputs documented

5. `[Catalog] Clarify c3agent multi-target resource-kind semantics`
- Labels: `enhancement`, `area:docs`, `area:adapter`
- Checklist:
  - [ ] clarify catalog output for multi-target generators
  - [ ] adjust metadata model if needed
  - [ ] parity tests updated if contract changes
