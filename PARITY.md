# cub-gen parity with `cub gitops`

This document tracks behavior parity between `cub-gen gitops` (local prototype)
and `cub gitops` (ConfigHub CLI implementation).

Status values:
- `matched`: behavior is intentionally mirrored
- `partial`: same contract shape, simplified implementation
- `deferred`: intentionally not implemented yet

Current lock level: `v0.2-preview-parity-locked` (2026-03-06 expanded contract baseline)

Contract lock means:
- command names/arity for `discover|import|cleanup` are frozen for v0.1
- JSON and table output contracts are golden-tested
- help/usage output for `gitops` and subcommands is golden-tested
- unsupported behavior must fail explicitly (never silently degrade)
- Helm import contract includes explicit DRY/WET surfaces (`dry_inputs`, `wet_manifest_targets`) and provenance lineage fields (`chart_path`, `values_paths`, `rendered_object_lineage`)
- Spring Boot import contract includes explicit DRY ownership (`dry_inputs.owner`) and platform-owned WET targets (`wet_manifest_targets.owner`) with app-team inverse-edit hints
- Backstage IDP import contract includes explicit catalog DRY ownership (`catalog-spec`) and platform-owned WET targets with inverse-edit hints for component metadata
- Ably app-config import contract includes app-team DRY ownership (`provider-config-*`) and platform-owned WET targets with inverse-edit hints for runtime provider fields
- Ops workflow import contract includes platform DRY ownership (`operations-*`) and platform-owned WET workflow/job targets with inverse-edit hints for execution intent fields

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
| Bridge digest fields | N/A | `digest_algorithm` + `bundle_digest` | matched (local contract) | Deterministic bundle verification handle for attestation pipelines |
| Optional verify command | N/A | `verify` (top-level) | matched (local contract) | Verifies schema + digest integrity; non-zero exit on mismatch |
| Optional attest command | N/A | `attest` (top-level) | matched (local contract) | Emits attestation envelope only from valid verified bundles |
| Optional verify-attestation command | N/A | `verify-attestation` (top-level) | matched (local contract) | Verifies attestation integrity and optional bundle linkage |
| Generator inventory command | N/A | `generators` (top-level) | matched (local contract) | Lists registry-backed supported generator families |

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
  - `cmd/cub-gen/testdata/parity/top-help.stdout.golden.txt`
  - `cmd/cub-gen/testdata/parity/publish-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/verify-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/attest-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/verify-attestation-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-help.stdout.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-discover-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-import-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-cleanup-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-discover.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-discover-score.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-discover-spring.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-discover-backstage.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-discover-ably.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-discover-ops.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import-score.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import-spring.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import-backstage.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import-ably.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-import-ops.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-cleanup.golden.json`
  - `cmd/cub-gen/testdata/parity/gitops-discover.table.golden.txt`
  - `cmd/cub-gen/testdata/parity/gitops-import.table.golden.txt`
  - `cmd/cub-gen/testdata/parity/publish-from-import.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-from-import-score.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-from-import-spring.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-from-import-backstage.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-from-import-ably.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-from-import-ops.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-direct-helm.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-direct-score.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-direct-spring.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-direct-backstage.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-direct-ably.golden.json`
  - `cmd/cub-gen/testdata/parity/publish-direct-ops.golden.json`
  - `cmd/cub-gen/testdata/parity/verify.stdout.golden.txt`
  - `cmd/cub-gen/testdata/parity/verify.json.golden.json`
  - `cmd/cub-gen/testdata/parity/verify-tampered.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/attest.json.golden.json`
  - `cmd/cub-gen/testdata/parity/attest-tampered.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/verify-attestation.json.golden.json`
  - `cmd/cub-gen/testdata/parity/verify-attestation-linked-helm.json.golden.json`
  - `cmd/cub-gen/testdata/parity/verify-attestation-linked-score.json.golden.json`
  - `cmd/cub-gen/testdata/parity/verify-attestation-linked-spring.json.golden.json`
  - `cmd/cub-gen/testdata/parity/verify-attestation-linked-backstage.json.golden.json`
  - `cmd/cub-gen/testdata/parity/verify-attestation-linked-ably.json.golden.json`
  - `cmd/cub-gen/testdata/parity/verify-attestation-linked-ops.json.golden.json`
  - `cmd/cub-gen/testdata/parity/generators.golden.json`
  - `cmd/cub-gen/testdata/parity/generators.table.golden.txt`
  - `cmd/cub-gen/testdata/parity/generators-help.stderr.golden.txt`
  - `cmd/cub-gen/testdata/parity/verify-attestation-tampered.stderr.golden.txt`
- Error-mode tests:
  - `cmd/cub-gen/command_surface_parity_test.go`
  - `cmd/cub-gen/gitops_parity_test.go`
