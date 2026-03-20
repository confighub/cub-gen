# Existing GitOps Adoption Plan

Status: proposed execution plan
Date: 2026-03-20
Owner: cub-gen maintainers

## Goal

Make `cub-gen` feel immediately useful to teams that already run GitHub plus
Argo CD or Flux, with real app or platform repos they already care about.

Within the first 10 minutes, a new user should be able to:

1. point `cub-gen` at an existing repo or a realistic demo repo,
2. see rendered-manifest provenance and ownership,
3. connect that repo-side view to ConfigHub and `cub-scout`,
4. understand why import matters before any migration or controller replacement.

## Product wedge

The near-term wedge is:

`GitHub + Helm/Score/Spring source repo + Argo/Flux + cub-gen + cub-scout + ConfigHub`

The story is additive:

1. Git remains the source authoring path,
2. Argo CD and Flux remain the reconcilers,
3. `cub-scout` explains cluster and controller reality,
4. `cub-gen` explains source-to-rendered provenance,
5. ConfigHub becomes the shared place to compare, validate, govern, and inspect evidence.

## Why this plan exists

The repo is stronger than it was:

1. the example catalog is broad,
2. the main examples have local and connected entrypoints,
3. live reconcile proof exists for both Flux and Argo,
4. the docs now describe the product more honestly.

But the current first-run experience still feels too internal and too
generator-centric.

The main gaps are:

1. the top entry docs still lead with `cub-gen` mechanics more than existing-app relevance,
2. there is no single obvious first-run path for "I already have an app in GitOps",
3. `cub-scout` continuity is missing from the primary entry surfaces,
4. demo breadth is better than demo focus,
5. mutation and lifecycle depth are more visible than the immediate "why import?" answer.

## North-star user stories

### 1. Repo-first platform team

"We already run Helm plus Flux or Argo. Show us which values file controls the
running thing, who owns it, and why ConfigHub is worth adding."

### 2. Repo-first app team

"We already ship a Spring Boot or Score-based app. Show us where to edit safely
without making us think in raw rendered YAML first."

### 3. Cluster-first GitOps operator

"We already have an Argo CD or Flux application running. Start from what exists
in cluster, inspect it with `cub-scout` and ConfigHub, then show how `cub-gen`
connects that back to source."

`cub-gen` owns the repo-first path directly and must link cleanly into the
cluster-first companion path.

## Non-negotiable requirements

1. Start from an existing app, repo, or GitOps setup users already recognize.
2. Show value immediately after import: inspect, validate, compare, explain.
3. Keep Flux, Argo, GitHub, Git, and OCI as first-class existing parts of the story.
4. Make `cub-scout` and ConfigHub continuity explicit in the first-run docs.
5. Keep AI plus CLI as the primary operating surface.
6. Use the GUI for evidence, tables, diffs, and review, not as the only entry path.
7. Do not make mutate/apply/status the critical path for first value.
8. Keep example claims conservative until the flagship proofs and gates really exist.

## Canonical first-run journeys

### Journey A: existing Helm plus Flux or Argo team

This should be the primary platform-first path.

Use:

1. `helm-paas` for repo-side DRY to WET provenance,
2. `live-reconcile` for WET to LIVE proof,
3. ConfigHub connected mode for validation, evidence, and governed decision state,
4. `cub-scout` for cluster-side inspection and comparison.

The first useful questions should be:

1. which values file controls this field,
2. what rendered output changed,
3. what is actually running,
4. what evidence or validation result should we look at next.

### Journey B: existing Spring Boot app team

This should be the primary app-first path.

Use:

1. `springboot-paas` as the most recognizable existing-app story,
2. one concrete config question a Spring team already asks,
3. one local import path,
4. one connected ConfigHub path,
5. one explicit handoff to cluster/runtime inspection.

`scoredev-paas` remains important, but Spring should be the more universal
entry point for new users.

### Journey C: cluster-first companion path

This path may span repos, but it must be visible from `cub-gen`.

Use:

1. ConfigHub GitOps import or existing cluster-side discovery,
2. `cub-scout` for cluster and controller inspection,
3. `cub-gen` to trace a chosen field or workload back to DRY source.

The point is continuity:

1. cluster-side discovery does not replace source-side provenance,
2. source-side provenance does not replace runtime inspection,
3. ConfigHub links both views into one operational story.

## Workstreams

## 1. Entry-surface reset

Primary files:

1. `README.md`
2. `docs/getting-started.md`
3. `examples/README.md`
4. `examples/demo/README.md`

Changes:

1. lead with existing app and existing GitOps wording,
2. highlight two canonical starts: platform-first and app-first,
3. add one visible cluster-first companion path using `cub-scout` plus ConfigHub,
4. make "why import?" concrete: inspect rendered manifests, run validation, compare, inspect evidence,
5. move advanced flow depth behind the first-run path instead of ahead of it.

Exit criteria:

1. a new user can choose a starting path in under 30 seconds,
2. the first commands feel tied to existing repos and apps,
3. `cub-scout` is visible in the main entry surfaces,
4. the docs answer "why import?" before asking for deeper adoption.

## 2. Flagship example reset

Primary examples:

1. `helm-paas`
2. `live-reconcile`
3. `springboot-paas`

Changes:

1. tighten `helm-paas` and `live-reconcile` into one coherent platform-first journey,
2. make `springboot-paas` the clearest app-first journey,
3. ensure each flagship example starts with a real user question,
4. ensure each flagship example includes one local first step, one connected first step, and one runtime inspection step,
5. show what ConfigHub adds without claiming reconciler replacement.

Exit criteria:

1. one platform-first flagship path is obviously stronger than the rest,
2. one app-first flagship path is obviously stronger than the rest,
3. each flagship example feels like a real adoption story, not a feature checklist.

## 3. Demo surface simplification

Primary files and scripts:

1. `examples/demo/README.md`
2. one new platform-first starter script
3. one new app-first starter script
4. existing advanced lifecycle and promotion scripts

Changes:

1. promote one primary script per canonical journey,
2. demote broad "run everything" paths for first-time users,
3. keep lifecycle, PR/MR, and promotion flows as follow-on proof,
4. add expected outcomes for the starter scripts in plain English.

Exit criteria:

1. the demo page no longer feels like a menu of equal-weight options,
2. first-time users start with one of two scripts,
3. advanced scripts remain available but clearly secondary.

## 4. `cub-scout` and ConfigHub continuity

Primary deliverable:

1. a documented repo-first to cluster-first handoff path

Changes:

1. add explicit references from `cub-gen` entry docs to the matching `cub-scout` and ConfigHub paths,
2. show the sequence "source repo -> rendered import -> cluster inspection -> evidence view",
3. make it clear which questions belong to `cub-gen`, which belong to `cub-scout`, and which belong to ConfigHub.

Exit criteria:

1. a new user can see how the tools fit together,
2. `cub-gen` no longer reads like a standalone island,
3. the additive story is stronger than the replacement story.

## 5. AI-first packaging

This matters, but it should package the canonical journeys rather than replace them.

Changes:

1. add `AI_START_HERE.md` once the primary journeys are stable,
2. add prompts and machine-readable contracts for the starter flows,
3. add repo-level agent guidance only after the first-run paths are clear,
4. make read-only versus mutating boundaries explicit in the starter flows.

Exit criteria:

1. an AI assistant can safely execute the same primary journeys a human would choose,
2. AI docs reinforce the main adoption wedge instead of introducing a parallel story.

## 6. Acceptance gate

Primary issues:

1. `#183` connected acceptance gate
2. `#182` example and demo entrypoint redesign
3. `#200` AI best practices from public examples

Changes:

1. enforce the new flagship entry criteria in docs and example checks,
2. verify the primary docs still point to the canonical journeys,
3. verify the starter scripts keep working,
4. keep connected checks strict about what is truly proven.

Exit criteria:

1. the release gate fails when the primary adoption story regresses,
2. the repo stops drifting back toward broad but unfocused example coverage.

## Recommended execution order

1. rewrite the entry surfaces around the existing GitOps adoption wedge,
2. strengthen `helm-paas` plus `live-reconcile` into the primary platform-first journey,
3. strengthen `springboot-paas` into the primary app-first journey,
4. document and link the `cub-gen` plus `cub-scout` plus ConfigHub handoff,
5. package the canonical journeys for AI-first execution,
6. lock the above into the acceptance gate.

## Issue alignment

This plan should drive the current issue stack rather than replace it:

1. `#182` is the entry-surface and demo simplification work,
2. `#183` is the enforcement and acceptance work,
3. `#177` plus `#187` are the platform-first flagship work,
4. `#179` is the app-first flagship work,
5. `#178` becomes the next app-platform story after the primary Spring path lands,
6. `#200` packages the proven journeys for AI-first usage.

## What to de-prioritize until this lands

1. adding more example families,
2. leading with abstract generator inventory,
3. making apply flows the main first-run story,
4. spending more effort on advanced lifecycle demos than on the first 10 minutes,
5. AI-doc packaging that is not tied to a concrete canonical journey.

## Success signal

`cub-gen` is in a useful, usable, relevant state when a new user can say:

"I already have a repo and a GitOps runtime. In a few minutes, I can see what
source controls the running thing, compare it with cluster reality, inspect
evidence in ConfigHub, and understand why these tools are worth adding without
changing how we already deploy."
