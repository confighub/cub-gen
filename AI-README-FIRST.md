# AI Read Me First

This is the repo-specific cold-start guide for Claude, Codex, and other AI coding agents.

If your AI host supports repo-local skills, load [skills/cub-gen/SKILL.md](skills/cub-gen/SKILL.md) after this file.

If you are starting work in this repo, read files in this order:

1. `AI-README-FIRST.md`
2. `HANDOVER.md`
3. `CLAUDE.md`
4. `docs/cli-reference.md`
5. `docs/contracts/change-cli-v1.md`

Use these for different AI scenarios:
- `docs/ai/cub-gen-skill.md` — capability-assistant profile (answering "can cub-gen do X?")
- `docs/ai/cub-gen-tasks.md` — task skill for *using* cub-gen against a real source repo

## What cub-gen is

`cub-gen` is the **repo-side traceability and governed-change CLI** for GitOps teams.

It maps DRY source files (Helm `values.yaml`, Score `score.yaml`, Spring Boot `application.yaml`, etc.) to WET rendered manifests and records:

- **Field origin** — for every deployed field, the source file/path that produced it
- **Inverse-edit hints** — given a deployed field, where to edit DRY source to change it
- **Confidence scores** — how certain the mapping is
- **Provenance bundles** — a verifiable record of what the rendering produced

## Tool boundaries

### `cub-gen`

Use it for:
- generator detection (`detect`, `generators`)
- rendering and provenance (`gitops discover`, `gitops import`, `publish`)
- bundle verification (`verify`, `attest`, `verify-attestation`)
- developer change workflows (`change preview`, `change run`, `change explain`)
- bridge to ConfigHub (`bridge ingest`, `bridge decision`, `bridge promote`)

`cub-gen` is local-first. It works without ConfigHub for source-side work.

### `cub`

`cub` is the ConfigHub CLI for intended-state workflows.

Use `cub gitops` for cluster-side import (Argo/Flux applications imported from a target).

`cub gitops import` and `cub-gen gitops import` are not the same:
- `cub gitops import` reads from a cluster target via a render-target
- `cub-gen gitops import` reads source repos and emits provenance from DRY/WET pairs

### `cub-scout`

`cub-scout` is the read-only Kubernetes/GitOps observer.

Together: `cub-gen` knows source → rendered, `cub-scout` knows cluster → owner. ConfigHub stitches them across teams and policy.

## Current shipped capabilities

As of 2026-04-09:

- 9 generator profiles supported (ApplicationSet, Helm, Score.dev, Spring Boot, Backstage, No-Config-Platform, Ops Workflow, C3 Agent, Swamp)
- Local field-origin tracing with confidence scores
- Inverse-edit pointers for safe DRY edits
- Provenance bundles with publish/verify/attest pipeline
- Change CLI: preview, run, explain (local + connected modes)
- Bridge workflow: ingest → decision → promote
- First-class ApplicationSet generator support for standalone deterministic repos, with layered Helm/ApplicationSet provenance still covered separately
- Live reconciler proofs against Flux and ArgoCD on kind

## Verification rule

Do not invent command surfaces or generator capabilities.

Verify from local help before claiming capability:

```bash
./cub-gen --help
./cub-gen generators --json
./cub-gen change --help
./cub-gen publish --help
./cub-gen verify --help
./cub-gen bridge --help
```

When the workflow crosses into ConfigHub:

```bash
cub gitops --help
cub gitops import --help
```

## Non-negotiables

1. Deterministic behavior — same input must give same output
2. Parse, don't guess — derive classifications from explicit artifacts
3. Local-first — no implicit deploys, no hidden control-plane side effects
4. Parity-first — preserve `cub gitops` command shape where declared in `PARITY.md`
5. Graceful degradation — unsupported paths must return explicit errors
6. Test every change — `go test ./...` and parity tests must pass

## Quick reality checks

Use these before answering capability or workflow questions:

```bash
./cub-gen --help
./cub-gen generators --markdown --details
./cub-gen detect --repo .
./cub-gen change --help
```

For cluster-side or policy questions:

```bash
cub gitops --help
```
