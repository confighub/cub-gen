# User Experience: Import from Git & Using Generators

**For:** Brian, Jesper
**From:** Alexis (via conversation with Claude, 2026-02-26)
**Context:** Brian's concern about DRY authors being unhappy when WET gets
customized. This doc describes what both paths should *feel like* to users.

---

## A. Import from Git

### Who is this user?

They already have YAML in Git. Helm charts, Kustomize overlays, raw manifests,
or a mix. They use Flux or Argo today. They're not going to rewrite anything.
They want to see what they have, compare across environments, and maybe get
governance later.

### The experience

**Step 1: Connect a repo**

```
$ confighub import connect --repo github.com/acme/platform-config

Scanning repository...
Found:
  12 Helm releases (charts/ + values files)
   4 Kustomize overlays (overlays/dev, staging, prod, canary)
   3 raw manifests (jobs/, one-off CRDs)

Import as Units? [y/n]
```

No migration. No restructuring. ConfigHub reads the repo, classifies what it
finds, and offers to import. The repo structure stays exactly as it is. Git
remains the source of truth.

**Step 2: See everything in one place**

```
$ confighub query --format table

APP              VARIANT   TYPE         SOURCE                        UPDATED
payments-api     dev       helm         charts/payments + values-dev   2h ago
payments-api     staging   helm         charts/payments + values-stg   2h ago
payments-api     prod      helm         charts/payments + values-prod  3d ago
orders-service   dev       kustomize    overlays/dev/orders            1d ago
orders-service   prod      kustomize    overlays/prod/orders           5d ago
cert-rotation    —         raw          jobs/cert-rotation.yaml        30d ago
```

This is the first "aha" moment. They've never seen all their config in one
table before. They've had to navigate repo directories, read kustomization.yaml
files, or run `helm list` per cluster.

**Step 3: Compare across environments**

```
$ confighub diff payments-api --from dev --to prod

  replicas:                1 → 5
  image.tag:               1.2.3 → 1.2.1        ← prod is behind
  resources.limits.memory: 256Mi → 1Gi
  tls:                     false → true
  values file:             values-dev → values-prod
  chart version:           same (33.2.1)
```

Second "aha." They can see *why* environments differ — not just that the YAML
is different, but which values-file knobs are turned differently. For Helm
imports, this is a diff of the values files, not the rendered output.

**Step 4: Detect staleness (passive, automatic)**

```
$ confighub check-freshness

⚠ payments-api (variant=prod)
  values-prod.yaml changed 3 days ago
  Last render: 5 days ago
  → WET output is stale. Re-render to pick up changes.

✓ orders-service (all variants fresh)
✓ cert-rotation (raw manifest, no generator)
```

They didn't have this before. They had "did Flux sync?" but not "did someone
change the values file and forget to re-render?"

**What import does NOT do:**

- Does not copy YAML into ConfigHub and delete it from Git
- Does not require changing repo structure
- Does not require changing Flux/Argo configuration
- Does not require adopting generators

Import is observation + queryability. Git is still the source. ConfigHub is a
read layer that makes the source queryable and comparable.

**What import enables next:**

- Provenance tracking (Level 2): ConfigHub knows which generator and inputs
  produced each Unit, so it can detect staleness and trace field origins
- Editing through ConfigHub (see section B): once field-origin maps exist,
  users can edit values through ConfigHub instead of hunting for files in Git
- Governance: policy evaluation on the imported config before it reaches clusters

### The pitch to this user

*"You keep your repos exactly as they are. ConfigHub reads them and gives you a
single query surface across all your environments. You can see what's deployed,
compare environments, and find stale renders — without changing your workflow."*

---

## B. Using Generators

### Who is this user?

Two personas:

**Platform engineer:** Builds the generator. Defines constraints (TLS required
in prod, min 2 replicas, images from approved registry). Wants app teams to
self-serve without breaking things.

**App developer:** Writes DRY intent (Score workload, Helm values, Spring Boot
config). Doesn't want to think about Kubernetes primitives. Wants to change a
value and have it deploy correctly.

### The platform engineer experience

**Step 1: Register an existing generator**

```
$ confighub generators register \
    --name helm \
    --version "3.14.0 / traefik@33.2.1" \
    --input-schema values-schema.json

Registered: helm
Input schema: 14 fields (replicas, image.tag, resources.*, tls, ...)
Field-origin map: auto-generated from schema
```

They're not building a new tool. They're naming what they already have (Helm,
Kustomize, Score) and telling ConfigHub about it. The field-origin map can be
auto-generated from the values schema for common cases — or hand-authored for
complex charts.

**Step 2: Define constraints**

```yaml
# platform-constraints.yaml
apiVersion: confighub.io/v1
kind: PlatformConstraints
metadata:
  name: acme-production
spec:
  when:
    variant: prod
  rules:
    - field: spec.replicas
      min: 2
      reason: "Production requires at least 2 replicas for availability"
    - field: spec.template.spec.containers[*].image
      pattern: "registry.acme.io/*"
      reason: "Production images must come from the approved registry"
    - field: spec.template.spec.containers[*].resources.limits
      required: true
      reason: "All production containers must have resource limits"
```

These are declarative rules, not scripts. They evaluate against the rendered
WET output before publish. The app developer never sees them unless they
violate one.

**Step 3: See what the generator produces across all teams**

```
$ confighub query --generator helm --format table

APP              VARIANT   CHART VERSION   VALUES DIGEST   RENDERED    STATUS
payments-api     prod      33.2.1          sha256:a1b2     3d ago      ⚠ stale
payments-api     dev       33.2.1          sha256:c3d4     2h ago      ✓ fresh
orders-service   prod      32.1.0          sha256:e5f6     1d ago      ✓ fresh
orders-service   staging   33.2.1          sha256:g7h8     1d ago      ✓ fresh
```

The platform engineer can see every team's generator output in one view. They
can spot version skew (orders-prod is still on chart 32.1.0), stale renders,
and constraint violations across the fleet.

### The app developer experience

**Step 1: Write DRY intent (they already do this)**

```yaml
# score.yaml (or values.yaml, or application.yaml — whatever they already use)
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
```

Nothing new here. They write what they already write.

**Step 2: See what gets generated**

```
$ confighub render --input score.yaml --variant prod --preview

Generator: score-gen@1.0.0
Context: variant=prod, platform=acme-production

Generated resources:
  Deployment/payments-api
    replicas: 2              (from: platform constraint min-replicas-2)
    image: ghcr.io/acme/payments-api:1.2.3  (from: score.yaml:6)
    resources.limits.memory: 1Gi             (from: platform default)
  Service/payments-api
    port: 8080               (from: score.yaml:8)
  Ingress/payments-api
    host: pay.example.com    (from: score.yaml:11)
    tls: true                (from: platform constraint tls-required-in-prod)

Constraint check:
  ✓ tls-required-in-prod
  ✓ min-replicas-2
  ✓ images-from-approved-registry

Field origins saved. You can trace any field back to its source.
```

This is the key moment. The developer sees exactly what gets produced, *why*
each field has its value (from their input vs. from platform constraint), and
whether it passes validation — all before anything is deployed.

**Step 3: Change a value through ConfigHub**

Later, they need to scale to 5 replicas in prod:

```
$ confighub edit --app payments-api --variant prod --field spec.replicas

Current value: 2
Source: score.yaml:8 → platform constraint min-replicas-2 (floor)
  Note: platform sets minimum 2. You can go higher.

New value: 5

→ Updating score.yaml in acme/payments-api repo...
→ Re-rendering with updated input...
→ Diff:
    spec.replicas: 2 → 5
    (no other fields affected)
→ Constraint check: ✓ all pass

→ Commit to acme/payments-api? [y/n]
```

They edited through ConfigHub. ConfigHub resolved the field-origin map, found
the DRY source (score.yaml), wrote the change there, re-rendered, validated,
and showed the diff. The developer didn't need to know that `replicas` in the
Deployment comes from their Score workload filtered through a platform
constraint. They just said "change replicas to 5" and the system did the right
thing.

**Step 4: See what's deployed (ongoing)**

```
$ confighub query --app payments-api --format table

VARIANT   REPLICAS   IMAGE TAG   TLS    GENERATOR          RENDERED    STATUS
dev       1          1.2.3       false  score-gen@1.0.0    2h ago      ✓
staging   2          1.2.3       true   score-gen@1.0.0    2h ago      ✓
prod      5          1.2.3       true   score-gen@1.0.0    just now    ✓
```

This becomes the daily view. Open ConfigHub, see your app across all
environments. Click any field to trace it or change it.

**Step 5: When something breaks**

```
$ confighub provenance payments-api --variant prod

generator:     score-gen@1.0.0
inputs.digest: sha256:a1b2c3...  (✓ fresh)
rendered.at:   2026-02-26T10:30:00Z

field_origins:
  spec.replicas             → score.yaml:8           (editable_by: app-team)
  spec.resources.limits     → platform-context.yaml  (editable_by: platform-team)
  spec.containers[0].image  → score.yaml:6           (editable_by: app-team)
  ingress.tls               → platform constraint    (editable_by: platform-team)

Last 3 changes:
  10:30  spec.replicas: 2 → 5      (by: alexis, decision: ALLOW)
  09:15  image.tag: 1.2.1 → 1.2.3  (by: ci/github-actions, decision: ALLOW)
  3d ago chart: score-gen 0.9 → 1.0 (by: platform-team, decision: ALLOW)
```

Full provenance. Every field traceable. Every change attributed. No archaeology.

---

## How A leads to B

Import (path A) is how most users start. They have YAML in Git, they connect
it, they see the queryable view. That builds trust.

Generators (path B) come when they're ready. Maybe they register their existing
Helm charts as generators to get provenance. Maybe they adopt Score for new
services. Maybe the platform team builds a Spring Boot generator. Each step is
incremental.

The adoption path:

```
1. Import from Git           → "I can see everything in one place"
2. Add provenance metadata   → "I can trace what produced this"
3. Add field-origin maps     → "I can trace any field to its source"
4. Edit through ConfigHub    → "I can change values without Git archaeology"
5. Add constraints           → "I can validate before deploy"
6. Add governance            → "I can prove who authorized what"
```

At no point does the user need to abandon their existing workflow. Git stays.
Flux/Argo stay. Helm stays. ConfigHub adds a layer on top — first for
visibility, then for traceability, then for editing, then for governance.

---

## Brian's concern, addressed

Brian worried that DRY authors won't be happy when WET gets customized via
overlays. The answer:

1. **ConfigHub is where they edit, but it writes back to DRY.** The user
   changes a value in ConfigHub. ConfigHub resolves the field-origin map, finds
   the DRY source file, commits the change there. The generator re-renders.
   DRY stays the source of truth. The user never touched WET.

2. **WET overlays are a transitional escape hatch, not a workflow.** If someone
   patches WET directly, the system flags it (cub-track `suggest`), classifies
   it as overlay drift, and creates friction to promote the change back to DRY.

3. **For fields that don't have a DRY source** (platform policy, emergency
   overrides, deployment topology), editing in ConfigHub directly is correct.
   The field-origin map distinguishes "this has a DRY upstream, edit there"
   from "this is control-plane-native, edit here."

The user experience is: **edit in one place (ConfigHub), the system routes the
change to the right layer.** DRY authors stay in DRY. Platform teams stay in
the control plane. Nobody edits generated output unless it's an emergency, and
emergencies are governed and tracked.
