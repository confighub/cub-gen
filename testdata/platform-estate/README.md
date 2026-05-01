# Platform Estate Fixture

Tiny read-only platform import fixture for `cub-gen platform import`.

The manifest intentionally includes:

- a Helm app repo with owner, component, variant, and target metadata;
- a Score app repo missing its owner;
- platform/env/rendered repos that exist but do not contain a supported generator;
- one missing repo path.

This proves that platform import builds a graph first and reports gaps instead
of guessing or rewriting anything.

Run it from the repo root:

```bash
./cub-gen platform import --json ./testdata/platform-estate/platform.yaml
```

The command is read-only. It should be the first step before adoption reports,
enrichment sidecars, variant fanout, or rewrite proposals.
