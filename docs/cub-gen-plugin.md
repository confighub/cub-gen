# `cub gen` Plugin

Status: implemented in this repo. The standalone command is still `cub-gen`,
but the same binary can now run as a `cub` plugin named `gen`.

## Why `cub gen`

`gen` means Generator. A Generator is a function on config data: it reads source
config such as Helm values, Score files, Spring config, OpenChoreo CRs, Argo
ApplicationSets, environment settings, and secret references, then returns
deployable Kubernetes config.

The product split is:

| Surface | Vantage point | Owns |
|---|---|---|
| `cub gen` | source repos and Generator inputs | provenance, inverse edits, bundles |
| `cub gitops` | ConfigHub intent | spaces, units, targets, governed import |
| `cub-scout` | live cluster | runtime ownership, health, troubleshooting |

## Install Shape

For local source testing:

```bash
make build-plugin
CUB_CONFIG="$PWD/.tmp/cub-plugin/config.yaml" cub gen --help
```

For a manual user-level install from this repo:

```bash
mkdir -p "$HOME/.confighub/plugins/gen"
go build -o "$HOME/.confighub/plugins/gen/main" ./cmd/cub-gen
cub gen --help
```

For release installs, the archive must contain both:

- `cub-gen`: the standalone binary
- `main`: the plugin entry point extracted to `$CUB_CONFIG/plugins/gen/main`

The `.goreleaser.yaml` in this repo builds that shape for Linux and macOS.
Windows keeps the standalone binary only because the current `cub` plugin exec
path uses `syscall.Exec`.

Once a plugin-compatible release is cut, the public install path is:

```bash
cub plugin install confighub/cub-gen
cub gen --help
```

## How `cub` Runs It

When a user runs:

```bash
cub gen --help
```

`cub` looks for a plugin named `gen`, sets plugin environment variables, and
execs the plugin binary. `cub-gen` detects `CUB_PLUGIN=1` and renders help as
`cub gen ...` instead of `cub-gen ...`.

The inherited variables are useful for connected flows:

| Variable | Use |
|---|---|
| `CUB_PLUGIN=1` | switch help and alias behavior to plugin mode |
| `CUB_CONFIG` | active `cub` config directory |
| `CUB_CONTEXT` | active `cub` context name |
| `CUB_SERVER` | default ConfigHub base URL for connected commands |
| `CUB_SPACE` | inherited default space from the host; commands still accept explicit `--space` |
| `CUB_TOKEN` | default bearer token for connected commands |

`CONFIGHUB_BASE_URL` and `CONFIGHUB_TOKEN` still work and take precedence over
`CUB_SERVER` and `CUB_TOKEN`.

## Command Mapping

The nested standalone commands still work in plugin mode. The plugin also adds
shorter aliases for the front-door `cub gen ...` shape.

| Standalone | Plugin |
|---|---|
| `cub-gen detect` | `cub gen detect` |
| `cub-gen generators` | `cub gen generators` |
| `cub-gen gitops discover` | `cub gen discover` |
| `cub-gen gitops import` | `cub gen import` or `cub gen render` |
| `cub-gen gitops cleanup` | `cub gen cleanup` |
| `cub-gen platform import` | `cub gen platform import` |
| `cub-gen platform fanout` | `cub gen fanout` |
| `cub-gen platform adapt` | `cub gen adapt` |
| `cub-gen enrich preview` | `cub gen enrich preview` |
| `cub-gen normalize preview` | `cub gen normalize preview` |
| `cub-gen publish` | `cub gen bundle` |
| `cub-gen verify` | `cub gen bundle verify` |
| `cub-gen attest` | `cub gen bundle attest` |
| `cub-gen verify-attestation` | `cub gen bundle verify-attestation` |
| `cub-gen proof events` | `cub gen bundle events` or `cub gen proof events` |
| `cub-gen change preview` | `cub gen preview` |
| `cub-gen change run` | `cub gen run` |
| `cub-gen change diff` | `cub gen diff` |
| `cub-gen change revision-diff` | `cub gen revision-diff` |
| `cub-gen change impact` | `cub gen impact` |
| `cub-gen change explain` | `cub gen explain` |
| `cub-gen bridge ingest` | `cub gen ingest` |
| `cub-gen bridge link` | `cub gen link` |
| `cub-gen bridge decision ...` | `cub gen decision ...` |
| `cub-gen bridge promote ...` | `cub gen promote ...` |

## Compatibility Contract

1. Existing `cub-gen ...` scripts continue to work.
2. Plugin aliases are front-door conveniences; they dispatch to the same
   underlying implementation.
3. `cub gen` preserves the local-first safety contract: no implicit deploys and
   no hidden control-plane writes.
4. Help text must make the source-side role clear so users do not confuse it
   with `cub gitops`.

## Proof

Current repo proof:

```bash
go test ./cmd/cub-gen -run '^(TestPluginModeTopLevelHelpRendersCubGen|TestPluginModeAliasesRouteToExistingCommands|TestCubHostDispatchesGenPluginWhenAvailable)$' -count=1 -v
make plugin-smoke
```

`TestCubHostDispatchesGenPluginWhenAvailable` stages the built binary under a
temporary `$CUB_CONFIG/plugins/gen/main` path and proves `cub gen --help`
dispatches through the real `cub` plugin host when `cub` is available on PATH.
