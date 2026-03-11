# Operations Workflow — User Stories

## 1. Ops team bumps the deploy image tag

The operations team needs to roll out v1.3.0 of the checkout service. They update `operations.yaml` with the new image tag. `cub-gen` traces the change: field-origin shows `actions.deploy.image_tag` in the DRY operations file, owned by the ops team. The change bundle captures the version bump with full provenance.

## 2. Schedule change requires policy check

The team moves the nightly maintenance from 2 AM to 3 AM in `operations-prod.yaml`. The platform's execution policy confirms the new time falls within the allowed production maintenance window (00:00-06:00 UTC). Decision: ALLOW. If they'd requested 10 AM, the decision engine would ESCALATE.

## 3. New action type needs platform approval

The team wants to add a `scale` action to the workflow. The platform's execution policy lists `scale` as an allowed action — no escalation needed. If they'd tried to add a `destroy` action, the policy would block it with a clear reason.

## 4. Audit across all maintenance workflows

Compliance asks: "which services have maintenance workflows targeting production?" ConfigHub's provenance index answers this instantly — every ops-workflow unit with `triggers.schedule` set traces back to the team that authored it, the approval that allowed it, and the last time the schedule was changed.

## 5. IDP portal renders operation forms from registry

Platform engineering publishes `platform/registry.yaml` with typed operations such as `scheduleWorkflow` and `deployServiceAction`. An internal portal can fetch the schema directly and render safe forms for ops teams without asking them to hand-edit every YAML path.
