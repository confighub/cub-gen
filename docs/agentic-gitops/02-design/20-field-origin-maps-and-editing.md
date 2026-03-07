# Field-Origin Maps and the Editing Experience

> The field-origin map is the bridge between provenance and usability.
> It turns "we know where this came from" into "we know where to change it."

**Part of:** [AI and GitOps v7 Document Set](/docs/agentic-gitops/00-index/00-gitops7-index.md)
**Status:** Planning doc (v7)
**Date:** 2026-02-28
**Audience:** Product team, platform engineers, app developers
**Purpose:** How field-origin maps enable traceable editing, and how ConfigHub becomes the editing surface without breaking DRY-source integrity

---

## Table of Contents

- [1. What Is a Field-Origin Map](#1-what-is-a-field-origin-map)
- [2. Two Capabilities](#2-two-capabilities)
- [3. Which Generators Produce Maps](#3-which-generators-produce-maps)
- [4. ConfigHub as the Editing Surface](#4-confighub-as-the-editing-surface)
- [5. The Adoption Ladder](#5-the-adoption-ladder)
- [6. What Gets Edited Where](#6-what-gets-edited-where)
- [7. cub-track as the Redirection Layer](#7-cub-track-as-the-redirection-layer)
- [8. Staleness Detection](#8-staleness-detection)
- [9. Per-Variant Changes Belong in DRY](#9-per-variant-changes-belong-in-dry)
- [10. Addressing DRY Author Concerns](#10-addressing-dry-author-concerns)
- [11. Cross-References](#11-cross-references)

---

## Qualification Rule

Use `Agentic GitOps` only when an active inner reconciliation loop
(`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler).

If this loop is absent, classify the flow as `governed config automation`.

---

## 1. What Is a Field-Origin Map

The four provenance fields (generator name, version, input digest, render timestamp) tell you *that* a generator produced the Unit. They do not tell you *which input field controls which output field*. A **field-origin map** bridges this gap -- a mapping from output fields to their DRY source, produced by the generator alongside the WET output:

```yaml
field_origins:
  spec.replicas:
    source: values-prod.yaml
    path: replicas
    line: 14
    editable_by: app-team
  spec.template.spec.containers[0].resources.limits.memory:
    source: platform-context.yaml
    path: constraints.memory-max
    editable_by: platform-team
  spec.template.spec.containers[0].image:
    source: values-prod.yaml
    path: image.tag
    line: 7
    editable_by: app-team
```

Each entry in the map answers three questions about a single WET field:

1. **Where did this value come from?** The `source` and `path` identify the DRY file and the key within it. The optional `line` gives an exact location for editors and tooling.

2. **Who is allowed to change it?** The `editable_by` field declares ownership. Platform-injected fields (resource limits, network policy) belong to the platform team. App-level fields (replicas, image tag, env vars) belong to the app team. This is not enforced at the map level -- it is consumed by ConfigHub's editing surface to route changes to the right people.

3. **Is this field generated or native?** If a field appears in the map, it has a DRY origin and edits should flow back to that origin. If a field does not appear in the map (or the Unit has no map at all), the field is WET-native and ConfigHub is the correct editing surface.

The map is a generator output artifact. It is produced at render time, stored alongside the WET manifests, and consumed by ConfigHub to power the editing experience described in the rest of this document.

### Relationship to Provenance

The field-origin map is complementary to the four provenance fields defined in the generator model ([02 -- Generators PRD](/docs/agentic-gitops/02-design/10-generators-prd.md)):

| Provenance field | What it answers | Granularity |
|-----------------|----------------|-------------|
| `generator.name` | "Which generator produced this?" | Whole Unit |
| `generator.version` | "Which version of the generator?" | Whole Unit |
| `inputs.digest` | "What inputs were consumed?" | Whole Unit |
| `rendered.at` | "When was this rendered?" | Whole Unit |
| **field-origin map** | "Which input controls this specific field?" | **Per field** |

Provenance tells you the Unit's lineage. The field-origin map tells you each field's lineage. Together they provide complete traceability from any WET field back to the DRY input, generator version, and render timestamp that produced it.

---

## 2. Two Capabilities

Field-origin maps enable two capabilities that provenance alone cannot:

### 2.1 Traceable Editing

When a user sees `replicas: 2` in a deployed manifest and wants to change it, the field-origin map resolves the DRY source instantly: "this value comes from `values-prod.yaml` line 14 -- edit there." Without the map, this resolution requires manual archaeology: grep through chart templates, find which values file feeds which environment, check which overlay directory applies. For teams with hundreds of services across multiple environments, this archaeology can take minutes or hours per field.

With the field-origin map, the resolution is a lookup. ConfigHub reads the map, shows the user the source file, line, and repo, and offers to open a PR. The entire flow -- from "I want to change this" to "here is the PR" -- is a single interaction.

### 2.2 Edit-Surface Routing

Fields with `editable_by: platform-team` vs `editable_by: app-team` tell ConfigHub who should change what. A platform-injected field (resource limits, network policy) is not the app team's to edit; the map makes this explicit.

This distinction matters because the editing surface differs by ownership:

- **App-team fields** route to DRY. ConfigHub resolves the source file and proposes a PR to the app team's config repo.
- **Platform-team fields** may route to a different DRY source (a platform-context overlay) or may be WET-native (ConfigHub is the editing surface directly).
- **Fields with no origin entry** are WET-native. There is no DRY to redirect to. ConfigHub owns them.

The routing is automatic from the user's perspective. They click a field, ConfigHub reads the map, and the right editing experience opens. No manual decision about where to make the change.

---

## 3. Which Generators Produce Maps

Not all generators can produce complete field-origin maps. Coverage depends on how transparent the mapping from input to output is:

**Identity generator** -- trivial. Every field maps to itself. The input is the output. The field-origin map is 1:1.

**Score and Spring Boot generators** -- clean mappings by design. The input schema is well-structured, so the mapping from input fields to output fields is straightforward. A Score workload's `containers[0].image` maps directly to the K8s Deployment's `spec.template.spec.containers[0].image`. There is no ambiguity.

**Helm** -- harder, but tractable for common fields. Helm templates can produce arbitrarily complex output from arbitrarily structured input. A full, automatic field-origin map for every Helm chart is not realistic. But the fields teams actually edit -- replicas, image, resources, ports, env vars -- have clear mappings from `values.yaml` to template output. Start with the fields people change, not every field.

### Progressive Coverage

Field-origin maps are optional at Level 2 (provenance). A Unit can have provenance metadata (generator name, version, digest, timestamp) without a field-origin map. The map becomes increasingly valuable at Level 3+ where ConfigHub can use it to route editing to the right source and validate field-level ownership.

The coverage model is progressive:

| Generator type | Map completeness | Strategy |
|---------------|-----------------|----------|
| Identity | 100% | Trivial -- every field maps to itself |
| Score | 90%+ | Schema-driven, well-structured inputs |
| Spring Boot | 90%+ | Convention-based, framework knowledge |
| Helm (common fields) | 60-80% | Top-edited fields first, expand over time |
| Helm (full chart) | 30-50% | Best-effort; complex template logic limits coverage |
| AI/LLM generators | Varies | Prompt + context fields tracked; output mapping is partial |

Incomplete maps are acceptable. A field without an origin entry is treated as WET-native -- ConfigHub becomes the editing surface. This is the safe default: if the system does not know where a field came from, it does not guess.

---

## 4. ConfigHub as the Editing Surface

### The Problem

Users who author config in DRY format -- Helm values, Score workloads, Spring Boot `application.yaml` -- naturally want to edit in DRY space. That is where they authored the intent; that is where changes should happen.

The risk: if variant-specific customizations happen as overlays on the WET output, the DRY source stops being the real source of truth. Over time, WET overlays accumulate, the WET diverges from what the generator would produce, and the DRY file becomes fiction -- it describes what the generator *would* produce, not what is actually deployed.

The answer is not to prevent WET-space customization entirely. The answer is to make editing through ConfigHub the natural path, and to have ConfigHub resolve the DRY source for the user.

### The Three-Step Experience

ConfigHub stores WET -- it is the queryable system of record for what is deployed. But for generator-backed config, ConfigHub can also be the surface that **writes back to DRY**.

The experience:

1. **View in ConfigHub.** The user sees the fully resolved WET -- what is actually deployed, across variants, with ownership and provenance. This is already better than reading raw YAML in Git because it is queryable, comparable, and shows the whole picture.

2. **Click a field to edit.** ConfigHub resolves the field-origin map (section 1). It shows the user: "this is `replicas`, it comes from `values-prod.yaml:14` in `acme/platform-config`, you have edit rights." The user changes the value in the ConfigHub UI.

3. **ConfigHub proposes upstream changes explicitly.** It opens a PR/MR to update the DRY source in Git, subject to policy and approval. After merge, the generator re-renders. WET updates. The reconciler applies. The user never opened an IDE or navigated a Git repo. They edited DRY *through* ConfigHub.

From the user's perspective, they edited in ConfigHub. Under the hood, the DRY-to-WET contract is preserved. DRY authors are happy because DRY is the source of truth. The product is happy because ConfigHub is the interface they use daily.

### Why This Matters

This is the key innovation of the editing model. Without field-origin maps, ConfigHub can only edit WET directly -- which breaks DRY-source integrity for generated config. With field-origin maps, ConfigHub becomes a **write-through editing surface**: the user edits in ConfigHub, and the system routes the change to the correct layer (DRY or WET) based on the map.

The experience is the same regardless of which layer the change flows to. The user sees a field, changes its value, and the system handles the rest. The routing is transparent.

### The Full Edit Flow

```
User clicks field in ConfigHub UI
        |
        v
ConfigHub reads field-origin map
        |
        +--- Field has DRY origin?
        |       |
        |       YES --> Show source file, line, repo
        |       |       User enters new value
        |       |       ConfigHub opens PR to DRY source
        |       |       PR merges --> generator re-renders
        |       |       WET updates --> reconciler applies
        |       |
        |       NO ---> Field is WET-native
        |               User edits directly in ConfigHub
        |               Change is recorded in mutation ledger
        |               Reconciler applies
        |
        v
    Result: user edited one field, system routed correctly
```

Both paths end with the same result: the intended state is updated and the reconciler applies. The difference is whether the change flows through the generator pipeline (DRY-origin fields) or goes directly to the WET store (WET-native fields). The user does not need to know which path applies -- ConfigHub handles the routing.

---

## 5. The Adoption Ladder

Building trust with DRY-first users is incremental. Start with viewing -- the lowest-friction entry point -- and let gravity pull toward editing.

| Stage | User does | Trust earned |
|-------|-----------|-------------|
| **View** | "Show me what's deployed for payments-api across envs" | ConfigHub knows the truth |
| **Compare** | "Why is prod different from staging?" | ConfigHub explains provenance |
| **Trace** | "Where does this value come from?" | ConfigHub navigates the DRY-to-WET chain |
| **Edit** | "Change replicas to 5 in prod" | ConfigHub is where I make changes |
| **Govern** | "Who approved this? Can I promote to prod?" | ConfigHub is how we operate |

### Starting with App Config

App config is the right starting category: highest-frequency edits, lowest stakes. Nobody is nervous about changing replicas or an env var. By the time users trust ConfigHub for that, the step to using it for variant creation, promotion, and governance feels incremental.

The progression is:

1. **View** requires no commitment. Import existing manifests, browse them in ConfigHub. Zero risk.
2. **Compare** builds confidence. Users see provenance and cross-environment diffs. They start trusting the data.
3. **Trace** creates the "aha" moment. "Where does this value come from?" resolves in one click instead of twenty minutes of Git archaeology.
4. **Edit** is the conversion point. The first time a user edits replicas through ConfigHub and sees the PR open automatically in their config repo, the value is concrete.
5. **Govern** follows naturally. Once editing happens through ConfigHub, governance (approvals, promotion, audit) is just configuration of the editing workflow.

Each stage is independently valuable. Users who never reach the Edit stage still benefit from View, Compare, and Trace.

---

## 6. What Gets Edited Where

Not everything has a DRY upstream. Some edits are legitimately WET-native -- ConfigHub *is* the right editing surface, and there is no DRY source to redirect to:

| Edit type | Where it happens | Why |
|-----------|-----------------|-----|
| **App config** (replicas, image, env vars, ports) | DRY source via generator then WET | Generator has a knob for it; ConfigHub resolves the DRY source via field-origin map |
| **Platform policy** (network rules, security, quotas) | ConfigHub directly | Platform team owns these; no generator upstream |
| **Emergency overrides** | ConfigHub with TTL | Temporary, governed, recorded; expires or promotes to DRY |
| **Deployment topology** (which clusters, target selection) | ConfigHub | Control plane decision, not generated |
| **Variant lifecycle** (create canary, regional override) | ConfigHub | Control plane orchestration |
| **Governance state** (approvals, locks, trust tiers) | ConfigHub | Native to the control plane |
| **Legacy / no-generator YAML** | ConfigHub | WET-already (identity generator); no DRY to go back to |

### The Routing Principle

If a field has a generator origin, you edit in DRY and the system guides you there. If a field is control-plane-native or platform-injected, ConfigHub is the right editing surface and there is no DRY to redirect to. The field-origin map (section 1) distinguishes these cases.

This taxonomy matters because it prevents a common failure mode: teams who treat all config as equivalent and try to manage it through a single editing pattern. Generated fields and native fields have different lifecycles, different ownership, and different change velocities. The field-origin map makes these differences explicit and routes edits accordingly.

### Emergency Overrides: A Special Case

Emergency overrides deserve specific attention. During an incident, an operator may need to change a generated field (e.g., scale replicas from 2 to 20) without waiting for a PR to merge and a generator to re-render. The system permits this:

1. The operator edits the field directly in ConfigHub (WET edit).
2. ConfigHub applies the change immediately. The reconciler deploys it.
3. cub-track records the override with a TTL and classifies it as "emergency overlay."
4. After the incident, the system surfaces the override: "This field has a DRY origin at `values-prod.yaml:14`. The emergency override has been active for 3 hours. Promote to DRY or revert?"

The TTL ensures emergency overrides do not silently become permanent. The mutation ledger ensures they are visible and auditable. The field-origin map ensures the system knows where to promote the change when the emergency is over.

---

## 7. cub-track as the Redirection Layer

cub-track's mutation ledger gains a new role: when someone edits WET directly for a field that has a DRY origin, cub-track can detect this and redirect.

### explain --fields

**`cub-track explain --commit <sha>`** already shows intent/decision/outcome. With field-origin awareness, it adds: "This commit changed `replicas` in the WET manifest. The DRY source for this field is `values-prod.yaml:14` in `acme/platform-config`." The user knows where to go next time.

This is not a hard block -- the WET change is already committed. It is post-hoc education. The next time the user wants to change replicas, they know the DRY source exists and where to find it.

### suggest

**`cub-track suggest`** (post-MVP): before committing a WET change, check provenance. If the field has a DRY origin, surface it: "You are editing generated output. The source for this field is X. Edit there instead?" This creates friction that pushes users back to DRY without blocking them.

The friction is intentional and calibrated:

- It does not prevent the commit. Emergency overrides must be possible.
- It does log the bypass. The mutation ledger records that a WET edit was made despite a DRY origin being available.
- It does create visibility. Dashboards can show "WET edits to generated fields" as a metric, surfacing where DRY-source integrity is eroding.

### Overlay Drift Classification

Drift classification gets richer with field-origin awareness. When a WET field changes without `inputs.digest` changing, that is a distinct signal: someone edited the output, not the input. cub-track can classify this as "overlay drift from DRY source" -- a different category from "runtime drift from intended state."

The classification taxonomy:

| Category | What happened | Signal |
|----------|--------------|--------|
| **Runtime drift** | Cluster state differs from intended state (WET) | cub-scout detects mismatch |
| **Overlay drift** | WET was edited directly for a field with DRY origin | `inputs.digest` unchanged, WET field changed |
| **Stale render** | DRY inputs changed but WET was not re-rendered | `inputs.digest` changed, WET unchanged |
| **WET-native edit** | WET field with no DRY origin was edited | No field-origin entry, change is expected |

Overlay drift is the category that matters most for DRY-source integrity. It is the signal that someone bypassed the generator pipeline. The system does not block it, but it classifies it, logs it, and creates friction to promote the change back to DRY.

---

## 8. Staleness Detection

"Is the WET in ConfigHub stale relative to DRY inputs?" is a ConfigHub/CI concern, not a cub-scout concern. cub-scout observes the cluster; it does not know about the generator pipeline or DRY sources.

### The Mechanism

The staleness check is straightforward: compare `inputs.digest` at render time vs current DRY inputs. If the digest changed but the WET has not been re-rendered, the stored WET is stale. This belongs in the publishing pipeline -- a pre-publish check that detects when generator inputs have changed but the output has not been re-rendered.

### Ownership Boundary

| Concern | Owned by | Mechanism |
|---------|----------|-----------|
| "Is the cluster running what we intended?" | **cub-scout** | Observes cluster, compares to WET |
| "Is the WET stale relative to DRY inputs?" | **ConfigHub** (publishing pipeline) | Compares `inputs.digest` at render time vs current |
| "Did someone edit WET directly for a generated field?" | **cub-track** | Overlay drift classification |

This boundary matters. cub-scout does not need to know about generators, DRY sources, or field-origin maps. It compares cluster state to intended state (WET). Staleness -- the gap between DRY and WET -- is a different concern, owned by a different component.

### Integration with the Publishing Pipeline

Staleness detection fits naturally into the publishing pipeline. The check point is:

1. A DRY input changes (e.g., someone updates `values-prod.yaml` in Git).
2. The publishing pipeline computes a new `inputs.digest` from the current DRY inputs.
3. It compares this digest against the `inputs.digest` stored with the most recent WET render.
4. If the digests differ, the WET is stale. The pipeline can: (a) auto-re-render if policy allows, (b) notify the owner, or (c) block promotion until re-rendered.

This is a CI/pipeline concern. It runs in the publishing workflow, not in the cluster observer. The separation keeps cub-scout focused on runtime observation and keeps ConfigHub focused on the DRY-to-WET pipeline.

---

## 9. Per-Variant Changes Belong in DRY

The default path for variant-specific changes should be the generator's input surface, not post-generation patches. Helm already supports this -- per-environment values files (`values-dev.yaml`, `values-prod.yaml`). Score and Spring Boot generators can support variant-specific inputs the same way.

### Extend the Input, Not the Output

When the generator does not have a knob for a needed change, the first response is to **extend the generator input schema** -- not to overlay the output. WET-space overlays are a transitional escape hatch. The system should create friction (evidence, staleness detection, cub-track redirection) that encourages promotion back to DRY.

### The cub edit Experience

The ideal UX makes this seamless:

```
$ cub edit payments-api --field spec.replicas --variant prod

> This field is generated from: values-prod.yaml:14 (repo: acme/platform-config)
> Current value: 2
> New value: 5

> Re-rendering with updated input...
> Diff:
    spec.replicas: 2 -> 5
    (no other fields affected)

> Commit to acme/platform-config? [y/n]
```

The user never touches WET. They express intent, the system resolves the DRY source via the field-origin map, re-renders, shows the WET diff, and commits to the right place. The generator contract is preserved. Provenance stays clean.

This is the same flow as ConfigHub's three-step editing experience (section 4), but from the CLI. The field-origin map powers both surfaces. The underlying mechanism is identical: resolve the DRY source, modify the input, re-render, show the diff, commit.

### When Overlays Are Acceptable

WET overlays are not forbidden. They are a transitional state with increasing friction:

1. The overlay is recorded in the mutation ledger (cub-track).
2. cub-track classifies it as overlay drift from DRY source.
3. Staleness detection flags that WET diverges from what the generator would produce.
4. `cub-track suggest` prompts the user to promote the change to DRY next time.

The system creates a gravity well that pulls changes back toward DRY. Overlays are permitted but not encouraged. Over time, the natural path becomes editing DRY through ConfigHub or `cub edit`.

---

## 10. Addressing DRY Author Concerns

The concern: DRY authors will not be happy when WET gets customized. If someone writes careful Helm values and then someone else edits the WET output directly, the DRY source becomes fiction. This is a legitimate worry.

Three answers:

### Answer 1: ConfigHub Edits Write Back to DRY

ConfigHub is where the user edits, but it writes back to DRY. The flow:

1. User sees `replicas: 2` in ConfigHub (the WET view)
2. User clicks to change it to `5`
3. ConfigHub resolves the field-origin map: this value comes from `values-prod.yaml:14` in `acme/platform-config`
4. ConfigHub opens a PR to update `values-prod.yaml` line 14 from `2` to `5`
5. PR merges, generator re-renders, WET updates

DRY stays the source of truth. The user just did not have to find the right file, line, and repo themselves. ConfigHub was the *interface*, not the *source*.

### Answer 2: WET Overlays Are a Transitional Escape Hatch, Not a Workflow

When someone does edit WET directly for a field that has a DRY origin, the system creates friction:

- **cub-track suggest** (post-MVP): "You are editing generated output. The source for this field is `values-prod.yaml:14`. Edit there instead?"
- **Drift classification**: cub-track classifies this as "overlay drift from DRY source" -- a distinct, visible category that signals the DRY source is going stale.
- **Staleness detection**: the `inputs.digest` has not changed but the WET has, making the divergence visible in dashboards and queries.

WET overlays are permitted (blocking them entirely would prevent emergency fixes), but the system flags them, classifies them, and creates friction to promote them back to DRY. They are not a sustainable workflow.

### Answer 3: Some Fields Correctly Have No DRY Source

For fields that are legitimately WET-native -- platform policy, emergency overrides, deployment topology, variant lifecycle, governance state -- editing in ConfigHub directly is correct. There is no DRY source to redirect to. The field-origin map distinguishes these cases:

- If `field_origins[spec.replicas].source` is `values-prod.yaml`, the edit routes to DRY.
- If `field_origins[spec.resources.limits.memory].source` is `platform-context.yaml` with `editable_by: platform-team`, the edit happens in ConfigHub directly.
- If there is no field-origin entry (identity generator, imported WET), ConfigHub is the editing surface and that is correct.

The user experience is: **edit in one place (ConfigHub), the system routes the change to the right layer.** DRY authors keep their source of truth. Platform teams keep their direct control. Emergency overrides happen where they need to happen, with TTLs and governance. The field-origin map is what makes this routing possible.

---

## 11. Cross-References

This document covers field-origin maps and the editing experience they enable. Related topics are covered in companion documents:

- **[01 -- Introducing Agentic GitOps](/docs/agentic-gitops/01-vision/01-introducing-agentic-gitops.md)** -- why agentic GitOps exists, classical GitOps gaps, the three invariants, and the generator concept
- **[02 -- Generators PRD](/docs/agentic-gitops/02-design/10-generators-prd.md)** -- generator definition, maturity levels, provenance fields, authoring landscape, and GeneratorOutput envelope
- **[04 -- App Model and Contracts](/docs/agentic-gitops/02-design/30-app-model-and-contracts.md)** -- entity definitions (App, Deployment, Unit, Variant), operating boundary, constraints, operations
- **[05 -- cub-track](/docs/agentic-gitops/05-rollout/10-cub-track.md)** -- standalone mutation ledger, Change Interaction Cards, overlay drift detection, the explain and suggest commands
- **[06 -- Governed Execution](/docs/agentic-gitops/02-design/40-governed-execution.md)** -- two-loop model, evidence bundles, trust tiers, attestation, write-back semantics
- **[07 -- User Experience](/docs/agentic-gitops/05-rollout/30-user-experience.md)** -- four surfaces, two personas, import from Git flow, field-origin edit mockups, AI tooling
- **[08 -- Adoption and Reference](/docs/agentic-gitops/05-rollout/40-adoption-and-reference.md)** -- adoption path, value-per-level table, pricing boundary, worked examples, FAQ

### Key Dependencies

| This document's concept | Depends on | Defined in |
|------------------------|------------|------------|
| Field-origin map schema | Generator model, provenance fields | [02 -- Generators PRD](/docs/agentic-gitops/02-design/10-generators-prd.md) |
| ConfigHub editing surface | App, Deployment, Unit entities | [04 -- App Model and Contracts](/docs/agentic-gitops/02-design/30-app-model-and-contracts.md) |
| cub-track redirection | Mutation ledger, Change Interaction Cards | [05 -- cub-track](/docs/agentic-gitops/05-rollout/10-cub-track.md) |
| Trust tiers for edit approvals | Governed execution model | [06 -- Governed Execution](/docs/agentic-gitops/02-design/40-governed-execution.md) |
| Edit UX mockups | Four surfaces, persona model | [07 -- User Experience](/docs/agentic-gitops/05-rollout/30-user-experience.md) |
| Adoption ladder business case | Value analysis, pricing boundary | [08 -- Adoption and Reference](/docs/agentic-gitops/05-rollout/40-adoption-and-reference.md) |
