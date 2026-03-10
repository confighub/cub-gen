# AI and GitOps: Intent, Generators, and Governed Execution

**Status:** Planning doc (v6)
**Date:** 2026-02-25
**Supersedes:** GitOps5.md (sectioned draft), GitOps4.md (sectioned draft), GitOps3.md (sectioned draft), GitOps2.md (structured draft), GitOps.docx (narrative draft)
**Context:** App Generators specs, cub-track decision memos, ConfigHub App model update

---

Qualification rule:
Use `Agentic GitOps` only when an active inner reconciliation loop (`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that loop, classify the flow as `governed config automation`.

---

# Part I: Narrative & Positioning

> Our central claim: **configuration is data** — literal values, not
> parameterized templates, stored where they can be queried, versioned, and
> governed. Generators are how you get there: deterministic functions that turn
> intent into explicit, deployable configuration. If you use Helm, you already
> have one.

## 1. You Already Have Generators

You use `helm template` every day:

```
chart/ + values.yaml  →  helm template  →  rendered manifests
```

That's a generator: a **deterministic, side-effect-free function** that, given a
set of inputs, produces deployment manifests.

Kustomize is the same pattern:

```
base/ + overlays/  →  kustomize build  →  patched manifests
```

What comes out is a **deployment manifest**: declarative, fully rendered, no
unresolved placeholders or macros (references like `secretKeyRef` are valid —
they point to runtime values, not template variables). No programming constructs:
no conditionals, loops, functions. This is **WET** (Write Everything Twice) —
explicit, deployable, auditable.

The formal definition:

```
WET = generate(intent, context)
```

Where:
- `intent` = DRY developer wishes (Score workload, Spring Boot config, Backstage form, Helm values)
- `context` = environment labels, platform constraints, policies
- `WET` = explicit manifests + provenance metadata

> **Note:**
> - Generators receive all context as explicit input.
> - They **never** discover or infer context from live systems.

### We want to track provenance

You already use generators, but you probably do not track their provenance.

Why does this matter? Because what comes out of the generator — the WET
manifest — is **configuration as data**. It is not a template to be rendered
later, and not a script to be interpreted. It contains literal values for every
field. You can diff it, query it across every cluster and environment, validate
it against platform constraints, and govern who changes it — all through an API,
not by grepping Git repos.

For many people in the cloud native industry, "templates" are known as how teams
*author* config. Meanwhile data is how platforms *govern* config. The generator
joins the two, and constitutes the boundary between the two.

Once the output is data, it belongs in a system that treats it as data — with
revision history, label-based queries, cross-environment comparison, and policy
evaluation at write time. That's what ConfigHub is: a database, API, and SDK for
configuration data.

---

## 2. Generator Maturity Levels

Generators are not all-or-nothing. ConfigHub's shipped **rendered manifests**
feature (`docs.confighub.com/guide/rendered-manifests/`) is the starting point.
Each level adds capability that answers a question the previous level cannot:

| Level | What | Today? | What You Can Now Answer |
|-------|------|--------|----------------------|
| **1. Capture** | Renderer unit → output unit (MergeUnits + UseLiveState) | **Shipped** | "What did Flux/Argo actually produce?" |
| **2. Provenance** | + generator name, version, input digest on the output unit | Planned | "What inputs produced this? Has the render gone stale?" |
| **3. Pre-publish** | + render before publish, validate against constraints | Planned | "Will this violate platform rules?" — before deploy, not after |
| **4. Governed** | + policy gates, attestation, trust tiers | Planned | "Who authorized this, what checks ran, what proof exists?" |

**Why Level 2?** Level 1 shows you the rendered output, but six months later
someone asks "what chart version and values produced this broken Deployment?"
Without an input digest, you're doing archaeology through CI logs. You also
can't detect stale renders — inputs changed but nobody re-rendered. Concretely:
when a chart version bumps from v28 to v29 but values stay the same, Level 1
shows the output changed but not why. Level 2's input digest changes because
the chart version is a different input — making the cause visible and
triggerable.

**Why Level 3?** "This Deployment has no resource limits" is an incident if
caught after deploy. It's a failed validation if caught before publish. Same
problem, different blast radius. Pre-publish rendering lets you diff the change
and validate it against platform constraints before it reaches the cluster.

**Why Level 4?** When AI agents make dozens of config changes a day, "who
authorized this and what proof exists?" becomes a compliance requirement. The
volume of agentic mutation makes governance non-optional — not because any
single change is dangerous, but because the aggregate risk of ungoverned
changes at that velocity is unacceptable.

Level 1 is valuable on its own. Each subsequent level is justified by the
questions it answers that the previous level cannot.

---

## 3. Four Fields That Complete the Picture

### Provenance metadata on Units

Units already have revision history, mutation sources, tags, and values maps.
Four more fields complete the picture — they record *how* the Unit's current
content was produced. Without them, a Unit is a snapshot with no memory of its
origin.

| Field | Type | Why You Need It |
|-------|------|-----------------|
| `generator.name` | string | Which generator produced this Unit? Without it, you can't answer "was this Helm, Score, or Spring Boot?" when three teams use different generators for the same app. |
| `generator.version` | string | Which version? For Helm: `(CLI version, chart repo, chart name, chart version)`. When a chart bump breaks prod, this tells you which Units were rendered by the bad version — and which need re-rendering. |
| `inputs.digest` | SHA-256 | Hash of all normalized inputs. Detects stale renders (inputs changed, nobody re-rendered). Also the cache key — skip rendering if the digest hasn't changed. |
| `rendered.at` | ISO 8601 | When was this rendered? Combined with `inputs.digest`, tells you whether a Unit is fresh or stale. |

> **Maturity note:** These fields apply at **Level 2+**. Level 1 (rendered
> manifests today) captures output without provenance. Adding these fields to
> existing Units is the natural next step.

### Field-origin map (optional, Level 2+)

The four provenance fields tell you *that* a generator produced the Unit. They
don't tell you *which input field controls which output field*. A **field-origin
map** bridges this gap — a mapping from output fields to their DRY source,
produced by the generator alongside the WET output:

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

Field-origin maps enable two capabilities that provenance alone cannot:

1. **Traceable editing.** When a user sees `replicas: 2` in a deployed manifest
   and wants to change it, the field-origin map resolves the DRY source
   instantly: "this value comes from `values-prod.yaml` line 14 — edit there."
   Without the map, this resolution requires manual archaeology.

2. **Edit-surface routing.** Fields with `editable_by: platform-team` vs
   `editable_by: app-team` tell ConfigHub who should change what. A platform-
   injected field (resource limits, network policy) is not the app team's to
   edit; the map makes this explicit.

Not all generators can produce complete field-origin maps. The identity generator's
map is trivial (every field maps to itself). Score and Spring Boot generators have
clean mappings by design. Helm is harder — but the common fields teams actually
edit (replicas, image, resources, ports, env vars) are tractable. Start with the
fields people change, not every field.

Field-origin maps are optional at Level 2. They become increasingly valuable at
Level 3+ where ConfigHub can use them to route editing to the right source and
validate field-level ownership. See section 7 (Editing Config Through ConfigHub)
for how this enables the full editing experience.

### Export envelope (GeneratorOutput)

When provenance data needs to leave ConfigHub — written to Git, published as an
OCI artifact, or consumed by external tooling — the Unit is exported as a
`GeneratorOutput` envelope that bundles resources with provenance into a
portable YAML document:

```yaml
# Export format (for Git / OCI / external tooling)
apiVersion: confighub.io/v1
kind: GeneratorOutput

metadata:
  name: string          # Unit name
  namespace: string     # Target namespace (optional)

generator:
  name: string          # from Unit provenance metadata
  version: string       # from Unit provenance metadata

inputs:
  digest: string        # from Unit provenance metadata
  raw: object           # Original input values (optional, redactable)

rendered:
  at: string            # from Unit provenance metadata
  by: string            # What triggered render (optional)
  commit: string        # Source commit (optional)

resources: []object     # WET manifests (the Unit's config data)

field_origins: object   # Field-origin map (optional, Level 2+)
```

This is what `cub export` or a ConfigHub API call produces — not what's stored
internally. Inside ConfigHub, the provenance lives as metadata on the Unit
itself.

**The determinism rule:** Given identical `generator.name`, `generator.version`, and
`inputs.digest`, the `resources` output must be byte-identical after normalization.

A practical test: if you cannot diff the output of two runs with identical inputs, the generator is not deterministic.

---

## 4. Worked Example: Traefik Across Three Environments

Helm is already a generator — it just isn't usually called one:

```
chart/ + values.yaml  →  helm template  →  rendered manifests
```

**Generator version for Helm** is not a single number — it has four axes:

```
generator version = (Helm CLI version, chart repo, chart name, chart version)
```

Most teams track only chart version. The other three axes are invisible without
Level 2 provenance. Concrete example with traefik across three environments:

| Env | Helm CLI | Chart | Values | Output |
|-----|----------|-------|--------|--------|
| dev | 3.14.0 | traefik/traefik **33.2.1** | `replicas: 1` | Deployment + IngressRoute (**v3** API) |
| staging | 3.14.0 | traefik/traefik **32.1.0** | `replicas: 2` | Deployment + IngressRoute (v2 API) |
| prod | **3.13.2** | traefik/traefik **32.1.0** | `replicas: 5` | Deployment + IngressRoute (v2 API) |

Three version axes diverge:

1. **Chart version** (dev vs staging/prod): traefik 33.x changed IngressRoute
   from v2 to v3 API. Same values, different chart → structurally different
   output. Without provenance, you see the diff but not the cause.

2. **Helm CLI version** (prod vs staging): Helm 3.14.0 changed template function
   behavior. Same chart and values, different CLI → potentially different output.
   At Level 2, `generator.version: 3.13.2` vs `3.14.0` makes this visible.

3. **Chart repo** (not shown, but real): switching from `traefik/traefik` to an
   internal fork `acme/traefik` at the same version number produces different
   chart bytes. The input digest catches this even when the version string matches.

At Level 2, the Unit's provenance metadata records all four axes:

```
generator.name:    helm
generator.version: 3.14.0 / traefik/traefik@33.2.1
inputs.digest:     sha256:... (chart bytes + values)
rendered.at:       2026-02-25T14:30:00Z
```

For export to Git or OCI, the full Unit is wrapped as a `GeneratorOutput`
envelope:

```yaml
# Export format
apiVersion: confighub.io/v1
kind: GeneratorOutput
metadata:
  name: traefik-dev
generator:
  name: helm
  version: 3.14.0           # Helm CLI version
inputs:
  digest: sha256:...         # hash of chart bytes + values
  raw:
    chart: traefik/traefik
    chartVersion: 33.2.1
    values: { replicas: 1 }
rendered:
  at: "2026-02-25T14:30:00Z"
resources:
  - kind: Deployment
  - kind: IngressRoute       # v3 API (from chart 33.x)
```

ConfigHub's **rendered manifests** feature already captures Helm output as a
Unit (Level 1). Adding provenance metadata to the Unit (Level 2) is the
difference between "I can see what Helm produced" and "I can trace which of the
four version axes caused this output to differ across environments."

---

## 5. Beyond Helm: Other Generator Types

### 5a. Generators Closest to Helm

**Kustomize** is the same pattern with different inputs: base manifests +
overlays instead of chart + values. Same provenance fields, same maturity levels.

```
generator.name:    kustomize
generator.version: 5.4.0
inputs.digest:     sha256:... (base + overlays)
rendered.at:       2026-02-25T14:30:00Z
```

**Why this is the best way to deal with Argo App-of-Apps and ApplicationSet.**

ApplicationSet and App-of-Apps run *inside* the cluster as part of
reconciliation. Their expansion happens after publish — you can't validate or
govern their output before it reaches the cluster.

```yaml
# ApplicationSet (lives in cluster)
kind: ApplicationSet
spec:
  generators:
    - clusters:
        selector:
          matchLabels:
            env: prod
  template:
    spec:
      source:
        repoURL: https://github.com/acme/apps
        path: "payments/{{name}}"
```

This expands at runtime into Application CRs — one per matching cluster. The
expansion happens *after* publish, so the individual Application CRs lack
pre-publish provenance unless the ApplicationSet inputs (selectors, templates)
are themselves governed in the outer loop.

If you want governance, you have two options:

**(a) Govern the inputs (pragmatic today).** Keep ApplicationSet, but manage the
ApplicationSet *spec* as a Unit in the outer loop. ConfigHub tracks its
provenance and changes; Argo handles expansion at runtime. cub-scout observes
the expanded Application CRs and classifies them as Argo-owned. If the selector
or template is changed manually in-cluster, that shows up as drift.

**(b) Pre-render the expansion (Level 3+).** Treat the expansion as an
authoritative generator: render the individual Application CRs in the outer loop,
publish each as a Unit with provenance. This gives you pre-publish governance but
requires moving expansion logic out of Argo.

Option (a) is where most teams start. Option (b) is where Level 3+ governance
leads.

### 5b. Framework Generators (The New Category)

Frameworks like Spring Boot, Django, and Rails already encode intent — ports,
probes, lifecycle, conventions. A framework generator reads this existing
knowledge and **narrates** it into explicit K8s configuration.

**Why Score.dev is a generator.**

A Score workload is DRY intent — the developer describes *what the app needs*
without specifying K8s primitives:

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

The platform team provides a rendering context — constraints, labels, policies.
The Score generator applies the context to produce WET manifests: Deployment with
resource limits, Service with TLS, Ingress with mesh injection, HPA with zone
spread. This is exactly what `helm template` does for charts: structured input →
deterministic output. The only difference is the abstraction level of the input.

```
generator.name:    score-generator
generator.version: 1.0.0
inputs.digest:     sha256:... (score.yaml + platform-context.yaml)
rendered.at:       2026-02-25T14:30:00Z
```

**Why a Spring Boot platform is "just a generator."**

Spring Boot apps already declare their operational intent — server ports,
actuator endpoints, graceful shutdown, health check paths. A framework generator
reads `application.yaml` and *narrates* this into K8s manifests:

```yaml
# application.yaml (already exists)           # intent.yaml (developer adds)
server.port: 8080                              image: ghcr.io/acme/pay:1.2.3
management.server.port: 9090                   route:
management.endpoint.health.probes.enabled: true   host: pay.example.com
spring.lifecycle.timeout-per-shutdown-phase: 30s   tls: true
```

Generator renders:

```yaml
# "DRY space" → "WET space"
containerPort: 8080               # from server.port
readinessProbe: /actuator/health/readiness, port 9090   # from probes.enabled + mgmt port
livenessProbe: /actuator/health/liveness, port 9090
terminationGracePeriodSeconds: 30  # from lifecycle timeout
```

The developer doesn't write a readiness probe; the generator knows Spring Boot
exposes `/actuator/health`. This isn't magic — it's the same thing `helm
template` does, except the "chart" is the framework's conventions.

```
generator.name:    spring-boot-generator
generator.version: 2.1.0
inputs.digest:     sha256:... (application.yaml + intent.yaml)
rendered.at:       2026-02-25T14:30:00Z
```

Full walkthrough: `spring-boot-generator-example.md`.

**Backstage / IDP: a client, not a generator.**

Backstage captures intent via a Software Template form:

```yaml
# What the developer fills in (Backstage form)
host: pay.example.com
tls: true
useMesh: true
minReplicas: 2
```

Backstage calls a custom Scaffolder action (`confighub:upsert-unit`) which:

1. Validates inputs against platform constraints
2. Invokes the appropriate generator (Score, Spring Boot, etc.)
3. Stores the resulting Unit in ConfigHub
4. Returns links to the Unit, diff, and provenance

Backstage **does not own state**. It shows receipts and drift evidence from
ConfigHub. If you delete Backstage, the Units, provenance, and evidence survive
intact. Full walkthrough: `backstage-idp-example.md`.

### 5c. The Identity Generator

For imported WET config that wasn't generated — the input *is* the output.

```
generator.name:    identity
generator.version: 1.0.0
inputs.digest:     sha256:... (the config blob itself)
rendered.at:       2026-02-25T14:30:00Z
```

The `identity` generator is the degenerate case. Its digest still matters: it
tells you whether the imported config changed since last time.

### 5d. The Pattern is Constant

The generator model is not opinionated about input format. Three different entry
points produce the same output:

| Entry Point | Input | Audience | Generator |
|-------------|-------|----------|-----------|
| **Spring Boot** | `application.yaml` + `intent.yaml` | Java/Spring developers | `spring-boot-generator` |
| **Score.dev** | `score.yaml` | Any containerized app | `score-generator` |
| **Backstage** | Template form parameters | All developers (portal UX) | Scaffolder action -> generator |

All three produce:
- Units with the same four provenance fields (generator name, version, input digest, render timestamp)
- Units stored in ConfigHub
- Evidence when drift occurs

The abstraction level varies across entry points, but the explicitness of the output
is constant. Different DRY inputs,
same WET output structure. Backstage is a **client** of ConfigHub, not a system of
record — it captures intent and shows receipts, but does not own state.

### Two Patterns, Not Many

| Pattern | When | Examples |
|---------|------|----------|
| **DRY → Transform → WET** | Config has semantic meaning to translate | Spring Boot → K8s, Helm values → manifests, Score → K8s |
| **WET Already Exists → Store** | Config is already consumable | Feature flags, runtime tuning, No Config Platform config |

An import worker is a degenerate generator (identity transform).

---

## 6. Generator Inputs as Units

The maturity levels above track generator *outputs*. But the inputs matter too.
Storing generator inputs as Units makes the full rendering pipeline visible:

**Values files** (strongest case). Values files are structured data — YAML
key-value pairs. As Units they get: revision history, cross-environment diff
via variants, ChangeSet governance, and trigger-based validation (e.g., a CEL
trigger that rejects `replicas: 0` in production).

**Chart references** (moderate case). Store the chart pointer (repo + name +
version) as a Unit, not the chart contents. "What chart version is staging
running vs prod?" becomes a Unit query instead of a kubectl question.

**Umbrella charts** (composition visibility). An umbrella is a dependency list.
As a Unit with links to child chart-reference Units, it gives a deployable
bundle with per-component version tracking and history.

The principle: ConfigHub answers "what changed?" across the full *input*
surface, not just the output surface that rendered manifests capture today.

**Adoption.** Git remains the source of truth for charts and values files.
ConfigHub *references* them and tracks which version is deployed where. Start
with values files — the thing teams change weekly — not charts, which change
quarterly. Opt-in per component, same as rendered manifests.

Generators produce configuration as data. The next section explains where users
*edit* that data — and why ConfigHub is the right surface for it.

---

## 7. Editing Config: DRY Sources Through ConfigHub

### The Problem

Users who author config in DRY format — Helm values, Score workloads, Spring
Boot `application.yaml` — naturally want to edit in DRY space. That's where
they authored the intent; that's where changes should happen.

The risk: if variant-specific customizations happen as overlays on the WET
output, the DRY source stops being the real source of truth. Over time, WET
overlays accumulate, the WET diverges from what the generator would produce, and
the DRY file becomes fiction — it describes what the generator *would* produce,
not what's actually deployed.

The answer is not to prevent WET-space customization entirely. The answer is to
make editing through ConfigHub the natural path, and to have ConfigHub resolve
the DRY source for the user.

### ConfigHub as the Editing Surface

ConfigHub stores WET — it is the queryable system of record for what's deployed.
But for generator-backed config, ConfigHub can also be the surface that **writes
back to DRY**.

The experience:

1. **View in ConfigHub.** The user sees the fully resolved WET — what's actually
   deployed, across variants, with ownership and provenance. This is already
   better than reading raw YAML in Git because it's queryable, comparable, and
   shows the whole picture.

2. **Click a field to edit.** ConfigHub resolves the field-origin map (section 3).
   It shows the user: "this is `replicas`, it comes from `values-prod.yaml:14`
   in `acme/platform-config`, you have edit rights." The user changes the value
   in the ConfigHub UI.

3. **ConfigHub proposes upstream changes explicitly.** It opens a PR/MR to update
   the DRY source in Git, subject to policy and approval. After merge, the
   generator re-renders. WET updates. The reconciler applies. The user never
   opened an IDE or navigated a Git repo. They edited DRY *through* ConfigHub.

From the user's perspective, they edited in ConfigHub. Under the hood, the
DRY→WET contract is preserved. DRY authors are happy because DRY is the source
of truth. The product is happy because ConfigHub is the interface they use daily.

### The Adoption Ladder

Building trust with DRY-first users is incremental. Start with viewing — the
lowest-friction entry point — and let gravity pull toward editing.

| Stage | User does | Trust earned |
|-------|-----------|-------------|
| **View** | "Show me what's deployed for payments-api across envs" | ConfigHub knows the truth |
| **Compare** | "Why is prod different from staging?" | ConfigHub explains provenance |
| **Trace** | "Where does this value come from?" | ConfigHub navigates the DRY→WET chain |
| **Edit** | "Change replicas to 5 in prod" | ConfigHub is where I make changes |
| **Govern** | "Who approved this? Can I promote to prod?" | ConfigHub is how we operate |

App config is the right starting category: highest-frequency edits, lowest
stakes. Nobody is nervous about changing replicas or an env var. By the time
users trust ConfigHub for that, the step to using it for variant creation,
promotion, and governance feels incremental.

### What Gets Edited in WET Space (Directly in ConfigHub)

Not everything has a DRY upstream. Some edits are legitimately WET-native —
ConfigHub *is* the right editing surface, and there's no DRY source to redirect
to:

| Edit type | Where it happens | Why |
|-----------|-----------------|-----|
| **App config** (replicas, image, env vars, ports) | DRY source → generator → WET | Generator has a knob for it; ConfigHub resolves the DRY source via field-origin map |
| **Platform policy** (network rules, security, quotas) | ConfigHub directly | Platform team owns these; no generator upstream |
| **Emergency overrides** | ConfigHub with TTL | Temporary, governed, recorded; expires or promotes to DRY |
| **Deployment topology** (which clusters, target selection) | ConfigHub | Control plane decision, not generated |
| **Variant lifecycle** (create canary, regional override) | ConfigHub | Control plane orchestration |
| **Governance state** (approvals, locks, trust tiers) | ConfigHub | Native to the control plane |
| **Legacy / no-generator YAML** | ConfigHub | WET-already (identity generator); no DRY to go back to |

The principle: if a field has a generator origin, you edit in DRY and the system
guides you there. If a field is control-plane-native or platform-injected,
ConfigHub is the right editing surface and there's no DRY to redirect to. The
field-origin map (section 3) distinguishes these cases.

### cub-track as the Redirection Layer

cub-track's mutation ledger gains a new role: when someone edits WET directly
for a field that has a DRY origin, cub-track can detect this and redirect.

**`cub-track explain --commit <sha>`** already shows intent/decision/outcome.
With field-origin awareness, it adds: "This commit changed `replicas` in the WET
manifest. The DRY source for this field is `values-prod.yaml:14` in
`acme/platform-config`." The user knows where to go next time.

**`cub-track suggest`** (post-MVP, new): before committing a WET change, check
provenance. If the field has a DRY origin, surface it: "You're editing generated
output. The source for this field is X. Edit there instead?" This creates
friction that pushes users back to DRY without blocking them.

**Drift classification gets richer.** When a WET field changes without
`inputs.digest` changing, that's a distinct signal: someone edited the output,
not the input. cub-track can classify this as "overlay drift from DRY source" —
a different category from "runtime drift from intended state."

### Staleness Detection

"Is the WET in ConfigHub stale relative to DRY inputs?" is a ConfigHub/CI
concern, not a cub-scout concern. cub-scout observes the cluster; it doesn't
know about the generator pipeline or DRY sources.

The staleness check is straightforward: compare `inputs.digest` at render time
vs. current DRY inputs. If the digest changed but the WET hasn't been
re-rendered, the stored WET is stale. This belongs in the publishing pipeline —
a pre-publish check that detects when generator inputs have changed but the
output hasn't been re-rendered.

### Per-Variant Changes Belong in DRY

The default path for variant-specific changes should be the generator's input
surface, not post-generation patches. Helm already supports this — per-environment
values files (`values-dev.yaml`, `values-prod.yaml`). Score and Spring Boot
generators can support variant-specific inputs the same way.

When the generator doesn't have a knob for a needed change, the first response
is to **extend the generator input schema** — not to overlay the output. WET-space
overlays are a transitional escape hatch. The system should create friction
(evidence, staleness detection, cub-track redirection) that encourages promotion
back to DRY.

The ideal UX makes this seamless:

```
$ cub edit payments-api --field spec.replicas --variant prod

→ This field is generated from: values-prod.yaml:14 (repo: acme/platform-config)
→ Current value: 2
→ New value: 5

→ Re-rendering with updated input...
→ Diff:
    spec.replicas: 2 → 5
    (no other fields affected)

→ Commit to acme/platform-config? [y/n]
```

The user never touches WET. They express intent, the system resolves the DRY
source via the field-origin map, re-renders, shows the WET diff, and commits to
the right place. The generator contract is preserved. Provenance stays clean.

---

## 8. What Classical GitOps Gets Wrong

Classical GitOps is strong at convergence, weak at governance detail. It answers what
changed and whether reconcile happened. It often cannot answer why a change was
proposed, why it was allowed, what checks ran, or what proof exists after execution.

Specific gaps:

**DRY vs WET confusion.** Generators may be applied in source control, during build,
or only at runtime. The availability of multiple rendering points creates confusion.
If generators and input values both change, updating dependent manifests is
unpredictable.

**Multiple stores and transports.** Git, OCI, Helm repos, cluster caches — what is
the total reconciliation state? Always partial or grey. This worsens as we add
mutation records for agentic changes.

**Push vs pull vs gates.** Classical GitOps is pull-only. Agentic operations need
decision gates and explicit authorization before apply.

**"Bidirectional GitOps."** The term is shorthand — the precise mechanism is
a **governed reverse flow**: live state changes produce evidence, evidence triggers
an explicit proposal (merge request) back to intended state, and that proposal is
reviewed and accepted or rejected like any other change. This is sometimes called
"approved sync-back." Neither direction is silent; both are governed.

**Platform abstraction hides too much.** Platform engineering wants to reduce
complexity, but hiding config entirely means no one can debug it. App teams want an
abstraction layer — they don't want to think about config — but they do need to see
what was generated when things break.

**No editing path back to DRY.** Users author in DRY (Helm values, Score workloads)
and generators produce WET. But when a variant-specific change is needed, users
often patch the WET output directly because there's no system that resolves which
DRY input controls which WET field. Over time, overlays accumulate and the DRY
source becomes fiction. Classical GitOps has no concept of a field-origin map or
a governed path from "I want to change this deployed value" back to the right
input file. (See section 7 for how ConfigHub addresses this.)

**Dynamic config gaps.** Secrets, feature flags, staged rollouts, app sets, batch
jobs — all forms of generated or dynamic config that exist outside the simple
"commit YAML to Git" model.

**Staging is hard.** No standard model for promoting config across environments
without variant clobbering. In this model, staging is variant-driven (labels, not
folders) and promotion is explicit copy/merge with approval policy — see the
Staging Model in section 12.

**Mixed controllers.** Using Terraform or Crossplane with Argo or Flux creates
"mixed configs" that no single reconciler understands fully.

---

## 9. How Agentic Engineering Changes GitOps

GitOps automates live operations by letting agents (Flux, Argo) control changes to
live state, driven by an external model. Changes to the model — a set of declarative
configuration statements in a versioned store — are continually synchronized with live
state. This is the reconciliation loop.

GitOps is already partially agentic. The question is what governance it lacks.

**Volume without governance.** We moved control from runtime updates to configuration
updates. Who says those config updates will be any better? Someone could write a bad
config and break everything. Now imagine AI agents making hundreds of changes a day.
Without governance over those mutations, the risk of a serious incident grows with
every unreviewed change.

**One-way reconciliation.** Classical GitOps prefers desired-state changes to drive
runtime. A direct "break-glass" fix shows up as drift and gets overwritten. If we want
AI Ops to work on live systems, we need a model where live-state changes can be
proposed back to the desired-state store, reviewed, and accepted or rejected explicitly.

**Accountability at scale.** Combine both problems. If we automate operations using a
mix of GitOps and AI, we need to know when agents acted on human authority, what plan
they followed, and whether the outcome matched the specification. An enterprise
agentic solution must support:

1. **Ask** — express intent via prompt or structured input
2. **Specify** — form an execution plan with preconditions
3. **Decide** — evaluate policy (`ALLOW | ESCALATE | BLOCK`)
4. **Execute** — enact the plan with scoped, attested authority
5. **Verify** — compare outcome against specification
6. **Attest** — produce a signed proof that the authority has seen and approved

This is agentic verification — a proof expressed as operational facts and
configuration changes, not as a chat transcript.

See also early ideas in the GitOpsConEU keynote (December 2023), which introduced
"Generative GitOps" — the idea that AI agents would generate configuration downward
while runtime anomalies would be explained upward.

---

## 10. Invariants (Unchanged)

Three rules that are never waived:

1. **Nothing implicit ever deploys.**
2. **Nothing observed silently overwrites intent.**
3. **Configuration is data, not code.**

The first two govern mutation flow. The third governs representation — and it
is the foundational claim.

**Configuration is data.** Configuration data is not parameterized. It contains
literal values for every field. You can read it, query it, diff it, and validate
it without rendering anything first.

ConfigHub exists because configuration, once rendered, is structured data that
deserves the same treatment as any other business-critical dataset: typed
queries, label-based filtering, revision history, cross-environment comparison,
retention policies, and policy evaluation at write time. Git stores text diffs;
ConfigHub stores queryable records. (See FAQ: "Why ConfigHub instead of building
all of this on Git?")

Every artifact in this model is data:
- **Generator output** (Units) — literal manifests, queryable by label
- **Evidence** — structured diffs with classification, not log lines
- **Mutation records** — intent + decision + outcome, not chat transcripts
- **Constraints** — declarative rules, not imperative scripts

The generator boundary is where code becomes data. Templates, SDK methods,
procedural scripts — all fine as authoring tools. But whatever enters as code
exits as explicit manifests: structured data. Nothing imperative crosses the
publish line.

Actions are the boundary case: they describe operational intent as declarative
manifests (structured data), but a runtime must interpret and execute them. The
action manifest is data; the execution is not. This is an intentional seam —
the manifest is diffable and governable *before* execution, and the outcome is
observable *after*.

These hold across all modes: standalone, connected, agentic.

---

# Part II: Normative Contracts

> The formal contracts that implementors build against: system responsibilities,
> entity definitions, governance models, evidence schemas, and the mutation ledger.

## 11. Operating Boundary

| Responsibility | Owned By |
|----------------|----------|
| Store and publish explicit intended state + provenance | **ConfigHub** |
| Resolve field-origin maps, route editing to DRY sources | **ConfigHub** |
| Detect stale renders (inputs changed, output not re-rendered) | **ConfigHub** (publishing pipeline) |
| Reconcile runtime from published artifacts | **Flux / Argo** (inner loop) |
| Observe cluster reality, capture evidence, detect drift from intended state | **cub-scout** |
| Record governed mutation history in Git; redirect WET edits to DRY sources | **cub-track** |
| Evaluate risk and policy signals | **confighub-scan** |
| Evaluate semantic assertions + issue verification attestation | **verified** |
| Issue decisions and attestations | **confighub** (decision authority) |
| Execute token-scoped runtime actions | **confighub-actions** |

ConfigHub is not a controller. It does not reconcile. It stores, governs, and
publishes. Flux/Argo remain the runtime reconcilers.

**cub-scout observes the cluster, not the generator pipeline.** cub-scout
compares cluster reality against the reconciler's intended state. It does not
know about DRY sources or generators. "Is the WET in ConfigHub stale relative
to DRY inputs?" is a ConfigHub publishing pipeline concern — detected via
`inputs.digest` comparison — not a cub-scout concern.

---

## 12. The ConfigHub App Model

ConfigHub is moving to app-centric language. The core entities are described
below as **current implementation mappings** — the concepts are stable, but
the API surface (whether App is a label query, a first-class object, or both)
may evolve. The cardinality invariants and staging model at the end of this
section are the durable rules; the specific API shape is not.

### App

A named collection of components owned by one team. The "App" is what users think
about. It emerges from querying Units by label:

```
App: payment-service
  Components: api, worker, redis
  Deployments: dev, staging, prod
```

An App is a label value (`app=payment-service`), not a separate API object today.

### Deployment

The junction of an App and a Target — what you get when you deploy an App to a
specific environment.

### Target

A Kubernetes cluster (or other managed system) connected via a Worker.

### Unit

The atomic element: a single deployable workload with labels, source mapping, and
provenance. Units remain the implementation primitive. Apps and Deployments are
queries over Units.

### Variant

A label indicating environment or configuration flavor (`variant=prod`,
`variant=staging`). Not a folder — Git paths like `overlays/prod` map to
`variant=prod` on the Unit.

### Cardinality Invariants

These structural rules hold regardless of API evolution:

| Relationship | Rule |
|-------------|------|
| App → Units | An App is a **grouping key** (label query), not a mutable container — adding a Unit to an App means labeling the Unit, not modifying the App |
| Deployment → Target | A Deployment binds **one** App to **one** Target; the same App in two clusters is two Deployments |
| Deployment → Reconciler | A Deployment has **exactly one** reconciler (Flux *or* Argo, not both) |
| Unit → App | A Unit belongs to **exactly one** App |
| Variant | A Variant is a **label** on a Unit, not a separate object — staging is variant-driven, not folder-fragmented |

### Staging Model

Promotion across environments is explicit, variant-driven, and policy-gated:

1. Environments are expressed as Variant labels (`variant=dev`, `variant=staging`,
   `variant=prod`), not as separate folders.
2. Promotion is an explicit copy/merge of Unit configuration from one Variant to
   another, subject to approval policy (e.g., `variant=prod` requires human approval;
   `variant=dev` may auto-promote).
3. The same Unit can exist with multiple Variants. Each Variant is a full,
   auditable snapshot — not a diff against another Variant.
4. Platform constraints may vary per Variant (e.g., `min-replicas-2` applies
   only when `variant=prod`).

This avoids the classical GitOps problem of "folder-per-environment" drift where
overlay directories diverge silently over time.

---

## 13. The Two-Loop Model

### From Classical to Governed GitOps

Classical GitOps covers commit-to-cluster convergence. Governed GitOps extends
every phase with intent capture, policy gates, and evidence:

| Phase | Classical GitOps | Governed (Agentic) GitOps | What's Added |
|-------|-----------------|--------------------------|--------------|
| **Author** | Human writes YAML, commits to Git | Human or agent expresses intent (DRY) — directly or through ConfigHub's editing surface; generator renders explicit config (WET) | Provenance: generator version, input digest, field-origin map |
| **Review** | PR review (human eyeballs) | PR review + policy evaluation (`ALLOW | ESCALATE | BLOCK`) | Automated constraint validation; trust tier gating |
| **Publish** | Merge to main; reconciler watches repo | ConfigHub stores Unit, publishes OCI artifact or Git source | Immutable intended state with provenance; not just "latest commit on main" |
| **Reconcile** | Flux/Argo pull and apply | Flux/Argo pull and apply *(unchanged)* | — (reconcilers are not replaced) |
| **Observe** | `flux get all` / Argo health check | cub-scout structured evidence: field-level diff + provenance link | Drift is typed, classified, and linked back to the operation that set the expected value |
| **Respond** | Manual fix or re-sync | Policy-driven: alert, propose-revert, propose-accept, require-approval | Governed reverse flow — no silent overwrite in either direction |
| **Record** | Git log + Flux/Argo events | cub-track ChangeInteractionCard: intent + decision + execution + outcome | Mutation ledger answers "why was this allowed?" not just "what changed" |
| **Attest** | *(not modeled)* | Signed attestation: actor, intent revision, artifact digest, observed result | Audit-grade proof that authority, action, and outcome are linked |

The left two columns are what Flux/Argo users already have. The right two columns
are what this model adds. Nothing in the left column is removed or replaced.

**Why governed GitOps is inherently bidirectional.** Classical GitOps is
one-directional by design: Git wins, always. But break-glass fixes, runtime-
discovered state (autoscaler adjustments, cert rotations), and AI agents acting
on live systems all produce legitimate changes that originate in the cluster, not
in Git. Without a governed reverse path, those changes are either silently
reverted or silently tolerated — both bad. This model adds the reverse direction
explicitly: observe, produce evidence, propose back to intended state, review,
accept or reject. Neither direction is automatic. See Appendix B for the full
explanation.

### Outer Loop (Authority and Governance)

```
ask -> specify -> plan -> decide -> authorize
```

Outputs:
1. Intended state artifact(s)
2. Policy decision (`ALLOW | ESCALATE | BLOCK`)
3. Verification verdict + attestation (`pass | fail | abstain`)
4. Scoped execution authority (short-lived token)

This loop is where **authoritative generators operate**. DRY intent is transformed
into explicit WET configuration, validated against platform constraints, and
authorized for publication. In-cluster templaters (ApplicationSet, App-of-Apps) run
in the inner loop instead; their outputs are observed by cub-scout but not governed
at this stage.

### Inner Loop (Runtime Reconciliation)

```
publish -> reconcile -> observe -> verify -> attest
```

Flux/Argo remain in this loop today. ConfigHub publishes OCI artifacts or Git source
updates. Flux/Argo detect changes and reconcile clusters. cub-scout observes the
result.

### Combined Flow

```
Developer/Agent Intent (DRY)
        |
        v                          ConfigHub Editing Surface
    Generator (deterministic,  <-- (resolves field-origin map,
     versioned)                     writes back to DRY source)
        |                                    ^
        v                                    |
ConfigHub (WET, system of record  ----------+
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

### Platform Engineering Flow

```
Platform team provides generator + constraints
        |
App team writes app code + app config (DRY intent)
        |     ^
        |     |--- ConfigHub editing surface (resolves field-origin
        |          map, commits change to DRY source in Git)
        v
Generator turns DRY into deployment manifests (WET)
        |
ConfigHub stores WET as Units with provenance + field-origin maps
        |
Workers publish artifacts (OCI default, or Git source)
        |
Flux/Argo reconcile cluster from published artifacts
        |
cub-scout observes runtime, produces evidence
```

---

## 14. Constraints and Governance

The platform team defines **constraints** — rules that apply to Apps, Deployments,
and Targets. App teams make choices within those rules.

```
Platform constraints (acme-platform):
|
+-- MUST: All Units have resource limits
+-- MUST: Images from approved registry
+-- CAN: May use Flux or ArgoCD
|
Applied to:
|
+-- App: payments-service
|   +-- Deployment: payments-dev   → Target: dev-cluster,  reconciler: flux
|   +-- Deployment: payments-prod  → Target: prod-cluster, reconciler: flux
|
+-- App: orders-service
    +-- Deployment: orders-dev     → Target: dev-cluster,  reconciler: argo
    +-- Deployment: orders-prod    → Target: prod-cluster, reconciler: argo
```

Platform teams set rules and app teams make choices within them. Constraints are
additive across multiple platform sources. Capabilities are intersected. When
constraints conflict, the system surfaces an error rather than resolving silently.

---

## 15. Operations (Intent as Code)

An operation is a **method in a framework SDK** that produces a configuration diff.
Operations are intent, not execution.

Operations exist in "DRY space" — they describe desired configuration changes using
the vocabulary of the framework or platform SDK. When a generator renders them, the
result crosses into "WET space": explicit, deployable manifests stored in ConfigHub.
The operation itself is never deployed; only its rendered output is.

```python
app = AcmeApp("payments-api", framework="spring-boot")
app.add_router("pay.example.com", tls=True)   # operation ("DRY space")
app.scale(min=2, max=10)                       # operation ("DRY space")
# ↓ generator renders ↓
# WET manifests: Ingress, Service, HPA          ("WET space")
```

Every operation produces four artifacts:

| Artifact | Purpose |
|----------|---------|
| **Plan** | What will change (preview) |
| **Patch** | Explicit config delta |
| **Explanation** | Why, risks, assumptions |
| **Provenance** | "Created by X at commit Y" |

Operations live in framework SDKs, not in ConfigHub. Frameworks own the operations;
ConfigHub stores the receipts.

The SDK registers its operations with ConfigHub for discovery, validation, and
audit. The registry (see `operation-registry-v1.md`) advertises each operation's
input schema, the resource kinds it produces, and the constraints it validates
against — making the "DRY space" → "WET space" boundary inspectable.

---

## 16. Evidence and the Drift Loop

### Three Sources of Truth

| Source | Contains | Authority For |
|--------|----------|---------------|
| **ConfigHub** | Intended WET | "What should exist" |
| **Cluster** | Running state | "What does exist" |
| **cub-scout** | Observed evidence | "What we can prove" |

Evidence is the structured diff between intended and observed state, captured at a
point in time. Evidence is observational — creating or exporting it never modifies
intended or runtime state.

**Scope of cub-scout observation.** cub-scout compares cluster reality against
the reconciler's intended state (what Flux/Argo is trying to apply). It does not
compare stored WET against what the generator *would* produce from current DRY
inputs — that is a staleness check, owned by ConfigHub's publishing pipeline via
`inputs.digest` comparison (see section 7). cub-scout's evidence feeds into the
broader system: cub-track can enrich mutation records with field-origin data, and
ConfigHub can correlate evidence with provenance to classify drift causes.

### Evidence Bundle Schema (v2 Proposed)

> **Migration note:** The current codebase uses `BundleSummary` (v1). The schema
> below is the **proposed v2** — it adds `observation.differences` and `provenance`
> fields. v1 remains valid; v2 is additive, behind a schema version flag.
>
> | v1 (current) | v2 (proposed) | Notes |
> |--------------|---------------|-------|
> | `BundleSummary` | `EvidenceBundle` | Rename + richer structure |
> | `summary` field | `summary.title` + `summary.severity` | Split for export routing |
> | N/A | `observation.differences[]` | Structured diff per resource/field |
> | N/A | `provenance.intended_operations` | Links to generator operations |

```yaml
apiVersion: confighub.io/v1
kind: EvidenceBundle

metadata:
  id: string
  created_at: string        # ISO 8601
  type: drift | verification | snapshot

subject:
  unit: string              # Unit name
  app: string               # App name
  deployment: string        # Deployment name
  cluster: string           # Target cluster observed

observation:
  intended: object          # What ConfigHub says should exist
  observed: object          # What cub-scout found
  differences:
    - resource: {apiVersion, kind, namespace, name}
      field: string         # JSONPath
      expected: any
      observed: any
      classification:
        type: added | removed | modified
        likely_cause: manual_edit | controller | unknown

provenance:
  intended_operations: []string
  intended_at: string
  intended_commit: string

summary:
  title: string
  severity: info | warning | critical
```

### Drift Response Policies

| Policy | Behavior |
|--------|----------|
| `alert` | Notify, don't change |
| `propose-revert` | Create PR to restore intended state |
| `propose-accept` | Create PR to accept observed state as new intent |
| `require-approval` | Create PR for human decision |

Policies can vary by label (`variant=dev` -> accept; `variant=prod` -> revert).

**ConfigHub is not a controller.** It does not reconcile. cub-scout observes,
ConfigHub records, and humans or policies decide. Evidence is the interface between
observation and decision.

---

## 17. cub-track: Git-Native Mutation Ledger

### Why cub-track Exists

AI increases mutation volume. Classical GitOps captures diffs and sync status but
cannot answer: Why was this mutation proposed? What evidence existed? What was
decided and executed?

cub-track closes this gap with a Git-native mutation ledger for AI-assisted GitOps
changes.

### What cub-track Is

- OSS, Git-native, local-first
- Compatible with Flux/Argo/Helm workflows (no controller replacement)
- Works without ConfigHub backend dependency in standalone mode
- Optional connected mode adds ConfigHub ingestion and governance

### MVP Commands

| Command | Purpose |
|---------|---------|
| `cub-track enable` | Install hooks, initialize metadata branch |
| `cub-track explain --commit <sha>` | Explain a commit in intent/decision/outcome terms |
| `cub-track search --text\|--file\|--agent\|--decision` | Search mutation history |

### Planned Commands (Post-MVP)

| Command | Purpose |
|---------|---------|
| `cub-track explain --commit <sha> --fields` | Enrich explanation with DRY origin info from field-origin maps: "This commit changed `replicas` in WET. DRY source: `values-prod.yaml:14` in `acme/platform-config`" |
| `cub-track suggest` | Before committing a WET change, check provenance. If the field has a DRY origin, surface it: "You're editing generated output. The source for this field is X. Edit there instead?" |

These commands make cub-track a **redirection layer**: when someone edits WET
directly for a field that has a DRY origin, cub-track detects this and guides
the user back to the right editing surface. This creates productive friction
that encourages editing in DRY space without blocking emergency changes.

### The Core Primitive: Change Interaction Card

The unit of governance is not a transcript. It is a **governed mutation record**:

```
intent + evidence + decision + execution + outcome
```

Represented as a `ChangeInteractionCard` with stable IDs linked to commits:

```json
{
  "schema_version": "change-interaction-card.v1",
  "card_id": "cic_7f8e9d0c1b2a3f44",
  "identity": {
    "repo": "github.com/acme/platform",
    "commit_sha": "9f29a5c...",
    "trailers": {
      "Cub-Checkpoint": "7f8e9d0c1b2a",
      "Cub-Agent": "codex"
    }
  },
  "intent": {
    "summary": "Roll out payment API config update",
    "domain": "app",
    "targets": [{"kind": "Deployment", "namespace": "payments", "name": "payment-api"}]
  },
  "decision": {
    "result": "ALLOW",
    "verification_result": "pass",
    "reason": "Policy checks passed",
    "policy_refs": ["policy.gitops.tier2.standard"]
  },
  "execution": {
    "status": "succeeded",
    "runtime": "confighub-actions"
  },
  "outcome": {
    "result": "applied",
    "message": "Deployment healthy after rollout"
  }
}
```

### DRY/WET Storage Boundary

**Git stores the DRY artifacts:** compact, reviewable, immutable linkage artifacts:

- Commit trailers: `Cub-Checkpoint`, `Cub-Intent`, `Cub-Agent`
- Compact receipts: `decision-receipt.v1`, `execution-receipt.v1`, `outcome-receipt.v1`
- Metadata branch: `cub/mutations/v1` (append-only; read alias: `cub/checkpoints/v1`)

> **Trailer alias:** MVP uses `Cub-Checkpoint`; long-term key is `Cub-Mutation`.
> Both resolve to the same card. Accept both on read; switch default post-GA.

**ConfigHub stores the WET artifacts:** full, rich, queryable operational and governance state:

- Full policy input/output graphs and rule traces
- Approval chain metadata
- Token issuance metadata
- Full execution records
- Pre/post scan findings and evidence links
- Cross-repo correlation and search indexes

**Anti-patterns:** storing full transcripts in Git, storing tokens or auth material
in Git, using Git as a telemetry store, or maintaining mutable indexes in Git.

### Trust Tier Model

| Tier | Capability |
|------|------------|
| 0 | Observe only (no apply rights) |
| 1 | Low-risk apply domains |
| 2 | Medium-risk with human approval |
| 3 | High-risk/prod with strong attestation + dual approval |

Execution rights are tier-bound. Policy evaluates tier before token issuance.

### Attestation Minimum Contract

Every governed execution (Tier 1+) must produce an attestation that includes at
minimum:

| Field | What It Records |
|-------|----------------|
| **actor** | Who or what initiated the mutation (user, agent ID, CI job) |
| **approved_intent_revision** | The exact revision of intended state that was authorized |
| **rendered_artifact_digest** | SHA-256 of the published OCI artifact or Git tree |
| **target_scope** | Cluster, namespace, and resource(s) the execution may touch |
| **observed_result** | Post-execution observation: applied, partial, failed, rolled back |
| **policy_checks** | Which policies were evaluated and their outcomes (ALLOW/ESCALATE/BLOCK) |

This is the minimum set required to answer "who did what, why were they allowed,
and what happened?" after the fact. Higher tiers may add fields (e.g., dual
approval chain at Tier 3, scan findings at Tier 2+), but these six are always
present.

### Adoption Stages

| Stage | Installs | Gets | Git Writes (DRY) | ConfigHub Writes (WET) |
|-------|----------|------|-------------------|------------------------|
| **0. OSS Local** | `cub-track` only | Commit-linked mutation history, explain/search | Trailers + local card + linkage receipts (no governance receipts yet) | None required |
| **1. Connected Evidence** | + ConfigHub credentials | Cross-repo search, centralized provenance | Same DRY artifacts | Ingested card index, evidence catalog |
| **2. Governed Apply** | + policy/runtime services | Policy-gated mutation: `ALLOW | ESCALATE | BLOCK` | + governance receipts (`decision-receipt.v1`, `execution-receipt.v1`, `outcome-receipt.v1`) | Full policy traces, attestation |
| **3. Enterprise Ops** | + org controls | Audit-grade reporting, retention, compliance exports | Same compact governance receipts | Full retention, RBAC, analytics |

> **Stage 0 note:** `cub-track` writes linkage receipts (trailer → card ID) but
> not governance receipts (decision, execution, outcome). Those require Stage 2+
> where `confighub-scan` provides policy decisions.

---

## 18. Write-Back Semantics (Not Silent Overwrite)

Observed runtime changes **do not** overwrite intent. They produce explicit proposals:

1. cub-scout detects drift and produces evidence bundle
2. Evidence is exported (Slack, Jira, ConfigHub)
3. If policy says `propose-accept`: a merge request is created to update intended state
4. If policy says `propose-revert`: a merge request is created to restore intended state
5. Human or automation reviews and accepts/rejects

Write-backs from agents follow the same pattern. An agent changes an operational
overlay value, proposes it as a merge request, and the mutation is logged in both
ConfigHub and Git (depending on scope).

### Overlay Edits: Transitional, Not Steady-State

Variant overlay edits on WET output do **not** require immediate write-back to
generator intent. But they are a **transitional escape hatch**, not a permanent
workflow.

The preferred path for per-variant changes is the generator's input surface —
per-environment values files, variant-specific context, or extended generator
input schemas. When a user needs a change that the generator doesn't support,
the first response should be to extend the generator input, not to overlay the
output (see section 7).

If an overlay is applied to WET:

1. The system records it as a governed mutation (cub-track ChangeInteractionCard)
2. `cub-track suggest` flags the field's DRY origin, if one exists
3. The overlay is classified as "overlay drift from DRY source" — distinct from
   runtime drift
4. Promote the overlay to DRY intent/generator when the change becomes reusable,
   default-worthy, or long-lived

The system should create friction — evidence, staleness detection, cub-track
redirection — that encourages promotion back to DRY rather than normalizing
WET-space editing for generator-backed config.

---

## 19. Failure Modes (Explicitly Handled)

| Failure | How It Surfaces | Resolution |
|---------|----------------|------------|
| **Drift** | cub-scout evidence bundle | Policy-driven: alert, revert, accept, or require-approval |
| **Conflicting platform rules** | Deployment creation fails with explicit error | Platform teams coordinate; system surfaces conflicts, does not resolve them |
| **Generator bug** | Output reproducible via input digest; diff available | Fix generator, re-render; change is explicit and diffable |
| **Invalid operation** | Validation rejects before rendering | Fix inputs or obtain platform exception |
| **Manual runtime change** | Detected as drift on next observation | Depends on drift policy; never silently accepted |
| **Overlay drift from DRY** | WET field changed without `inputs.digest` change; cub-track classifies as overlay | Promote to DRY input or expire; `cub-track suggest` redirects to DRY source |
| **Stale render** | `inputs.digest` changed but WET not re-rendered; detected by ConfigHub publishing pipeline | Re-render from current DRY inputs; diff shows what changed |
| **Generator version mismatch** | Provenance shows old version; version pinning prevents silent upgrade | Explicit re-render; diff shows changes |

In each case, the failure is explicit, surfaced early, and non-destructive.

---

# Part III: Examples, Adoption & Reference

> Worked examples, adoption guidance, and frequently asked questions.

## 20. Concrete Example: Score.dev End-to-End

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
OCI artifact — exported as a `GeneratorOutput` envelope if it needs to be
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

## 21. Day 2: What ConfigHub Changes in Practice

The model above describes the architecture. These four scenarios describe what
changes in practice — the questions you can answer on Day 2 that you couldn't
answer on Day 1.

### Scenario 1: 3am incident — "What changed?"

A Deployment is crashing in production. The on-call engineer opens ConfigHub
and asks: "What changed in `payments-api` in the last 24 hours?"

**Without ConfigHub:** Check Git log across 3 repos. Run `kubectl describe` on
the Deployment. Cross-reference Helm release history with Flux events. Grep
Slack for who merged what. Reconstruct the timeline manually. This takes 30-60
minutes — during an outage.

**With ConfigHub:** One query returns the Unit's revision history: the chart
version bumped from 32.1.0 to 33.2.1, the values file changed `replicas` from
3 to 2, and the generator version shows a Helm CLI upgrade from 3.13.2 to
3.14.0. The evidence bundle shows the specific field-level diff between
intended and observed state. The cub-track mutation record shows who merged it,
what policy evaluated it, and what the decision was. Time to root cause: under
5 minutes.

### Scenario 2: Fleet visibility — "What's deployed where?"

The VP of Engineering asks: "Which version of the payments API is running in
each environment? Are any clusters running a version more than a week old?"

**Without ConfigHub:** Someone writes a script to iterate over 40 repos and
kubectl contexts, parsing labels and annotations. The script breaks when a
team uses a different naming convention. The answer arrives three days later,
already stale.

**With ConfigHub:** One API call: `GET /units?app=payments-api` returns every
Unit across every variant and target, with the generator version, render
timestamp, and current image tag. Filter by `rendered.at < 7d` to find stale
renders. The answer is instant, live, and queryable by anyone with dashboard
access.

### Scenario 3: AI agent audit — "What did the agent do?"

An AI agent made 47 configuration changes overnight — scaling, image updates,
feature flag toggles. The compliance team asks: "Can you prove each change was
authorized and validated?"

**Without ConfigHub:** The evidence is scattered: Git commits (some with useful
messages, some not), Flux sync events, Argo health checks, and a chat
transcript in Claude Code. Reconstructing which change was authorized by what
policy requires manual archaeology across systems.

**With ConfigHub + cub-track:** Each of the 47 changes has a
ChangeInteractionCard with structured fields: intent (what the agent proposed),
decision (which policy evaluated it, ALLOW/ESCALATE/BLOCK), execution (what
actually ran), and outcome (what cub-scout observed afterward). The mutation
ledger is the compliance export — every change has a who, what, why, and proof
chain. The audit takes minutes, not days.

### Scenario 4: Config change — "I need to change replicas in prod"

A developer needs to scale `payments-api` from 2 to 5 replicas in production.

**Without ConfigHub:** The developer knows the app runs on Helm, but isn't sure
which repo has the values file, what it's called, or whether there are
per-environment overrides. They find a `values.yaml` in the platform repo, change
`replicas: 5`, open a PR — and learn from code review that production uses
`values-prod.yaml` in a different directory, and their change only affected
staging. They fix it, wait for CI, merge, wait for Flux to sync. Elapsed time:
45 minutes, two PR cycles.

**With ConfigHub:** The developer opens ConfigHub, finds `payments-api` variant
`prod`, clicks `spec.replicas`. ConfigHub shows: "This field is generated from
`values-prod.yaml:14` in `acme/platform-config`. Current value: 2." The developer
changes it to 5. ConfigHub commits the change to the correct file in the correct
repo, the generator re-renders, and the diff shows exactly one field changed.
Elapsed time: 2 minutes. The developer never needed to know which repo, which
file, or which values overlay — ConfigHub resolved it via the field-origin map.

---

## 22. Adoption Path

### Step 1: Add today (no migration)

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
3. Users edit app config *through* ConfigHub — it commits to the DRY source in Git
4. Generator re-renders, WET updates, reconciler applies

This step builds trust incrementally (View → Compare → Trace → Edit → Govern)
starting with app config — the highest-frequency, lowest-stakes edits. See
section 7 for the full editing model.

### Step 4: Full governed execution

1. Maintain explicit intended state and policy gates
2. Support execution paths with attested, token-scoped runtime execution
3. Preserve the same invariants and audit chain whether reconciled by Flux/Argo or
   executed directly

### Value at Each Level

Each adoption step maps to a maturity level, a business outcome, and a boundary
between what's free and what requires ConfigHub.

| Level | Capability | Business Outcome | Free / ConfigHub |
|-------|-----------|-----------------|------------------|
| **0. OSS tools** | cub-scout observation, cub-track local mutation history | Dev goodwill, community adoption, per-repo visibility | Free (OSS) |
| **1. Capture** | Rendered manifests stored as Units | "What did Flux/Argo produce?" — answer in one query, not kubectl archaeology | ConfigHub (shipped) |
| **2. Provenance** | Four fields + field-origin maps on Units | "What inputs produced this?" / stale render detection / trace any field to DRY source | ConfigHub (planned) |
| **2a. Edit** | ConfigHub resolves field origins, commits to DRY source in Git | "Change this value" — without knowing which repo/file/line; ConfigHub handles it | ConfigHub (planned) |
| **3. Pre-publish** | Validate rendered output against platform constraints before publish | "Will this break?" — caught at render time, not after deploy | ConfigHub (planned) |
| **4. Governed** | Policy gates, trust tiers, attestation, mutation ledger | Audit-grade compliance: who authorized, what checks ran, what proof exists | ConfigHub (planned) |
| **Enterprise** | Retention, RBAC, fleet analytics, compliance exports | Org-wide visibility, regulatory reporting, cross-team governance | ConfigHub Enterprise |

---

## 23. What This Is Not

- Not a controller or reconciliation engine
- Not a runtime reconciler or orchestrator
- Not a portal-driven IDP
- Not a replacement for Git
- Not a replacement for Flux/Argo (today)

It is a disciplined way to turn intent into explicit configuration, govern mutations,
and produce evidence when reality diverges from intent.

---

## 24. Summary of Key Concepts

| Concept | Definition |
|---------|------------|
| **App** | Named collection of components, queried by label |
| **Deployment** | App x Target (environment instance) |
| **Unit** | Atomic deployable config with labels and provenance |
| **Generator** | Deterministic function: intent + context -> WET |
| **Field-origin map** | Generator-produced mapping from WET output fields to DRY input sources; enables tracing and editing |
| **Operation** | SDK method that produces diffable config (intent, not execution) |
| **WET** | Explicit manifests — what actually deploys |
| **DRY** | Developer intent: templates, values, workload specs — authored, not deployed directly |
| **Evidence** | cub-scout observation: structured diff + provenance (cluster vs. intended state) |
| **Overlay drift** | WET field changed without DRY input change; transitional state, not steady-state |
| **Change Interaction Card** | cub-track mutation record: intent + decision + execution + outcome |
| **Receipt** | Compact DRY write-back to Git, linking to full WET in ConfigHub |

---

## 25. Related Documents

> **Note:** The files listed below are working documents in the author's local
> project directories (`~/Desktop/App Generators/` and the `cub-scout` repo).
> They are not published to any shared repository yet. If you are reading this
> document outside that context, treat the references as pointers to companion
> specs that may not be available to you.

> **Alignment note:** The published ConfigHub docs page
> (`docs.confighub.com/background/config-as-data/`) currently frames config-as-data
> by critiquing Helm and templates. This document takes the opposite approach:
> embrace Helm as a generator, and position ConfigHub as the governance layer for
> what Helm produces. The published docs should evolve to match this framing —
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
| `gitops-checkpoint-prd.md` | GitOps checkpoint proposal |
| `gitops-checkpoint-schemas.md` | Schema contracts for checkpoint objects |
| `ai-and-gitops-v2-draft.md` | Earlier v2 draft (superseded by this doc) |

### cub-scout and Evidence (cub-scout/docs/)

| File | Purpose |
|------|---------|
| `reference/evidence-export-v1.md` | Evidence bundle export format (current: BundleSummary) |
| `reference/next-gen-gitops-ai-era.md` | Explainer: next-gen GitOps in AI era |
| `getting-started/app-and-ai-gitops-plain-english.md` | Plain English explainer |
| `reference/glossary.md` | ConfigHub concepts glossary |

---

## Appendix B: Frequently Asked Questions

### Why is agentic GitOps bidirectional when classical GitOps is not?

Classical GitOps is deliberately one-directional: Git is the single source of
truth, reconcilers pull from it and converge the cluster toward it. If someone
changes something in the cluster directly, the reconciler overwrites it on the
next sync. That's the design — Git wins, always.

The problem is that this breaks down in three real-world scenarios:

**1. Break-glass fixes.** An operator patches a production deployment at 3am to
stop an outage. Classical GitOps treats this as drift and reverts it. The fix
disappears. The operator learns to disable reconciliation before making emergency
changes — which means the cluster is now unmanaged during the most critical
moments.

**2. Runtime-discovered state.** Some configuration only becomes known at
runtime — autoscaler adjustments, certificate rotations, admission webhook
mutations. These are legitimate changes that originate in the cluster, not in
Git. Classical GitOps either fights them (revert loop) or ignores them (fields
excluded from sync, which creates invisible blind spots).

**3. AI agents acting on live systems.** An agent observes a problem, proposes a
config change, and wants to apply it. In classical GitOps, the agent must commit
to Git first and wait for reconciliation. But for time-sensitive operational
changes, that round-trip may be too slow — and the agent may not have Git write
access, or the change may need approval before it reaches Git.

In all three cases, the information flow needs to go **cluster → intended
state**, not just **intended state → cluster**. That's the reverse direction.

The reason agentic GitOps makes this more acute is volume: a human might make a
break-glass fix once a month. An AI agent might propose runtime-informed changes
dozens of times a day. Without a governed reverse path, you either block the
agent from acting (defeating the purpose) or let it act outside the system of
record (defeating governance).

The governed reverse flow in this model works like this: the change happens,
cub-scout observes it, evidence is produced, and that evidence triggers an
explicit proposal back to intended state — a merge request, not a silent
overwrite. The proposal is reviewed (by human or policy) and accepted or
rejected. If accepted, intended state is updated to match. If rejected, a revert
is proposed instead.

Neither direction is automatic. Both are governed. That's what makes it
bidirectional rather than just "Git wins" or "cluster wins."

---

### How is this different from Helm/Kustomize?

Helm and Kustomize are generators — they produce config from templates and
overlays. But they don't give you the provenance wrapper (generator version +
input digest + operations list), they don't store the output as intended state in
a system of record separate from "latest commit on main," and they don't produce
evidence when things drift. This model completes the loop: generate, store with
provenance, publish, reconcile, observe, and record evidence.

You don't replace Helm or Kustomize. You store their output as a Unit with
provenance metadata (generator name, version, input digest), and gain the audit
trail they don't provide on their own. For Git or OCI, export as a
`GeneratorOutput` envelope.

---

### How is this different from ArgoCD/Flux?

Flux and Argo are reconcilers — they make runtime match a declared source. This
model doesn't replace them; reconciliation remains their job (the "Reconcile" row
in the comparison table is unchanged). What this model adds is the governance
layer around reconciliation: capturing intent before publish, producing structured
evidence after reconcile, and recording the full mutation chain (who proposed,
what policy evaluated, what outcome resulted) in a way that Git logs and
Flux/Argo events alone cannot.

---

### Should I adopt config-as-data if I'm doing agentic GitOps?

Yes. Agentic GitOps increases mutation volume — AI agents may author dozens or
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
   scanning file trees across repos — a project, not a query.

The generator framing makes adoption incremental. If you already use Helm or
Kustomize, you already produce config-as-data — `helm template` outputs literal
manifests. The adoption path starts by capturing that output as Units (Level 1,
no migration required) and adds provenance, validation, and governance as
maturity increases.

For non-deterministic generators (LLMs), the same provenance fields apply but
the determinism guarantee does not hold — re-rendering the same prompt may
produce different output. Governance compensates: pre-publish validation catches
constraint violations regardless of how the output was produced, and trust tiers
require human review for AI-authored changes. See section 10 (Invariants) and
section 2 (Generator Maturity Levels) for details.

---

### How would an enterprise verification system work?

At minimum, it runs as a gated chain:

1. Capture explicit intent and render deterministic WET artifacts.
2. Evaluate risk signals (`confighub-scan`) and semantic assertions (`verified`).
3. Produce a policy decision (`ALLOW | ESCALATE | BLOCK`) with reason codes.
4. Issue short-lived execution authority only on `ALLOW`.
5. Reconcile via Flux/Argo, observe outcome via cub-scout, and record attestations.

The key property is enforceability: verification evidence must be required before
execution authority is granted, not added after the fact as documentation.

---

### This sounds like an IDP. Is it?

No. IDPs hide infrastructure behind portals. This model exposes infrastructure
through explicit, diffable configuration. The abstraction compresses (Score
workload → K8s manifests); it does not conceal. If you cannot `cat` the output
and read every line, the abstraction is broken.

Backstage can be a client of this model — capturing intent via forms and showing
receipts and evidence from ConfigHub — but it is not the system of record. If you
delete Backstage, Units, provenance, and evidence survive intact.

---

### Won't this become a controller?

Controllers reconcile — they watch state and converge toward it. ConfigHub stores,
governs, and publishes. cub-scout observes and records. Neither reconciles.
Evidence informs decisions but never enacts them autonomously. The operating
boundary (section 11) is explicit: reconciliation belongs to Flux/Argo, not to
ConfigHub or cub-scout.

---

### What about secrets?

Secrets management is explicitly out of scope (section 23). ConfigHub does not
manage secrets at rest. Generators may reference secret names or external secret
store paths, but the secret values themselves are never stored in Units or
provenance metadata. Integration with external secret operators (External Secrets
Operator, Sealed Secrets, Vault) is via reference, not by value.

---

### Can I use this without ConfigHub?

Yes. Stage 0 (OSS Local) requires only `cub-track` and works with any Git
repository. You get commit-linked mutation history, explain, and search — with no
backend dependency. cub-scout also works standalone against any kubectl context.
ConfigHub adds centralized storage, cross-repo search, policy evaluation, and
governed execution — but the local tools are useful on their own.

---

### What happens if I stop using ConfigHub?

Your generators and templates live in Git — always yours. Nothing about the
authoring workflow changes if ConfigHub disappears.

What you keep:
- All DRY inputs (charts, values files, Score workloads, framework config) — in Git
- All generator code — in Git or your own registries
- All cub-track mutation history — in Git (trailers, receipts, metadata branch)
- All Units — exportable as `GeneratorOutput` envelopes (structured YAML)
- All evidence bundles — structured YAML, exportable via API

What you lose:
- Cross-repo, cross-cluster queries ("show me everything labeled `app=payments`")
- Policy evaluation at write time
- Centralized provenance index and stale render detection
- Evidence correlation across environments
- Retention policies and compliance exports
- Trust tiers and governed execution

The value is in the centralized queries and governance — the platform layer on
top of your data. If you leave, you lose the platform, not the data. All formats
are open and all data is exportable.

---

### Why ConfigHub instead of building all of this on Git?

Git is excellent at what it does: versioned text diffs, review workflows (PRs),
immutable history, distributed collaboration. The problem is what Git does not do
well — and what you would have to build on top of it if you tried to make Git the
entire system of record for governed operations.

**What Git cannot do natively:**

- **Structured query across repos.** "Show me every Unit labeled `app=payments`
  across all 40 team repos" is not a Git operation. You would need an external
  index — which is a database, which is what ConfigHub is.
- **Policy evaluation at write time.** Git accepts any commit that passes hooks.
  Governing what *content* is allowed — constraint validation, trust tier checks,
  approval chains — requires a layer that understands the content semantically,
  not just as text diffs.
- **Rich metadata and correlation.** Linking a commit to the evidence bundle that
  triggered it, the policy decision that approved it, and the runtime outcome that
  resulted — across repos, across clusters — requires relational or graph queries
  that Git's content-addressed store cannot express.
- **Retention and compliance.** Git retains everything forever (or loses it on
  rebase). Operational data needs retention policies, redaction, and export
  controls. Bolting these onto Git means building a database layer on top of Git.
- **Fast label-based lookups.** "What is deployed to `variant=prod` in
  `cluster=east-1`?" is an instant query in a database. In Git, it requires
  scanning file trees across branches and repos.

**The DRY/WET boundary is the answer.** Git stores compact, reviewable, immutable
linkage artifacts — trailers, receipts, intent diffs. ConfigHub stores the full,
rich, queryable operational state — policy traces, approval chains, execution
records, evidence indexes. Neither duplicates the other. Git remains the
collaboration surface; ConfigHub is the operational database.

The anti-pattern is trying to make Git do both jobs. That path leads to bloated
repos full of operational metadata that no one reviews in PRs, custom tooling to
query across repos, and eventually a database-on-Git that would have been simpler
to build as an actual database.

---

### What happens when a generator has a bug?

Generator output is reproducible: given the same `generator.name`,
`generator.version`, and `inputs.digest`, the output must be byte-identical.
This means a bug is reproducible too. You can re-render the same inputs with the
same generator version and see the same bad output, then diff it against the
fixed version. The provenance record tells you exactly which Units were produced
by the buggy version, so you know what to re-render. See section 19 (Failure
Modes) for the full list.

---

### How does this handle multi-cluster?

Each cluster is a Target. A Deployment binds one App to one Target — the same
App deployed to three clusters produces three Deployments, each with its own
Unit(s), Variant(s), and evidence. cub-scout in standalone mode observes one
cluster at a time. Connected mode (via ConfigHub) aggregates evidence across
clusters, enabling fleet-wide drift detection and cross-cluster comparison.

---

### How should platform engineers and portals adopt this?

Platform engineers are the primary authors of the governed layer. Their adoption
path:

**Step 1: Name what you already have.** Most platform teams already run Helm
charts, Kustomize overlays, or internal scripts that produce K8s manifests. These
are generators — they just aren't called that yet. Register them: give each a
name, a version, and start wrapping their output with an input digest. This is
discovery, not invention.

**Step 2: Define constraints.** Write down the rules that are currently enforced
by PR review, tribal knowledge, or post-deploy panic. Express them as platform
constraints: `tls-required-in-prod`, `min-replicas-2`, `images-from-approved-
registry`. These become inputs to generators and validation gates.

**Step 3: Build framework generators.** For teams using Spring Boot, Score, or
other frameworks with strong conventions, build generators that read the framework
config and produce explicit manifests. This is where the abstraction pays off —
the developer writes what they already write, and the generator narrates it into
K8s wiring.

**Step 4: Add a portal (optional).** Backstage or any other portal can be a
client — capturing intent via forms, invoking generators, and displaying receipts
and evidence from ConfigHub. The portal is a UX layer, not a system of record.
If you delete the portal, everything survives. If you never build a portal, the
CLI and Git workflow still work.

The key principle: platform engineers provide generators and constraints.
Application teams provide intent. The boundary between them is explicit and
inspectable.

---

### Do I keep using Helm?

Yes. Helm is a generator. You keep using it.

The change is what happens to Helm's output. Today, `helm template` renders
manifests and either a human reviews them or Flux/Argo runs Helm directly in the
cluster. In this model, the rendered output is stored as a Unit with provenance
metadata — generator name, chart version, values digest, render timestamp.

What you gain:

- **Audit trail.** Six months from now, you can answer "what chart version and
  values file produced this Deployment?" by looking at the Unit's provenance,
  without digging through CI logs or Helm release history.
- **Drift detection.** cub-scout can compare what Helm intended with what's
  actually running and produce evidence when they diverge.
- **Constraint validation.** Platform constraints can evaluate the rendered output
  before it publishes — catching violations at render time, not after deploy.

What you don't gain by switching away from Helm: nothing. Helm is a fine
generator. The model is about what surrounds the generator, not what replaces it.

For Flux HelmRelease users specifically: Flux runs Helm inside the cluster (inner
loop). This means the rendered output lacks pre-publish provenance unless the
values and chart reference are themselves governed in the outer loop. The adoption
path: keep HelmRelease, but consider also storing the expected rendered output as
a Unit so you can detect drift between what Helm intended and what actually exists.

---

### Where do I edit config? DRY source or ConfigHub?

Both — through the same surface.

If you author config in DRY format (Helm values, Score workloads, Spring Boot
`application.yaml`), you should edit in DRY space. That's where your intent
lives. But you don't need to hunt for the right file in Git. ConfigHub resolves
it for you.

The experience: you open ConfigHub, find the deployed config, click the field you
want to change. ConfigHub's field-origin map (section 3) tells you where that
value comes from — which file, which line, in which repo. You edit the value in
ConfigHub's UI. ConfigHub commits the change to the DRY source in Git. The
generator re-renders. The WET updates. The reconciler applies.

You edited DRY, but you did it *through* ConfigHub. The generator contract is
preserved. Provenance stays clean. And you didn't need to know the repo
structure, file layout, or which values overlay applies to which environment.

**What about fields with no DRY source?** Platform-injected fields (network
policy, security constraints), emergency overrides, deployment topology, and
governance state are edited directly in ConfigHub — there's no upstream to
redirect to. ConfigHub *is* the right editing surface for these. The field-origin
map distinguishes "this has a DRY source, edit there" from "this is
control-plane-native, edit here."

**What about WET overlays?** You can patch WET output directly when needed —
emergency overrides, one-off variant customizations. But overlays on
generator-backed fields are a transitional state, not a permanent workflow. The
system creates friction (cub-track redirection, overlay drift classification,
staleness detection) that encourages promotion back to DRY. See section 7 and
section 18 for the full model.

---

### My company wants their own internal Heroku. How do I do that?

The "internal Heroku" pattern is: developers say "deploy my app" and the platform
handles everything else. This is a legitimate goal. The question is whether the
platform hides or exposes the resulting infrastructure.

In this model, you build the Heroku experience as a generator + constraints layer:

**What the developer sees:**

```yaml
# Something simple — Score, a custom app manifest, or even a Backstage form
name: payments-api
image: ghcr.io/acme/payments:1.2.3
port: 8080
expose: true
```

**What the platform does:**

1. A generator reads this intent plus platform context (constraints, capabilities,
   environment labels)
2. The generator produces explicit K8s manifests: Deployment, Service, Ingress,
   HPA, PDB, NetworkPolicy — whatever the platform requires
3. The output is stored as a Unit with full provenance
4. Flux/Argo reconcile the cluster

**How this differs from actual Heroku (and from most IDPs):**

The developer *can* see the generated output. They don't have to — the
abstraction is designed so they normally don't need to. But when something breaks
at 3am, they (or the on-call engineer) can `cat` the generated manifests and read
every line. The abstraction compresses; it does not conceal.

The platform team controls the generator and the constraints. They can change how
`expose: true` is implemented (Ingress today, Gateway API tomorrow) without
changing the developer interface. The change shows up as a diff in the generated
output, not as invisible infrastructure magic.

This is the same pattern as Score.dev (section 20) or Spring Boot generators
(section 5b) — the developer writes familiar, minimal config; the platform
translates it into explicit infrastructure wiring. The "Heroku feel" comes from
the simplicity of the input, not from hiding the output.

---

### What is app config in this model?

App config is the runtime configuration that the application reads — environment
variables, config maps, feature flags, connection strings, tuning parameters. It
is distinct from *deployment config* (the K8s manifests that describe how the
application is deployed).

In practice, the line between them blurs. This model handles both:

**App config embedded in DRY intent.** Environment variables in a Score workload
or Spring Boot `application.yaml` are part of the generator input. The generator
renders them into ConfigMaps or env entries in the Deployment spec. They follow
the same generate → store → publish → observe cycle as everything else.

**App config as a separate Unit.** Feature flags, runtime tuning, and service
config that change independently of deployments can be managed as their own Units.
They have their own provenance, their own Variants (different flags per
environment), and their own evidence when they drift.

**App config from external systems.** Secrets from Vault, flags from LaunchDarkly,
connection strings from a service mesh — these are referenced, not stored.
Generators can produce the *references* (ExternalSecret CRs, ConfigMap entries
pointing to external sources), but the values themselves are never in ConfigHub.

The important point is that app config is just config in this model. Whether it arrives via a generator
or via direct import, it becomes a Unit with labels, provenance, and a Variant.
The same staging model applies — promote feature flags from `variant=dev` to
`variant=prod` with the same approval policy as any other config change.

The distinction between "app config" and "deployment config" matters to the
developer (they think about them differently) but not to the storage and
governance model (both are Units).

---

### What is the right model for ConfigHub Actions?

ConfigHub Actions are the execution layer — the part of the system that actually
changes runtime state. The question is how to model them so they fit the same
discipline as everything else: explicit, governed, auditable.

**The principle: actions are operational config, not imperative scripts.**

Just as a generator takes DRY intent and produces deployment manifests, an action
generator takes operational intent and produces *action manifests* — explicit,
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

All three are expressed as config — declarative descriptions of desired
operational state changes, not shell scripts or imperative code.

**How actions fit the governed model:**

1. An action manifest is *generated* like any other config — from intent, through
   a generator, with provenance. The generator for an action might be a framework
   SDK method (`app.rollout(image="1.3.0")`) or a workflow template.
2. The manifest is *governed* like any other config — policy evaluates it against
   the trust tier, checks preconditions, and issues a scoped execution token.
3. The runtime (`confighub-actions`) *interprets* the manifest — it reads the
   steps and executes them within the token's scope. The runtime is the only
   component that touches the cluster.
4. The outcome is *observed* — cub-scout captures post-execution state, and the
   attestation records what happened against what was intended.

**Why config manifests, not scripts?**

Scripts are opaque — you can't diff them meaningfully, you can't validate their
effects before execution, and you can't compare intended vs actual at the field
level. Action manifests are structured data: you can diff them, validate them
against constraints, preview their effects, and compare the intended operation
against the observed outcome field by field.

This keeps the same invariant: nothing implicit ever executes. The action manifest
is the plan. The attestation is the receipt. The evidence is the proof.
