# OpenChoreo Example

OpenChoreo is a good example of a platform that acts as a Generator: it reads
component/workload config, environment bindings, platform contracts, and secret
references, then returns deployable Kubernetes config.

This example is the discoverable wrapper for the fixture-backed hardgate in
[`testdata/openchoreo-hardgate`](../../testdata/openchoreo-hardgate/). It is
intentionally honest: this proves the hard shape for OpenChoreo-style config,
not full upstream OpenChoreo repo conformance.

## What this proves today

| Slice | Status | How to prove it now |
|---|---|---|
| OpenChoreo CRD-shaped detection | Real | `./examples/openchoreo/demo-local.sh` |
| Workload to rendered-resource provenance | Real | `./examples/openchoreo/demo-local.sh` |
| Generated-resource ownership and edit routing | Real | `./examples/openchoreo/demo-local.sh` |
| Connected evidence smoke | Wrapper | `cub auth login && ./examples/openchoreo/demo-connected.sh` |
| Full upstream OpenChoreo import | Not claimed | follow-on repo validation |

## If you already run OpenChoreo

Start here if you want to see whether cub-gen's Generator model matches the
shape of OpenChoreo:

- `Workload` is app-owned source config.
- `ComponentType` is platform-owned contract config.
- `ReleaseBinding` is environment-owned deployment context.
- `SecretReference` is security-owned binding config.
- `RenderedRelease` and generated Kubernetes resources are rendered config.

The useful question is:

```text
If this generated Kubernetes field is wrong, should I edit the app workload,
the environment binding, the platform component type, a secret reference, or
the rendered object?
```

## Why this maps cleanly to the cub-gen framework

| OpenChoreo concept | cub-gen concept | Why it matters |
|---|---|---|
| `Workload` | Component/app source config | app-owned fields can route back to app config |
| `ReleaseBinding` | Deployment Variant context | environment-specific choices stay visible |
| `ComponentType` | platform Generator contract | platform-owned structure is not silently editable by app teams |
| `SecretReference` | Connection evidence | secret wiring has owner and route proof |
| `RenderedRelease` | rendered config proof | generated Kubernetes resources can trace back to inputs |

## Local and Connected Entrypoints

From the repo root:

```bash
go build -o ./cub-gen ./cmd/cub-gen

./examples/openchoreo/demo-local.sh

cub auth login
./examples/openchoreo/demo-connected.sh
```

The connected wrapper validates authentication and produces the same
deterministic evidence locally. It does not deploy and does not rewrite
OpenChoreo resources.

## Direct Commands

```bash
./cub-gen gitops discover --space platform --adoption-report --json \
  ./testdata/openchoreo-hardgate

./cub-gen gitops import --space platform --json \
  ./testdata/openchoreo-hardgate

./cub-gen publish --space platform ./testdata/openchoreo-hardgate \
  | ./cub-gen verify --in -
```
