# Platform Generators Manifesto

Status: plain-English synthesis of the current `cub-gen` thesis and gaps.

Short answer: very close in intent, partly real in examples, not yet fully
productized.

## Front-Door Doctrine

Start with the user model, not the implementation model:

```text
Component
  -> Deployable Variant
      -> Target
      -> Connections
      -> Proof
```

The vocabulary should stay small:

| Term | Plain-English meaning |
|---|---|
| Component | reusable app, service, workflow, or platform base |
| Deployable Variant | concrete deployable copy/context of a Component: env, region, tenant, customer, cluster |
| AI Variant | Deployable Variant whose delta, wiring, or operation is AI-assisted and governed |
| Target | place the variant runs or reconciles |
| Connection | dependency, secret, database, API, service binding, or platform contract the variant is wired to |
| Change | requested alteration to source, rendered config, wiring, or operation |
| Proof | evidence of origin, ownership, rendered impact, decision, and runtime state |

This is the product surface people should understand first. The generator model
explains how `cub-gen` can produce that surface from existing repos.

The manifesto is the clean version of what the early `cub-gen` planning docs
were already reaching for:

1. Import existing platform and app repos, not replace them.
2. Model them as generators from source intent to deployable config.
3. Preserve provenance and inverse edit paths.
4. Route changes as apply-here, lift-upstream, or block/escalate.
5. Link GitHub PRs and ConfigHub MRs with one `change_id`.
6. Keep Git as intent/source, ConfigHub as governance/query/runtime evidence.

## The Core Claim

Most Kubernetes app config is generated now, but teams do not have a clean way
to govern the generator boundary.

Many "platform abstractions" are actually generators:

```text
source intent + environment context + platform contract -> deployable config
```

That matters because a generator can be inspected. A bad abstraction hides
Kubernetes behind knobs until every missing use case becomes another feature
request. A clean generator records the source layers and routes requested
changes to the right owner.

## Why This Is Not a New Direction

The original generator PRD already says the product must import Helm, Score,
Spring Boot, and custom platform patterns, convert them into DRY/WET generator
contracts, and govern changes through ConfigHub MR flow.

The canonical triple already formalizes the same idea:

| Contract artifact | What it answers |
|---|---|
| `GeneratorContract` | What generator ran, with which inputs and behavior |
| `ProvenanceRecord` | What WET output was produced, with field lineage |
| `InverseTransformPlan` | How a WET/live change can become a governed DRY proposal |

The manifesto is therefore not a pivot. It is a better teaching layer over the
same architecture.

## Where Examples Are Strongest

| Manifesto claim | Current proof |
|---|---|
| Platforms are generators | Helm, Score, Spring, Backstage, app-config, workflow examples import as generator families |
| Trace rendered config back to source | `field_origin_map` and `inverse_edit_pointers` exist in import output |
| Route changes to the right layer | Spring Boot example proves apply-here / lift-upstream / block-escalate |
| Link PR and MR | Bridge promote commands and demo scripts exist |
| Keep Git plus Argo/Flux | Helm example has Flux/Argo live proof |
| Use ConfigHub for evidence/governance | connected demos produce bundles, attestations, and decision state |

## The Spring Ancestor

The clearest ancestor is the external `spring-platform` example. It teaches the
same model with fixed inputs:

```text
Spring app config + platform policy -> operational ConfigHub/Kubernetes state
```

It names the key properties:

| Property | Meaning |
|---|---|
| invertible mapping | every operational field traces back to one source field |
| field provenance | the system can explain why a field has its value |
| ownership boundaries | fields are app-owned or platform-owned |
| mutation from provenance | routing follows source ownership |

It also names the three routes:

| Route | Meaning |
|---|---|
| apply here | app-owned field can be mutated directly in ConfigHub |
| lift upstream | source change is required, so route back to Git |
| block/escalate | platform-owned boundary requires platform decision |

`cub-gen/examples/springboot-paas` is the product path for that model. It
computes source-side provenance, validates routes client-side, mutates embedded
ConfigMap payloads for allowed app-owned fields, and has connected/live proof.

## Where It Is Still Far

| Gap | Current state | Desired product shape |
|---|---|---|
| Generic platform import | import is repo/generator oriented | import a platform estate and stitch app/platform repos into one graph |
| OpenChoreo adapter | fixture-backed initial adapter | read Workload, ReleaseBinding, SecretReference, ComponentType, RenderedRelease |
| App-of-apps adapter | fixture-backed initial adapter | broader root/child app catalog analysis |
| Automatic annotation/enrichment | not generic | generate sidecar provenance and optional PRs for annotations |
| "Better config" generation | limited to Spring starter scaffolding | propose normalized variants, secret references, ownership metadata, and source lifts |
| Multi-env/tenant fanout | no one-command fanout | one invocation emits one bundle per env/tenant/app variant |
| PR/MR magic | contracts, commands, and demos exist | seamless creation/linkage with evidence attached |
| Server-side policy enforcement | client-side proof is stronger than backend enforcement | ConfigHub rejects/blocks governed writes authoritatively |

## How To Explain This to Skeptics

The skeptical concern:

```text
The abstraction blocks a real Kubernetes need.
Users ask for a parameter.
The platform adds a knob.
Repeat until the abstraction is not an abstraction.
```

The generator answer:

```text
Assume the model is incomplete.
When a generated field changes, classify the source and owner.
If it is app-owned, route it to app source.
If it is environment-owned, route it to variant source.
If it is platform-owned, route it to platform review.
If it violates policy, block it.
If it is an emergency, record an overlay with TTL and later promote or revert.
```

The goal is not a perfect abstraction. The goal is a governed escape path.

## Example Generator Families

| Platform style | Generator input | WET output | Notes |
|---|---|---|---|
| Helm platform | chart, values, overlays, invocation args | Kubernetes manifests | broadest adoption, hardest provenance |
| Score.dev | `score.yaml` plus platform contracts | Kubernetes manifests | clean app intent |
| Spring Boot platform | `application*.yaml`, build metadata, platform policy | Deployment, Service, ConfigMap | strongest route teaching |
| OpenChoreo | Workload, ReleaseBinding, SecretReference, ComponentType | RenderedRelease and K8s resources | fixture-backed initial adapter; not full upstream conformance |
| Argo ApplicationSet | selectors, list/git/cluster generators, template | Argo Applications | already partially supported |
| Argo app-of-apps | root app and child app catalog | child Argo Applications | fixture-backed initial adapter |
| app-config manager | provider config or literal app config | explicit app config data | identity generator case |
| workflow platform | workflow graph, schedules, approvals | workflow/action manifests | operations as config |

## Useful Reading Order

1. [Generators PRD](10-generators-prd.md)
2. [Field-Origin Maps and Editing](20-field-origin-maps-and-editing.md)
3. [Canonical Triple and Storage Boundary](../../contracts/canonical-triple-and-storage-boundary.md)
4. [OpenChoreo as a Clean Generator](../03-worked-examples/05-openchoreo-generator-worked-example.md)
5. [Argo ApplicationSet and App-of-Apps as Generators](../03-worked-examples/06-argo-generators-worked-example.md)
6. [Platform Generator Product Worklist](../../plans/2026-04-30-platform-generator-product-worklist.md)
