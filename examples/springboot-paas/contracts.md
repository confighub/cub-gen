# Contracts: `springboot-paas`

Treat these as stable inspection contracts for human operators and AI
assistants. If any `expects` check fails, stop and report the mismatch.

## Read-only generator preview

```yaml
id: spring_generator_preview
command: ./examples/springboot-paas/generator/render.sh --explain
mutates:
  repo: false
  backend: false
  live: false
expects:
  stdout_contains:
    - "application"
    - "platform"
inspect_with:
  - cat
```

## Repo provenance import

```yaml
id: spring_repo_import
command: ./cub-gen gitops import --space platform --json ./examples/springboot-paas
mutates:
  repo: false
  backend: false
  live: false
expects:
  generator_profile: springboot-paas
  dry_inputs_min: 1
  wet_manifest_targets_min: 1
  inverse_edit_pointers_min: 1
inspect_with:
  - jq '.discovered[0].generator_profile'
  - jq '.dry_inputs | length'
  - jq '.wet_manifest_targets | length'
  - jq '.provenance[0].inverse_edit_pointers[0]'
```

## Local route proof

```yaml
id: spring_governed_routes
command: ./examples/springboot-paas/demo-governed-routes.sh
mutates:
  repo: false
  backend: false
  live: false
expects:
  stdout_contains:
    - "[spring-routes] success"
    - "allowed field"
    - "lift-upstream field"
    - "blocked field"
  files_exist:
    - ".tmp/springboot-governed-routes/<run>/allow.json"
    - ".tmp/springboot-governed-routes/<run>/lift.json"
    - ".tmp/springboot-governed-routes/<run>/block.json"
inspect_with:
  - jq '{route: .route.kind, decision: .decision.state}' .tmp/springboot-governed-routes/<run>/allow.json
  - jq '{route: .route.kind, decision: .decision.state}' .tmp/springboot-governed-routes/<run>/lift.json
  - jq '{route: .route.kind, decision: .decision.state}' .tmp/springboot-governed-routes/<run>/block.json
```

## Local embedded payload proof

```yaml
id: spring_embedded_config
command: ./examples/springboot-paas/demo-embedded-config-mutation.sh
mutates:
  repo: scratch_clone_only
  backend: false
  live: false
expects:
  stdout_contains:
    - "[spring-embedded] success"
  files_exist:
    - ".tmp/springboot-embedded-config/<run>/allow.txt"
    - ".tmp/springboot-embedded-config/<run>/compare.json"
    - ".tmp/springboot-embedded-config/<run>/refresh.json"
    - ".tmp/springboot-embedded-config/<run>/block.txt"
  evidence:
    apply_here: "feature.inventory.reservationMode is changed directly in ConfigMap.data[\"application.yaml\"]"
    blocked: "spring.datasource.url is rejected"
inspect_with:
  - cat allow.txt
  - jq '.\"feature.inventory.reservationMode\".values.prod' compare.json
  - jq '[.[] | select(.field == \"feature.inventory.reservationMode\")] | first' refresh.json
  - cat block.txt
```

## Connected smoke preflight

```yaml
id: spring_connected_smoke
command: cub auth login && ./examples/demo/run-connected-smoke.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  files_exist:
    - ".tmp/connected-smoke/<run>/springboot-paas/summary.json"
  summary_fields:
    change_id: non_empty
    verify_valid: true
    attestation_valid: true
    linked_bundle_check: true
inspect_with:
  - jq '{change_id, verify_valid, attestation_valid, linked_bundle_check}' .tmp/connected-smoke/<run>/springboot-paas/summary.json
```

## Deep connected walkthrough

```yaml
id: spring_connected_deep
command: OUTPUT_DIR=.tmp/springboot-connected ./examples/springboot-paas/demo-connected.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  files_exist:
    - ".tmp/springboot-connected/create/summary.json"
    - ".tmp/springboot-connected/update/summary.json"
  decision_state: terminal_allow_escalate_or_block
inspect_with:
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/springboot-connected/create/summary.json
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/springboot-connected/update/summary.json
```

## Live standalone app proof

```yaml
id: spring_live_runtime
command: ./examples/springboot-paas/verify-e2e.sh
mutates:
  repo: false
  backend: false
  live: true
expects:
  live_checks:
    - "inventory-api is reachable"
    - "verify-e2e.sh exits 0"
inspect_with:
  - kubectl get pods,svc -A
  - curl http://localhost:18080/actuator/health
```
