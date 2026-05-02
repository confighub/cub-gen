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

The fixture deliberately relies on the current ConfigHub rule:

```text
no Target -> Base Variant
Target    -> Deployment Variant
```

`cub-gen` emits `variant_kind` as an explanatory label in import/fanout output.
If a manifest supplies an explicit kind later, it must still agree with Target
presence.

Run:

```bash
./cub-gen platform import --json ./testdata/variant-topology/platform.yaml \
  | jq '.variants'
```

Expected lesson: both entries are real Variants of the same Component. The Base
Variant has no Target. The Deployment Variant has a Target.
