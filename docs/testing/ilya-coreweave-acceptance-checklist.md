# Ilya/CoreWeave Acceptance Checklist

Source: Ilya feedback session, 6 March 2026
Status: active acceptance criteria

## Context

CoreWeave platform teams validated the core pain:

> GitOps repos are the transport and audit log, but not a usable platform API.
> "No write API" is directionally true for platform needs.

They want API-first platform engineering while keeping Flux/Argo and Git workflows.

## Core acceptance criteria

These criteria must be demonstrable in examples and connected flows.

### 1. Git is storage, ConfigHub is the operational API

| Criterion | Test |
|-----------|------|
| Read from Git | `cub-gen gitops discover/import` works on any repo |
| Mutate via API | ConfigHub mutation API changes config without manual YAML edit |
| Write back to Git | API mutation produces deterministic commit/PR |
| Governance at API | ALLOW/ESCALATE/BLOCK decisions happen at ConfigHub, not in Git |

**Example proof**: Import a messy Git repo, mutate via ConfigHub API, show generated PR + provenance.

### 2. No manual YAML editing for platform operations

| Criterion | Test |
|-----------|------|
| Semantic path mutation | Patch by field path, not file path |
| Label-based targeting | Mutate "all apps with label X" without knowing file locations |
| Safe write-back | Generated commits are deterministic and reviewable |

**Example proof**: Change a resource limit for all production deployments using a single API call.

### 3. Label/query model is first-class

| Criterion | Test |
|-----------|------|
| Query by labels | List apps/services/targets by labels across repos |
| Reorganize without surgery | Change platform structure via labels, not repo moves |
| Cross-repo visibility | See all deployments of an app across clusters |

**Example proof**: Query "all eshop deployments in EU region" without knowing repo structure.

### 4. Platform as APIs, not templates

| Criterion | Test |
|-----------|------|
| Services have APIs | Every platform service is callable, not just templated |
| Deterministic generators | DRY → WET transformation is reproducible |
| Provenance/inverse maps | For any WET field, show which DRY source to edit |

**Example proof**: Show field-origin tracing from deployed configmap back to values.yaml.

### 5. Governance is explicit

| Criterion | Test |
|-----------|------|
| ALLOW/ESCALATE/BLOCK | Every change gets an explicit decision |
| Attestation links | Changes are signed and linked to evidence |
| Promotion gates | Cross-environment promotion requires separate approval |

**Example proof**: Show a blocked change (privileged escalation) with explanation.

## Demo flow (CoreWeave pattern)

This is the demo sequence that addresses their specific pain:

```
1. Start with messy Git repo (typical CW state)
2. Discover/import with cub-gen (generator detection, provenance)
3. Mutate via ConfigHub API call (no manual YAML edit)
4. Show generated PR + provenance + decision + rollout path
5. Show promotion to upstream DRY with separate review gate
```

## What this is NOT

- This is NOT cub-track (mutation ledger for agent sessions)
- This is NOT full agentic GitOps (agent workflow orchestration)
- This IS: ConfigHub + cub-gen with Flux/Argo for WET→LIVE

cub-track can be added later for deeper agent workflow and mutation evidence.

## Separation of concerns

| Tool | Question it answers |
|------|---------------------|
| cub-gen | "What should exist?" (generator intelligence) |
| cub-track | "How did this change happen?" (mutation/evidence intelligence) |
| ConfigHub | "Is this allowed?" (decision/governance authority) |

## When to use cub-track without cub-gen

cub-track is the fast governance-on-ramp; cub-gen is the structured generator-on-ramp.

Use cub-track without cub-gen when:

- Repos are WET-first (raw manifests, Kustomize, hand-edited YAML)
- No deterministic generator exists yet for that stack
- You need AI/human change auditability now (compliance, incident review)
- Changes include non-generator artifacts (runbooks, CI, scripts)
- You are piloting agentic governance first, generator modeling later

## Checklist for example validation

Every featured example must pass these checks:

- [ ] Can be discovered/imported without manual configuration
- [ ] Produces field-origin map with inverse-edit guidance
- [ ] Supports label-based queries (when connected to ConfigHub)
- [ ] Shows at least one ALLOW and one ESCALATE/BLOCK path
- [ ] Demonstrates governance without requiring manual YAML edits
- [ ] Works with existing Flux/Argo reconciliation (no reconciler changes)

## Related documentation

- [Universal Example Contract](../workflows/example-checklist.md)
- [App/Deployment Concepts](app-deployment-concepts.md)
- [Prompt as DRY](../workflows/prompt-as-dry.md)
