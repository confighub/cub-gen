# Change CLI Contract v1 (Draft)

Status: Draft
Version: `1.0.0-draft`

This contract defines the developer-facing `cub-gen change` surface.

Goal: one stable interface for terminal users, CI jobs, and agent tool-calls.

## Design principles

1. One `change_id` lifecycle across preview, run, and explain.
2. Same JSON shape across local and connected execution.
3. No shell parsing of multiple intermediate files required.
4. Additive over existing `discover/import/publish/verify/attest` pipeline.

## Commands

### 1) `cub-gen change preview`

Purpose:
- Compute the proposed governed change and return a compact mutation card.

Proposed flags:

- `--repo <path>` (required)
- `--target <path>` (optional, default=`--repo`)
- `--space <name>` (optional, default=`platform`)
- `--out <path>` (optional JSON output file)
- `--json` (default output format)

Expected output fields (`ChangePreviewResult`):

- `change_id`
- `bundle_digest`
- `detected_profiles[]`
- `top_edit_recommendation.{owner,wet_path,dry_path,edit_hint,confidence}`
- `counts.{dry_inputs,wet_targets,inverse_patches}`
- `artifacts.{import,bundle,verify,attestation}`

### 2) `cub-gen change run`

Purpose:
- Execute a governed change end-to-end in one call.

Proposed flags:

- `--repo <path>` (required)
- `--target <path>` (optional, default=`--repo`)
- `--space <name>` (optional)
- `--mode <local|connected>` (required)
- `--base-url <url>` (required for `connected` unless resolved from `cub context`)
- `--token <token>` (required for `connected` unless resolved from `cub auth`)
- `--out <path>` (optional)
- `--json` (default output format)

Expected output fields (`ChangeRunResult`):

- `change_id`
- `bundle_digest`
- `decision.{state,authority,source}`
- `verification.{bundle_valid,attestation_valid}`
- `promotion_ready` (boolean)
- `top_edit_recommendation.{owner,wet_path,dry_path,edit_hint,confidence}`
- `artifacts.{bundle,verify,attestation,decision_query}`

### 3) `cub-gen change explain`

Purpose:
- Explain exactly what to edit for a specific field/resource.

Proposed flags:

- `--change-id <id>` (required)
- `--wet-path <path>` (optional, one of `--wet-path` or `--resource` required)
- `--resource <kind/name>` (optional)
- `--mode <local|connected>` (required)
- `--base-url <url>` (required for `connected` unless resolved from context)
- `--token <token>` (required for `connected` unless resolved from auth)
- `--json` (default output format)

Expected output fields (`ChangeExplainResult`):

- `change_id`
- `query.{wet_path,resource}`
- `owner`
- `source.{file,path,line}`
- `confidence`
- `edit_hint`
- `evidence_refs[]`

## Exit code contract

- `0`: success (`ALLOW` or successful local run)
- `2`: policy `BLOCK`
- `3`: policy `ESCALATE`
- `4`: auth/context missing for connected mode
- `5`: backend unreachable/endpoint missing
- `6`: invalid user input
- `7`: contract/triple validation failed
- `8`: internal execution error

## MVP compatibility mapping (today)

Until native `cub-gen change` subcommands are implemented, wrappers map to existing commands:

- `change preview`
  - `cub-gen gitops import --json ...`
  - `cub-gen publish --in ...`
  - extract mutation card fields from import/provenance
- `change run (local)`
  - `examples/demo/app-ai-change-run.sh`
- `change run (connected)`
  - `examples/demo/run-confighub-lifecycle-connected.sh`
- `change explain`
  - read `inverse_edit_pointers` from import output or mutation card projection

## Determinism requirements

1. Same input repo+target produces stable output structure.
2. `change_id` uniqueness is guaranteed per run.
3. `bundle_digest` and `attestation_digest` are content-addressed and reproducible.
4. Connected mode must record decision source (`confighub-backend` vs explicit fallback source).

## Non-goals for v1

1. Replacing Flux/Argo reconciliation.
2. Defining backend HTTP API schemas (handled in API contract doc).
3. Hiding policy outcomes (`BLOCK`/`ESCALATE` remain explicit).
