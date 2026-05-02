# Deployment Adaptation Fixture

This fixture shows the gap after cloning a Base Variant into a new Deployment
Variant.

```text
clone Variant -> add Target -> placeholders remain -> apply gate blocks
```

The platform manifest declares the placeholder tokens and the target context
needed to replace them. `cub-gen platform adapt` reads that explicit data and
emits a review-only plan. It does not edit Git, deploy, or bypass the
`vet-placeholders` gate.

Run:

```bash
cub-gen platform adapt --json ./testdata/deployment-adaptation/platform.yaml
```

Expected lesson: a Deployment Variant is known because it has a Target, but it
is not ready to apply until its placeholders are replaced through a reviewed
adaptation plan.
