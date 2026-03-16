# Universal Example Contract

Every example in `examples/` must satisfy this contract. This is the enforceable
standard that separates product-quality examples from internal fixtures.

Status: canonical contract (replaces prior 5-question checklist)
Date: 2026-03-16

## Contract purpose

Turn the `cub-gen` example catalog into the main user-facing product surface:
real-cluster, real-app, two-audience examples that clearly show why someone
would add `cub-gen` + ConfigHub to an existing platform/app workflow.

The dominant audience assumption is:

- existing platform-tool users
- adding ConfigHub + `cub-gen` + AI-assisted change workflows together
- without replacing their current reconciler or platform framework

## The canonical pattern

All cub-gen examples follow the same structure regardless of generator kind,
app type, or workload shape:

```
examples/<name>/
  <dry-source>              DRY intent (what the team authors)
  <dry-source>-prod.yaml    Production overlay (env-specific)
  platform/                 Platform contracts and policies
  gitops/                   Transport fixtures (Flux + ArgoCD)
  docs/                     User stories and narrative
  README.md                 Satisfies universal contract below
```

The generator changes. The app type changes. The pattern stays the same.

## Nine sections every README must include

### 1. Who this is for

Two explicit audience paths:

- **existing ConfigHub user** adding this platform tool/scenario
- **existing platform-tool user** adding ConfigHub + `cub-gen`

Each path should have a distinct "start here" instruction. Neither audience
should feel like an afterthought.

### 2. What runs

Concrete answers to:

- real app/runtime/workflow engine (not a stub)
- real cluster objects (not mock YAML)
- real live thing to inspect (URL, pod, dashboard, decision)

### 3. Why ConfigHub + cub-gen helps here

Three concrete answers:

- **one concrete pain** this example addresses
- **one concrete answer** `cub-gen` provides
- **one concrete governed change win** ConfigHub enables

### 4. Run it from ConfigHub (connected-first path)

Connected mode instructions for users who already have ConfigHub:

```bash
cub auth login
./cub-gen publish --space <space> ./examples/<name> ./examples/<name> > /tmp/bundle.json
./cub-gen bridge ingest --in /tmp/bundle.json ...
```

### 5. Run it from the platform tool (tool-first path)

Local mode instructions for users starting from their existing platform tool:

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space <space> ./examples/<name>
./cub-gen gitops import --space <space> --json ./examples/<name> ./examples/<name>
```

Then: how to add ConfigHub connection when ready.

### 6. Inspect the result

What to look at after running:

- URL, pods, or app/deployment objects
- ConfigHub evidence, decision, or query
- Proof artifacts (bundle, attestation, verification)

### 7. Try one governed change

At minimum:

- **one `ALLOW` path**: a change that should pass governance
- **one `ESCALATE` or `BLOCK` path**: a change that should require review or be denied

Show both outcomes with actual commands and expected results.

### 8. Show the generation chain (layered examples only)

For Helm/Argo/Kubara-like platforms with multi-layer generation:

- labels, overlays, umbrella charts, ApplicationSets, or other intermediate layers
- how to trace from `config.yaml` or cluster labels through overlays to live deployment

### 9. Explain the ownership boundary (layered examples only)

For layered platforms:

- where platform-owned security defaults can be weakened by downstream edits
- what should be edited upstream instead of downstream
- which invariants are enforced at which layer

## Day-2 story requirement

Every example must answer what happens after import:

- **Day 1**: import and explain (covered by sections 1-6)
- **Day 2**: governed change, promotion, or live-origin proposal (section 7)
- **Day 3**: optional AI-assisted lane with mutation-ledger evidence (if applicable)

The day-2 story should be prominent, not buried in appendices.

## AI prompt-as-DRY visibility

For workflow and AI examples, the README should explain:

- prompt + context can be DRY input
- the LLM or agent layer can behave like a non-deterministic generator
- verification, attestation, and governance are what make that safe
- the mutation ledger is the compliance and forensics proof

This is a first-class product lane, not planning-only content.

## Directory structure checklist

| Item | Required | Notes |
|------|----------|-------|
| DRY source file(s) | Yes | The human-editable intent file |
| Production overlay | Yes | Shows env-specific override pattern |
| `platform/` directory | Yes | At least one contract or policy file |
| `gitops/flux/` | Recommended | HelmRelease or Kustomization fixture |
| `gitops/argo/` | Recommended | ArgoCD Application fixture |
| `docs/user-stories.md` | Yes | Day-1/day-2/day-3 narrative turns |
| `README.md` | Yes | Satisfies all 9 contract sections |
| `.cub-gen/` discover cache | Auto-generated | Created by `gitops discover` |

## Contract compliance checklist

| Section | Pass criteria |
|---------|---------------|
| 1. Who this is for | Two explicit audience paths (ConfigHub-first + tool-first) |
| 2. What runs | Real app/runtime, real cluster objects, real inspection target |
| 3. Why ConfigHub + cub-gen helps | One pain, one answer, one governed change win |
| 4. Run from ConfigHub | Connected mode instructions work |
| 5. Run from platform tool | Local mode instructions work from repo root |
| 6. Inspect the result | URL/pods/evidence artifacts documented |
| 7. Governed change | At least one ALLOW + one ESCALATE/BLOCK shown |
| 8. Generation chain | (Layered only) Multi-hop tracing documented |
| 9. Ownership boundary | (Layered only) Platform vs downstream edit guidance |

## Content quality checklist

| Check | Pass criteria |
|-------|---------------|
| Two-audience explicit | Neither audience feels like an afterthought |
| Real app, not stub | Real field names, real structure, deployable |
| Platform contracts exist | At least one policy or constraint file |
| Day-2 story prominent | Governed change, promotion, or live-origin visible |
| Commands actually work | `discover` + `import` + bridge flow succeed |
| Canonical pattern visible | Same DRY+WET+LIVE structure as all other examples |
| AI lane visible (if applicable) | Prompt-as-DRY, verification, mutation ledger explained |

## Generator-specific contract requirements

### App generators (helm, score, springboot)

These generate Kubernetes workload manifests.

Contract requirements:
- Platform contracts enforce resource limits, required probes, network policies
- GitOps transport is Flux HelmRelease or Kustomization + ArgoCD Application
- README must show: resource limit change (ALLOW) + privileged escalation (BLOCK)

For Helm/Kubara-like layered stacks, sections 8 and 9 are mandatory:
- Show trace from cluster labels through overlays to deployed resources
- Explain where platform-owned security can be weakened by downstream edits

### Operations generators (ops-workflow, confighub-actions)

These generate governed execution plans.

Contract requirements:
- DRY source describes *what operations to run*, not what manifests to deploy
- Platform contracts enforce approval gates, scheduling windows, action policies
- README must show: schedule change (ALLOW) + unapproved action (ESCALATE)

### Integration generators (no-config-platform, backstage-idp)

These generate configuration for external services or developer platforms.

Contract requirements:
- DRY source describes *what the app needs from the service*
- Platform contracts enforce allowed channels, API policies, catalog standards
- README must show: metadata update (ALLOW) + catalog standard violation (BLOCK)

### Automation generators (swamp, c3agent)

These generate configuration for AI-native automation platforms.

Contract requirements:
- DRY source describes *agent fleet config or workflow definitions*
- Platform contracts enforce model approval, budget ceilings, credential hygiene
- README must show: budget adjustment (ALLOW) + unapproved model (BLOCK)
- AI prompt-as-DRY section is mandatory for these generators

## AI lane contract requirements

For swamp, c3agent, and ai-ops-paas examples, the README must include:

1. **Prompt/context as DRY**: Show that prompts and context are treated as DRY input
2. **LLM as non-deterministic generator**: Explain how the AI layer fits the generator model
3. **Verification boundary**: Show how verification, attestation, and governance make this safe
4. **Mutation ledger proof**: Explain where compliance/forensics evidence lives

This is a first-class product lane. Users adding AI workflows should see this
immediately, not discover it in design docs.

## Verifying an example

Run the verification commands from repo root:

```bash
# Build
go build -o ./cub-gen ./cmd/cub-gen

# Discover (should detect the generator with confidence > 0.7)
./cub-gen gitops discover --space test --json ./examples/<name>

# Import (should produce provenance with field-origin map)
./cub-gen gitops import --space test --json ./examples/<name> ./examples/<name>

# Bridge flow (should produce bundle + verification + attestation)
./cub-gen publish --space test ./examples/<name> ./examples/<name> > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen verify-attestation --in /tmp/attestation.json --bundle /tmp/bundle.json
```

All commands must exit 0. The bridge flow proves the example produces ConfigHub-ready artifacts.

## Non-negotiable requirements (from execution plan)

These requirements are enforced by acceptance tests and release gates:

1. Every featured example must support two audiences explicitly
2. Every featured example must use a real cluster (for connected runs)
3. Every featured example must deploy a real app, runtime, or workflow engine
4. Every connected example must use ConfigHub in a way that adds visible value
5. Every example must use current app/deployment concepts from ConfigHub
6. Every example must provide a live inspection target
7. Every example must include one governed `ALLOW` path and one governed `ESCALATE` or `BLOCK` path
8. Examples are the primary discovery surface; supporting docs are secondary
9. Layered platform frameworks must show multi-layer tracing
10. AI prompt/context as DRY input must be visible as a first-class product lane
