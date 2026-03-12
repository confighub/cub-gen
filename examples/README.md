# Examples — Governed Config for Every Stack

You already have a deployment pipeline: Git, Helm, Flux, Argo, Spring Boot, Score, or internal workflows.

`cub-gen` adds the missing answers:

- who changed this field,
- what source produced this deployed value,
- who owns it,
- what file to edit safely.

Each example in this directory is runnable and maps to a real platform/app pattern.

**Who this is for:** Platform engineers, SREs, and app developers who want config
traceability without changing their existing deployment workflow.

**What runs locally vs what needs ConfigHub:** Everything in these examples runs
locally with no backend. Generator detection, field-origin tracing, and evidence
bundles work offline. Cross-repo queries, policy enforcement, and governed decisions
require [ConfigHub](https://confighub.github.io/cub-gen/platform/).

## Pick your domain POV first

Start with the example that matches how your team already thinks:

| If you are... | Start here | First value you should see |
|---|---|---|
| Spring Boot platform or app lead | [`springboot-paas`](./springboot-paas/) | "Which Spring property changed, who owns it, and what file do I edit?" |
| Helm/Flux/Argo platform team (umbrella charts, overlays) | [`helm-paas`](./helm-paas/) | Ownership + field trace map without chart archaeology |
| Score.dev platform team | [`scoredev-paas`](./scoredev-paas/) | Visibility from `score.yaml` intent to rendered runtime fields |
| Ops/SRE workflow owner | [`ops-workflow`](./ops-workflow/) | Governed schedule/action changes with explicit ALLOW/BLOCK outcomes |
| AI workflow / Swamp-style team | [`swamp-automation`](./swamp-automation/) | Structural workflow-change classification and policy-ready evidence |
| AI fleet platform owner | [`c3agent`](./c3agent/) or [`ai-ops-paas`](./ai-ops-paas/) | Model/budget/credential governance over fleet config changes |
| Backstage catalog owner | [`backstage-idp`](./backstage-idp/) | Catalog ownership/lifecycle changes become traceable and reviewable |
| Reconciler reliability owner | [`live-reconcile`](./live-reconcile/) | Real Flux+Argo create/update/drift-correction proof harness |

If you are unsure, start with `helm-paas` (platform POV) or `springboot-paas` (app-team POV).

Deeper persona framing (from domain feedback): [Domain POV Matrix](../docs/workflows/domain-pov-matrix.md)

## What `cub-gen` does

`cub-gen` is a deterministic, Git-native generator importer. It reads existing config and emits:

1. generator detection (what type of project this is),
2. DRY/WET classification (authoring intent vs rendered targets),
3. field-origin tracing (WET field -> DRY source path),
4. inverse-edit guidance (where to edit),
5. evidence bundles (`publish`, `verify`, `attest`).

If confidence scores are new to your team, use:
- [Confidence score guide](../docs/workflows/confidence-scores.md)

## Run modes

## Local mode (fastest, no login required)

```bash
go build -o ./cub-gen ./cmd/cub-gen
./examples/demo/run-all-modules.sh
./examples/demo/run-all-confighub-lifecycles.sh
```

## Connected mode (ConfigHub)

```bash
cub auth login
TOKEN="$(cub auth get-token)"
cub context get --json | jq -r '.coordinate.user'
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"

./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN" > /tmp/ingest.json
```

Use local mode for first value. Use connected mode for centralized governance state and cross-repo visibility.

## Use your own repo quickly

```bash
REPO=/path/to/your/repo
./cub-gen change preview --space platform "$REPO" "$REPO"
./cub-gen change run --mode local --space platform "$REPO" "$REPO"
./cub-gen change explain --space platform --owner app-team "$REPO" "$REPO"
```

Connected run against the same repo:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"
./cub-gen change run --mode connected --base-url "$BASE_URL" --token "$TOKEN" --space platform "$REPO" "$REPO"
```

Connected full-entrypoint runner:

```bash
cub auth login
./examples/demo/run-all-connected-entrypoints.sh
```

Per-example wrappers:

- Local: `./examples/<example>/demo-local.sh`
- Connected: `./examples/<example>/demo-connected.sh` (starts with `cub auth login`)

## Verifier identity

When you run `cub-gen attest --verifier <name>`, the verifier name records who/what attested the bundle.

| Verifier | When to use |
|----------|-------------|
| `ci-bot` | CI pipeline attestation |
| `platform-lead` | Human platform sign-off |
| `security-review` | Security approval |
| `deploy-agent` | Automated deployment agent |

## Pick your starting point

## Choose your starting view

| If you are... | Start here | Direct viewpoint section | First command |
|---------|-----------|--------------------------|---------------|
| Helm / umbrella-chart platform team | [helm-paas](helm-paas/) | [If you already run Helm heavily](helm-paas/README.md#if-you-already-run-helm-heavily) | `./examples/helm-paas/demo-local.sh` |
| Spring Boot platform/app lead | [springboot-paas](springboot-paas/) | [If you already ship Spring Boot services](springboot-paas/README.md#if-you-already-ship-spring-boot-services) | `./examples/springboot-paas/demo-local.sh` |
| Score.dev platform team | [scoredev-paas](scoredev-paas/) | [If you already use Score.dev in production](scoredev-paas/README.md#if-you-already-use-scoredev-in-production) | `./examples/scoredev-paas/demo-local.sh` |
| Backstage/IDP owner | [backstage-idp](backstage-idp/) | [If you already run Backstage catalogs at scale](backstage-idp/README.md#if-you-already-run-backstage-catalogs-at-scale) | `./examples/backstage-idp/demo-local.sh` |
| Ops workflow/SRE automation team | [ops-workflow](ops-workflow/) | [If you already run operational workflows at scale](ops-workflow/README.md#if-you-already-run-operational-workflows-at-scale) | `./examples/ops-workflow/demo-local.sh` |
| AI agent fleet platform team | [c3agent](c3agent/) | [If you already run agent fleets operationally](c3agent/README.md#if-you-already-run-agent-fleets-operationally) | `./examples/c3agent/demo-local.sh` |
| Full AI PaaS builder | [ai-ops-paas](ai-ops-paas/) | [If you already run AI/ops platforms on Kubernetes](ai-ops-paas/README.md#if-you-already-run-aiops-platforms-on-kubernetes) | `./examples/ai-ops-paas/demo-local.sh` |
| Workflow automation platform team | [swamp-automation](swamp-automation/) | [If you already build workflow automation systems](swamp-automation/README.md#if-you-already-build-workflow-automation-systems) | `./examples/swamp-automation/demo-local.sh` |
| Helm-based AI runtime team | [swamp-project](swamp-project/) | [If you already operate Helm-based AI runtimes](swamp-project/README.md#if-you-already-operate-helm-based-ai-runtimes) | `./examples/swamp-project/demo-local.sh` |
| Reconciler/platform reliability engineer | [live-reconcile](live-reconcile/) | [If you already operate Flux/Argo at scale](live-reconcile/README.md#if-you-already-operate-fluxargo-at-scale) | `RECONCILER=both ./examples/live-reconcile/demo-local.sh` |

If you want copy/paste 5-minute paths per persona, use:

- [Persona 5-minute runbooks](../docs/workflows/persona-5-minute-runbooks.md)

## Workflow-first quick path (Ops + Swamp)

If your users mostly run workflows (not app manifests), start with these two:

```bash
# Ops workflows: actions/schedules/approval-gates as governed config
./examples/ops-workflow/demo-local.sh
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '.provenance[0].ops_workflow_analysis'

# Swamp workflows: model/method/required-step structural governance
./examples/swamp-automation/demo-local.sh
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '.provenance[0].swamp_workflow_analysis'
```

Operation-registry walkthrough (AI Ops + Ops Workflow + Swamp):
- [docs/workflows/operation-registry-real-apps.md](../docs/workflows/operation-registry-real-apps.md)
Now includes Helm + Spring Boot registry-backed platform examples too.

## Platform + app patterns (Kubernetes workloads)

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**helm-paas**](helm-paas/) | Helm charts + values overlays | Chart contract tracing, values ownership, ALLOW/BLOCK governance |
| [**scoredev-paas**](scoredev-paas/) | Score.dev workload specs | DRY intent with field-origin mapping |
| [**springboot-paas**](springboot-paas/) | Spring Boot + `application.yaml` | App/platform ownership boundaries |

## Integration patterns (external services + developer portals)

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**backstage-idp**](backstage-idp/) | Backstage software catalog | Catalog governance with ownership standards |
| [**just-apps-no-platform-config**](just-apps-no-platform-config/) | SaaS provider config | App-only config governance without platform layer |

## AI + automation patterns

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**c3agent**](c3agent/) | AI agent fleets | Fleet config governance, model policy, budget controls |
| [**ai-ops-paas**](ai-ops-paas/) | Full AI platform + constraints | Registry + constraints + governed lifecycle |
| [**swamp-automation**](swamp-automation/) | Swamp agent-authored workflows | Workflow-graph change governance (models/methods/required steps) |
| [**swamp-project**](swamp-project/) | Helm chart for AI runtime | Helm-based runtime policy mapping |

## Operations patterns

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**ops-workflow**](ops-workflow/) | Scheduled maintenance workflows | Structural workflow governance (actions/schedules/approval gates) |
| [**confighub-actions**](confighub-actions/) | ConfigHub lifecycle automation | Recursive governance (ConfigHub governing itself) |

## Infrastructure

| Example | Purpose |
|---------|---------|
| [**live-reconcile**](live-reconcile/) | Flux + Argo e2e fixtures proving WET->LIVE reconciliation |
| [**demo**](demo/) | Runnable demo script index |

## How to read each example

Every example README should answer:

1. What scenario it models.
2. Who owns which fields.
3. How to run local mode.
4. How to run connected mode.
5. What proof artifacts to inspect.

For acceptance criteria across examples, see [Example Checklist](../docs/workflows/example-checklist.md).
