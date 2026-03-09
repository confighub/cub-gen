# Adoption Path, Value Analysis, and Reference

> How to adopt, what value each level delivers, worked examples, key concepts, and FAQ.

**Part of:** [AI and GitOps v7 Document Set](../00-index/00-gitops7-index.md)
**Status:** Planning doc (v7)
**Date:** 2026-02-28
**Audience:** Decision makers, implementation teams
**Purpose:** How to adopt, what value each level delivers, pricing boundary, worked examples, key concepts, and FAQ

---

Qualification rule:

Use `Agentic GitOps` only when an active inner reconciliation loop
(`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that
loop, classify the flow as `governed config automation`.

---

## Table of Contents

- [1. Adoption Path](#1-adoption-path)
- [2. Value at Each Level](#2-value-at-each-level)
- [3. The Pricing Boundary](#3-the-pricing-boundary)
- [4. Lock-In Addressed](#4-lock-in-addressed)
- [5. Worked Example: Score.dev End-to-End](#5-worked-example-scoredev-end-to-end)
- [6. Summary of Key Concepts](#6-summary-of-key-concepts)
- [7. Frequently Asked Questions](#7-frequently-asked-questions)
  - [7.1 Why is agentic GitOps bidirectional when classical GitOps is not?](#71-why-is-agentic-gitops-bidirectional-when-classical-gitops-is-not)
  - [7.2 How is this different from Helm/Kustomize?](#72-how-is-this-different-from-helmkustomize)
  - [7.3 How is this different from ArgoCD/Flux?](#73-how-is-this-different-from-argocdflux)
  - [7.4 Should I adopt config-as-data if I'm doing agentic GitOps?](#74-should-i-adopt-config-as-data-if-im-doing-agentic-gitops)
  - [7.5 How would an enterprise verification system work?](#75-how-would-an-enterprise-verification-system-work)
  - [7.6 This sounds like an IDP. Is it?](#76-this-sounds-like-an-idp-is-it)
  - [7.7 Won't this become a controller?](#77-wont-this-become-a-controller)
  - [7.8 What about secrets?](#78-what-about-secrets)
  - [7.9 Can I use this without ConfigHub?](#79-can-i-use-this-without-confighub)
  - [7.10 What happens if I stop using ConfigHub?](#710-what-happens-if-i-stop-using-confighub)
  - [7.11 Why ConfigHub instead of building all of this on Git?](#711-why-confighub-instead-of-building-all-of-this-on-git)
  - [7.12 What happens when a generator has a bug?](#712-what-happens-when-a-generator-has-a-bug)
  - [7.13 How does this handle multi-cluster?](#713-how-does-this-handle-multi-cluster)
  - [7.14 How should platform engineers and portals adopt this?](#714-how-should-platform-engineers-and-portals-adopt-this)
  - [7.15 Do I keep using Helm?](#715-do-i-keep-using-helm)
  - [7.16 Where do I edit config? DRY source or ConfigHub?](#716-where-do-i-edit-config-dry-source-or-confighub)
  - [7.17 My company wants their own internal Heroku. How do I do that?](#717-my-company-wants-their-own-internal-heroku-how-do-i-do-that)
  - [7.18 What is app config in this model?](#718-what-is-app-config-in-this-model)
  - [7.19 What is the right model for ConfigHub Actions?](#719-what-is-the-right-model-for-confighub-actions)
- [8. Related Documents](#8-related-documents)
- [9. Cross-References](#9-cross-references)

---

## 1. Adoption Path

Adoption is incremental. Each step adds capability without requiring migration from
the previous step. You keep your existing tools and gain governance around them.

### Step 1: Add today (no migration)

Assumed baseline for these demos/examples: Kubernetes + Flux/Argo + Git + OCI
already exist. Helm is common but optional.

1. Keep current Flux/Argo runtime flow
2. Add `cub-track` for commit-linked explain/search
3. Gain immediate "why" and provenance visibility for AI-assisted changes

Message: *"Anyone can add this to Flux or Argo today, and we'll soon have an agentic-only model too."*

### Step 2: Add governance controls

1. Connect policy scan and decisioning (`confighub-scan`)
2. Enforce trust tiers for apply rights
3. Write compact receipts back to Git

### Step 3: Generators as explicit platform contract

1. Name existing Helm/Kustomize/Argo rendering as generators (discovery, not invention)
2. Add framework generators for Spring Boot, Score, etc.
3. Store generator output as Units with provenance metadata + field-origin maps

### Step 3a: ConfigHub as the editing surface for app config

1. Users view deployed state in ConfigHub (WET, queryable, cross-variant)
2. Users trace fields back to DRY sources via field-origin maps
3. Users edit app config *through* ConfigHub -- it commits to the DRY source in Git
4. Generator re-renders, WET updates, reconciler applies

This step builds trust incrementally (View -> Compare -> Trace -> Edit -> Govern)
starting with app config -- the highest-frequency, lowest-stakes edits.

### Step 4: Full governed execution

1. Maintain explicit intended state and policy gates
2. Support execution paths with attested, token-scoped runtime execution
3. Preserve the same invariants and audit chain whether reconciled by Flux/Argo or executed directly

---

## 2. Value at Each Level

Each adoption step maps to a maturity level, a business outcome, and a boundary
between what is free and what requires ConfigHub.

| Level | Capability | Business Outcome | Free / ConfigHub |
|-------|-----------|-----------------|------------------|
| **0. OSS tools** | cub-scout observation, cub-track local mutation history | Dev goodwill, community adoption, per-repo visibility | Free (OSS) |
| **1. Capture** | Rendered manifests stored as Units | "What did Flux/Argo produce?" -- answer in one query, not kubectl archaeology | ConfigHub (shipped) |
| **2. Provenance** | Four fields + field-origin maps on Units | "What inputs produced this?" / stale render detection / trace any field to DRY source | ConfigHub (planned) |
| **2a. Edit** | ConfigHub resolves field origins, commits to DRY source in Git | "Change this value" -- without knowing which repo/file/line; ConfigHub handles it | ConfigHub (planned) |
| **3. Pre-publish** | Validate rendered output against platform constraints before publish | "Will this break?" -- caught at render time, not after deploy | ConfigHub (planned) |
| **4. Governed** | Policy gates, trust tiers, attestation, mutation ledger | Audit-grade compliance: who authorized, what checks ran, what proof exists | ConfigHub (planned) |
| **Enterprise** | Retention, RBAC, fleet analytics, compliance exports | Org-wide visibility, regulatory reporting, cross-team governance | ConfigHub Enterprise |

---

## 3. The Pricing Boundary

**The generator boundary still defines value, but storage is now dual-mode.**

Authoring-side workflows remain open and OSS-friendly. ConfigHub captures control-plane value by storing and governing both dry units and wet units, with WET as the deployment contract.

```
Authoring ingress (free / OSS-friendly) | Control plane (paid / ConfigHub)
                                         |
Git repos, PRs, existing generator toolchains | Dry units + wet units as queryable Units
Helm, Kustomize, Score, Spring Boot, scripts  | Cross-repo, cross-cluster queries + provenance graph
cub-scout standalone (OSS)                    | Verification + attestation indexing
cub-track Stage 0 (OSS, local)                | Policy decisioning and trust-tier governance
Git trailers + receipts                       | Retention, compliance export, fleet analytics
```

- **Authoring ingress remains free / OSS-friendly.** Developers keep their tools: Helm, Kustomize, Score, Spring Boot, and internal scripts. Git remains the primary ingress for most teams, but it is not the only possible DRY entry point.
- **ConfigHub control-plane value is paid.** ConfigHub stores dry units and wet units, keeps WET authoritative for deployment, and adds queryability, provenance, stale render detection, field-origin editing, constraint validation, policy governance, verification, attestation, and audit.

### Value by Buyer Persona

| Buyer Persona | What They Care About | Value ConfigHub Delivers |
|---------------|---------------------|--------------------------|
| **Platform Engineer** | Fleet visibility, constraint enforcement, generator catalog | Cross-team query, constraint validation, generator version tracking. "I can define what my platform enforces and see what every team actually runs." Replaces tribal knowledge and PR review as governance. |
| **App Team Lead** | "Where does this value come from?", editing without Git archaeology | Field-origin maps, ConfigHub editing surface, cross-env diff. "I can see what's different between staging and prod without kubectl." Replaces manual debugging. |
| **SRE / On-Call** | "What changed in the last 24 hours?", 3am incident response | Provenance audit trail, evidence bundles, revision history. "When a Deployment breaks at 3am, I can see what changed, what generator produced it, and what inputs went in." Reduces MTTR. |
| **CISO / Compliance** | "Who authorized this change? What proof exists?" | Trust tiers, attestation, mutation ledger, compliance exports. "Every config change has a who, what, why, and proof chain." Audit-grade governance. |
| **VP Engineering** | "What's deployed where? How many stale renders?" | Fleet-wide dashboard, version tracking, stale detection. "We have 200 microservices across 40 repos. I can see what's deployed everywhere, what's drifted, and what's ungoverned." |

---

## 4. Lock-In Addressed

Lock-in is a legitimate concern. This section addresses it directly.

### What you keep if you leave ConfigHub

- **All Git-authored DRY inputs** (charts, values files, Score workloads, framework config) -- still in Git
- **Any ConfigHub-authored dry units** -- exportable as open structured artifacts
- **All generator code** -- in Git or your own registries
- **All cub-track mutation history** -- in Git (trailers, receipts, metadata branch)
- **All Units** -- exportable as `GeneratorOutput` envelopes (structured YAML)
- **All evidence bundles** -- structured YAML, exportable via API

### What you lose

- Cross-repo, cross-cluster queries ("show me everything labeled `app=payments`")
- Policy evaluation at write time
- Centralized provenance index and stale render detection
- Evidence correlation across environments
- Retention policies and compliance exports
- Trust tiers and governed execution

### The principle

The value is in the centralized queries and governance -- the platform layer on
top of your data. If you leave, you lose the platform, not the data. All formats
are open and all data is exportable.

Nothing is proprietary format. Your generators and templates can stay in Git -- always yours. ConfigHub-authored dry units are exportable. Units can be exported as `GeneratorOutput` envelopes (YAML). Evidence bundles are structured YAML. If you stop using ConfigHub, your authoring workflow does not change. What you lose is the centralized visibility and governance that makes the data actionable at scale.

---

## 5. Worked Example: Score.dev End-to-End

This walkthrough follows a single change through all seven stages of the model,
from developer intent to recorded evidence.

### Step 1: Developer writes DRY intent

```yaml
# score.yaml
apiVersion: score.dev/v1b1
kind: Workload
metadata:
  name: payments-api
spec:
  image: ghcr.io/acme/payments-api:1.2.3
  ports:
    - port: 8080
  service:
    expose: true
    host: pay.example.com
  networking:
    useMesh: true
```

### Step 2: Platform provides context

```yaml
# platform-context.yaml
labels:
  environment: production
constraints:
  - tls-required-in-prod
  - min-replicas-2
  - mesh-allowed
  - require-zone-spread
```

### Step 3: Generator renders WET

```bash
score-gen render --input score.yaml --context platform-context.yaml --out generated.yaml
```

Output includes:
- Deployment with `sidecar.istio.io/inject: "true"` (mesh = visible annotation)
- Service, Ingress with TLS (constraint enforced)
- `replicas: 2` (constraint enforced)
- Pod anti-affinity for zone spread (constraint enforced)
- Provenance metadata on the Unit: generator name, version, and input digest

### Step 4: ConfigHub stores Unit, publishes artifact

ConfigHub stores the output as a Unit with labels (`app=payments-api`,
`variant=prod`, `framework=score`) and provenance metadata (generator name,
version, input digest). For Flux/Argo consumption, the Unit is published as an
OCI artifact -- exported as a `GeneratorOutput` envelope if it needs to be
written to Git.

### Step 5: Flux/Argo reconcile

The published artifact is detected. Flux or Argo reconcile the cluster.

### Step 6: cub-scout observes

If someone disables mesh injection manually:

```yaml
# Evidence
kind: EvidenceBundle
type: drift
observation:
  differences:
    - resource: Deployment/payments-api
      field: metadata.annotations.sidecar.istio.io/inject
      expected: "true"
      observed: "false"
      classification:
        type: modified
        likely_cause: manual_edit
provenance:
  intended_operations: ["require(mesh)"]
```

The system does not auto-fix or silently accept the drift. Humans or policy decide
how to respond.

### Step 7: cub-track records mutation context

If the original change was AI-assisted, `cub-track` links the commit to a
ChangeInteractionCard recording: what was intended, what risk checks ran, what
decision was made, and what the outcome was.

---

## 6. Summary of Key Concepts

| Concept | Definition |
|---------|------------|
| **App** | Named collection of components, queried by label |
| **Deployment** | App x Target (environment instance) |
| **Unit** | Atomic deployable config with labels and provenance |
| **Generator** | Deterministic function: intent + context -> WET |
| **Field-origin map** | Generator-produced mapping from WET output fields to DRY input sources; enables tracing and editing |
| **Operation** | SDK method that produces diffable config (intent, not execution) |
| **WET** | Explicit manifests -- what actually deploys |
| **DRY** | Developer intent: templates, values, workload specs -- authored, not deployed directly |
| **Evidence** | cub-scout observation: structured diff + provenance (cluster vs. intended state) |
| **Overlay drift** | WET field changed without DRY input change; transitional state, not steady-state |
| **Change Interaction Card** | cub-track mutation record: intent + decision + execution + outcome |
| **Receipt** | Compact DRY write-back to Git, linking to full WET in ConfigHub |

---

## 7. Frequently Asked Questions

### 7.1 Why is agentic GitOps bidirectional when classical GitOps is not?

Classical GitOps is deliberately one-directional: Git is the single source of
truth, reconcilers pull from it and converge the cluster toward it. If someone
changes something in the cluster directly, the reconciler overwrites it on the
next sync. That is the design -- Git wins, always.

The problem is that this breaks down in three real-world scenarios:

**1. Break-glass fixes.** An operator patches a production deployment at 3am to
stop an outage. Classical GitOps treats this as drift and reverts it. The fix
disappears. The operator learns to disable reconciliation before making emergency
changes -- which means the cluster is now unmanaged during the most critical
moments.

**2. Runtime-discovered state.** Some configuration only becomes known at
runtime -- autoscaler adjustments, certificate rotations, admission webhook
mutations. These are legitimate changes that originate in the cluster, not in
Git. Classical GitOps either fights them (revert loop) or ignores them (fields
excluded from sync, which creates invisible blind spots).

**3. AI agents acting on live systems.** An agent observes a problem, proposes a
config change, and wants to apply it. In classical GitOps, the agent must commit
to Git first and wait for reconciliation. But for time-sensitive operational
changes, that round-trip may be too slow -- and the agent may not have Git write
access, or the change may need approval before it reaches Git.

In all three cases, the information flow needs to go **cluster -> intended
state**, not just **intended state -> cluster**. That is the reverse direction.

The reason agentic GitOps makes this more acute is volume: a human might make a
break-glass fix once a month. An AI agent might propose runtime-informed changes
dozens of times a day. Without a governed reverse path, you either block the
agent from acting (defeating the purpose) or let it act outside the system of
record (defeating governance).

The governed reverse flow in this model works like this: the change happens,
cub-scout observes it, evidence is produced, and that evidence triggers an
explicit proposal back to intended state -- a merge request, not a silent
overwrite. The proposal is reviewed (by human or policy) and accepted or
rejected. If accepted, intended state is updated to match. If rejected, a revert
is proposed instead.

Neither direction is automatic. Both are governed. That is what makes it
bidirectional rather than just "Git wins" or "cluster wins."

---

### 7.2 How is this different from Helm/Kustomize?

Helm and Kustomize are generators -- they produce config from templates and
overlays. But they do not give you the provenance wrapper (generator version +
input digest + operations list), they do not store the output as intended state in
a system of record separate from "latest commit on main," and they do not produce
evidence when things drift. This model completes the loop: generate, store with
provenance, publish, reconcile, observe, and record evidence.

You do not replace Helm or Kustomize. You store their output as a Unit with
provenance metadata (generator name, version, input digest), and gain the audit
trail they do not provide on their own. For Git or OCI, export as a
`GeneratorOutput` envelope.

---

### 7.3 How is this different from ArgoCD/Flux?

Flux and Argo are reconcilers -- they make runtime match a declared source. This
model does not replace them; reconciliation remains their job (the "Reconcile" row
in the comparison table is unchanged). What this model adds is the governance
layer around reconciliation: capturing intent before publish, producing structured
evidence after reconcile, and recording the full mutation chain (who proposed,
what policy evaluated, what outcome resulted) in a way that Git logs and
Flux/Argo events alone cannot.

---

### 7.4 Should I adopt config-as-data if I'm doing agentic GitOps?

Yes. Agentic GitOps increases mutation volume -- AI agents may author dozens or
hundreds of configuration changes per day. This creates four requirements that
templates-in-Git alone cannot satisfy:

1. **Queryable output.** When an agent produces a config change, reviewers and
   policies need to inspect the actual field values, not a template that still
   needs rendering. Config-as-data means the output is literal, queryable, and
   diffable without running a generator first.

2. **Pre-publish validation.** Platform constraints (resource limits, image
   registries, TLS requirements) need to evaluate the rendered output before it
   reaches a cluster. This requires the output to exist as structured data, not
   as a pending render.

3. **Provenance and audit.** When something breaks, you need to trace which
   generator version, which inputs, and which agent produced the current state.
   Provenance metadata on Units (generator name, version, input digest, render
   timestamp) answers this directly. Git commit history records *that* a file
   changed; provenance records *why* and *from what*.

4. **Cross-fleet visibility.** "What is deployed where?" across 40 repos and
   multiple clusters is an instant query over Units. In Git, it requires
   scanning file trees across repos -- a project, not a query.

The generator framing makes adoption incremental. If you already use Helm or
Kustomize, you already produce config-as-data -- `helm template` outputs literal
manifests. The adoption path starts by capturing that output as Units (Level 1,
no migration required) and adds provenance, validation, and governance as
maturity increases.

For non-deterministic generators (LLMs), the same provenance fields apply but
the determinism guarantee does not hold -- re-rendering the same prompt may
produce different output. Governance compensates: pre-publish validation catches
constraint violations regardless of how the output was produced, and trust tiers
require human review for AI-authored changes.

---

### 7.5 How would an enterprise verification system work?

At minimum, it runs as a gated chain:

1. Capture explicit intent and render deterministic WET artifacts.
2. Evaluate risk signals (`confighub-scan`) and semantic assertions (`verified`).
3. Produce a policy decision (`ALLOW | ESCALATE | BLOCK`) with reason codes.
4. Issue short-lived execution authority only on `ALLOW`.
5. Reconcile via Flux/Argo, observe outcome via cub-scout, and record attestations.

The key property is enforceability: verification evidence must be required before
execution authority is granted, not added after the fact as documentation.

---

### 7.6 This sounds like an IDP. Is it?

No. IDPs hide infrastructure behind portals. This model exposes infrastructure
through explicit, diffable configuration. The abstraction compresses (Score
workload -> K8s manifests); it does not conceal. If you cannot `cat` the output
and read every line, the abstraction is broken.

Backstage can be a client of this model -- capturing intent via forms and showing
receipts and evidence from ConfigHub -- but it is not the system of record. If you
delete Backstage, Units, provenance, and evidence survive intact.

---

### 7.7 Won't this become a controller?

Controllers reconcile -- they watch state and converge toward it. ConfigHub stores,
governs, and publishes. cub-scout observes and records. Neither reconciles.
Evidence informs decisions but never enacts them autonomously. The operating
boundary is explicit: reconciliation belongs to Flux/Argo, not to ConfigHub or
cub-scout.

---

### 7.8 What about secrets?

Secrets management is explicitly out of scope. ConfigHub does not manage secrets
at rest. Generators may reference secret names or external secret store paths, but
the secret values themselves are never stored in Units or provenance metadata.
Integration with external secret operators (External Secrets Operator, Sealed
Secrets, Vault) is via reference, not by value.

---

### 7.9 Can I use this without ConfigHub?

Yes. Stage 0 (OSS Local) requires only `cub-track` and works with any Git
repository. You get commit-linked mutation history, explain, and search -- with no
backend dependency. cub-scout also works standalone against any kubectl context.
ConfigHub adds centralized storage, cross-repo search, policy evaluation, and
governed execution -- but the local tools are useful on their own.

---

### 7.10 What happens if I stop using ConfigHub?

Your generators and templates can stay in Git -- always yours. ConfigHub-authored dry units are exportable. Nothing about the core authoring workflow changes if ConfigHub disappears.

**What you keep:**
- All Git-authored DRY inputs (charts, values files, Score workloads, framework config) -- still in Git
- Any ConfigHub-authored dry units -- exportable as open structured artifacts
- All generator code -- in Git or your own registries
- All cub-track mutation history -- in Git (trailers, receipts, metadata branch)
- All Units -- exportable as `GeneratorOutput` envelopes (structured YAML)
- All evidence bundles -- structured YAML, exportable via API

**What you lose:**
- Cross-repo, cross-cluster queries ("show me everything labeled `app=payments`")
- Policy evaluation at write time
- Centralized provenance index and stale render detection
- Evidence correlation across environments
- Retention policies and compliance exports
- Trust tiers and governed execution

The value is in the centralized queries and governance -- the platform layer on
top of your data. If you leave, you lose the platform, not the data. All formats
are open and all data is exportable.

---

### 7.11 Why ConfigHub instead of building all of this on Git?

Git is excellent at what it does: versioned text diffs, review workflows (PRs),
immutable history, distributed collaboration. The problem is what Git does not do
well -- and what you would have to build on top of it if you tried to make Git the
entire system of record for governed operations.

**What Git cannot do natively:**

- **Structured query across repos.** "Show me every Unit labeled `app=payments`
  across all 40 team repos" is not a Git operation. You would need an external
  index -- which is a database, which is what ConfigHub is.
- **Policy evaluation at write time.** Git accepts any commit that passes hooks.
  Governing what *content* is allowed -- constraint validation, trust tier checks,
  approval chains -- requires a layer that understands the content semantically,
  not just as text diffs.
- **Rich metadata and correlation.** Linking a commit to the evidence bundle that
  triggered it, the policy decision that approved it, and the runtime outcome that
  resulted -- across repos, across clusters -- requires relational or graph queries
  that Git's content-addressed store cannot express.
- **Retention and compliance.** Git retains everything forever (or loses it on
  rebase). Operational data needs retention policies, redaction, and export
  controls. Bolting these onto Git means building a database layer on top of Git.
- **Fast label-based lookups.** "What is deployed to `variant=prod` in
  `cluster=east-1`?" is an instant query in a database. In Git, it requires
  scanning file trees across branches and repos.

**The DRY/WET type boundary is the answer.** Git stores compact, reviewable, immutable linkage artifacts -- trailers, receipts, intent diffs -- and remains the primary collaboration surface for many teams. ConfigHub stores dry units, wet units, and rich queryable operational state -- policy traces, approval chains, execution records, verification and attestation indexes. WET in ConfigHub is the authoritative deployment contract.

The anti-pattern is trying to make Git do both jobs. That path leads to bloated
repos full of operational metadata that no one reviews in PRs, custom tooling to
query across repos, and eventually a database-on-Git that would have been simpler
to build as an actual database.

---

### 7.12 What happens when a generator has a bug?

Generator output is reproducible: given the same `generator.name`,
`generator.version`, and `inputs.digest`, the output must be byte-identical.
This means a bug is reproducible too. You can re-render the same inputs with the
same generator version and see the same bad output, then diff it against the
fixed version. The provenance record tells you exactly which Units were produced
by the buggy version, so you know what to re-render.

---

### 7.13 How does this handle multi-cluster?

Each cluster is a Target. A Deployment binds one App to one Target -- the same
App deployed to three clusters produces three Deployments, each with its own
Unit(s), Variant(s), and evidence. cub-scout in standalone mode observes one
cluster at a time. Connected mode (via ConfigHub) aggregates evidence across
clusters, enabling fleet-wide drift detection and cross-cluster comparison.

---

### 7.14 How should platform engineers and portals adopt this?

Platform engineers are the primary authors of the governed layer. Their adoption
path:

**Step 1: Name what you already have.** Most platform teams already run Helm
charts, Kustomize overlays, or internal scripts that produce K8s manifests. These
are generators -- they just are not called that yet. Register them: give each a
name, a version, and start wrapping their output with an input digest. This is
discovery, not invention.

**Step 2: Define constraints.** Write down the rules that are currently enforced
by PR review, tribal knowledge, or post-deploy panic. Express them as platform
constraints: `tls-required-in-prod`, `min-replicas-2`, `images-from-approved-
registry`. These become inputs to generators and validation gates.

**Step 3: Build framework generators.** For teams using Spring Boot, Score, or
other frameworks with strong conventions, build generators that read the framework
config and produce explicit manifests. This is where the abstraction pays off --
the developer writes what they already write, and the generator narrates it into
K8s wiring.

**Step 4: Add a portal (optional).** Backstage or any other portal can be a
client -- capturing intent via forms, invoking generators, and displaying receipts
and evidence from ConfigHub. The portal is a UX layer, not a system of record.
If you delete the portal, everything survives. If you never build a portal, the
CLI and Git workflow still work.

The key principle: platform engineers provide generators and constraints.
Application teams provide intent. The boundary between them is explicit and
inspectable.

---

### 7.15 Do I keep using Helm?

Yes. Helm is a generator. You keep using it.

The change is what happens to Helm's output. Today, `helm template` renders
manifests and either a human reviews them or Flux/Argo runs Helm directly in the
cluster. In this model, the rendered output is stored as a Unit with provenance
metadata -- generator name, chart version, values digest, render timestamp.

What you gain:

- **Audit trail.** Six months from now, you can answer "what chart version and
  values file produced this Deployment?" by looking at the Unit's provenance,
  without digging through CI logs or Helm release history.
- **Drift detection.** cub-scout can compare what Helm intended with what is
  actually running and produce evidence when they diverge.
- **Constraint validation.** Platform constraints can evaluate the rendered output
  before it publishes -- catching violations at render time, not after deploy.

What you do not gain by switching away from Helm: nothing. Helm is a fine
generator. The model is about what surrounds the generator, not what replaces it.

For Flux HelmRelease users specifically: Flux runs Helm inside the cluster (inner
loop). This means the rendered output lacks pre-publish provenance unless the
values and chart reference are themselves governed in the outer loop. The adoption
path: keep HelmRelease, but consider also storing the expected rendered output as
a Unit so you can detect drift between what Helm intended and what actually exists.

---

### 7.16 Where do I edit config? DRY source or ConfigHub?

Both -- through the same surface.

If you author config in DRY format (Helm values, Score workloads, Spring Boot
`application.yaml`), you should edit in DRY space. That is where your intent
lives. But you do not need to hunt for the right file in Git. ConfigHub resolves
it for you.

The experience: you open ConfigHub, find the deployed config, click the field you
want to change. ConfigHub's field-origin map tells you where that value comes
from -- which file, which line, in which repo. You edit the value in ConfigHub's
UI. ConfigHub commits the change to the DRY source in Git. The generator
re-renders. The WET updates. The reconciler applies.

You edited DRY, but you did it *through* ConfigHub. The generator contract is
preserved. Provenance stays clean. And you did not need to know the repo
structure, file layout, or which values overlay applies to which environment.

**What about fields with no DRY source?** Platform-injected fields (network
policy, security constraints), emergency overrides, deployment topology, and
governance state are edited directly in ConfigHub -- there is no upstream to
redirect to. ConfigHub *is* the right editing surface for these. The field-origin
map distinguishes "this has a DRY source, edit there" from "this is
control-plane-native, edit here."

**What about WET overlays?** You can patch WET output directly when needed --
emergency overrides, one-off variant customizations. But overlays on
generator-backed fields are a transitional state, not a permanent workflow. The
system creates friction (cub-track redirection, overlay drift classification,
staleness detection) that encourages promotion back to DRY.

---

### 7.17 My company wants their own internal Heroku. How do I do that?

The "internal Heroku" pattern is: developers say "deploy my app" and the platform
handles everything else. This is a legitimate goal. The question is whether the
platform hides or exposes the resulting infrastructure.

In this model, you build the Heroku experience as a generator + constraints layer:

**What the developer sees:**

```yaml
# Something simple -- Score, a custom app manifest, or even a Backstage form
name: payments-api
image: ghcr.io/acme/payments:1.2.3
port: 8080
expose: true
```

**What the platform does:**

1. A generator reads this intent plus platform context (constraints, capabilities,
   environment labels)
2. The generator produces explicit K8s manifests: Deployment, Service, Ingress,
   HPA, PDB, NetworkPolicy -- whatever the platform requires
3. The output is stored as a Unit with full provenance
4. Flux/Argo reconcile the cluster

**How this differs from actual Heroku (and from most IDPs):**

The developer *can* see the generated output. They do not have to -- the
abstraction is designed so they normally do not need to. But when something breaks
at 3am, they (or the on-call engineer) can `cat` the generated manifests and read
every line. The abstraction compresses; it does not conceal.

The platform team controls the generator and the constraints. They can change how
`expose: true` is implemented (Ingress today, Gateway API tomorrow) without
changing the developer interface. The change shows up as a diff in the generated
output, not as invisible infrastructure magic.

This is the same pattern as Score.dev ([section 5](#5-worked-example-scoredev-end-to-end) in this document) or Spring Boot
generators -- the developer writes familiar, minimal config; the platform
translates it into explicit infrastructure wiring. The "Heroku feel" comes from
the simplicity of the input, not from hiding the output.

---

### 7.18 What is app config in this model?

App config is the runtime configuration that the application reads -- environment
variables, config maps, feature flags, connection strings, tuning parameters. It
is distinct from *deployment config* (the K8s manifests that describe how the
application is deployed).

In practice, the line between them blurs. This model handles both:

**App config embedded in DRY intent.** Environment variables in a Score workload
or Spring Boot `application.yaml` are part of the generator input. The generator
renders them into ConfigMaps or env entries in the Deployment spec. They follow
the same generate -> store -> publish -> observe cycle as everything else.

**App config as a separate Unit.** Feature flags, runtime tuning, and service
config that change independently of deployments can be managed as their own Units.
They have their own provenance, their own Variants (different flags per
environment), and their own evidence when they drift.

**App config from external systems.** Secrets from Vault, flags from LaunchDarkly,
connection strings from a service mesh -- these are referenced, not stored.
Generators can produce the *references* (ExternalSecret CRs, ConfigMap entries
pointing to external sources), but the values themselves are never in ConfigHub.

The important point is that app config is just config in this model. Whether it
arrives via a generator or via direct import, it becomes a Unit with labels,
provenance, and a Variant. The same staging model applies -- promote feature flags
from `variant=dev` to `variant=prod` with the same approval policy as any other
config change.

The distinction between "app config" and "deployment config" matters to the
developer (they think about them differently) but not to the storage and
governance model (both are Units).

---

### 7.19 What is the right model for ConfigHub Actions?

ConfigHub Actions are the execution layer -- the part of the system that actually
changes runtime state. The question is how to model them so they fit the same
discipline as everything else: explicit, governed, auditable.

**The principle: actions are operational config, not imperative scripts.**

Just as a generator takes DRY intent and produces deployment manifests, an action
generator takes operational intent and produces *action manifests* -- explicit,
inspectable descriptions of what should happen:

```yaml
apiVersion: confighub.io/v1
kind: ActionManifest

metadata:
  name: rollout-payments-api
  action_type: rollout

spec:
  # What must be true before execution (assertions)
  preconditions:
    - type: health-check
      target: Deployment/payments-api
      expect: healthy
    - type: evidence-check
      expect: no-critical-drift

  # What to execute (workflow steps)
  steps:
    - name: update-image
      operation: patch
      target: Deployment/payments-api
      field: spec.template.spec.containers[0].image
      value: ghcr.io/acme/payments:1.3.0
    - name: wait-healthy
      operation: observe
      target: Deployment/payments-api
      timeout: 300s
      expect: available

  # What must be true after execution (assertions)
  postconditions:
    - type: health-check
      target: Deployment/payments-api
      expect: healthy
    - type: evidence-snapshot
      store: true
```

**Three kinds of operational content, one manifest shape:**

| Content | Purpose | Example |
|---------|---------|---------|
| **Assertions** | What must be true (pre/post) | "Deployment is healthy," "No critical drift exists" |
| **Workflow steps** | What to execute, in order | "Patch image," "Wait for rollout," "Run smoke test" |
| **Ops tasks** | Standalone operational actions | "Scale to 5 replicas," "Rotate certificate" |

All three are expressed as config -- declarative descriptions of desired
operational state changes, not shell scripts or imperative code.

**How actions fit the governed model:**

1. An action manifest is *generated* like any other config -- from intent, through
   a generator, with provenance. The generator for an action might be a framework
   SDK method (`app.rollout(image="1.3.0")`) or a workflow template.
2. The manifest is *governed* like any other config -- policy evaluates it against
   the trust tier, checks preconditions, and issues a scoped execution token.
3. The runtime (`confighub-actions`) *interprets* the manifest -- it reads the
   steps and executes them within the token's scope. The runtime is the only
   component that touches the cluster.
4. The outcome is *observed* -- cub-scout captures post-execution state, and the
   attestation records what happened against what was intended.

**Why config manifests, not scripts?**

Scripts are opaque -- you cannot diff them meaningfully, you cannot validate their
effects before execution, and you cannot compare intended vs actual at the field
level. Action manifests are structured data: you can diff them, validate them
against constraints, preview their effects, and compare the intended operation
against the observed outcome field by field.

This keeps the same invariant: nothing implicit ever executes. The action manifest
is the plan. The attestation is the receipt. The evidence is the proof.

---

### 7.20 How does this work if CI is our main platform interface?

Treat CI as the orchestration surface, not the system of record.

Your pipeline calls ConfigHub APIs for semantic operations ("set replicas to 5",
"promote change_id X", "re-render affected Units"), instead of editing files by
path. ConfigHub applies governance (policy, trust tier, decision gate), performs
deterministic write-back to Git via PR/MR, and publishes WET for Flux/Argo.

You keep CI. You remove brittle file-path logic from CI jobs.

---

### 7.21 Can we evolve labels/taxonomy without reorganizing repos?

Yes. That is one of the main reasons to store governed operational state in
ConfigHub.

Repo structures can stay team-local. Query and governance semantics are driven by
labels and typed Unit metadata (App, Deployment, Target, Variant). You can evolve
taxonomy (`service`, `domain`, `tier`, `owner`, `risk`) as policy/query contracts
without forcing repository rewrites.

---

### 7.22 How do we run fleet-wide changes (for example CVE patch waves)?

Use a governed change wave:

1. Select affected Units by query (`label`, `generator`, `image`, `variant`).
2. Create one logical wave (`change_id`) with per-target execution plans.
3. Evaluate policy per trust tier (`ALLOW | ESCALATE | BLOCK`).
4. Execute only allowed targets, with explicit escalations for protected domains.
5. Record verification + attestation for each target outcome.

This gives one auditable wave instead of dozens of ad hoc PRs with no common
identity.

---

### 7.23 Does ConfigHub bypass Git protections?

No. The model assumes Git protections remain enforced:

1. Branch protection and required reviews stay active.
2. Signed-commit requirements stay active where configured.
3. ConfigHub write-back happens via normal PR/MR workflows.
4. Merge approval and deploy decision remain separate controls.

ConfigHub adds governance around changes; it does not grant an implicit bypass.

---

### 7.24 How does LIVE-origin (kargo-style) fit this model?

LIVE-origin is a proposal path, not an overwrite path.

When cub-scout detects meaningful live drift, ConfigHub opens a proposal MR with
drift classification and inverse plan. Reviewers decide:

1. accept drift and promote it back to DRY, or
2. reject drift and revert to declared intent.

Either way, the outcome is explicit, governed, and linked to the same `change_id`
chain as forward-path changes.

---

### 7.25 Do humans, CI bots, and AI agents follow the same governance model?

Yes. Actor type changes identity metadata, not governance semantics.

Human users, CI bots (for example Renovate), and AI agents all produce mutations
that flow through the same decision states (`ALLOW | ESCALATE | BLOCK`), trust
tiers, verification, and attestation contracts. This is how audit stays coherent
at scale.

---

### 7.26 Can this run on-prem or air-gapped?

Yes. The model does not require SaaS control planes to be valid.

In on-prem/air-gapped deployments, Git, ConfigHub, OCI registry, Flux/Argo,
policy, and evidence stores run in your environment. Governance semantics stay
the same; only deployment topology changes. Evidence exports remain available for
audit and compliance workflows.

---

## 8. Related Documents

> **Note:** The files listed below are working documents in the author's local
> project directories (`~/Desktop/App Generators/` and the `cub-scout` repo).
> They are not published to any shared repository yet. If you are reading this
> document outside that context, treat the references as pointers to companion
> specs that may not be available to you.

> **Alignment note:** The published ConfigHub docs page
> (`docs.confighub.com/background/config-as-data/`) currently frames config-as-data
> by critiquing Helm and templates. This document takes the opposite approach:
> embrace Helm as a generator, and position ConfigHub as the governance layer for
> what Helm produces. The published docs should evolve to match this framing --
> "Helm is how you author. ConfigHub is how you govern. Templates produce data;
> we store and query that data." This is more persuasive than "Helm is bad."

### Generators and App Config (Desktop/App Generators/) *(External)*

| File | Purpose |
|------|---------|
| `[External] generators-v1.md` | Full conceptual guide: generators, operations, spaces, evidence |
| `[External] generator-contract-v1.md` | Output schema, determinism rules |
| `[External] operation-registry-v1.md` | How frameworks advertise operations |
| `[External] unified-app-config-model.md` | Two patterns (DRY->WET and direct WET) |
| `[External] score-generator-example.md` | Score.dev end-to-end walkthrough |
| `[External] spring-boot-generator-example.md` | Spring Boot end-to-end walkthrough |
| `[External] backstage-idp-example.md` | Backstage as ConfigHub client |
| `[External] failure-modes.md` | How failures surface and resolve |
| `[External] mental-model.md` | Core relationships |
| `[External] platform-inheritance.md` | Multi-platform constraint resolution |

### cub-track and Governed Execution (cub-scout/docs/reference/)

| File | Purpose |
|------|---------|
| `cub-track-internal-decision-memo.md` | Decision summary for Labs project |
| `cub-track-mvp-roadmap-note.md` | MVP phases and success criteria |
| `cub-track-mvp-upsell-and-dual-store.md` | Adoption stages and DRY/WET data model |
| `stored-in-git-vs-confighub.md` | Storage boundary definition |
| `gitops-checkpoint-prd.md` | Legacy naming draft (checkpoint term; superseded by mutation-ledger terminology) |
| `gitops-checkpoint-schemas.md` | Legacy checkpoint schemas (reference during migration to mutation/card schema) |
| `ai-and-gitops-v2-draft.md` | Earlier v2 draft (superseded by this document set) |

### cub-scout and Evidence (cub-scout/docs/)

| File | Purpose |
|------|---------|
| `reference/evidence-export-v1.md` | Evidence bundle export format (current: BundleSummary) |
| `reference/next-gen-gitops-ai-era.md` | Explainer: next-gen GitOps in AI era |
| `getting-started/app-and-ai-gitops-plain-english.md` | Plain English explainer |
| `reference/glossary.md` | ConfigHub concepts glossary |

---

## 9. Cross-References

This document is part of an 8-document set for ConfigHub's agentic GitOps strategy:

| Doc | Title | Focus |
|-----|-------|-------|
| [01](../01-vision/01-introducing-agentic-gitops.md) | Introducing Agentic GitOps | Why: classical gaps, agentic changes, invariants |
| [02](../02-design/10-generators-prd.md) | Generators and Provenance PRD | How: DRY->WET pipeline, maturity levels, generator contract |
| [03](../02-design/20-field-origin-maps-and-editing.md) | Field-Origin Maps and the Editing Experience | Tracing: field-origin maps, editing surface, DRY-source integrity |
| [04](../02-design/30-app-model-and-contracts.md) | ConfigHub App Model and Contracts | What: entities, operating boundary, constraints, operations |
| [05](../05-rollout/10-cub-track.md) | cub-track: Git-Native Mutation Ledger | Provenance: commit-linked mutation history, change interaction cards |
| [06](../02-design/40-governed-execution.md) | Governed Execution | Trust: two-loop model, evidence, attestation, write-back semantics |
| [07](../05-rollout/30-user-experience.md) | User Experience | Feel: four surfaces, two personas, import flow, AI tooling |
| **[08](../05-rollout/40-adoption-and-reference.md)** | **Adoption Path, Value Analysis, and Reference** | **This document** |
