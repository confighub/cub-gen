# 03. Generator-Owned

Request:

"Change `spring.datasource.*` or bypass the managed datasource boundary."

Why this must be blocked or escalated:

- datasource connectivity is platform-owned
- the field is not safe for app-local divergence
- direct mutation would bypass the runtime policy contract

Relevant files:

- upstream platform policy: [`../platform/base/runtime-policy.yaml`](../platform/base/runtime-policy.yaml)
- operational config: [`../operational/configmap.yaml`](../operational/configmap.yaml)
- route rule: [`../operational/field-routes.yaml`](../operational/field-routes.yaml)
- boundary bundle: [`../block-escalate.sh`](../block-escalate.sh)
