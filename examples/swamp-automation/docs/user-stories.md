# Swamp Automation — User Stories

## 1. Team adds a workflow step

The SRE team wants to add a health check before deployment. They edit `workflow-deploy.yaml`, adding a new step that calls the `app-healthcheck` model. `cub-gen` detects the change, traces the new step back to the DRY workflow file, and produces a change bundle. The platform's workflow policy confirms `app-healthcheck` is in the approved models list.

## 2. Platform enforces execution windows

The platform team updates `platform/workflow-policy.yaml` to restrict production deployments to business hours. When the SRE team next publishes a workflow change targeting production, the decision engine checks the execution window policy. Workflows scheduled outside the window get an ESCALATE decision requiring platform-owner approval.

## 3. Vault credential rotation

Security policy requires vault encryption keys to rotate every 90 days. The platform team updates `.swamp.yaml` with a new key reference. `cub-gen` traces the change through the DRY/WET boundary — the vault config is platform-owned DRY, so the change bundle shows platform-team ownership. The attestation record links the rotation to the CI verification that ran.

## 4. Cross-repo workflow audit

Compliance asks: "which workflows across all teams reference the `app-deployer` model?" ConfigHub's provenance index answers this from governed decision history — every workflow that imports `app-deployer` has a provenance record linking the model reference back to the DRY workflow file, the team that authored it, and the decision that approved it.
