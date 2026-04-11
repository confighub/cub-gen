# Contracts: `swamp-automation`

Treat these as stable inspection contracts for human operators and AI
assistants. If any `expects` check fails, stop and report the mismatch.

## Repo preview

```yaml
id: swamp_repo_preview
command: ./cub-gen gitops import --space platform --json ./examples/swamp-automation
mutates:
  repo: false
  backend: false
  live: false
expects:
  generator_profile: swamp
  dry_inputs_min: 1
  wet_manifest_targets_min: 1
  inverse_edit_pointers_min: 1
inspect_with:
  - jq '.discovered[0].generator_profile'
  - jq '.dry_inputs | length'
  - jq '.wet_manifest_targets | length'
  - jq '.provenance[0].inverse_edit_pointers[0]'
```

## AI-safe local prompt-as-DRY loop

```yaml
id: swamp_prompt_as_dry_local
command: OUT_DIR=.tmp/swamp-prompt-local ./examples/demo/prompt-as-dry-local.sh ./examples/swamp-automation
mutates:
  repo: false
  backend: false
  live: false
expects:
  files_exist:
    - ".tmp/swamp-prompt-local/import.json"
    - ".tmp/swamp-prompt-local/bundle.json"
    - ".tmp/swamp-prompt-local/attestation.json"
    - ".tmp/swamp-prompt-local/mutation-card.json"
  mutation_card:
    change_id: non_empty
    verification.bundle_valid: true
    verification.attestation_valid: true
inspect_with:
  - jq '{change, edit_recommendation, verification}' .tmp/swamp-prompt-local/mutation-card.json
```

## Local structural walkthrough

```yaml
id: swamp_local_structural_walkthrough
command: ./examples/swamp-automation/demo-local.sh
mutates:
  repo: false
  backend: false
  live: false
expects:
  stdout_contains:
    - "[phase:create]"
    - "[phase:update]"
    - "\"attested_valid\": true"
  note: "this script uses a temporary directory that is cleaned automatically"
inspect_with:
  - stdout
```

## Connected preflight

```yaml
id: swamp_connected_preflight
command: cub auth login && ./examples/demo/run-connected-smoke.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  shared_smoke_examples:
    - helm-paas
    - springboot-paas
  purpose: "confirm auth and backend health before the workflow-specific connected lanes"
inspect_with:
  - jq '{change_id, verify_valid, attestation_valid}' .tmp/connected-smoke/<run>/helm-paas/summary.json
  - jq '{change_id, verify_valid, attestation_valid}' .tmp/connected-smoke/<run>/springboot-paas/summary.json
```

## AI-safe connected prompt-as-DRY loop

```yaml
id: swamp_prompt_as_dry_connected
command: OUTPUT_DIR=.tmp/swamp-prompt-connected ./examples/demo/prompt-as-dry-connected.sh ./examples/swamp-automation
mutates:
  repo: false
  backend: true
  live: false
expects:
  files_exist:
    - ".tmp/swamp-prompt-connected/create/summary.json"
    - ".tmp/swamp-prompt-connected/update/summary.json"
  decision_state: terminal_allow_escalate_or_block
inspect_with:
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/swamp-prompt-connected/create/summary.json
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/swamp-prompt-connected/update/summary.json
```

## Deep connected walkthrough

```yaml
id: swamp_connected_deep
command: OUTPUT_DIR=.tmp/swamp-connected ./examples/swamp-automation/demo-connected.sh
mutates:
  repo: false
  backend: true
  live: false
expects:
  files_exist:
    - ".tmp/swamp-connected/create/summary.json"
    - ".tmp/swamp-connected/update/summary.json"
  decision_state: terminal_allow_escalate_or_block
  note: "runtime execution artifacts remain outside the in-repo proof for this example"
inspect_with:
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/swamp-connected/create/summary.json
  - jq '{phase, change_id, decision_state, attestation_valid}' .tmp/swamp-connected/update/summary.json
```
