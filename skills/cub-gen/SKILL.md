---
name: cub-gen
description: Use when working in the cub-gen repo or when explaining cub-gen's source-side role versus ConfigHub cub and cub-scout. Covers generator discovery, Component -> Variant -> Base/Deployment proof, DRY/WET field-origin tracing, inverse-edit routing, platform import/fanout, enrichment/normalization previews, publish/verify/attest bundles, and ConfigHub bridge workflows.
tier: plugin-candidate
plugin: cub-gen
canonical_command: cub-gen
future_command: cub gen
---

# cub-gen

Use this skill when the task is about:

- what `cub-gen` does today
- how `cub-gen` differs from `cub`, `cub gitops`, and `cub-scout`
- generator detection and supported generator families
- Component, Variant, Base Variant, Deployment Variant, Target, Connection, Change, and Proof mapping
- field-origin tracing, confidence scores, and inverse-edit guidance
- platform estate import, variant fanout, enrichment, and normalization previews
- publish, verify, attest, and verify-attestation bundles
- ConfigHub bridge workflows for ingest, decisions, PR/MR links, and promotion
- the future `cub gen` plugin surface

## Read First

1. `AI-README-FIRST.md`
2. `HANDOVER.md`
3. `README.md`
4. `docs/cli-reference.md`
5. `docs/cub-gen-plugin.md`
6. `docs/ai/cub-gen-tasks.md`

When the user wants you to use `cub-gen` against a real source repo, prefer
`docs/ai/cub-gen-tasks.md`; it is task-oriented and has concrete command flows.

## Product Value In One Breath

`cub-gen` is the repo-side traceability and governed-change tool. It starts
from source/config repos, maps source config to rendered deployable config,
records field-level provenance with confidence scores, and emits inverse-edit
hints so app, platform, environment, and security owners know where a change
belongs.

The user model should stay small:

```text
Component
  -> Variant
      -> Base Variant
      -> Deployment Variant
          -> Target
          -> Connections
          -> Change
          -> Proof
```

## Command Status

Today the canonical command is `cub-gen`. The target product surface is
`cub gen`, but that requires the external `cub` plugin loader/distribution
mechanism. Do not claim `cub gen --help` works unless you have verified it in
the user's environment.

Use this mapping when discussing the migration:

| Today | Future plugin surface |
|---|---|
| `cub-gen detect` | `cub gen detect` |
| `cub-gen generators` | `cub gen generators` |
| `cub-gen gitops discover` | `cub gen discover` |
| `cub-gen gitops import` | `cub gen import` or `cub gen render` |
| `cub-gen platform import` | `cub gen platform import` |
| `cub-gen platform fanout` | `cub gen fanout` |
| `cub-gen enrich preview` | `cub gen enrich preview` |
| `cub-gen normalize preview` | `cub gen normalize preview` |
| `cub-gen publish` | `cub gen bundle` |
| `cub-gen verify` | `cub gen bundle verify` |
| `cub-gen attest` | `cub gen bundle attest` |
| `cub-gen verify-attestation` | `cub gen bundle verify-attestation` |
| `cub-gen change preview` | `cub gen preview` |
| `cub-gen change run` | `cub gen run` |
| `cub-gen change explain` | `cub gen explain` |
| `cub-gen bridge ingest` | `cub gen ingest` |
| `cub-gen bridge link` | `cub gen link` |
| `cub-gen bridge promote ...` | `cub gen promote ...` |

## Tool Boundaries

Use `cub-gen` for source-side work:

- generator detection and registry inspection
- source-to-rendered provenance and inverse-edit routing
- read-only platform estate import and variant fanout
- enrichment sidecars and normalization preview proposals
- governed evidence bundles and attestations
- bridge records that connect GitHub PRs and ConfigHub MRs

Use `cub` / `cub gitops` for ConfigHub intended-state workflows:

- spaces, units, targets, workers, and ConfigHub control-plane operations
- cluster-integrated GitOps import where ConfigHub owns the operation

Use `cub-scout` for runtime observation:

- live cluster discovery
- ownership detection from Flux, Argo, Helm, and Kubernetes state
- runtime health and troubleshooting

Do not blur these:

- `cub-gen gitops import` reads source-side repos and emits local provenance.
- `cub gitops import` is the ConfigHub product surface.
- `cub-gen` is local-first: no implicit deploys and no hidden control-plane
  side effects.

## Shipped Generator Families

Verify the exact current list with:

```bash
./cub-gen generators --markdown --details
```

Current families include:

- Helm
- Score.dev
- Spring Boot
- Backstage IDP
- No-config platform
- OpenChoreo-style platform
- Argo ApplicationSet
- Argo app-of-apps
- Ops workflow
- C3 Agent
- Swamp

## Current High-Signal Workflows

```bash
./cub-gen gitops discover --space platform --json <repo>
./cub-gen gitops import --space platform --json <repo> [<render-target>]
./cub-gen change explain --space platform --owner app-team <repo>
```

```bash
./cub-gen platform import --json <platform-manifest>
./cub-gen platform fanout --json <variant-manifest>
./cub-gen change explain --change-id <id> --bundle fanout.json --variant <component>/<variant>
```

```bash
./cub-gen enrich preview --space platform --patch <repo>
./cub-gen normalize preview --space platform --patch <repo>
```

```bash
./cub-gen publish --space platform <repo> > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json
./cub-gen verify-attestation --in attestation.json --bundle bundle.json
```

```bash
./cub-gen bridge link \
  --bundle bundle.json \
  --git-provider github \
  --git-repo confighub/cub-gen \
  --git-pr 300 \
  --mr-id <confighub-mr-id>
```

## Confidence Score Routing

- `>= 0.90`: normal app/team edit flow
- `0.75 - 0.89`: run `change preview` and `change explain` before merge
- `< 0.75`: escalate for platform review

Always surface confidence when reporting field origins or inverse-edit guidance.

## Verification Rule

Do not invent command surfaces or generator capabilities. Verify from local help
before claiming capability:

```bash
./cub-gen --help
./cub-gen generators --markdown --details
./cub-gen gitops import --help
./cub-gen platform --help
./cub-gen change --help
./cub-gen publish --help
./cub-gen bridge --help
```

When the workflow crosses into ConfigHub or runtime state:

```bash
cub gitops --help
cub-scout --help
```

## Safety Rule

- `cub-gen` is local-first by default.
- Connected mode talks to a ConfigHub backend but does not deploy.
- Bridge workflows ingest provenance bundles into ConfigHub for governed
  decisions.
- `cub-gen` never directly modifies a cluster.
