# Kubara-Like Platform Example

This is the discoverable entrypoint for the Kubara-like platform pattern in the
docs.

It does not claim to be an actual Kubara product import. It shows the pattern:
a platform or app-config manager reads source config, environment context, and
shared machinery, then produces governed config data that ConfigHub can reason
about.

## What this proves today

| Slice | Status | How to prove it now |
|---|---|---|
| App config without a Kubernetes platform | Real | `./examples/kubara/demo-local.sh` |
| Multi-repo platform graph before rewrites | Real | `./examples/kubara/demo-local.sh` |
| Base/Deployment Variant topology | Real | `./examples/kubara/demo-local.sh` |
| Connected evidence smoke | Wrapper | `cub auth login && ./examples/kubara/demo-connected.sh` |
| Actual Kubara repo import | Not claimed | use this as the pattern, not a conformance test |

## If you already run a Kubara-like platform

Start here if your platform is mostly an app config manager plus shared
machinery:

- app teams own provider/app config,
- platform teams own shared defaults and policy,
- environments or tenants provide deployment context,
- the platform produces deployable or runtime config.

The useful question is:

```text
If this deployed field is wrong, is the durable fix in app config, platform
defaults, environment context, or a deployment-specific override?
```

## Why this maps cleanly to the cub-gen framework

| Kubara-like concept | cub-gen concept | Why it matters |
|---|---|---|
| app-owned config file | Component source config | app teams keep editing familiar files |
| shared machinery/defaults | Generator/platform contract | platform ownership stays explicit |
| env/tenant context | Deployment Variant + Target | generated copies can be compared and governed |
| rendered/runtime config | rendered config proof | every field can get owner and route evidence |

## Local and Connected Entrypoints

From the repo root:

```bash
go build -o ./cub-gen ./cmd/cub-gen

./examples/kubara/demo-local.sh

cub auth login
./examples/kubara/demo-connected.sh
```

The connected wrapper validates authentication and runs the same deterministic
proof locally. It does not deploy and does not rewrite actual Kubara repos.

## Direct Commands

```bash
./cub-gen gitops import --space platform --json \
  ./examples/just-apps-no-platform-config

./cub-gen platform import --json \
  ./testdata/platform-estate/platform.yaml

./cub-gen platform import --json \
  ./testdata/variant-topology/platform.yaml
```
