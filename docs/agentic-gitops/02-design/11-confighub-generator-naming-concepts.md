# ConfigHub Generator Naming Concepts

**Status:** Naming workshop draft
**Date:** 2026-03-04
**Purpose:** Choose externally clear and internally extensible naming for generator capabilities.

## 1. Naming Objectives

Names should:

1. be understandable to Kubernetes + GitOps + Helm users,
2. make sense for internal platform generators (not only Helm),
3. avoid confusion with controllers/reconcilers,
4. fit AI-assisted workflows and CH MR governance language,
5. work as both product language and API object names.

## 2. Naming Surfaces to Decide

1. Umbrella feature name (user-facing).
2. Core object names (internal/API-facing).
3. User command verbs (`import`, `evaluate`, `promote`, etc.).

## 3. Candidate Umbrella Names

### Option A: ConfigHub Generators

Positioning:

1. "Bring your generator; ConfigHub governs it."

Pros:

1. direct and obvious for technical users,
2. broad enough for Helm/Score/Spring/custom,
3. easy mapping to schema names already in use.

Risks:

1. sounds implementation-heavy, less product-polished.

### Option B: ConfigHub Render Pipeline

Positioning:

1. "From DRY intent to governed WET output."

Pros:

1. clear process framing,
2. aligns with dry->wet pipeline model.

Risks:

1. can imply CI-only rendering rather than governance system.

### Option C: ConfigHub Compose

Positioning:

1. "Compose platform intent into deployable contracts."

Pros:

1. product-friendly,
2. works across frameworks.

Risks:

1. may be vague for Helm-first users.

### Option D: ConfigHub Forge

Positioning:

1. "Forge deployable artifacts with provenance and policy."

Pros:

1. strong brand tone,
2. memorable.

Risks:

1. less obvious meaning without explanation.

### Option E: ConfigHub Source-to-Deploy

Positioning:

1. "Track and govern source-to-runtime transformation."

Pros:

1. explicit value chain.

Risks:

1. long and clunky for CLI/docs.

## 4. Recommended Shortlist

### 1) Primary recommendation: ConfigHub Generators

Why:

1. most technically honest,
2. lowest ambiguity for early adopters,
3. cleanly supports contract triple naming already established.

### 2) Productized alternative: ConfigHub Render Pipeline

Why:

1. good for roadmap and architecture communication,
2. naturally explains DRY->WET flow.

### 3) Brand-forward alternative: ConfigHub Compose

Why:

1. friendlier for external narrative,
2. still broad enough for custom platform generators.

## 5. Suggested Naming Stack (if Option A chosen)

User-facing:

1. Feature: `ConfigHub Generators`
2. Workflow: `Import -> Evaluate -> Promote`

Object names:

1. `GeneratorContract`
2. `ProvenanceRecord`
3. `InverseTransformPlan`
4. `GeneratorUnit`

CLI naming:

1. Prototype binary: `cub-gen detect|import|evaluate|origin|inverse-plan|promote`
2. Official integrated surface: `cub gitops detect|import|evaluate|origin|inverse-plan|promote`

UI terms:

1. "Generator"
2. "Origin Map"
3. "Inverse Plan"
4. "Promotion"

## 6. Anti-Patterns to Avoid

1. Names implying reconciler replacement (`orchestrator`, `controller`).
2. Names tied only to one tool (`helm-*`) at umbrella level.
3. Overly abstract names with no source/deploy semantics.
4. Different names for same concept across CLI/UI/API.

## 7. Fast Decision Framework

Decision criteria (1-5 score):

1. Immediate clarity for Helm/GitOps users.
2. Extensibility to internal generators.
3. Alignment with existing schemas/docs.
4. Marketing usability.
5. Low risk of architectural confusion.

Use weighted total:

1. Clarity (30%)
2. Extensibility (25%)
3. Schema alignment (20%)
4. Marketing usability (15%)
5. Confusion risk (10%, inverted)

## 8. Recommendation

Adopt:

1. External + internal primary name: `ConfigHub Generators`
2. Supporting phrase: `Generator Render Pipeline`
3. Keep contract triple names unchanged.
4. Treat `cub-gen` as prototype packaging and `cub gitops` as official integrated surface.

This gives immediate clarity now and preserves room for stronger branded packaging later.
