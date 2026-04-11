# Contracts: `helm-paas`

Treat these as stable inspection contracts for human operators and AI
assistants. If any `expects` check fails, stop and report the mismatch.

## Repo preview

```yaml
id: helm_repo_preview
command: ./cub-gen gitops import --space platform --json ./examples/helm-paas
mutates:
  repo: false
  backend: false
  live: false
expects:
  generator_profile: helm-paas
  dry_inputs_min: 1
  wet_manifest_targets_min: 1
  inverse_edit_pointers_min: 1
inspect_with:
  - jq '.discovered[0].generator_profile'
  - jq '.dry_inputs | length'
  - jq '.wet_manifest_targets | length'
  - jq '.provenance[0].inverse_edit_pointers[0]'
```

## Local governed ownership proof

```yaml
id: helm_local_governed_change
command: ./examples/helm-paas/demo-governed-change.sh
mutates:
  repo: scratch_clone_only
  backend: false
  live: false
expects:
  stdout_contains:
    - "[helm-governed] success"
  files_exist:
    - ".tmp/helm-paas-governed-change/<run>/allow-explain.json"
    - ".tmp/helm-paas-governed-change/<run>/allow-report.json"
    - ".tmp/helm-paas-governed-change/<run>/block-report.json"
    - ".tmp/helm-paas-governed-change/<run>/block-gate.log"
  evidence:
    allow_path: "app-team edit in values.yaml passes"
    block_path: "app-team edit in templates/deployment.yaml fails"
inspect_with:
  - jq '.explanation.owner, .explanation.dry_path' allow-explain.json
  - jq '{status, changed_files, failures}' allow-report.json
  - jq '{status, changed_files, failures}' block-report.json
```

## Connected smoke preflight

```yaml
id: helm_connected_smoke
command: cub auth login && ./examples/demo/run-connected-smoke.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  files_exist:
    - ".tmp/connected-smoke/<run>/helm-paas/summary.json"
  summary_fields:
    change_id: non_empty
    verify_valid: true
    attestation_valid: true
    linked_bundle_check: true
inspect_with:
  - jq '{change_id, verify_valid, attestation_valid, linked_bundle_check}' .tmp/connected-smoke/<run>/helm-paas/summary.json
```

## Deep connected walkthrough

```yaml
id: helm_connected_deep
command: OUTPUT_DIR=.tmp/helm-paas-connected ./examples/helm-paas/demo-connected.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  files_exist:
    - ".tmp/helm-paas-connected/create/summary.json"
    - ".tmp/helm-paas-connected/update/summary.json"
    - ".tmp/helm-paas-connected/create/decision-final.json"
    - ".tmp/helm-paas-connected/update/decision-final.json"
  decision_state: terminal_allow_escalate_or_block
inspect_with:
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/helm-paas-connected/create/summary.json
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/helm-paas-connected/update/summary.json
```

## Live runtime proof

```yaml
id: helm_live_runtime
command: RECONCILER=both ./examples/helm-paas/demo-runtime.sh
mutates:
  repo: false
  backend: true
  live: true
expects:
  live_checks:
    - "Flux and Argo report healthy reconciliation"
    - "kubectl get deploy,pods,svc shows payments-api objects"
  artifact_root: ".tmp/helm-paas-runtime"
inspect_with:
  - kubectl get deploy,pods,svc -n payments-platform
  - flux get helmreleases -A
  - kubectl get applications -A
```
