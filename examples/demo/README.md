# Demo entrypoints

## Core platform/app track

- `module-1-helm-import.sh`
- `module-2-score-field-map.sh`
- `module-3-spring-ownership.sh`
- `module-4-bridge-governance.sh`
- `module-5-ably-platform.sh`
- `run-all-modules.sh`

## GUI-wizard simulation

- `simulate-repo-wizard.sh <repo-path> <render-target-path> [profile-hint]`

## ConfigHub lifecycle simulation

- `simulate-confighub-lifecycle.sh <repo-path> <render-target-path> [example-slug]`
- `run-all-confighub-lifecycles.sh`

This flow runs for each example:

1. Create/import path (`discover -> import -> publish -> verify -> attest`)
2. Decision + promote path (`bridge decision` + `bridge promote`)
3. Update source config and re-run governance chain
4. Surface summaries for:
   - OCI bundle output URIs
   - Flux fixture files (if present)
   - Argo fixture files (if present)
   - cub-scout watchlist (from wet targets)

## AI work platform track

- `ai-work-platform/scenario-1-c3agent.sh` (11-target c3agent metadata coverage)
- `ai-work-platform/scenario-2-swamp.sh`
- `ai-work-platform/scenario-3-confighub-actions.sh`
- `ai-work-platform/scenario-4-operations.sh`
- `ai-work-platform/run-all.sh`
