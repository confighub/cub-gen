# `cub gen` Plugin Readiness

Status: readiness contract. The shipped command in this repo is still
`cub-gen`. The product target is `cub gen`, but that depends on the external
`cub` plugin loader and distribution mechanism.

## Why `cub gen`

`cub-gen` is the source-side companion to ConfigHub GitOps:

| Surface | Vantage point | Owns |
|---|---|---|
| `cub gen` | source repo and generator inputs | provenance, inverse edits, bundles |
| `cub gitops` | ConfigHub intent | spaces, units, targets, governed import |
| `cub-scout` | live cluster | runtime ownership, health, troubleshooting |

The goal is not to rename for its own sake. The goal is to make the product
model obvious:

```text
source -> generated deployable config -> governed ConfigHub change -> runtime proof
```

## Compatibility Contract

When `cub gen` ships:

1. Existing `cub-gen ...` scripts continue to work for at least one release.
2. `cub-gen` remains useful for offline-first and no-backend workflows unless a
   later release explicitly removes it.
3. `cub gen` must preserve the local-first safety contract: no implicit deploys
   and no hidden control-plane writes.
4. Help text must name the source-side role directly so users do not confuse it
   with `cub gitops`.

## Command Mapping

| Current command | Target plugin command | Notes |
|---|---|---|
| `cub-gen detect` | `cub gen detect` | source repo detection |
| `cub-gen generators` | `cub gen generators` | generator registry |
| `cub-gen gitops discover` | `cub gen discover` | drop nested `gitops`; source-side is implied |
| `cub-gen gitops import` | `cub gen import` or `cub gen render` | final verb needs product choice |
| `cub-gen platform import` | `cub gen platform import` | multi-repo estate graph |
| `cub-gen platform fanout` | `cub gen fanout` | Component to Deployable Variant bundles |
| `cub-gen enrich preview` | `cub gen enrich preview` | sidecar provenance proposal |
| `cub-gen normalize preview` | `cub gen normalize preview` | review-only config rewrite proposal |
| `cub-gen publish` | `cub gen bundle` | evidence bundle creation |
| `cub-gen verify` | `cub gen bundle verify` | bundle verification |
| `cub-gen attest` | `cub gen bundle attest` | attestation creation |
| `cub-gen verify-attestation` | `cub gen bundle verify-attestation` | attestation verification |
| `cub-gen change preview` | `cub gen preview` | local governed-change preview |
| `cub-gen change run` | `cub gen run` | local or connected decision flow |
| `cub-gen change explain` | `cub gen explain` | field-to-source explanation |
| `cub-gen bridge ingest` | `cub gen ingest` | ConfigHub bundle ingest |
| `cub-gen bridge link` | `cub gen link` | GitHub PR to ConfigHub MR correlation |
| `cub-gen bridge promote ...` | `cub gen promote ...` | promotion flow |

## Distribution Decision

The right destination is distribution through `cub` itself:

```bash
cub plugin install cub-gen
cub gen --help
```

That cannot be completed inside this repository alone. Until the `cub` plugin
loader is available, this repo can only prepare:

1. stable command mapping,
2. conservative compatibility language,
3. an in-repo `skills/cub-gen/SKILL.md`,
4. tests and docs that keep the standalone `cub-gen` contract honest.

## Done Means

Issue [#236](https://github.com/confighub/cub-gen/issues/236) should remain
open until all of these are true:

| Requirement | Current status |
|---|---|
| `cub gen --help` works through the `cub` plugin system | blocked on external plugin loader |
| standalone `cub-gen` compatibility is documented | ready in this repo |
| command mapping is documented | ready in this repo |
| `skills/cub-gen/SKILL.md` exists and reflects current command truth | ready in this repo |
| existing `cub-gen ...` scripts continue to work | true today |

The readiness work is still valuable because it prevents the migration from
becoming another naming puzzle. But the actual plugin shipment belongs with the
`cub` plugin mechanism.
