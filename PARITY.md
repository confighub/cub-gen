# cub-gen parity with `cub gitops`

This document tracks behavior parity between `cub-gen gitops` (local prototype)
and `cub gitops` (ConfigHub CLI implementation).

Status values:
- `matched`: behavior is intentionally mirrored
- `partial`: same contract shape, simplified implementation
- `deferred`: intentionally not implemented yet

Current lock level: `v0.1-parity-locked` (2026-03-05 baseline)

Contract lock means:
- command names/arity for `discover|import|cleanup` are frozen for v0.1
- JSON and table output contracts are golden-tested
- help/usage output for `gitops` and subcommands is golden-tested
- unsupported behavior must fail explicitly (never silently degrade)
- Helm import contract includes explicit DRY/WET surfaces (`dry_inputs`, `wet_manifest_targets`) and provenance lineage fields (`chart_path`, `values_paths`, `rendered_object_lineage`)
- Spring Boot import contract includes explicit DRY ownership (`dry_inputs.owner`) and platform-owned WET targets (`wet_manifest_targets.owner`) with app-team inverse-edit hints

## Command contract parity

| Area | `cub gitops` | `cub-gen gitops` | Status | Notes |
|---|---|---|---|---|
| Command group | `gitops` | `gitops` | matched | Same top-level grouping |
| Discover command | `gitops discover <target-slug>` | `gitops discover <target-slug>` | matched | Same arity and purpose |
| Import command | `gitops import <target-slug> <render-target-slug>` | `gitops import <target-slug> <render-target-slug>` | matched | Same arity |
| Cleanup command | `gitops cleanup <target-slug>` | `gitops cleanup <target-slug>` | matched | Same arity |
| `--space` | required context in cub | accepted label in prototype | partial | Used for discover slug/state partitioning |
| `--where-resource` | full where expression support | subset support (`kind`, `name`, `root`, `id`, `LIKE`, `IN`, `AND`) | partial | Unsupported clauses return explicit error |
| `--wait` on import | controls async wait | accepted no-op | partial | Kept for CLI compatibility |
| `--json` output mode | structured output | structured output | matched | Golden-tested |
| Default/table output mode | human-oriented command output | human-oriented command output | matched | Golden-tested for discover + import |
| Optional bridge command | N/A | `publish` (top-level) | deferred/parity-safe | Added outside `gitops` contract so v0.1 parity surface remains stable |

## Flow parity

| Flow step | `cub gitops` | `cub-gen gitops` | Status | Notes |
|---|---|---|---|---|
| Discover stage | creates/reuses discover unit and dry-run import | persists discover state in `.cub-gen/discover/*.json` | partial | Same lifecycle concept, local file instead of API unit |
| Import stage starts from discover | yes | yes | matched | `Import` calls `Discover` first |
| Create dry units | yes | yes | partial | Local artifacts only |
| Render to wet units | yes | yes | partial | Local derived artifacts only |
| Create links | yes | yes | partial | Local link records |
| Cleanup discover state | yes | yes | matched | Removes discover-state file |

## Target resolution parity

| Capability | Status | Notes |
|---|---|---|
| Direct path target (`<target-slug>` as repo path) | matched | Directory path resolves directly |
| Alias target (`<target-slug>` from config) | matched | Uses `CUB_GEN_TARGETS_FILE` or `.cub-gen/targets.json` |
| Toolchain + provider capability checks | partial | Enforced from local target metadata (not ConfigHub server targets) |
| ConfigHub target lookup (`cub target list`, target IDs) | deferred | Planned bridge step after parity baseline |

## Intentionally deferred integration features

| Feature | Status | Reason |
|---|---|---|
| ConfigHub API calls (`CreateUnit`, `BulkApply`, links in server state) | deferred | This phase is local-only parity foundation |
| Render-target provider validation against live target capabilities | partial | Validation implemented from local metadata only |
| Async queued operation handling (`--wait` behavior) | deferred | No server queue in local mode |
| Real bridge-worker render execution | deferred | Prototype uses synthetic local artifacts |

## Proof references

- Command parity golden tests:
  - `cmd/cub-gen/testdata/parity/gitops-discover.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import-spring.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-cleanup.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-discover.table.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-import.table.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-help.stdout.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-discover-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-import-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-cleanup-help.stderr.golden.txt`
- Error-mode tests:
  - `cmd/cub-gen/gitops_parity_test.go`
