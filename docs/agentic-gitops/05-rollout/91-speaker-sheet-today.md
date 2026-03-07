# Speaker Sheet (Today)

Date: 2026-03-07  
Audience: Platform engineering + architecture stakeholders  
Length: 45 minutes (modular)

## Core line (repeat often)

`Flux/Argo reconcile. ConfigHub decides. Git records.`

## Baseline assumption (state once at start)

We assume teams already run Kubernetes + Flux/Argo + Git + OCI. Helm may be used, but it is optional.

Naming guardrail:

1. If there is no active reconciler loop (`WET -> LIVE`), call it `governed config automation` rather than Agentic GitOps.

## Run-of-show (modular, jump-in anywhere)

1. 0:00-3:00 — 1-slide framing only
2. 3:00-35:00 — All 4 modules (~8 min each, order flexible)
3. 35:00-41:00 — Q&A / one deep-dive follow-up
4. 41:00-45:00 — Close

Each module is standalone:

1. 20-second setup ("what you already have")
2. 2-minute walkthrough
3. 30-second value line

Use this 8-minute module rhythm:

1. 0:00-0:20 setup line
2. 0:20-2:20 core script (from `92-two-minute-modules.md`)
3. 2:20-5:30 live example walk-through (show existing artifact/screens/flow)
4. 5:30-6:30 objection handling (pick one likely question)
5. 6:30-7:30 value line + boundary reminder
6. 7:30-8:00 jump to next module

## Jump-in starter lines (use before any module)

1. "You already have Kubernetes, Flux/Argo, Git, and OCI. We are adding a usable platform API and governance, not replacing your runtime."
2. "This module is standalone; we can jump to any other module after this."

## Optional 30-second transitions (if chaining)

### Transition to Demo 1 (Score.dev)

"Let’s start with the cleanest generator case. Score shows the core model: DRY intent in, explicit WET out, with provenance and field-origin mapping. This is the shortest path from abstraction to governed deployment."

### Transition to Demo 2 (Traefik/Helm)

"Now we prove we do not break existing GitOps. Same Flux/Argo runtime, same Git and OCI transport. We only add explainability and governance around what already runs."

### Transition to Demo 3 (Spring Boot)

"Next is framework-native onboarding. Developers keep the config they already write, while platform still gets explicit, auditable deployment output."

### Transition to Demo 4 (CI Hub + CVE wave)

"Now the enterprise wedge: CI remains the hub, but it calls semantic platform APIs instead of patching files. Then we scale that to one governed fleet wave with one change identity."

### Transition to close

"So this is not a controller replacement story. It is an API and governance upgrade for teams already running GitOps."

## Standalone modules (pick any 3)

### Module A: Score.dev

1. DRY intent -> explicit WET + provenance.
2. Field-origin map supports safe edit routing.
3. Reusable app change promotes upstream with separate review.
4. Easy value line: "You keep Score, but now every generated field is explainable and governable."

### Module B: Traefik/Helm

1. Existing Helm + Flux/Argo flow unchanged.
2. OCI/Git transport unchanged.
3. Governance and explainability added without reconciler replacement.
4. Easy value line: "No migration. Same controllers. Better visibility and control."

### Module C: Spring Boot

1. Framework config stays natural for app teams.
2. Output remains explicit and reviewable.
3. Platform gets deterministic lineage + policy gates.
4. Easy value line: "Developers keep framework config; platform gets auditable deployment state."

### Module D: CI Hub + Fleet CVE wave

1. CI calls semantic API, not file paths.
2. One `change_id` spans many repos/targets.
3. Per-target `ALLOW|ESCALATE|BLOCK` + attestation closes audit loop.
4. Easy value line: "Replace YAML patch scripts with one governed API path."

## Objection quick responses (one-liners)

1. "Are you replacing Flux/Argo?"  
No. They remain the reconciler.

2. "Why not just Git?"  
Git is great for history and review, but not for cross-repo query, policy-time decisions, or attestation joins.

3. "Does ConfigHub bypass PR controls?"  
No. Write-back stays inside normal PR/MR protections.

4. "Can this run on-prem/air-gapped?"  
Yes. Same model, different deployment topology.

5. "Is this only for AI agents?"  
No. Same governance model applies to humans, CI bots, and AI.

## Close slide (say exactly)

1. Day 1: import + query + governed mutation API.
2. Day 2: deterministic write-back PR/MR + promotion.
3. Day 3: optional `cub-track` mutation ledger for human/CI/AI history.

## One-sentence close

"Pick any entry point: app abstraction, Helm compatibility, framework onboarding, or CI automation; the operating boundary stays the same."

## Hard stops (to stay tight)

1. No deep schema walkthrough in live session.
2. No new architecture proposals during Q&A.
3. If discussion drifts: return to boundary line and phased adoption.
