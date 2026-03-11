# Confidence Scores (How to Read Them)

`cub-gen` emits confidence on field-origin mappings and inverse-edit guidance.
Use confidence as an execution hint, not as a replacement for review.

## Quick interpretation

| Confidence range | Meaning | Recommended action |
|---|---|---|
| `>= 0.90` | Strong mapping confidence | Safe default for direct DRY edits in normal app-team flow. |
| `0.75 - 0.89` | Likely correct, some ambiguity | Use `change preview` + `change explain` before merge. |
| `< 0.75` | Ambiguous mapping | Escalate to platform owner or reviewer before applying. |

## Why confidence varies

Confidence depends on generator structure:

- High confidence: explicit schema and stable transforms.
- Medium confidence: overlays and inferred ownership.
- Lower confidence: multiple possible source paths for one WET field.

## Operational rule of thumb

1. Treat confidence as a routing signal.
2. Low confidence means "human review required", not "mapping is wrong".
3. Keep ownership and policy gates authoritative in connected mode.

## Commands to inspect confidence directly

```bash
# Top recommendations with confidence
./cub-gen change explain --space platform --owner app-team ./examples/helm-paas ./examples/helm-paas

# Raw provenance data (includes confidence fields)
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas
```

## Related

- [User story acceptance](user-story-acceptance.md)
- [Prompt as DRY](prompt-as-dry.md)
