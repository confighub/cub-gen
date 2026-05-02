# Research: Spring Boot on Kubernetes and Platform Engineering Landscape

Last reviewed: 2026-03-30

This note gathers current external guidance for Spring Boot workloads on Kubernetes and compares adjacent platform-engineering approaches that are relevant to `cub-gen`.

## Executive takeaways

- Spring's own docs position plain Spring Boot plus Kubernetes as enough for many teams to get started. The baseline is: build OCI images with buildpacks, expose Actuator probes, use graceful shutdown, and externalize config with ConfigMaps or Secrets.
- Spring Cloud Kubernetes is optional. Its own reference guide explicitly says it is useful, but not required, to run a Spring Boot app on Kubernetes.
- Spring Cloud Kubernetes adds a specific set of cluster-native integrations: discovery, ConfigMap and Secret property sources, reload and config watcher behavior, leader election, load balancer, and service-registry style capabilities.
- There is an important current limitation: the Spring Cloud Kubernetes reference docs say it does not support Spring Boot AOT transformations or native images at this point.
- Cloud-provider tutorials are mostly about cluster bootstrapping, image build and push, and deployment mechanics. They do not solve platform ownership, field provenance, or change-governance questions by themselves.
- The platform-engineering ecosystem breaks into layers rather than one unified category:
  - developer portal and scaffolding: Backstage
  - workload specification: Score
  - control plane and orchestration: Kratix, Crossplane, Humanitec Platform Orchestrator
  - opinionated app platform and delivery UX: Devtron

## Current Spring baseline

As of 2026-03-30:

- [Spring Boot docs](https://docs.spring.io/spring-boot/reference/packaging/container-images/cloud-native-buildpacks.html) list stable branches `4.0.5`, `3.5.13`, `3.4.13`, and `3.3.13`.
- [Spring Cloud Kubernetes docs](https://docs.spring.io/spring-cloud-kubernetes/reference/) list stable branches `5.0.1`, `3.3.1`, `3.2.3`, and `3.1.6`.

## What Spring itself recommends for Kubernetes

### 1. Start with buildpacks and standard Kubernetes objects

The official [Spring on Kubernetes guide](https://spring.io/guides/topicals/spring-on-kubernetes/) frames the basic path as:

- build a container image with Spring Boot's Cloud Native Buildpacks support
- deploy with standard `Deployment` and `Service` resources
- add readiness and liveness probes
- externalize configuration with `ConfigMap`
- use Kubernetes service discovery and scaling

The guide also notes that, according to the [2024 State of Spring Survey](https://spring.io/blog/2024/11/19/2024-state-of-spring-survey-results), 65% of respondents use Kubernetes in their Spring environment.

The current Spring Boot reference guide says Boot supports [Cloud Native Buildpacks directly from Maven and Gradle](https://docs.spring.io/spring-boot/reference/packaging/container-images/cloud-native-buildpacks.html), and that this gives you container images you can run anywhere.

### 2. Treat probes as a first-class runtime contract

The official [Spring Boot Actuator endpoints reference](https://docs.spring.io/spring-boot/reference/actuator/endpoints.html#actuator.endpoints.kubernetes-probes) documents Kubernetes probe support:

- Boot exposes `/actuator/health/liveness` and `/actuator/health/readiness`
- these health groups are automatically enabled
- if startup is slow, `startupProbe` may still be useful
- if management endpoints run on a separate port, Spring warns that probe success may not prove the main app is healthy, so surfacing probe paths on the main server port is often better

This is especially relevant to the existing `springboot-paas` example because probe endpoints are part of the contract between Spring intent and platform-owned runtime behavior.

### 3. Externalize config with `spring.config.import` and `configtree:`

The official [Spring Boot externalized configuration docs](https://docs.spring.io/spring-boot/reference/features/external-config.html#features.external-config.files.configtree) recommend using mounted config volumes when platforms such as Kubernetes provide them.

Important details from the docs:

- Kubernetes can mount both ConfigMaps and Secrets as volumes.
- Boot supports importing whole files and directory trees.
- For directory-style mounts, use `spring.config.import=optional:configtree:/etc/config/`.
- Folder and file names become property names.

This fits the `cub-gen` model well because it keeps Spring application config as DRY input while still letting the platform contribute mounted operational config.

### 4. Graceful shutdown is part of the default runtime story now

The current [Spring Boot graceful shutdown docs](https://docs.spring.io/spring-boot/reference/web/graceful-shutdown.html) say graceful shutdown is enabled by default with Jetty, Reactor Netty, and Tomcat, for both reactive and servlet apps.

The docs also point to `spring.lifecycle.timeout-per-shutdown-phase` as the main tuning knob. That matters for Kubernetes rollouts because it affects whether old pods drain cleanly under rolling updates.

## Spring Cloud Kubernetes: when it helps and when it is optional

The official [Spring Cloud Kubernetes reference guide](https://docs.spring.io/spring-cloud-kubernetes/reference/) is unusually clear about scope:

- it provides implementations of familiar Spring Cloud interfaces for Kubernetes
- it may be useful for cloud-native apps
- it is not a requirement to deploy Spring Boot on Kubernetes

The reference docs list these major feature areas:

- discovery client for Kubernetes
- Kubernetes-native service discovery
- ConfigMap and Secret property sources
- property source reload
- pod health indicator
- info contributor
- leader election
- load balancer for Kubernetes
- service registry implementation
- configuration watcher
- config server and discovery server variants

### Configuration reload is the sharp edge

The [Configuration Watcher docs](https://docs.spring.io/spring-cloud-kubernetes/reference/spring-cloud-kubernetes-configuration-watcher.html) are one of the most useful pages in the set.

They say:

- mounted ConfigMap and Secret content can change without restarting the container
- Spring Boot will not automatically refresh the application context when those files change
- the watcher can trigger refresh through `/refresh` or Spring Cloud Bus
- watched resources need labels such as `spring.cloud.kubernetes.config=true` or `spring.cloud.kubernetes.secret=true`
- mounted-volume refresh timing needs a cluster-specific delay because projected volume updates are only eventually consistent

That is a concrete signal that "config on disk changed" and "Spring runtime behavior changed" are not the same thing. Any platform contract around runtime config needs to decide whether the desired behavior is restart-based reconciliation or in-place refresh.

### Native-image caveat

The same [reference guide](https://docs.spring.io/spring-cloud-kubernetes/reference/) currently states that Spring Cloud Kubernetes does not support Spring Boot AOT transformations or native images. Partial support may come later.

Inference: if a future `cub-gen` Spring path wants to cover GraalVM-native or AOT-heavy Spring workloads, Spring Cloud Kubernetes should be treated as an optional add-on surface, not the default model.

## Managed-cloud guidance

### Google Cloud

The older [GKE Spring Boot codelab](https://codelabs.developers.google.com/codelabs/cloud-springboot-kubernetes#0) is still useful for the basic mechanical flow:

- package the app as a container
- create a GKE cluster
- deploy to Kubernetes
- scale the service
- roll out an upgrade

The more current direction is Google's [Build an application with buildpacks](https://docs.cloud.google.com/docs/buildpacks/build-application) and [builders overview](https://docs.cloud.google.com/docs/buildpacks/builders):

- Google supports local and remote buildpack builds
- the generic builder targets GKE, GKE Enterprise, Cloud Run, App Engine, and Cloud Run functions
- the `google-24` builder currently supports Java `17`, `21`, and `25`

Inference: Google is pushing a buildpack-first supply-chain story that lines up cleanly with Spring Boot's own buildpack stance.

### Azure

The official [Deploy Spring Boot Application to Azure Kubernetes Service](https://learn.microsoft.com/en-us/azure/developer/java/spring-framework/deploy-spring-boot-java-app-on-kubernetes) page contains a notable recommendation:

- for Spring Boot applications, Microsoft recommends Azure Container Apps
- AKS remains an option when you specifically want Kubernetes as the destination

This is useful framing. Even vendors with strong Kubernetes offerings now distinguish between:

- "run my Spring app on a higher-level managed runtime"
- "give me raw Kubernetes control because I need platform-level policy, multi-service composition, or infrastructure integration"

## Platform-engineering framework map

### Backstage

Official docs:

- [Backstage Software Templates overview](https://backstage.io/docs/features/software-templates/)
- [Adding your own Templates](https://backstage.io/docs/features/software-templates/adding-templates/)

Backstage's abstraction is a catalog plus scaffolding workflow:

- templates are stored in the software catalog as kind `Template`
- templates gather inputs, execute scaffolder steps, publish to Git providers, and register catalog entities

Good fit:

- onboarding and golden-path entry point for Spring services
- repo creation, template-driven generation, docs, ownership metadata

Not the whole answer:

- Backstage is a front door and workflow UI, not the deeper control plane by itself

This maps cleanly to the existing `backstage-idp` example in this repo.

### Score

Official docs:

- [Get started with Score](https://docs.score.dev/docs/get-started/)

Score's abstraction is an environment-agnostic workload spec:

- developers declare containers, services, and resource dependencies in `score.yaml`
- implementations such as `score-compose` and `score-k8s` turn that into runnable output

Good fit:

- Spring teams describe app source config and dependencies without owning raw manifests
- strong alignment with DRY intent feeding rendered WET output

Important boundary:

- Score is a spec plus implementations, not a full platform control plane on its own

This maps directly to the existing `scoredev-paas` example in this repo.

### Kratix

Official docs:

- [Introduction to Kratix](https://docs.kratix.io/)
- [Quick Start](https://docs.kratix.io/main/quick-start)
- [Installing and using a Promise](https://docs.kratix.io/main/guides/installing-a-promise)

Kratix's abstraction is the `Promise`:

- a Promise is a contract between the platform and its users
- Promises expose self-service APIs
- Promises embed workflows and business rules
- Promises reconcile fleets, not just one-off instances

The docs explicitly position Kratix as an open-source platform framework and note that APIs can be exposed through portals such as Backstage.

Good fit:

- platform teams want clear self-service APIs plus built-in governance
- platform wants to update a fleet of app-adjacent services by changing the Promise itself

This is conceptually close to `cub-gen`'s interest in governed contracts, field ownership, and platform-mediated change.

### Crossplane

Official docs:

- [What's Crossplane?](https://docs.crossplane.io/latest/whats-crossplane/)

Crossplane calls itself a control plane framework for platform engineering. Its core concepts are:

- composition
- managed resources
- operations
- package manager

The key value proposition from the docs is that platform engineers can build custom APIs and abstractions without writing controllers for each one.

Good fit:

- platform-owned APIs that compose Kubernetes resources and external cloud resources
- teams that want Kubernetes-native reconciliation and custom resource APIs

Inference: Crossplane is strongest when the platform wants Kubernetes itself to be the control-plane substrate. That is adjacent to `cub-gen`, but it sits more on the platform-owned resource orchestration side than on app-source provenance.

### Humanitec Platform Orchestrator

Official docs:

- [Platform Orchestrator overview](https://developer.humanitec.com/platform-orchestrator/docs/what-is-the-platform-orchestrator/overview/)

The Orchestrator's model is:

- developers provide a lean workload manifest
- platform teams register reusable modules and rules, usually around Terraform or OpenTofu
- the orchestrator generates environment-specific app and infrastructure configs
- deployments produce a live resource graph

Good fit:

- organizations with significant existing Terraform or OpenTofu estates
- teams that want a workload-spec-to-infra orchestration layer without replacing existing CI or IaC

It is especially relevant here because Humanitec strongly overlaps with Score-based workflows.

### Devtron

Secondary source:

- [How to Deploy Spring Boot Applications on Kubernetes Effectively](https://devtron.ai/blog/how-to-deploy-spring-boot-application-on-kubernetes/)

Devtron is better understood as an opinionated Kubernetes application platform than as a generic framework:

- app creation and Git wiring
- build and deploy pipelines
- generated manifests
- health, rollback, vulnerabilities, and deployment status

Good fit:

- teams that want a delivery platform and operational UX quickly

Less relevant to `cub-gen`'s core differentiation:

- it is not primarily about DRY-to-WET provenance, field ownership, or inverse edit mapping

## Useful comparison

| Tool | Primary layer | Main abstraction | Best fit for Spring teams | Relevance to `cub-gen` |
| --- | --- | --- | --- | --- |
| Backstage | Portal and scaffolding | Catalog entities and templates | Golden paths, repo generation, ownership, docs | Strong complement |
| Score | Workload spec | `score.yaml` | Environment-agnostic app source config | Strong complement |
| Kratix | Platform framework | Promise plus workflows | Self-service APIs with built-in governance | Strong conceptual peer |
| Crossplane | Control plane framework | XRD/XR plus composition | Kubernetes-native custom APIs across app and infra | Adjacent peer |
| Humanitec PO | Orchestration layer | Manifest plus module and rule orchestration | Keep existing Terraform/OpenTofu and CI/CD | Useful comparator |
| Devtron | App platform | Build and deploy UX | Fast operationalization of K8s delivery | More operational than provenance-oriented |

## Secondary practitioner signals

### Reddit migration thread

The Reddit thread [Advice on migrating Spring Boot apps to Kubernetes](https://www.reddit.com/r/kubernetes/comments/1ijhe5o/advice_on_migrating_spring_boot_apps_to_kubernetes/) is not canonical guidance, but it does surface a realistic migration concern:

- phased migration with stable FQDN and path contracts
- coexistence of Kubernetes-hosted and VM-hosted services
- ingress path matching as a simpler migration bridge than overloading Spring-specific gateway logic
- GitOps tools such as Argo CD or Flux entering the conversation early

That is a useful real-world reminder that migration strategy is often more about traffic shape and coexistence than about Java packaging.

### Devtron blog

The Devtron post is a secondary source, but it lines up with the official guidance on:

- containerization
- resource management
- autoscaling
- rolling updates
- graceful shutdown

Its value is less in the manual YAML examples and more in showing what an opinionated app platform chooses to automate.

## Implications for `cub-gen`

### 1. Keep Spring Boot as app source config, not the platform runtime contract

The Spring docs reinforce the direction already present in `examples/springboot-paas`:

- Spring configuration is the app-facing authoring surface
- Kubernetes resources are the rendered operational surface
- the platform should own operational boundaries such as datasource wiring, policy overlays, and reconciler transport

### 2. Model Spring Cloud Kubernetes as an optional integration profile

Inference from the official docs:

- a base Spring-on-Kubernetes profile should not require Spring Cloud Kubernetes
- a second optional profile could cover discovery, property sources, config reload, and config watcher semantics

That would let `cub-gen` distinguish:

- "plain Spring Boot on Kubernetes"
- "Spring Boot with Spring Cloud Kubernetes"

instead of forcing both stories into one example.

### 3. Strengthen provenance around runtime config boundaries

The strongest technical opportunities for `cub-gen` look like:

- tracing `spring.config.import` usage, especially `configtree:` mounts
- distinguishing restart-required config from refreshable config
- recognizing probe endpoints and graceful-shutdown settings as part of runtime ownership
- keeping app-owned namespaces distinct from platform-owned namespaces such as datasource, secret, ingress, or SLO fields

### 4. Use the platform-framework examples as layered complements, not substitutes

A clean mental model for future examples would be:

- Backstage for the entry UX
- Score or Spring config for workload intent
- `cub-gen` for provenance and governed source-to-runtime tracing
- Kratix, Crossplane, or another control-plane framework for platform-owned orchestration when needed

### 5. Candidate future research or demo paths

- Backstage template that scaffolds a Spring Boot service plus `cub-gen`-friendly metadata
- Spring Boot plus Spring Cloud Kubernetes example showing ConfigMap reload semantics explicitly
- Kratix Promise wrapped around a Spring workload contract
- Crossplane `App` API that composes Kubernetes and cloud dependencies around a Spring service

## Source list

Primary sources:

- Spring: [Spring on Kubernetes guide](https://spring.io/guides/topicals/spring-on-kubernetes/)
- Spring Boot: [Cloud Native Buildpacks](https://docs.spring.io/spring-boot/reference/packaging/container-images/cloud-native-buildpacks.html)
- Spring Boot: [Actuator Kubernetes probes](https://docs.spring.io/spring-boot/reference/actuator/endpoints.html#actuator.endpoints.kubernetes-probes)
- Spring Boot: [Externalized configuration and config trees](https://docs.spring.io/spring-boot/reference/features/external-config.html#features.external-config.files.configtree)
- Spring Boot: [Graceful shutdown](https://docs.spring.io/spring-boot/reference/web/graceful-shutdown.html)
- Spring Cloud Kubernetes: [Reference guide](https://docs.spring.io/spring-cloud-kubernetes/reference/)
- Spring Cloud Kubernetes: [Configuration Watcher](https://docs.spring.io/spring-cloud-kubernetes/reference/spring-cloud-kubernetes-configuration-watcher.html)
- Google Cloud: [Spring Boot on GKE codelab](https://codelabs.developers.google.com/codelabs/cloud-springboot-kubernetes#0)
- Google Cloud: [Build an application with buildpacks](https://docs.cloud.google.com/docs/buildpacks/build-application)
- Google Cloud: [Buildpacks builders](https://docs.cloud.google.com/docs/buildpacks/builders)
- Microsoft Azure: [Deploy Spring Boot Application to AKS](https://learn.microsoft.com/en-us/azure/developer/java/spring-framework/deploy-spring-boot-java-app-on-kubernetes)
- Backstage: [Software Templates](https://backstage.io/docs/features/software-templates/)
- Backstage: [Adding your own Templates](https://backstage.io/docs/features/software-templates/adding-templates/)
- Score: [Get started](https://docs.score.dev/docs/get-started/)
- Kratix: [Introduction](https://docs.kratix.io/)
- Kratix: [Quick Start](https://docs.kratix.io/main/quick-start)
- Kratix: [Installing and using a Promise](https://docs.kratix.io/main/guides/installing-a-promise)
- Crossplane: [What's Crossplane?](https://docs.crossplane.io/latest/whats-crossplane/)
- Humanitec: [Platform Orchestrator overview](https://developer.humanitec.com/platform-orchestrator/docs/what-is-the-platform-orchestrator/overview/)

Secondary and practitioner sources:

- Devtron: [How to Deploy Spring Boot Applications on Kubernetes Effectively](https://devtron.ai/blog/how-to-deploy-spring-boot-application-on-kubernetes/)
- Reddit: [Advice on migrating Spring Boot apps to Kubernetes](https://www.reddit.com/r/kubernetes/comments/1ijhe5o/advice_on_migrating_spring_boot_apps_to_kubernetes/)
