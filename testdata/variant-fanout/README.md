# Variant Fanout Fixture

This fixture proves the v0.4 spine:

```text
Component
  -> Deployable Variant
      -> Target
      -> Proof bundle
```

`platform.yaml` declares three Components:

| Component | Generator | Variants | Variant source |
|---|---|---|---|
| `checkout-api` | Helm | dev, stage, prod | same chart plus explicit Helm CLI overrides |
| `score-checkout` | Score.dev | dev, stage, prod | one Score repo per environment |
| `spring-checkout` | Spring Boot | dev, stage, prod | one Spring config repo per environment |

Run:

```bash
cub-gen platform fanout --json --out fanout.json \
  ./testdata/variant-fanout/platform.yaml
```

Each `variants[]` entry contains the stable `variant_id`, separated
`shared_inputs` and `variant_inputs`, a distinct `change_id`, and a standard
`change-bundle/v1` payload. A variant can be explained directly:

```bash
CHANGE_ID="$(jq -r '.variants[] | select(.variant_id=="checkout-api/dev") | .change_id' fanout.json)"
cub-gen change explain --change-id "$CHANGE_ID" --bundle fanout.json \
  --variant checkout-api/dev
```

The command is read-only. It does not deploy, mutate repos, or infer variants
from globs.
