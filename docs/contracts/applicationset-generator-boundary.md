# ApplicationSet Generator Boundary

`cub-gen` now models Argo `ApplicationSet` as a first-class generator family
when the repo itself is ApplicationSet-centric and the child expansion can be
explained from explicit inputs.

This is intentionally narrower than the broader layered Helm/ApplicationSet work
from `#187`.

## What is in scope

First milestone support is:

1. **Authoritative list expansion**
   If the `ApplicationSet` uses a `list` generator, `cub-gen` expands child
   `Application` names from the explicit list elements in the repo.
2. **Authoritative clusters expansion**
   If the `ApplicationSet` uses a `clusters` generator and the repo includes a
   pinned cluster inventory snapshot, `cub-gen` expands child `Application`
   names from that inventory.
3. **Explicit degraded states**
   If the child set cannot be reproduced from repo inputs alone, `cub-gen`
   reports a deterministic degraded mode instead of pretending the expansion is
   authoritative.

## Modes

`application_set_analysis.mode` in `provenance[]` is the contract surface for
the boundary:

1. `authoritative`
   The repo contains enough explicit inputs to explain the child `Application`
   set deterministically.
2. `authoritative-expansion-unavailable`
   The parent `ApplicationSet` spec is governed, but deterministic child
   expansion is unavailable because a required inventory input is missing.
3. `observed-only`
   `cub-gen` recognizes the `ApplicationSet` boundary, but unsupported
   generator types such as `git`, `matrix`, or `merge` keep child expansion in
   observation mode.

## What the importer records

For `applicationset` generators, `cub-gen gitops import --json` now records:

1. a first-class `applicationset` generator contract
2. `rendered_object_lineage[]` entries for the parent `ApplicationSet` and any
   deterministically generated child `Application` objects
3. `field_origin_map[]` and `inverse_edit_pointers[]` that route child
   `Application` edits back to `spec.template.*` in the parent
4. `application_set_analysis` metadata with:
   - `generator_types`
   - `unsupported_generator_types`
   - `cluster_inventory_paths`
   - `matched_clusters`
   - `list_element_names`
   - `generated_applications`
   - `mode` / `mode_reason`

That means the repo can answer both:

1. "Why does this child `Application` exist?"
2. "Where do I edit to change it safely?"

## Relation to layered Helm work

The standalone `applicationset` family does **not** replace the existing layered
Helm provenance path.

Today the split is:

1. **Standalone ApplicationSet repo**
   detect/import as `applicationset`
2. **Helm repo with ApplicationSet as a layered input**
   keep the primary family as `helm`, and record the selector/inventory/overlay
   chain under `helm_layered_analysis`

That keeps this issue focused on the generator boundary itself while `#187`
continues to own the deeper Kubara-like multi-layer story.

## Proof commands

Deterministic standalone fixture:

```bash
./cub-gen gitops discover --space platform --json ./testdata/applicationset-standalone
./cub-gen gitops import --space platform --json ./testdata/applicationset-standalone
```

Look for:

1. `generator_kind = "applicationset"`
2. `resource_kind = "ApplicationSet"`
3. `provenance[0].application_set_analysis.mode = "authoritative"`
4. generated child `Application` lineage tied back to `applicationset.yaml`

Graceful degradation proof:

```bash
go test ./internal/importer -run '^TestImportRepoApplicationSetGracefulDegradationWithoutClusterInventory$' -count=1 -v
```
