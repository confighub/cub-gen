# 2026-03-08 Swamp/C3Agent Import and Detection Corrections

## Context

During post-merge review of `main`, three correctness gaps were identified in the Swamp workflow path mapping and detection behavior:

1. Swamp model binding inverse paths were missing the `jobs[]` segment.
2. Swamp model binding edit guidance pointed to base config even when workflow overlay file existed.
3. Swamp detection did not include nested `workflow-*.yaml` files despite comments indicating child-directory support.

These issues impact provenance accuracy and inverse-edit reliability for agentic workflow repos.

## What was changed

### 1) Correct Swamp DRY path for model bindings

Updated all Swamp model-binding references from:

- `steps[].task.modelIdOrName`

to:

- `jobs[].steps[].task.modelIdOrName`

This update was applied in:

- inverse patch plan generation
- field origin map generation
- inverse edit pointer generation

Files:

- `internal/importer/importer.go`

### 2) Use workflow-specific inverse hint when overlay workflow exists

Added conditional hint key selection in Swamp inverse pointer generation:

- base-only: `model_binding_base`
- workflow overlay present: `model_binding_workflow`

Also updated Swamp hint templates to support path rendering:

- `model_binding_base`: `Edit model references in {{base_config_path}}.`
- `model_binding_workflow`: `Edit model method bindings in {{workflow_path}} for task-specific overrides.`

Files:

- `internal/importer/importer.go`
- `internal/registry/registry.go`

### 3) Make Swamp detection include child-directory workflows

Replaced non-recursive glob behavior with recursive `WalkDir` under the Swamp root.

Now detection collects:

- sibling `workflow-*.yaml|yml`
- nested child-directory `workflow-*.yaml|yml`

while still honoring skip dirs (`.git`, `vendor`, etc.).

File:

- `internal/detect/detect.go`

## Tests added/updated

### New/expanded tests

- Added c3agent and swamp coverage in detect example matrix.
- Added nested workflow detection test for Swamp.
- Added c3agent and swamp coverage in importer example/capability matrices.
- Added explicit Swamp contract assertions for:
  - corrected DRY path
  - workflow-specific inverse hint text
  - field-origin source mapping to workflow file

Files:

- `internal/detect/detect_test.go`
- `internal/importer/importer_test.go`
- `internal/registry/registry_test.go`

### Golden updates

Swamp parity snapshots updated to reflect corrected DRY paths and hints:

- `cmd/cub-gen/testdata/parity/gitops-import-swamp.golden.json`
- `cmd/cub-gen/testdata/parity/publish-from-import-swamp.golden.json`
- `cmd/cub-gen/testdata/parity/publish-direct-swamp.golden.json`

## Verification

Executed successfully:

- `make ci`
- full parity suite in `cmd/cub-gen`
- all internal package tests

No failing tests remain after these corrections.

## Why this matters

These fixes make the Swamp generator triple reliable for:

- provenance trust (accurate field-origin DRY paths)
- inverse editing UX (correct file guidance)
- real-world repo topology (nested workflow directories)

This brings Swamp behavior in line with the project rule that generator triples must be governance-grade and path-accurate.
