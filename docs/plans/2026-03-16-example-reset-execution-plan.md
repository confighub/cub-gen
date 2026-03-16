# Example Reset Execution Plan

Status: active execution plan
Date: 2026-03-16
Owner: cub-gen maintainers

## Goal

Turn the `cub-gen` example catalog into the main user-facing product surface:
real-cluster, real-app, two-audience examples that clearly show why someone
would add `cub-gen` + ConfigHub to an existing platform/app workflow.

This plan replaces the weaker standard of "good explanatory docs" with the
stronger standard of "compelling runnable examples that deploy something real and
show concrete ConfigHub value."

The dominant audience assumption for this plan is:

- existing platform-tool users
- adding ConfigHub + `cub-gen` + AI-assisted change workflows together
- without replacing their current reconciler or platform framework

That means the product surface must make three things obvious:

1. most code and prompts are DRY,
2. generators are how teams get from DRY to WET,
3. human and AI-assisted changes should run through the same governed ConfigHub
   MR path.

It also needs to give platform users credible day-2 stories:

- after import, what do I do next?
- how do governed change, promotion, live-origin proposals, or AI-assisted
  changes help me now?

## Non-negotiable requirements

1. Every featured example must support two audiences explicitly:
   - existing ConfigHub user adding the platform tool/scenario
   - existing platform-tool user adding `cub-gen` + ConfigHub
2. Every featured example must use a real cluster.
3. Every featured example must deploy a real app, runtime, or workflow engine.
4. Every connected example must use ConfigHub in a way that adds visible value.
5. Every example must use current app/deployment concepts from ConfigHub.
6. Every example must provide a live inspection target.
7. Every example must include one governed `ALLOW` path and one governed
   `ESCALATE` or `BLOCK` path.
8. Examples are the primary discovery surface; supporting docs are secondary.
9. Argo-first users must see Argo as a first-class path, not an afterthought.
10. Layered platform frameworks must show multi-layer tracing where the stack
    genuinely has more than one generation hop.
11. Story 2 (import), story 10 (ConfigHub promotion value), and story 11
    (live-origin proposal flow) must be prominent user-facing workflows.
12. The meaning of invariants and the enforcement model must be explained in
    workflow terms, not left as abstract internal language.
13. AI prompt/context as DRY input must be visible as a first-class product
    lane, not left inside planning docs only.

## Universal example contract

Every example README and entrypoint must include these sections and behaviors:

1. `Who this is for`
   - existing ConfigHub user adding this platform tool/scenario
   - existing platform-tool user adding ConfigHub
2. `What runs`
   - real app/runtime/workflow engine
   - real cluster objects
   - real live thing to inspect
3. `Why ConfigHub + cub-gen helps here`
   - one concrete pain
   - one concrete answer
   - one concrete governed change win
4. `Run it from ConfigHub`
   - connected-first path
5. `Run it from the platform tool`
   - tool-first path, then add `cub-gen`, then connect ConfigHub
6. `Inspect the result`
   - URL, pods, app/deployment objects, ConfigHub evidence, decision, or query
7. `Try one governed change`
   - at least one `ALLOW`
   - at least one `ESCALATE` or `BLOCK`

For layered examples such as Helm/Argo/Kubara-like platforms, the contract also
requires:

8. `Show the generation chain`
   - labels, overlays, umbrella charts, ApplicationSets, or other intermediate
     layers if they materially affect what gets deployed
9. `Explain the ownership boundary`
   - especially where platform-owned security defaults can be weakened by
     downstream edits unless governed

## Execution waves

### Wave 0: contract and foundations

Purpose: define what a "good example" is and make it enforceable.

Issues:
- [#174](https://github.com/confighub/cub-gen/issues/174) universal example contract
- [#175](https://github.com/confighub/cub-gen/issues/175) shared real-cluster connected harness
- [#176](https://github.com/confighub/cub-gen/issues/176) app/deployment concept alignment + Ilya checklist capture
- [#183](https://github.com/confighub/cub-gen/issues/183) connected acceptance suite and release gate
- [#185](https://github.com/confighub/cub-gen/issues/185) custom-generator onboarding path for framework-specific generators
- [#190](https://github.com/confighub/cub-gen/issues/190) clarify DRY->WET, AI governance, invariants, and enforcement in primary entrypoints
- [#192](https://github.com/confighub/cub-gen/issues/192) AI prompt-as-DRY with verification, attestation, and mutation-ledger proof

Exit criteria:
- example contract is written and testable
- shared harness exists for connected runs
- current app/deployment concepts are encoded
- Ilya complaint checklist is written and mapped to tests
- release gate can fail on example regressions
- primary entrypoints explain DRY->WET generators and ConfigHub MR governance in plain English
- primary entrypoints expose credible day-2 stories for stuck platform users
- AI prompt-as-DRY is visible as a first-class product lane

### Wave 1: flagship app-platform examples

Purpose: fix the examples most users will recognize first.

Issues:
- [#187](https://github.com/confighub/cub-gen/issues/187) Kubara-like layered provenance across overlays, ApplicationSet, and label-driven generation
- [#177](https://github.com/confighub/cub-gen/issues/177) Helm / Argo / Kubara-like flagship example
- [#178](https://github.com/confighub/cub-gen/issues/178) Score.dev flagship example
- [#179](https://github.com/confighub/cub-gen/issues/179) Spring Boot flagship example

Exit criteria:
- Helm, Score, and Spring all satisfy the universal example contract
- each one deploys a real app to a real cluster
- each one has both audience paths
- each one shows a real ConfigHub connected value path
- Helm/Kubara-like example demonstrates layered provenance, not only flat field mapping

### Wave 2: governed entry/promotion flows

Purpose: make the most important ConfigHub MR workflows runnable and visible.

Issues:
- [#188](https://github.com/confighub/cub-gen/issues/188) real Git PR <-> ConfigHub MR pairing flows A and B
- [#189](https://github.com/confighub/cub-gen/issues/189) FR8 promotion and live->wet->dry first-class flow

Exit criteria:
- Flow A (`Git PR -> ConfigHub MR`) is demoed on a real cluster
- Flow B (`ConfigHub MR -> Git PR`) is demoed on a real cluster
- story 10 and FR8 are easy to point at from examples
- story 11 remains a prominent, runnable live-origin flow

### Wave 3: workflow platform examples

Purpose: make workflow/automation stories equally strong and credible.

Issues:
- [#180](https://github.com/confighub/cub-gen/issues/180) Ops Workflow + Swamp stories

Exit criteria:
- workflow examples are real workflow-platform stories, not just structural scans
- runtime and governance concerns are separated and both made concrete
- each example ends with a real inspectable outcome

### Wave 4: supporting example catalog

Purpose: bring the rest of the catalog up to the same user-facing quality bar.

Issues:
- [#181](https://github.com/confighub/cub-gen/issues/181) supporting examples (`backstage-idp`, `just-apps-no-platform-config`, `confighub-actions`, `c3agent`, `ai-ops-paas`, `live-reconcile`)
- [#182](https://github.com/confighub/cub-gen/issues/182) examples/demo entrypoint redesign

Exit criteria:
- no example remains a second-class citizen
- examples index helps users find a relevant live demo quickly
- demo index is organized around outcomes, not just script names

## Issue map

### Umbrella tracking issue
- [#173](https://github.com/confighub/cub-gen/issues/173) turn examples into real-cluster, two-audience product surfaces

### Foundation issues
- [#174](https://github.com/confighub/cub-gen/issues/174) define and enforce a universal example contract
- [#175](https://github.com/confighub/cub-gen/issues/175) build shared real-cluster connected harness for examples
- [#176](https://github.com/confighub/cub-gen/issues/176) align examples to current app/deployment concepts and capture Ilya acceptance checklist
- [#185](https://github.com/confighub/cub-gen/issues/185) add a custom-generator onboarding path for Kubara-like frameworks
- [#183](https://github.com/confighub/cub-gen/issues/183) add connected acceptance suite and release gate for the example reset
- [#190](https://github.com/confighub/cub-gen/issues/190) clarify DRY->WET, AI governance, invariants, and enforcement in primary entrypoints
- [#192](https://github.com/confighub/cub-gen/issues/192) make AI prompt-as-DRY with verification, attestation, and mutation-ledger proof first-class

### Primary example issues
- [#187](https://github.com/confighub/cub-gen/issues/187) support Kubara-like layered provenance across overlays, ApplicationSet, and label-driven generation
- [#177](https://github.com/confighub/cub-gen/issues/177) upgrade `helm-paas` into the flagship Helm/Argo/Kubara-like example
- [#178](https://github.com/confighub/cub-gen/issues/178) upgrade `scoredev-paas` for ConfigHub-first and Score-first adoption
- [#179](https://github.com/confighub/cub-gen/issues/179) upgrade `springboot-paas` into a real app + governed config example
- [#180](https://github.com/confighub/cub-gen/issues/180) upgrade `ops-workflow` and Swamp examples into runnable workflow platform stories

### Flow and promotion issues
- [#188](https://github.com/confighub/cub-gen/issues/188) demo real Git PR <-> ConfigHub MR pairing flows A and B
- [#189](https://github.com/confighub/cub-gen/issues/189) make FR8 promotion and live->wet->dry a first-class example flow

### Catalog and UX issues
- [#181](https://github.com/confighub/cub-gen/issues/181) upgrade supporting examples to the same product standard
- [#182](https://github.com/confighub/cub-gen/issues/182) redesign example and demo entrypoints around audience and live outcomes

## Priority stack

1. Universal example contract and acceptance gates
2. Shared real-cluster connected harness
3. Product framing: DRY->WET, AI governance, invariants, enforcement
4. App/deployment concept alignment + Ilya checklist capture
5. AI prompt-as-DRY user-facing lane
6. Custom-generator onboarding path for framework-specific platforms
7. Helm/Kubara layered provenance capability
8. Helm flagship example
9. Score flagship example
10. Spring flagship example
11. Real Flow A / Flow B pairing demos
12. FR8 promotion and live->wet->dry example
13. Workflow examples
14. Supporting catalog cleanup
15. Entry-point/index redesign
16. Connected release gate hardening

## Detailed expectations by example family

### Helm / Kubara-like

We should treat `helm-paas` as the closest current answer for users of:
- Helm + Argo/Flux
- umbrella charts
- overlays
- ApplicationSets and cluster-label targeting
- secure-by-default platform baselines
- Kubara-like platform frameworks

The example must feel like a real platform engineering story, not a template demo.
It should answer concrete Kubara-style questions such as:

- why does this cluster have this addon enabled?
- which label or overlay caused this deployment?
- who is allowed to weaken a platform-owned security setting?
- what should be edited upstream instead of downstream?

It also needs to answer the deeper chain question:

- how do I trace from `config.yaml` or cluster labels through overlays/ApplicationSet
  logic to the live deployment on a specific cluster?

### Score.dev

This is the cleanest two-audience split in the catalog:
- ConfigHub-first: govern `score.yaml` before it becomes opaque cluster/app state
- Score-first: keep Score as the contract, add `cub-gen` for traceability, then add ConfigHub for decisions/evidence

### Spring Boot

The example must speak in Spring terms first:
- profiles
- properties
- actuator/health
- app-team vs platform/DBA ownership

The Kubernetes layer should support the story, not lead it.

### Workflow platforms

The workflow examples must no longer read like static YAML governance demos.
They need to show a real workflow engine or real runnable workflow artifact,
plus the value of ConfigHub-connected governance.

They also need to carry the stronger AI story:

- prompt + context can be DRY input,
- the LLM or agent layer can behave like a non-deterministic generator,
- verification, attestation, and governance are what make that safe,
- the mutation ledger is the compliance and forensics proof.

## Flow expectations from the PRD

The product surface should make these flows easy to find and run:

- story 2: import as a concrete first step for brownfield users
- story 10 / FR8: promotion to reusable base DRY and upstream/default config
- story 11: live-origin proposal flow, because it is one of the clearest unique
  ConfigHub strengths

The goal is not just to mention them in design docs. The goal is to make them
obvious from examples and entrypoints.

It should also tell a believable day-2 story:

- Day 1: import and explain
- Day 2: governed change, promotion, or live-origin proposal
- Day 3: optional AI-assisted lane with mutation-ledger evidence

## Dependencies and open inputs

1. Ilya complaint details must be captured explicitly and added to the checklist
   before we can declare the reset complete.
2. Jesper's current app/deployment concepts must be treated as the canonical
   model for example language and live proof.
3. Some supporting examples may need to split into sub-issues if their runtime
   story becomes too large for one PR.
4. Kubara-like/platform-framework users need a clear custom-generator path from
   the user-facing surface, not from deep design docs only.
5. The strongest product story is often \"existing platform-tool user adds
   ConfigHub + `cub-gen` + AI in one step\"; examples and entrypoints should be
   optimized for that reader.
6. The AI-specific value from the DRY/WET analysis needs to be surfaced in
   examples and entrypoints, not left in planning text.

## Definition of done

We can call this reset complete when all of the following are true:

1. A user can open `examples/README.md`, recognize their stack, and find a
   fitting example immediately.
2. Primary examples run in connected mode on real clusters.
3. Primary examples deploy real apps/runtimes/workflow platforms.
4. Every primary example supports both audiences explicitly.
5. Every primary example shows one `ALLOW` and one `ESCALATE`/`BLOCK` path.
6. Every primary example exposes a real live inspection target.
7. CI/release gates prove the claims.
8. We no longer need hidden supporting docs to explain the examples' value.
