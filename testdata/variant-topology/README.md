# Base and Deployment Variant Fixture

This fixture shows the ConfigHub topology language directly:

```text
Component
  -> Variant
      -> Base Variant        # no Target, no live address
      -> Deployment Variant  # has Target and live address
```

`checkout-api/base` is a Base Variant. It keeps the reusable Helm shape and
placeholder-like defaults, but it is not meant to deploy.

`checkout-api/prod-us` is a Deployment Variant. It has concrete values and a
`prod-us` Target, so its rendered resources can be mapped to a live address.

Run:

```bash
./cub-gen platform import --json ./testdata/variant-topology/platform.yaml \
  | jq '.variants'
```

Expected lesson: both entries are real Variants of the same Component, but only
the Deployment Variant has a Target.
