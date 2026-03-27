# 02. Lift Upstream

Request:

"This service now needs Redis-backed caching."

Why this should be lifted upstream:

- the app code must gain a Redis dependency
- the Spring app inputs must grow cache configuration
- the platform-rendered operational shape changes as a consequence

Relevant files:

- upstream build input: [`../pom.xml`](../pom.xml)
- upstream app config: [`../src/main/resources/application.yaml`](../src/main/resources/application.yaml)
- operational deployment shape: [`../operational/deployment.yaml`](../operational/deployment.yaml)
- route rule: [`../operational/field-routes.yaml`](../operational/field-routes.yaml)
- read-only Redis bundle: [`../lift-upstream.sh`](../lift-upstream.sh)
