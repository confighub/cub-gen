# AI and GitOps: Intent, Generators, and Governed Execution

**Status:** Planning doc (v7)
**Date:** 2026-02-28
**Supersedes:** GitOps6.md (monolithic draft)
**Context:** Comprehensive update incorporating UX design, value analysis, authoring landscape, and Claude skill concept from conversations of 2026-02-25/26/28.

---

## What's New Since v6

v6 was a 67-page monolith. v7 restructures the same content (plus significant new
material) into **eight focused documents**, each with a clear audience, purpose, and
format. Nothing from v6 is lost; everything is reorganized and expanded.

### New material in v7

| Topic | Source | Where It Lives |
|-------|--------|---------------|
| **Four UI surfaces × two personas** | Conversation 2026-02-26 | 07-user-experience.md |
| **GUI mockups** (generator catalog, repo view, app view, field-origin edit, app graph) | Conversation 2026-02-26 | 07-user-experience.md |
| **Import from Git experience** (4-step UX flow, no migration required) | Conversation 2026-02-26 (for Brian/Jesper) | 07-user-experience.md |
| **Full authoring landscape** (6 patterns: templates, framework generators, workload abstractions, app platforms, ops/actions, AI agentic) | dry-wet-config-analysis.md | 02-generators-prd.md |
| **Value analysis / pricing boundary** (OSS authoring ingress + ConfigHub dry/wet control-plane value, value-per-buyer-persona) | dry-wet-config-analysis.md | 08-adoption-and-reference.md |
| **Claude skill for agentic GitOps** (tool routing, architecture oracle, doc consistency, 5 skill flavors) | Conversation 2026-02-26 | 07-user-experience.md |
| **Standalone cub-track introduction** (KEP-quality, full schema, examples, FAQ) | Extracted from v6 + expanded | 05-cub-track.md |
| **Field-origin maps and editing as standalone doc** | Extracted from generators doc | 03-field-origin-maps-and-editing.md |
| **Generators doc as PRD** (product requirements, acceptance criteria, milestones) | Rewritten from v6 | 02-generators-prd.md |
| **Brian's DRY/WET concern addressed** | Conversation 2026-02-26 | 03-field-origin-maps-and-editing.md |
| **Day 2 scenarios** (3am incident, fleet visibility, AI audit, config change) | Retained from GitOps6.md | [07-user-experience.md](../05-rollout/30-user-experience.md#11-day-2-scenarios) |
| **Lock-in addressed head-on** | dry-wet-config-analysis.md gap analysis | 08-adoption-and-reference.md |

### Structural changes

v6 had three parts (Narrative, Normative, Examples) in one file. v7 splits by
**topic, audience, and format**:

| Document | Format | Audience | Purpose |
|----------|--------|----------|---------|
| **01** Introducing Agentic GitOps | Blog-post intro | Anyone new to the model | Why: the problem, the claim, the invariants |
| **02** Generators PRD | Product requirements | Product, platform engineers | What: generator model, maturity levels, requirements |
| **03** Field-Origin Maps and Editing | Product innovation | Product, UX, engineers | How: editing through ConfigHub, adoption ladder |
| **04** App Model and Contracts | Reference | Implementors, API consumers | What: entities, contracts, constraints, operations |
| **05** cub-track | Standalone intro (KEP-quality) | Engineers, compliance, SREs | What: mutation ledger, schema, trust tiers |
| **06** Governed Execution | Architecture | Security, compliance, SREs | Trust: two-loop model, verification, attestation, write-back |
| **07** User Experience | Design | Product, UX, dev advocates | Feel: surfaces, personas, import flow, AI tooling |
| **08** Adoption and Reference | Business case | Decision makers, teams | Path: adoption steps, value, examples, FAQ |

### Reading order

```
01 Introduction  → Read this first. "What is this and why does it matter?"
02 Generators    → PRD. "What are we building? Requirements, scope, milestones."
03 Editing       → Innovation. "How does editing through ConfigHub work?"
04 Data model    → Reference. "What are the entities and contracts?"
05 cub-track     → Product intro. "What is cub-track, specifically?"
06 Governance    → Architecture. "How does the full governance model work?"
07 UX            → Design. "What does it feel like across surfaces and personas?"
08 Adoption      → Business case. "How do I start? What's the value?"
```

### Illustrated companions

1. `00-index/02-illustrated-cheat-sheet.md` for visual architecture and enforcement flows.
2. `03-worked-examples/04-eight-example-story-cards.md` for the 2x4 example map and user-story alignment.
3. `05-rollout/94-demo-illustration-pack.md` for reusable demo diagrams and talk tracks.

---

## The Central Claim

**Configuration is data** — literal values, not parameterized templates, stored where
they can be queried, versioned, and governed. Generators are how you get there:
deterministic functions that turn intent into explicit, deployable configuration.

If you use Helm, you already have one.

---

## Three Invariants (Never Waived)

1. **Nothing implicit ever deploys.**
2. **Nothing observed silently overwrites intent.**
3. **Configuration is data, not code.**

The first two govern mutation flow. The third governs representation — and it is the
foundational claim.

These hold across all modes: standalone, connected, agentic.

Operational corollary for agentic GitOps: no execution authority is issued without verification evidence, and every governed mutation emits an attestation that links intent, decision, execution, and observed outcome.

Qualification rule: if there is no active GitOps inner loop (`WET -> LIVE`
reconciliation via Flux/Argo or equivalent), describe the system as
`governed config automation`, not Agentic GitOps.

---

## The Model in One Diagram

```
Developer/Agent Intent (DRY)
        |
        v                          ConfigHub Editing Surface
    Generator (deterministic,  <-- (resolves field-origin map,
     versioned)                     writes back to DRY source)
        |                                    ^
        v                                    |
ConfigHub (dry+wet Units; WET deployment contract) ----------+
     for intended state)           (user views, compares,
        |                          traces, edits here)
        v
    Publish (OCI artifact or Git source update)
        |
        v
    GitOps Reconcile (Flux/Argo, inner loop)
        |
        v
    Runtime (what actually exists)
        |
        v
    cub-scout Observes
        |
        v
    Evidence Bundle (structured diff + provenance)
        |
        +---> Export (Slack, Jira, S3, ConfigHub history)
        |
        +---> Decision (human review, policy engine, or workflow)
```

---

## Operating Boundary (Quick Reference)

| Responsibility | Owned By |
|----------------|----------|
| Store dry+wet Units, keep WET authoritative for deployment, publish, resolve field-origin maps, route editing to DRY ingress | **ConfigHub** |
| Detect stale renders (inputs changed, output not re-rendered) | **ConfigHub** (publishing pipeline) |
| Import generator-style DRY inputs, produce field-origin maps + inverse-edit guidance, and emit governed change bundles | **cub-gen** |
| Reconcile runtime from published artifacts | **Flux / Argo** (inner loop) |
| Observe cluster, capture evidence, detect drift from intended state | **cub-scout** |
| Record governed mutation history; redirect WET edits to DRY sources | **cub-track** |
| Evaluate risk and policy signals | **confighub-scan** |
| Execute token-scoped runtime actions | **confighub-actions** |

**Never cross these boundaries:**
- cub-scout observes the CLUSTER, not the generator pipeline
- cub-gen analyzes/imports DRY->WET intent, not LIVE runtime state
- cub-track records MUTATIONS, not runtime state
- ConfigHub STORES and GOVERNS, it does not reconcile
- Flux/Argo RECONCILE, they are not replaced
- Staleness (DRY changed, WET not re-rendered) is ConfigHub's concern, not cub-scout's
- Drift (cluster differs from intended state) is cub-scout's concern, not ConfigHub's

**Clarity check (not one tool in disguise):**
- `cub-gen` + `cub-scout` are complementary surfaces in one operating model, not duplicates.
- `cub-track` is a separate mutation-ledger surface (Labs/planned), not a hidden rename of either tool.
- Packaging may converge later (`cub track`), but the responsibilities stay separate.

---

## Generator Maturity Levels (Quick Reference)

| Level | What | Today? | What You Can Now Answer |
|-------|------|--------|----------------------|
| **1. Capture** | Renderer unit → output unit | **Shipped** | "What did Flux/Argo actually produce?" |
| **2. Provenance** | + generator name, version, input digest, field-origin map | Planned | "What inputs produced this? Is it stale? Where does this field come from?" |
| **3. Pre-publish** | + render before publish, validate against constraints | Planned | "Will this violate platform rules before deploy?" |
| **4. Governed** | + policy gates, attestation, trust tiers | Planned | "Who authorized this, what checks ran, what proof exists?" |

---

## Adoption Ladder (Quick Reference)

| Stage | User does | Trust earned |
|-------|-----------|-------------|
| **View** | "Show me what's deployed across envs" | ConfigHub knows the truth |
| **Compare** | "Why is prod different from staging?" | ConfigHub explains provenance |
| **Trace** | "Where does this value come from?" | ConfigHub navigates the DRY→WET chain |
| **Edit** | "Change replicas to 5 in prod" | ConfigHub is where I make changes |
| **Govern** | "Who approved this? Can I promote to prod?" | ConfigHub is how we operate |

---

## Value at Each Level

| Level | Capability | Business Outcome | Free / ConfigHub |
|-------|-----------|-----------------|------------------|
| **0. OSS tools** | cub-scout observation, cub-track local mutation history | Community adoption, per-repo visibility | Free (OSS) |
| **1. Capture** | Rendered manifests stored as Units | Answer "what did Flux produce?" in one query | ConfigHub (shipped) |
| **2. Provenance** | Four fields + field-origin maps | Stale render detection, trace any field to DRY source | ConfigHub (planned) |
| **2a. Edit** | ConfigHub resolves field origins, commits to DRY in Git | Change values without knowing repo/file/line | ConfigHub (planned) |
| **3. Pre-publish** | Validate against platform constraints before publish | Catch violations at render time, not after deploy | ConfigHub (planned) |
| **4. Governed** | Policy gates, trust tiers, attestation, mutation ledger | Audit-grade compliance: who authorized, what proof exists | ConfigHub (planned) |
| **Enterprise** | Retention, RBAC, fleet analytics, compliance exports | Org-wide visibility, regulatory reporting | ConfigHub Enterprise |

---

## Document Set

1. **[01-introducing-agentic-gitops.md](../01-vision/01-introducing-agentic-gitops.md)** — The problem, the claim, and the invariants
2. **[02-generators-prd.md](../02-design/10-generators-prd.md)** — Generator model, maturity levels, provenance requirements
3. **[03-field-origin-maps-and-editing.md](../02-design/20-field-origin-maps-and-editing.md)** — Field-origin maps, editing experience, adoption ladder
4. **[04-app-model-and-contracts.md](../02-design/30-app-model-and-contracts.md)** — Entities, operating boundary, constraints, operations
5. **[05-cub-track.md](../05-rollout/10-cub-track.md)** — Git-native mutation ledger, ChangeInteractionCard, trust tiers
6. **[06-governed-execution.md](../02-design/40-governed-execution.md)** — Two-loop model, evidence, write-back semantics
7. **[07-user-experience.md](../05-rollout/30-user-experience.md)** — Four surfaces, two personas, import flow, AI tooling
8. **[08-adoption-and-reference.md](../05-rollout/40-adoption-and-reference.md)** — Adoption path, value analysis, Day 2 scenarios, FAQ

---

## Key Concepts (Quick Reference)

| Concept | Definition |
|---------|------------|
| **App** | Named collection of components, queried by label |
| **Deployment** | App × Target (environment instance) |
| **Unit** | Atomic deployable config with labels and provenance |
| **Generator** | Deterministic function: intent + context → WET |
| **Field-origin map** | Generator-produced mapping from WET fields to DRY sources; enables tracing and editing |
| **WET** | Explicit manifests — what actually deploys |
| **DRY** | Developer intent: templates, values, workload specs — authored, not deployed directly |
| **Evidence** | cub-scout observation: structured diff + provenance (cluster vs. intended state) |
| **Overlay drift** | WET field changed without DRY input change; transitional state, not steady-state |
| **Change Interaction Card** | cub-track mutation record: intent + decision + execution + outcome |

---

## Related Documents

### v7 Document Set (this repository)

| File | Purpose |
|------|---------|
| `GitOps7.md` | This file — master overview and index |
| `01-introducing-agentic-gitops.md` | Positioning and invariants |
| `02-generators-prd.md` | Generator PRD |
| `03-field-origin-maps-and-editing.md` | Editing innovation |
| `04-app-model-and-contracts.md` | App model and contracts |
| `05-cub-track.md` | Mutation ledger |
| `06-governed-execution.md` | Governance and evidence |
| `07-user-experience.md` | UX design and surfaces |
| `08-adoption-and-reference.md` | Adoption, value, and FAQ |

### Superseded

| File | Status |
|------|--------|
| `GitOps6.md` | Superseded by this document set |
| `GitOps5.md` | Superseded |
| `GitOps4.md` | Superseded |
| `GitOps3.md` | Superseded |
| `GitOps2.md` | Superseded (retains useful compact contract format as reference) |
| `GitOps.docx` | Superseded |

### Supporting Analysis

| File | Purpose |
|------|---------|
| `ux-four-surfaces-two-personas.md` | Source material for 07-user-experience.md |
| `ux-import-and-generators-for-brian-jesper.md` | Source material for 07-user-experience.md |
| `claude-idea-for-agentic-gitops-skill.md` | Source material for 07-user-experience.md |
| `dry-wet-config-analysis.md` | Source material for value analysis in 08-adoption-and-reference.md |
