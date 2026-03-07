# Introducing Agentic GitOps

> Configuration is data. Generators are how you get there.

**Part of:** [AI and GitOps v7 Document Set](/docs/agentic-gitops/00-index/00-gitops7-index.md)
**Status:** Planning doc (v7)
**Date:** 2026-02-28

---

You use `helm template` every day. That is a generator.

```
chart/ + values.yaml  →  helm template  →  rendered manifests
```

A generator is a deterministic function: it takes some intent (your Helm values, your Kustomize overlays, your Score workload) and produces explicit configuration. Same inputs, same outputs, every time. No cluster access required, no runtime discovery, no inference. Kustomize is the same pattern. So is `cdk8s synth`, or any script that reads a config file and writes Kubernetes YAML.

The formal version looks like this:

```
WET = generate(intent, context)
```

**DRY** is the shorthand you author in — Helm values, Score workloads, Backstage forms. It is compact and parameterized. **WET** is what comes out of the generator — fully rendered manifests where every field has a literal value. No template expressions, no unresolved variables, no conditionals. You can `cat` it and read every field.

If you track what went in and what came out, you have **provenance**. If you can trace any output field back to the specific input that produced it, you have a **field-origin map**. These two ideas — provenance and field-origin maps — are the foundation of everything in this document set.

---

## The Problem, in Plain English

Classical GitOps is great at convergence. Git holds desired state, a reconciler watches for changes, the cluster converges. Simple, powerful, battle-tested. But the model only answers two questions well: "What is deployed?" and "Has the cluster converged?" It does not answer "Why was this change proposed?", "What evidence existed when it was approved?", or "What happened after it was applied?"

For a small team making a handful of changes a week, the answers to those harder questions live in PR descriptions and Slack threads. You remember the context because you were there. But for a platform team managing forty services across six environments — or for an AI agent making forty-seven changes overnight — those answers need to be structured, searchable, and provable. They cannot live in human memory.

The gaps are specific, and platform engineers hit them constantly. You can see what Helm produced, but not which of the four version axes (CLI version, chart repo, chart name, chart version) caused the output to differ between Tuesday and Wednesday. You can diff files in Git, but you cannot answer "What is actually deployed to production across our forty repos?" as a query — it is a research project. A break-glass hotfix gets applied directly to the cluster, and the reconciler reverts it on its next cycle, because there is no governed way to propose that fix back to desired state. Users edit rendered manifests directly because no system tells them which input file controls which output field — so the DRY source drifts from reality and becomes fiction. Staging means a folder per environment, and those folders drift silently over time because promotion is a copy operation that drags environment-specific values along for the ride.

These are not theoretical problems. They are what platform engineers deal with every day, and they get worse with every additional environment, team, and automation. The model described in this document set addresses them not by replacing GitOps, but by adding governance around it.

---

## The Claim

Once a generator renders output, the result contains literal values for every field. No unresolved variables, no template syntax, no conditionals. At that point, configuration is no longer code — it is data. And data can be queried, diffed, validated against constraints, and governed through an API. You do not need to re-render a template to understand what is deployed. You do not need to grep across Git repos to compare environments. You query a system that already holds the rendered, structured, labeled result.

ConfigHub exists because configuration artifacts are structured data that deserve database-grade treatment: typed queries, label-based filtering, revision history, cross-environment comparison, retention policies, and policy evaluation at write time. Git remains a primary collaboration and ingress surface; ConfigHub is the control-plane store for dry units and wet units, with WET authoritative for deployment.

---

## Hard Qualification Rule

`Agentic GitOps` has a strict requirement:

1. It **MUST** include an active GitOps reconciliation loop where WET desired
   state is continuously reconciled to LIVE runtime by Flux/Argo (or an
   equivalent reconciler).

The complete model is three loops:

1. Outer loop: `DRY -> WET` (generator + governance).
2. Inner loop: `WET <-> LIVE` (GitOps reconciliation).
3. Evidence loop: `LIVE -> provenance/decision`.

If a system has the outer loop but not the inner reconciliation loop, label it
as **governed config automation**, not Agentic GitOps.

---

## How It Works

Here is the full flow, from intent to evidence:

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

The forward path works like this: a developer (or an AI agent) writes intent in DRY format — Helm values, a Score workload, a Backstage form submission. A generator renders that intent into explicit WET manifests. ConfigHub stores dry and wet Units as queryable records with full provenance — what generator ran, what version, what inputs, what came out. WET Units are the authoritative deployment contract. From there, the manifests are published (as an OCI artifact or a Git commit), and Flux or Argo does what it does best: reconcile the cluster toward desired state. cub-scout observes the actual runtime state. If something drifts — a field changed, a resource disappeared, a replica count shifted — structured evidence is produced. A human or a policy engine decides what to do about it.

But the flow is not one-way. When a user wants to change a deployed value, they do not need to hunt through Git repos, figure out which of twelve files controls the replica count for the staging variant, and manually edit the right line. ConfigHub resolves which input file controls that output field — the field-origin map — and routes the change to the correct DRY source. The user edits in one place. The system commits the change to the right layer, re-renders, and the forward path takes over again.

This means the same system that lets you query "What is deployed?" also lets you act on the answer. You see a value, you trace where it came from, you change it at the source. No context-switching between a dashboard and a Git repo. No guessing which overlay to edit.

---

## Three Rules

Three invariants that are never waived, regardless of mode, toolchain, or organizational context.

**1. Nothing implicit ever deploys.**

Every deployed artifact is explicit, diffable, and traceable to a generator run with known inputs. If you cannot `cat` the manifest and read every field as a literal value, it should not be deployed. No silent side effects, no implicit defaults that bypass review, no reconciler applying changes without a corresponding record in the desired-state store.

**2. Nothing observed silently overwrites intent.**

When the cluster diverges from intended state — whether from a break-glass fix, an autoscaler adjustment, or an AI agent acting on live state — the change is observed, evidence is produced, and an explicit proposal goes back to intended state. Neither direction is automatic. The forward path (intent to cluster) goes through generators and reconciliation. The reverse path (cluster to intent) goes through evidence collection, proposal generation, and human or policy review. Both are governed.

**3. Configuration is data, not code.**

Templates, SDKs, procedural scripts — all fine as authoring tools. Authors should use whatever produces correct configuration most efficiently. But whatever enters the generator as code exits as explicit manifests: literal values, queryable by API, diffable without rendering. The generator boundary is where code becomes data. Nothing imperative crosses the publish line.

The first two invariants govern mutation flow — they define what is allowed to change configuration and under what conditions. The third governs representation — it defines what configuration fundamentally is.

For Agentic GitOps, this is enforced as a hard rule: propose -> evaluate ->
approve -> execute -> verify -> attest. Verification and attestation are
mandatory governance primitives, not optional reporting features.

---

## What This Changes

Adopting this model does not require ripping out your existing setup. Your Helm charts stay. Your Argo or Flux configuration stays. You do not migrate anything on day one. What changes is what happens around the tools you already use:

You start capturing what generators produce as queryable Units with provenance — not just as files in a Git repo, but as structured records with labels, version history, and cross-environment comparison built in.

You can answer "What is deployed where?" as a query, not as a research project that involves cloning six repos and grepping for image tags.

You can trace any field in any deployment back to the specific input that produced it. When someone asks "Why is the replica count 5 in staging?", the answer is a lookup, not an archaeology expedition.

You can edit configuration through ConfigHub without knowing which repo, file, or line to change. The field-origin map resolves it for you. You change the value; the system routes the commit to the right place.

AI agents get a governed mutation path instead of unsupervised access. The pattern is: propose, evaluate against policy, approve (or escalate, or block), execute within scoped authority, verify the outcome, and attest with a signed record. Every step produces structured evidence.

Every change — human or automated — has a structured audit trail: who proposed it, what policy evaluated it, what decision was made, and what actually happened. Not a chat transcript. Not a PR description that someone forgot to write. Queryable, machine-readable records.

---

## What This Is Not

To prevent misunderstanding, explicit boundaries:

- **Not a controller or reconciliation engine.** This model does not replace Flux, Argo, or any Kubernetes controller. Controllers reconcile desired state to live state. This model governs how desired state is produced, how mutations are authorized, and how evidence is recorded.

- **Not a replacement for Git.** Git remains a primary collaboration and ingress surface for configuration source material. This model governs how Git-authored DRY sources and ConfigHub dry units produce a WET deployment contract, and what happens when reality diverges (evidence and governed reverse flow).

- **Not a replacement for Flux or Argo.** Existing reconcilers continue to do what they do well: pull desired state from a store and converge clusters toward it. This model adds the layers that reconcilers lack: generation governance, mutation authorization, evidence production, and field-origin tracking.

- **Not a portal-driven IDP.** It is not a self-service portal with forms and wizards. While it can be consumed through portals (Backstage, custom UIs), the model itself is about configuration data and governance, not user interface.

- **Not a runtime reconciler or orchestrator.** It does not watch clusters and apply changes. It does not manage pod scheduling, rolling updates, or health checks. Runtime reconciliation remains the domain of existing controllers.

It is a disciplined way to turn intent into explicit configuration, govern mutations, and produce evidence when reality diverges from intent.

---

## Where to Go Next

This document establishes the "why" and the foundational principles. The remaining documents in the set cover the "how":

| If you want to understand... | Read |
|-----|------|
| The generator model and maturity levels | [02 — Generators PRD](/docs/agentic-gitops/02-design/10-generators-prd.md) |
| Field-origin maps and the editing experience | [03 — Field-Origin Maps and Editing](/docs/agentic-gitops/02-design/20-field-origin-maps-and-editing.md) |
| The ConfigHub data model | [04 — App Model and Contracts](/docs/agentic-gitops/02-design/30-app-model-and-contracts.md) |
| cub-track, the mutation ledger | [05 — cub-track](/docs/agentic-gitops/05-rollout/10-cub-track.md) |
| The governed execution architecture | [06 — Governed Execution](/docs/agentic-gitops/02-design/40-governed-execution.md) |
| What it looks like to use | [07 — User Experience](/docs/agentic-gitops/05-rollout/30-user-experience.md) |
| How to adopt and the business case | [08 — Adoption and Reference](/docs/agentic-gitops/05-rollout/40-adoption-and-reference.md) |
