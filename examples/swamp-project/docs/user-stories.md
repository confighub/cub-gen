# Swamp Project — User Stories

## 1. Project team switches model gateway

A project team needs to switch from `llama-gateway` to `mistral-gateway` for better performance. They update `values-prod.yaml`. `cub-gen` traces the change: field-origin shows `runtime.modelGateway` overridden in prod overlay. The platform's allowlist confirms `mistral-gateway` is approved.

## 2. Platform bumps chart version

The platform team releases a new chart version with improved health checks. They update `Chart.yaml` version. All project teams using the chart get the update on next Flux reconciliation. ConfigHub records the version bump as a platform-owned change with full provenance.

## 3. Scaling for production load

The project team increases `replicaCount` in `values-prod.yaml` from 4 to 8 for anticipated load. The platform's runtime policy requires minimum 3 replicas for production — the change passes. The inverse-edit pointer confirms: "to change replica count, edit `values-prod.yaml` replicaCount."

## 4. Cost audit across runtimes

Finance asks: "how many Swamp runtime replicas are running across all environments?" ConfigHub's cross-repo query surfaces every swamp-project unit with its `replicaCount` value, traced back to the values files and the teams that authored them.
