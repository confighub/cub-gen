# Contracts: `scoredev-paas`

Treat these as stable inspection contracts for human operators and AI
assistants. If any `expects` check fails, stop and report the mismatch.

## Repo preview

```yaml
id: score_repo_preview
command: ./cub-gen gitops import --space platform --json ./examples/scoredev-paas
mutates:
  repo: false
  backend: false
  live: false
expects:
  generator_profile: scoredev-paas
  dry_inputs_include:
    - score.yaml
  wet_manifest_targets_min: 1
  inverse_edit_pointers_min: 1
inspect_with:
  - jq '.discovered[0].generator_profile'
  - jq '.dry_inputs'
  - jq '.wet_manifest_targets | length'
  - jq '.provenance[0].inverse_edit_pointers[0]'
```

## Read-only workload contract check

```yaml
id: score_validate_workload
command: ./cub-gen score validate-workload --score ./examples/scoredev-paas/score.yaml --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml
mutates:
  repo: false
  backend: false
  live: false
expects:
  exit_code: 0
  stdout_contains:
    - "ALLOW"
inspect_with:
  - cat
```

## Local governed workload proof

```yaml
id: score_local_governed_workload
command: ./examples/scoredev-paas/demo-governed-workload.sh
mutates:
  repo: scratch_clone_only
  backend: false
  live: false
expects:
  stdout_contains:
    - "[score-governed] success"
  files_exist:
    - ".tmp/scoredev-governed-workload/<run>/allow-explain.json"
    - ".tmp/scoredev-governed-workload/<run>/allow.txt"
    - ".tmp/scoredev-governed-workload/<run>/escalate.txt"
  evidence:
    allow_path: "image update stays allowed"
    escalate_path: "unapproved resource type escalates"
inspect_with:
  - jq '.explanation.owner, .explanation.dry_path' allow-explain.json
  - cat allow.txt
  - cat escalate.txt
```

## Standalone runtime proof

```yaml
id: score_standalone_runtime
command: ./examples/scoredev-paas/demo-runtime.sh
mutates:
  repo: false
  backend: false
  live: kind_cluster_only
expects:
  stdout_contains:
    - "E2E verification PASSED"
    - "[score-runtime] success"
  files_exist:
    - "examples/scoredev-paas/var/runtime-manifests.yaml"
  live_checks:
    - "checkout-api deployment has a ready replica"
    - "/healthz returns ok"
    - "/ returns service=checkout-api and logLevel=warn"
inspect_with:
  - kubectl -n checkout-api get deployment checkout-api -o yaml
  - kubectl -n checkout-api get service checkout-api -o yaml
  - ./examples/scoredev-paas/verify-e2e.sh
```

## Connected preflight

```yaml
id: score_connected_preflight
command: cub auth login && ./examples/demo/run-connected-smoke.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  shared_smoke_examples:
    - helm-paas
    - springboot-paas
  purpose: "confirm auth and backend health before the Score deep walkthrough"
inspect_with:
  - jq '{change_id, verify_valid, attestation_valid}' .tmp/connected-smoke/<run>/helm-paas/summary.json
  - jq '{change_id, verify_valid, attestation_valid}' .tmp/connected-smoke/<run>/springboot-paas/summary.json
```

## Deep connected walkthrough

```yaml
id: score_connected_deep
command: OUTPUT_DIR=.tmp/scoredev-connected ./examples/scoredev-paas/demo-connected.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  files_exist:
    - ".tmp/scoredev-connected/create/summary.json"
    - ".tmp/scoredev-connected/update/summary.json"
  decision_state: terminal_allow_escalate_or_block
  note: "connected proof complements the standalone live-cluster proof; it does not replace it"
inspect_with:
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/scoredev-connected/create/summary.json
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/scoredev-connected/update/summary.json
```
