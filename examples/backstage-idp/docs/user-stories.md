# Backstage IDP — User Stories

## 1. New service registration

A team creates a new microservice and adds `catalog-info.yaml` to register it in Backstage. `cub-gen` detects the catalog entity, classifies it as app DRY, and traces the owner field back to the team. The platform's catalog standard confirms all required fields are present.

## 2. Lifecycle stage promotion

A service moves from `experimental` to `production` lifecycle. The team edits `catalog-info.yaml` spec.lifecycle. The change bundle captures the lifecycle transition with full provenance. ConfigHub records this as a governance-significant event — production services have different policy requirements.

## 3. Ownership transfer audit

During a team reorg, 15 services need to change owners. ConfigHub's cross-repo query shows all services affected. Each ownership change goes through the governed pipeline with explicit ALLOW decisions, creating a complete audit trail of the transfer.

## 4. Platform enforces catalog standards

A new team tries to register a service without a lifecycle field. The platform's catalog standard policy flags the missing field. The decision engine returns ESCALATE — the team must add the lifecycle field before the catalog change can be approved.
