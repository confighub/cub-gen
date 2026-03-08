# Decision: Add Markdown Generator Visibility Surface

Date: 2026-03-08
Status: accepted
Related PRs: #112, #113

## Context

Generator triples (contract + provenance + inverse plan) are central to cub-gen, but platform owners needed a readable surface without inspecting Go structs or building `jq` pipelines.

Before this decision:

1. `cub-gen generators` only had table and JSON output.
2. Rich policy/provenance templates were available only via JSON details.
3. Practical review in GitHub/Markdown contexts was awkward.

## Decision

Add a first-class Markdown output mode for generator inventory and triple inspection.

Implemented command surface:

1. `cub-gen generators --markdown`
2. `cub-gen generators --markdown --details`

Guardrails:

1. `--markdown` cannot be combined with `--json`.
2. `--details` requires `--json` or `--markdown`.

`--markdown --details` includes deterministic, sorted sections for:

1. input role rules and defaults
2. role ownership
3. inverse patch templates
4. inverse pointer templates
5. field-origin confidences
6. hint defaults
7. inverse reasons and edit hints
8. WET targets
9. rendered lineage templates

## Rationale

1. Improves platform-owner readability immediately with minimal architectural churn.
2. Preserves existing Go registry source-of-truth while reducing inspection friction.
3. Keeps output contract stable via golden tests.
4. Creates a bridge toward generated docs (Approach B) and potential YAML-first authoring (Approach A/C).

## Consequences

Positive:

1. Faster review/adoption for new generator families.
2. Better copy/paste artifacts for design reviews and docs.
3. Deterministic output suitable for CI parity locking.

Trade-offs:

1. Authoring still requires Go edits in `internal/registry/registry.go`.
2. Markdown output is verbose for `--details` (intended for inspection, not terse CLI output).

## Follow-up

1. Evaluate whether to add `cub-gen generators --markdown --kind <kind>` snippets directly into docs automation.
2. Decide between YAML-first registry (Approach A) and combined YAML + generated docs (Approach C).
3. If Approach A/C is selected, keep Markdown surface as compatibility layer during migration.
