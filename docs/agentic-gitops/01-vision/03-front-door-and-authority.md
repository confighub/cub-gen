# Many Front Doors, One Authority Layer

> Teams will keep many ways to create config. ConfigHub is the place where that config becomes authoritative, governable, and safe to operate.

**Status:** Draft
**Date:** 2026-03-30
**Audience:** GitHub readers, internal teams, platform teams, app teams

---

## Plain-English thesis

The industry does not have one way to build and run apps on Kubernetes.

Some teams still hand-write YAML. Some use Helm. Some use framework-native
inputs like Spring Boot `application.yaml`. Some use workload contracts like
Score. Some use internal developer platforms such as Backstage. Some use
workflow systems. Some now use AI to propose code or config directly from a
prompt.

That diversity is not going away.

So the right question is not:

"What is the one correct front door?"

The right question is:

"How do all of these front doors converge on one authoritative, governable,
queryable model of intended state?"

Our answer is ConfigHub.

ConfigHub is not just another front door.
It is the authority and governance layer behind all front doors.

Teams should be free to author intent in the tool that makes sense for them.
But before that intent becomes deployable operational state, it must cross a
boundary where it becomes:

- explicit
- queryable
- provenance-linked
- policy-evaluable
- verifiable
- attestable

That boundary is the value of ConfigHub.

---

## The front-door axioms

These are the core ideas behind the model.

### 1. There will never be one authoring surface

Real organizations have:

- framework-native apps such as Spring Boot services
- ad hoc apps with handwritten config
- platform-generated apps created through IDPs
- ops and workflow systems that emit config
- AI-assisted systems that generate code or config from prompts

Trying to force all of them into one authoring UI is not realistic.

### 2. Generated config is unavoidable

Even teams that say they want "configuration as data" still need some process
that turns compact intent into deployable reality.

Examples:

- a Spring app plus environment settings becomes a `Deployment`, `Service`, and
  runtime `ConfigMap`
- a workload contract becomes Kubernetes resources
- a prompt becomes a proposed patch or operational workflow

The question is not whether generation exists.
The question is whether it is disciplined.

### 3. Any front door is acceptable; not every publish boundary is

We can accept many ways to create intent:

- handwritten YAML
- framework-native config
- workload specs
- internal developer portals
- scripts and workflows
- AI-generated proposals

But we should not accept every kind of output as authoritative.

Nondeterministic output, hidden logic, and untraceable post-processing are not
safe publish boundaries.

### 4. Configuration becomes data at the authority boundary

Upstream authoring can look like code, templates, forms, prompts, or framework
conventions.

Downstream of the authority boundary, intended state must be explicit data.

That means:

- literal values
- stable identity
- versioned revisions
- provenance back to inputs
- ownership boundaries
- machine-readable policy and verification status

ConfigHub is where this conversion becomes durable and queryable.

### 5. AI makes the authority layer more important, not less

Prompt-to-code and prompt-to-config increase change volume and introduce
nondeterminism.

That means AI output should be treated as a proposal, not as authority.

The more generation happens upstream, the more valuable it becomes to have one
system that records:

- what was proposed
- what it rendered into
- what policy decided
- what was verified
- what was attested

### 6. ConfigHub is the authority and governance layer

This is the strongest claim in the model:

- front doors stay plural
- authority becomes singular

Git remains a collaboration surface.
Backstage remains a portal.
Spring Boot remains a framework.
Score remains a workload contract.
AI remains a proposal engine.
Flux and Argo remain reconcilers.

ConfigHub is the system of record for governed intended state.

---

## 1. State of the industry: apps on Kubernetes

The modern Kubernetes estate is a mix of several app creation styles.

### Ad hoc apps

These teams often have:

- handwritten manifests
- Kustomize overlays
- shell scripts or repo-specific conventions

They usually want control and familiarity, but pay for it with inconsistency,
copy-paste drift, and difficult cross-repo governance.

### Platform-based apps

These teams use some kind of app platform model:

- a developer writes a small amount of app-facing intent
- the platform fills in the operational details
- deployment YAML is generated or assembled automatically

This can be built with Backstage, Score, internal frameworks, platform APIs,
Kratix, Crossplane, Humanitec, or other systems.

The exact tooling varies, but the shape is the same:

team intent -> platform contract -> rendered operational state

### AI-assisted apps

These teams increasingly use:

- prompt-to-code
- prompt-to-config
- agent-generated PRs
- AI-proposed operational changes

This is powerful, but it pushes the same old problem to a new extreme:
generated config now appears faster, in larger volumes, and with weaker
predictability.

### Framework-native apps

Spring Boot is a good example of this class.

A Spring team often thinks in:

- `application.yaml`
- environment profiles
- Actuator behavior
- datasource settings
- feature flags

They do not naturally think in raw `Deployment` YAML first.
That does not make them less "Kubernetes-native". It means their app-facing
authoring surface is the framework, not the manifest.

### The common pattern

Across all of these styles, the same underlying pattern shows up:

- humans or tools author compact intent
- another layer fills in defaults, policies, and runtime details
- something deployable is produced

That is the app platform model in plain English, even when teams do not call it
that.

---

## 2. Vision: ConfigHub as authority for configuration as data

We believe in configuration as data.

That does not mean "nobody may generate config."
It means the output that matters operationally must be explicit, structured
data, not hidden logic.

This distinction matters.

People will continue to use different techniques upstream because they optimize
for different things:

- frameworks optimize for developer familiarity
- workload specs optimize for compact intent
- portals optimize for self-service UX
- scripts optimize for local speed
- AI optimizes for exploratory change

Those are all acceptable front doors.

But they are not all good places to stop.

ConfigHub exists to be the place where intent from any of those front doors is
normalized into governed configuration data.

In that model, ConfigHub provides the things upstream authoring tools usually do
not:

- one authoritative record of intended state
- provenance from input to rendered output
- field ownership and edit routing
- policy evaluation at write time
- verification and attestation linkage
- cross-repo and cross-environment queryability

This is why the right framing is:

- many front doors
- one authority layer

---

## 3. Why generated config is hard

Generated config is powerful, but it creates real operational problems.

### Problem 1: hidden logic

Teams can see the rendered YAML, but not always the chain of logic that produced
it.

Examples:

- a Helm chart plus values plus helper templates plus chart version
- a portal workflow that applies multiple hidden defaults
- an AI agent that edits several files based on unstated assumptions

If you cannot explain why a field exists, you do not really control it.

### Problem 2: broken ownership

Once config is generated, it is easy for teams to edit the wrong layer.

Examples:

- editing a rendered manifest instead of the Spring property that owns it
- patching a downstream YAML instead of the platform policy that should own it
- hotfixing a live object that gets silently reverted by reconciliation

This is how app source config and operational truth drift apart.

### Problem 3: too many version axes

Generated output often depends on more than one thing:

- input config version
- generator version
- policy/default set version
- platform contract version
- environment or target context

If these are not tracked together, teams cannot explain why Tuesday's render is
different from Wednesday's.

### Problem 4: AI amplifies nondeterminism

Prompt-based generation adds a new failure mode:

- the same user intent may produce different output on different runs
- the agent may silently make architectural assumptions
- the change may look reasonable in a PR but still violate policy or ownership

This is not a reason to avoid AI.
It is a reason to insist on a stronger authority boundary.

---

## 4. Good and bad generated config

Generated config is not all the same.

### Bad generated config

By "bad", we mean generation that makes the system harder to reason about and
harder to govern.

Examples:

- text templating with hidden branching and implicit defaults
- copy-paste overlays that drift across environments
- scaffolds that generate code nobody maintains
- scripts whose real behavior depends on external runtime state
- AI-generated YAML with no stable provenance or verification boundary

Helm templating is the canonical example here.
It is widely used and often useful, but it normalizes a style where important
operational behavior is spread across:

- values files
- helper templates
- chart version changes
- local rendering context

That makes ownership and field-origin reasoning harder than it should be.

### Good generated config

By "good", we mean generation that stays aligned with configuration as data.

Characteristics:

- typed or convention-based input surfaces
- deterministic generation
- explicit rendered output
- versioned generator behavior
- provenance from input to output
- ownership boundaries by field or object
- policy and verification applied after render, before authority

Examples:

- Spring Boot app config plus platform contract rendered into operational state
- Score workload intent rendered into Kubernetes resources
- buildpacks producing an image from a known source tree
- controller-driven synthesis from a typed app or workload API

The point is not "never generate."
The point is "generate in a way that preserves authority, traceability, and
governance."

---

## 5. Spring Boot on Kubernetes: a concrete app platform model

Spring Boot on Kubernetes is a very good example because the split is easy to
understand.

An app team naturally wants to author:

- Spring properties
- profile-specific overrides
- feature flags
- app behavior

A platform team naturally wants to own:

- health and readiness expectations
- datasource and secret boundaries
- replica and SLO policy
- service exposure
- reconciler packaging

That gives us a simple model:

1. The app author writes app source config.
2. The platform defines the contract, defaults, and guardrails.
3. A generator or platform control plane combines both into deployable
   operational state.
4. Flux or Argo reconciles that state to the cluster.

This is what we mean by the app platform model.

It is not "developers stop owning apps."
It is "developers own app source config, while the platform owns how that config is
made safe and operable."

This model lines up with current industry practice:

- Spring itself expects separate operational concerns such as probes,
  graceful shutdown, and externalized config
- platform frameworks expect app source config to be smaller and more stable than the
  rendered deployment bundle
- AI workflows make it even more valuable to separate proposal from authority

---

## 6. What this looks like in our Spring Boot example

The example in this repo makes the split visible.

App-facing inputs:

- [examples/springboot-paas/src/main/resources/application.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/src/main/resources/application.yaml)
- [examples/springboot-paas/src/main/resources/application-prod.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/src/main/resources/application-prod.yaml)

Platform-facing inputs:

- [examples/springboot-paas/platform/base/runtime-policy.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/platform/base/runtime-policy.yaml)
- [examples/springboot-paas/platform/overlays/prod/slo-policy.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/platform/overlays/prod/slo-policy.yaml)
- [examples/springboot-paas/platform/registry.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/platform/registry.yaml)

Rendered operational state:

- [examples/springboot-paas/confighub/inventory-api-prod.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/confighub/inventory-api-prod.yaml)

Transport to reconciliation:

- [examples/springboot-paas/gitops/flux/kustomization.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/gitops/flux/kustomization.yaml)
- [examples/springboot-paas/gitops/argo/application.yaml](https://github.com/confighub/cub-gen/blob/main/examples/springboot-paas/gitops/argo/application.yaml)

In plain English:

- the app team writes Spring config
- the platform contributes runtime and policy defaults
- the rendered output becomes the deployment bundle
- Flux or Argo deploys it
- ConfigHub is where the rendered state, provenance, and governance become
  authoritative

That is the compromise we want:

- upstream authoring stays familiar
- downstream operational truth becomes explicit and governable

---

## 7. Why App-Deployment-Target makes the model simpler

The current ConfigHub conceptual model is easier to understand when expressed as
App-Deployment-Target.

People do not naturally think in:

- repo paths
- folder conventions
- namespace fragments
- raw manifests

They think in:

- what app is this
- where is it deployed
- what is the deployment state there

That is why App-Deployment-Target is useful.

It gives teams a simple, queryable shape:

- **App**: the software service or product
- **Target**: where it runs
- **Deployment**: the app running on that target with a specific config state

This is a better organizing model for governance than "find the right folder in
the right repo."

It also makes platform claims easier to explain:

- "show me all prod deployments for this app"
- "compare staging and prod for this app"
- "which target is carrying this policy exception"

ConfigHub makes those questions first-class.

---

## 8. The experimental next step: make Platform explicit too

App-Deployment-Target is simpler than repo-centric thinking.
But in real app platforms, there is often one more thing that matters a lot:
the platform contract itself.

Many important fields are not owned by the app and are not explained by the
target alone.
They come from a reusable platform layer.

Examples:

- managed datasource policy
- approved runtime class
- SLO defaults
- service exposure rules
- secret binding patterns
- platform-provided operations

That is why we are experimenting with extending the conceptual model to make
Platform explicit:

- **App**: what the team is building
- **Platform**: the contract, defaults, and guardrails
- **Deployment**: the realized app instance
- **Target**: where it runs

This makes the ownership story clearer.

Without an explicit platform concept, too many fields look like they belong
either to the app or to the environment, when in reality they come from a
shared platform contract.

In practice, this helps explain:

- why some changes are direct app mutations
- why some changes must be lifted upstream into app source config
- why some changes are generator-owned or platform-owned and should be blocked

We should treat this as an experimental refinement, not as a breaking rename.
The important idea is simple:

not every generated field comes from the app
and not every non-app field comes from the target

some of it comes from the platform

---

## Closing

The future will have more front doors, not fewer.

There will be more framework-native config.
More internal developer platforms.
More workload contracts.
More workflow automation.
More AI-generated proposals.

That makes it more important to have one authority layer that can normalize all
of them into governed configuration data.

That is the role of ConfigHub.

ConfigHub is the place where:

- generated config becomes explicit data
- ownership becomes clear
- policy becomes enforceable
- AI output becomes governable
- deployment intent becomes queryable
- verification and attestation become durable

Many front doors.
One authority layer.
