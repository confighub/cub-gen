# Persona 5-Minute Runbooks

Use this page when you already know a platform/app pattern and want to test how it maps into `cub-gen` quickly.

Each runbook has two paths:

- `Local mode`: offline, no login required.
- `Connected mode`: real ConfigHub round-trip, starts with `cub auth login`.

Build once before local runs:

```bash
go build -o ./cub-gen ./cmd/cub-gen
```

## 1. Helm Platform Team (Umbrella Charts, Overlays, Flux/Argo)

Goal: prove you can trace rendered Kubernetes fields back to Helm values and ownership boundaries.

Local mode:

```bash
./examples/helm-paas/demo-local.sh
```

Connected mode:

```bash
cub auth login
./examples/helm-paas/demo-connected.sh
```

Why this maps:

- Your teams already use `Chart.yaml`, `values.yaml`, and overlays.
- `cub-gen` maps these files to WET outputs and shows inverse-edit guidance for real drift/debug work.

What you should see:

- Helm generator detection.
- Field-origin map from WET field to values path.
- App-team vs platform-team ownership hints.

## 2. Spring Boot Shop (application.yaml First)

Goal: keep Spring developers in Spring config while platform teams keep Kubernetes governance.

Local mode:

```bash
./examples/springboot-paas/demo-local.sh
```

Connected mode:

```bash
cub auth login
./examples/springboot-paas/demo-connected.sh
```

Why this maps:

- App teams keep editing `application.yaml` and profile overlays.
- Platform teams still get ownership, policy, and attested decision flow before deploy.

What you should see:

- Spring generator detection.
- Ownership split for app-owned and platform-owned property families.
- Clear inverse-edit hints for production overrides.

## 3. Score.dev Platform Team

Goal: prove Score intent can stay DRY while generated platform output remains governable.

Local mode:

```bash
./examples/scoredev-paas/demo-local.sh
```

Connected mode:

```bash
cub auth login
./examples/scoredev-paas/demo-connected.sh
```

Why this maps:

- Developers author `score.yaml` intent.
- Platform teams need traceability from rendered fields back to Score-level intent and policy boundaries.

What you should see:

- Score generator detection.
- Field-origin mappings for key Deployment fields.
- Connected ingest and decision-query artifacts for governance state.

## 4. AI Agent Platform Team (c3agent / AI Ops PaaS)

Goal: prove short fleet config can safely produce multi-resource Kubernetes output.

Local mode:

```bash
./examples/c3agent/demo-local.sh
./examples/ai-ops-paas/demo-local.sh
```

Connected mode:

```bash
cub auth login
./examples/c3agent/demo-connected.sh
./examples/ai-ops-paas/demo-connected.sh
```

Why this maps:

- AI teams define fleet/model/budget at intent level.
- Platform teams need governance for credentials, storage, RBAC, and replica policy before LIVE reconcile.

What you should see:

- Multi-target import coverage (Deployments, Services, ConfigMaps, Secret, PVC, RBAC resources).
- Policy decision state attached to each connected run.

## 5. Ops Workflow Team (Runbooks, Maintenance, Automation)

Goal: prove workflow configs can be governed like apps, with explicit edit paths and decision state.

Local mode:

```bash
./examples/ops-workflow/demo-local.sh
./examples/swamp-automation/demo-local.sh
```

Connected mode:

```bash
cub auth login
./examples/ops-workflow/demo-connected.sh
./examples/swamp-automation/demo-connected.sh
```

Why this maps:

- Ops teams already encode workflow intent in YAML/automation configs.
- `cub-gen` gives ownership boundaries and governed promotion signals before runtime actions.

What you should see:

- Workflow-specific detection with field-origin and inverse-edit output.
- Connected decision-query artifacts tied to change IDs.

## 6. Reconciler Reliability Engineer (Flux and Argo LIVE Proof)

Goal: prove governed WET can reconcile and self-correct in real controllers.

Local mode:

```bash
./examples/live-reconcile/demo-local.sh
```

Connected mode:

```bash
cub auth login
./examples/live-reconcile/demo-connected.sh
```

Why this maps:

- Your controllers stay Flux/Argo.
- `cub-gen` and ConfigHub add pre-reconcile governance and post-change evidence.

What you should see:

- Create, update, and drift-correction across both Flux and Argo paths.
- Same DRY-WET governance model feeding either reconciler.

## Fast Full-Coverage Paths

Run all local lifecycle fixtures:

```bash
./examples/demo/run-all-confighub-lifecycles.sh
```

Run all connected lifecycle fixtures:

```bash
cub auth login
./examples/demo/run-all-connected-lifecycles.sh
```
