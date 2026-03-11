#!/usr/bin/env bash
set -euo pipefail

replace_once() {
  local file="$1"
  local from="$2"
  local to="$3"

  if [ ! -f "$file" ]; then
    echo "error: update target missing: $file" >&2
    return 1
  fi
  if ! grep -Fq "$from" "$file"; then
    echo "error: update pattern not found in $file: $from" >&2
    return 1
  fi

  FROM="$from" TO="$to" perl -0777 -i -pe 's/\Q$ENV{FROM}\E/$ENV{TO}/' "$file"
}

write_file() {
  local file="$1"
  local content="$2"
  mkdir -p "$(dirname "$file")"
  printf '%s\n' "$content" > "$file"
}

apply_update() {
  local slug="$1"
  local repo="$2"

  case "$slug" in
    helm-paas|helm)
      write_file "$repo/values-canary.yaml" "replicaCount: 3
image:
  tag: v1.0.9-canary
appConfig:
  featureFlags:
    fastCheckout: true"
      ;;
    scoredev-paas|score)
      replace_once "$repo/score.yaml" "LOG_LEVEL: info" "APP_LOG_LEVEL: info"
      ;;
    springboot-paas|springboot)
      write_file "$repo/src/main/resources/application-canary.yaml" "server:
  port: 8082
feature:
  inventory:
    reservationMode: relaxed"
      ;;
    backstage-idp|backstage)
      mv "$repo/catalog-info.yaml" "$repo/catalog-info.yml"
      ;;
    just-apps-no-platform-config|no-config-platform)
      write_file "$repo/no-config-platform-canary.yaml" "app:
  environment: canary
channels:
  inbound: checkout.inbound.canary"
      ;;
    ops-workflow|opsworkflow)
      write_file "$repo/operations-canary.yaml" "triggers:
  schedule: \"*/30 * * * *\"
actions:
  deploy:
    image_tag: v1.2.9-canary"
      ;;
    c3agent)
      write_file "$repo/c3agent-canary.yaml" "fleet:
  agent_model: claude-opus-4-20250514
  max_concurrent_tasks: 1
agent_runtime:
  max_budget_usd: 2.0"
      ;;
    ai-ops-paas|aiops)
      replace_once "$repo/c3agent.yaml" "max_concurrent_tasks: 3" "max_concurrent_tasks: 4"
      replace_once "$repo/c3agent.yaml" "max_budget_usd: 8.0" "max_budget_usd: 12.0"
      ;;
    swamp-automation|swamp)
      write_file "$repo/workflow-canary.yaml" "id: canary-rollout
name: canary-rollout
jobs:
  - name: canary-validate
    steps:
      - name: validate
        task:
          type: model_method
          modelIdOrName: app-validator
          methodName: canaryCheck"
      ;;
    swamp-project|swampproject)
      write_file "$repo/values-canary.yaml" "replicaCount: 3
image:
  tag: v0.1.1-canary
runtime:
  modelGateway: llama-gateway-canary"
      ;;
    confighub-actions)
      write_file "$repo/operations-canary.yaml" "workflow:
  name: confighub-actions-canary
actions:
  verify:
    approvals:
      required: 1
  deploy:
    window: off-hours"
      ;;
    *)
      echo "error: no update recipe for example slug: $slug" >&2
      return 1
      ;;
  esac
}
