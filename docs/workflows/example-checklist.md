# Example Checklist

Every example in `examples/` must follow the canonical DRY+WET+LIVE pattern and answer the five questions a new user has. This checklist verifies parity across all current and future examples.

## The canonical pattern

All cub-gen examples follow the same structure regardless of generator kind, app type, or workload shape:

```
examples/<name>/
  <dry-source>              DRY intent (what the team authors)
  <dry-source>-prod.yaml    Production overlay (env-specific)
  platform/                 Platform contracts and policies
  gitops/                   Transport fixtures (Flux + ArgoCD)
  docs/                     User stories and narrative
  README.md                 Answers the five questions below
```

The generator changes. The app type changes. The pattern stays the same.

## Five questions every README must answer

### 1. What is this?

One paragraph explaining the real-world scenario in plain English. Not "this is a fixture" — tell me what a team would actually use this for.

### 2. Who does what?

Explicit ownership map:

- **App team** authors DRY intent (the config they control)
- **Platform team** owns contracts, policies, and guardrails
- **GitOps reconciler** (Flux/ArgoCD) syncs WET to LIVE — unchanged

### 3. What does cub-gen add?

Show the DRY-to-WET mapping with runnable commands:

```bash
./cub-gen gitops discover --space <space> ./examples/<name>
./cub-gen gitops import --space <space> --json ./examples/<name> ./examples/<name>
```

Explain what the output shows: field-origin confidence, inverse-edit pointers, rendered lineage.

### 4. How do I run it?

Copy-paste commands from repo root. Must actually work:

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space <space> ./examples/<name>
./cub-gen gitops import --space <space> --json ./examples/<name> ./examples/<name>
```

### 5. Show me a real-world example using ConfigHub

Walk through the governed pipeline from DRY edit to production deploy:

1. Team edits DRY source
2. `cub-gen` detects changes, emits provenance
3. `publish` creates a change bundle with digest
4. `verify` + `attest` produce evidence chain
5. ConfigHub ingests bundle, decision engine evaluates
6. Bridge worker applies via Flux/ArgoCD after ALLOW decision

This is the bridge from "I have a local tool" to "I have a governed platform."

## Directory structure checklist

| Item | Required | Notes |
|------|----------|-------|
| DRY source file(s) | Yes | The human-editable intent file |
| Production overlay | Yes | Shows env-specific override pattern |
| `platform/` directory | Yes | At least one contract or policy file |
| `gitops/flux/` | Recommended | HelmRelease or Kustomization fixture |
| `gitops/argo/` | Recommended | ArgoCD Application fixture |
| `docs/user-stories.md` | Yes | 3-4 narrative turns showing the workflow |
| `README.md` | Yes | Answers all five questions |
| `.cub-gen/` discover cache | Auto-generated | Created by `gitops discover` |

## Content quality checklist

| Check | Pass criteria |
|-------|---------------|
| README answers Q1 (What is this?) | Plain-English scenario, not "this is a fixture" |
| README answers Q2 (Who does what?) | Explicit ownership map with app/platform/reconciler roles |
| README answers Q3 (What does cub-gen add?) | DRY→WET mapping with actual output snippets |
| README answers Q4 (How do I run it?) | Commands work from repo root after `go build` |
| README answers Q5 (ConfigHub real-world?) | Bridge pipeline walkthrough for this specific example |
| DRY source is realistic | Real field names, real structure, not placeholder stubs |
| Platform contracts exist | At least one policy or constraint file |
| User stories show workflow | 3-4 narrative turns: edit → check → deploy → promote |
| Commands actually work | `discover` + `import` succeed without errors |
| Canonical pattern visible | Same DRY+WET+LIVE structure as all other examples |

## Generator-specific notes

### App generators (helm, score, springboot)

These generate Kubernetes workload manifests. Platform contracts enforce resource limits, required probes, network policies. GitOps transport is Flux HelmRelease or Kustomization + ArgoCD Application.

### Operations generators (ops-workflow, confighub-actions)

These generate governed execution plans. The DRY source describes *what operations to run*, not what manifests to deploy. Platform contracts enforce approval gates, scheduling windows, and action policies. The operations model treats workflow steps as configuration — diffable, governed, attested.

### Integration generators (ably-config, backstage-idp)

These generate configuration for external services or developer platforms. The DRY source describes *what the app needs from the service*. Platform contracts enforce allowed channels, API policies, catalog standards.

### Automation generators (swamp, c3agent)

These generate configuration for AI-native automation platforms. The DRY source describes *agent fleet config or workflow definitions*. Platform contracts enforce model approval, budget ceilings, credential hygiene.

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
