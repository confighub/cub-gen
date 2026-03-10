# C3 Agent Fleet — User Stories

## 1. Team provisions a new agent fleet

A QA team wants to run automated test generation agents. They author `c3agent.yaml` with their fleet config — model, concurrency, budget limits. `cub-gen` detects the c3agent source, maps it to 11 WET Kubernetes targets, and traces every field back to the DRY config. The platform's fleet policy confirms the model is approved and the budget is within ceiling.

## 2. Production budget increase

The code review team needs a higher per-task budget for complex PRs. They update `c3agent-prod.yaml` with `agent_runtime.max_budget_usd: 25.0`. The field-origin map shows this as a prod overlay, owned by the app team. The platform's budget ceiling confirms $25 is within the $25 per-task limit.

## 3. Model upgrade rollout

Anthropic releases a new model. The platform team adds it to the approved models list. Teams update their fleet configs one by one. ConfigHub's cross-repo query shows the rollout progress: "3 of 5 fleets upgraded."

## 4. Credential rotation audit

Security requires all API key references to rotate every 90 days. The platform's credential hygiene policy flags fleets where `credentials.anthropic_key_ref` hasn't changed in over 90 days. Each rotation goes through the governed pipeline with provenance linking the old and new credential references.
