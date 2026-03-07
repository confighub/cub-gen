# Today Demo Plan (Simple, High-Value, Reuse Existing Examples)

Date: 2026-03-07

## Goal for today

Show that Agentic GitOps gives teams:

1. a usable platform API (not file-path editing),
2. governed AI/human/CI change flow,
3. zero disruption to Flux/Argo reconciliation.

Boundary line to repeat in every demo:

`Flux/Argo reconcile. ConfigHub decides. Git records.`

Qualification rule:

1. If active reconciler loop (`WET -> LIVE`) is missing, describe the flow as
   `governed config automation`, not Agentic GitOps.

## Baseline assumptions for all demos/examples

Assume users already have:

1. Kubernetes
2. GitOps reconciler (Flux or Argo)
3. Git
4. OCI registry/transport

Helm is optional (common, but not required).

## Demo set (reuse-first)

Use existing worked examples as standalone modules (order optional):

1. `03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md`
2. `03-worked-examples/02-traefik-helm-dry-wet-unit-worked-example.md`
3. `03-worked-examples/03-spring-boot-dry-wet-unit-worked-example.md`
4. `05-rollout/30-user-experience.md` (Day 2 Scenarios 5 and 6)

Run all 4 modules; do not force sequential dependency between them.

## Story per demo (what to prove)

### Demo 1: Score.dev (core narrative)

Prove:

1. DRY intent -> explicit WET with provenance and field-origin map.
2. App-team change goes through ConfigHub governance and decision gate.
3. Reusable change promotes upstream to platform DRY with separate review.

### Demo 2: Traefik Helm (GitOps compatibility)

Prove:

1. Existing Helm + Flux/Argo flow stays intact.
2. OCI/Git transport remains standard.
3. ConfigHub adds explainability and governance, not reconciler replacement.

### Demo 3: Spring Boot (framework generator value)

Prove:

1. Developers keep familiar framework config.
2. Platform still gets explicit, auditable WET.
3. Field-origin routing removes repo/file/overlay archaeology.

### Demo 4: CI Hub + Fleet CVE wave (enterprise wedge)

Prove:

1. CI calls semantic ConfigHub API (no brittle YAML file patch scripts).
2. One `change_id` can govern a multi-repo wave.
3. Per-target `ALLOW|ESCALATE|BLOCK` + attestation closes audit loop.

## Minimal run-of-show (45 minutes)

1. 3 min: problem framing ("Git is not a service-level write API")
2. 32 min: four standalone modules (~8 min each)
3. 6 min: audience-driven deep dive/Q&A
4. 4 min: close with adoption path

## Adoption close slide (keep simple)

1. Day 1: import + query + governed mutation API
2. Day 2: deterministic write-back PR/MR + promotion flow
3. Day 3: optional cub-track mutation ledger for AI/human/CI

## Do not overcomplicate today

1. No new architecture branches.
2. No new abstractions beyond `change_id`, decision states, and field-origin maps.
3. Reuse existing examples; only improve narration and API/governance callouts.
4. Keep each module independently understandable in under 3 minutes.
