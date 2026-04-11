---
name: cub-gen
description: Use when working in the cub-gen repo or when answering what cub-gen can do versus ConfigHub/cub/cub-scout. Covers cub-gen's source-side provenance role, generator detection, DRY/WET field origin tracing, change CLI (preview/run/explain), publish/verify/attest pipeline, and how to verify current command truth before claiming capability.
---

# cub-gen

Start here when the task is about:

- what `cub-gen` does today
- how `cub-gen` differs from `cub` and `cub-scout`
- generator detection and support (Helm, Score, Spring Boot, etc.)
- field-origin tracing and inverse-edit guidance
- the publish/verify/attest provenance bundle pipeline
- change preview/run/explain workflows
- bridge workflows (ingest, decision, promote)

## Read first

1. `AI-README-FIRST.md`
2. `HANDOVER.md`
3. `docs/cli-reference.md`
4. `docs/contracts/change-cli-v1.md`
5. `docs/workflows/confidence-scores.md`

If the user is asking you to *use* cub-gen against a real source repo (not work on the repo), load `docs/ai/cub-gen-tasks.md` instead — that file is task-oriented with concrete command flows for source-side investigation.

## Product value in one breath

`cub-gen` is the source-side provenance and governed-change companion. It maps DRY source files to WET rendered manifests, records field-level origins with confidence scores, and emits inverse-edit hints so app/platform teams know exactly where to edit safely.

## Tool boundaries

### Use `cub-gen` for

- generator detection (`detect`, `generators`)
- DRY/WET classification and field-origin tracing
- provenance bundles (`publish`, `verify`, `attest`, `verify-attestation`)
- developer change workflows (`change preview|run|explain`)
- bridge workflows to ConfigHub (`bridge ingest|decision|promote`)

### Use `cub` for

- ConfigHub intended-state workflows
- spaces, units, targets, workers
- `cub gitops discover`
- `cub gitops import` (cluster + render-target based)

### Use `cub-scout` for

- cluster-side read-only observation
- ownership detection (Flux, Argo, Helm, etc.)
- live cluster troubleshooting

### Do not blur these

- `cub-gen gitops import` reads source repos and emits provenance from DRY/WET pairs
- `cub gitops import` reads from a cluster target via a render-target
- both have the same command shape but different inputs and different outputs
- `cub-gen` is local-first: no implicit deploys, no control-plane side effects

## High-signal shipped capabilities

- 8 generator profiles: Helm, Score.dev, Spring Boot, Backstage, No-Config-Platform, Ops Workflow, C3 Agent, Swamp
- Field-origin map with confidence scores per field
- Inverse-edit pointers (which DRY file/path/owner to edit)
- Provenance bundle pipeline: publish → verify → attest → verify-attestation
- Change CLI: preview, run (local|connected), explain
- Bridge workflow: ingest, decision query, promote init
- Live reconciler proofs against Flux and ArgoCD on kind clusters

## Variants and overlays today

Current status: partial support.

- One `gitops import` or `publish` invocation works on one repo path pair.
- Supported generators can pick up overlay files that already live in that repo, such as Helm `values-prod.yaml`, Spring `application-dev.yaml`, or generator-specific overlay files.
- Provenance records can distinguish base vs overlay source paths when the generator emits separate overlay transforms.
- `change explain` can point operators toward overlay-specific edits, but it does not expose a full CLI surface for "render N explicit variants and emit N bundles" yet.
- There is no `--values`, `--overlay`, or `--variant` fan-out flag today.

## Confidence score routing

- `>= 0.90` — proceed with normal app/team edit flow
- `0.75 - 0.89` — run `change preview` and `change explain` before merge
- `< 0.75` — escalate for platform review

See `docs/workflows/confidence-scores.md` for the full guide.

## Verification rule

Do not invent command surfaces or generator capabilities.

Verify from local help before claiming capability:

```bash
./cub-gen --help
./cub-gen generators --markdown --details
./cub-gen detect --help
./cub-gen change --help
./cub-gen publish --help
./cub-gen verify --help
./cub-gen bridge --help
```

When the workflow crosses into ConfigHub or cluster-side observation:

```bash
cub gitops --help
cub-scout --help
```

## Safety rule

- `cub-gen` is local-first by default
- Connected mode (`change run --mode connected`) talks to a ConfigHub backend but does not deploy
- Bridge workflows ingest provenance bundles into ConfigHub for governed decisions
- `cub-gen` never directly modifies a cluster
