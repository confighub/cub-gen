# 2-Minute Module Scripts (A/B/C/D)

Use any module in any order. Each script is designed to fit ~2 minutes.

Common opener (10 seconds):

"You already have Kubernetes, Flux/Argo, Git, and OCI. We are adding a usable platform API and governance, not replacing your runtime."

Common closer (10 seconds):

"`Flux/Argo reconcile. ConfigHub decides. Git records.`"

Qualification line (use if asked):

"If there is no active `WET -> LIVE` reconciler loop, this is governed config automation, not Agentic GitOps."

---

## Module A (Score.dev): abstraction -> governed explicit config

### Setup (20s)

"This is the cleanest example. Developers keep writing Score intent, but platform gets explicit deployment state with provenance."

### Talk track (80s)

"Here is the DRY input: Score workload intent.  
Generator renders explicit WET manifests.  
ConfigHub stores the output as queryable Units with provenance and field-origin mapping.  
If we change an app field, ConfigHub resolves the DRY source and writes back deterministically.  
If the change is reusable, it promotes upstream to platform DRY through separate review."

### Value line (20s)

"You keep Score simplicity, but every generated field becomes explainable, governable, and auditable."

---

## Module B (Traefik/Helm): no-migration GitOps compatibility

### Setup (20s)

"This module proves we do not break existing GitOps. Same controller, same transport, better control."

### Talk track (80s)

"Helm remains the generator. Flux/Argo remains the reconciler.  
Git and OCI remain transport and collaboration surfaces.  
ConfigHub adds provenance, policy decisioning, and attestation around the existing flow.  
So teams do not rewrite pipelines or platform repos. They gain explainability and governed execution over what they already run."

### Value line (20s)

"No migration project. Same runtime model. Stronger operational API and governance."

---

## Module C (Spring Boot): framework-native onboarding

### Setup (20s)

"This is for teams who do not want to hand-author deployment YAML."

### Talk track (80s)

"Developers keep Spring Boot config conventions.  
Generator translates that framework intent into explicit deployment manifests.  
ConfigHub stores DRY/WET lineage and field-origin maps.  
When someone asks to change a running value, the system routes edits to the correct source and preserves governance boundaries.  
Platform still gets deterministic output and policy checks before execution."

### Value line (20s)

"Developers stay in familiar app config; platform gets explicit, reviewable, policy-governed deployment state."

---

## Module D (CI Hub + Fleet CVE wave): enterprise wedge

### Setup (20s)

"This is for teams where CI is already the operational hub."

### Talk track (80s)

"Instead of patching files by path, CI calls semantic ConfigHub APIs.  
ConfigHub evaluates policy and creates deterministic write-back PR/MR.  
For a fleet CVE response, one `change_id` can span many repos and targets.  
Each target gets explicit `ALLOW|ESCALATE|BLOCK`, with verification and attestation recorded.  
So you get one governed campaign, not dozens of disconnected PRs and audit gaps."

### Value line (20s)

"Replace brittle YAML scripting with one governed API path and one auditable change identity."

---

## Fast handoff lines between modules

1. "If you prefer no-migration proof, jump to Helm compatibility."
2. "If you prefer developer productivity proof, jump to Spring Boot."
3. "If you prefer enterprise operations proof, jump to CI + fleet wave."

---

## 30-second wrap

"Pick any entry point: app abstraction, Helm compatibility, framework onboarding, or CI automation.  
The boundary stays fixed: Flux/Argo reconcile, ConfigHub decides, Git records.  
Day 1 import and query. Day 2 write-back and promotion. Day 3 optional mutation ledger."
