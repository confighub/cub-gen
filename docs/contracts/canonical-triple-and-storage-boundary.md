# Canonical Contract Triple and Storage Boundary

Status: Sprint 2 (`#77` + `#78` + `#79` + `#80`)

This document is the canonical reference for:

1. The contract triple emitted by `cub-gen` import flows.
2. The runtime gate: no contract triple, no governed import.
3. The storage boundary: Git (DRY linkage) vs ConfigHub (WET governance).

## Canonical triple

`cub-gen gitops import --json` emits three linked artifacts per detected generator:

1. `contracts[]` (`GeneratorContract`)
2. `provenance[]` (`ProvenanceRecord`)
3. `inverse_transform_plans[]` (`InverseTransformPlan`)

Schema assets:

1. `internal/contracts/schemas/generator-contract.v1.schema.json`
2. `internal/contracts/schemas/provenance.v1.schema.json`
3. `internal/contracts/schemas/inverse-transform-plan.v1.schema.json`

Runtime validator:

1. `internal/contracts/validator.go`

## Required schema versions

1. `contracts[].schema_version = "cub.confighub.io/generator-contract/v1"`
2. `provenance[].schema_version = "cub.confighub.io/provenance/v1"`
3. `inverse_transform_plans[].schema_version = "cub.confighub.io/inverse-transform-plan/v1"`

## No-triple-no-governed-import gate

Governed import is blocked when:

1. Any triple slice is missing while generators were detected.
2. Triple cardinality does not match detected generator count.
3. Any triple element fails schema validation.

Gate implementation:

1. `contracts.ValidateGovernedImportTriples(...)` in `internal/contracts/validator.go`
2. Called from importer pipeline in `internal/importer/importer.go`

Deterministic blocker messages are locked by tests in:

1. `internal/contracts/validator_test.go`
2. `internal/importer/importer_test.go`

## Output mapping (field-level)

`GeneratorContract` (`contracts[]`) captures deterministic generator execution contract:

1. Identity: `generator_id`, `name`, `kind`, `profile`, `version`
2. Source: `source_repo`, `source_ref`, `source_path`
3. IO: `inputs[]`, `output_format`, `transport`
4. Behavior: `capabilities[]`, `deterministic`

`ProvenanceRecord` (`provenance[]`) captures DRY->WET lineage and inverse edit guidance:

1. Identity: `provenance_id`, `change_id`, `generator_*`, `version`
2. Content references: `sources[]`, `outputs[]`, `input_digest`
3. Render lineage: `rendered_object_lineage[]`
4. Editability trace: `field_origin_map[]`, `inverse_edit_pointers[]`
5. Helm-specialized source hints: `chart_path`, `values_paths`

`InverseTransformPlan` (`inverse_transform_plans[]`) captures governed reverse-edit plan:

1. Identity: `plan_id`, `change_id`, `source_kind`, `target_unit_id`, `status`
2. Planned reverse mutations: `patches[]`
3. Timing: `created_at`

## Storage boundary (Git DRY vs ConfigHub WET)

Boundary model (agentic-gitops aligned):

1. Git stores DRY intent and source-of-change linkage.
2. ConfigHub stores explicit WET governance state and decisions.
3. Flux/Argo reconcile WET -> LIVE runtime state.

Git side (today, implemented):

1. DRY generator sources (`Chart.yaml`, `values*.yaml`, `score.yaml`, Spring app config, etc.).
2. Import/read path into triple artifacts via `gitops discover/import`.
3. Optional bridge bundle output from `publish`.

ConfigHub side (bridge path, intentionally separate):

1. Governed WET unit state and approval/attestation authority.
2. Decision path and execution rights after policy gates.
3. Not a reconciler; Flux/Argo remain runtime reconciliation engines.

Operational rule:

1. Nothing implicit deploys.
2. Nothing observed silently overwrites intent.

## Copy/paste verification commands

Build:

```bash
go build ./cmd/cub-gen
```

Inspect canonical triple from Helm example:

```bash
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{
      contract: (.contracts[0] | {schema_version,generator_id,name,kind,profile,source_repo,source_ref,inputs,capabilities,deterministic}),
      provenance: (.provenance[0] | {schema_version,provenance_id,change_id,generator_id,sources,outputs,field_origin_map,inverse_edit_pointers,chart_path,values_paths}),
      inverse_plan: (.inverse_transform_plans[0] | {schema_version,plan_id,change_id,source_kind,target_unit_id,status,patches})
    }'
```

Validate no-triple-no-governed-import blocker path:

```bash
go test ./internal/importer -run '^TestImportDetectionFailsOnInvalidContractTriple$' -count=1 -v
```

Run full gate set:

```bash
go test ./...
make ci
```
