# Examples — Governed Config for Every Stack

You already have a deployment pipeline: Git, Helm, Flux, Argo, Spring Boot, Score, or internal workflows.

`cub-gen` adds the missing answers:

- who changed this field,
- what source produced this deployed value,
- who owns it,
- what file to edit safely.

Each example in this directory is runnable and maps to a real platform/app pattern.

## Pick a path in 30 seconds

Do not start by scanning the whole catalog. Start with the path that matches
what you already run:

| If you already run... | Start here | Then do this | What you can inspect | Current truth |
|---|---|---|---|---|
| Helm plus Argo/Flux platform repos | `./examples/demo/start-platform-first.sh` | `cub auth login && ./examples/helm-paas/demo-connected.sh` | values ownership, rendered targets, then paired runtime proof via [`live-reconcile`](./live-reconcile/) | Strongest platform-first source-side path today |
| Spring Boot app repos | `./examples/demo/start-app-first.sh` | `cub auth login && ./examples/springboot-paas/demo-connected.sh` | config ownership now, live `inventory-api` proof next | Strongest standalone end-to-end example in the repo today |
| Score.dev workloads | `./examples/demo/start-score-first.sh` | `cub auth login && ./examples/scoredev-paas/demo-connected.sh` | `score.yaml` field origin now, governed connected output next | Strongest Score source-side path today; standalone live Score proof is still open work |
| A running cluster and GitOps controller | ConfigHub GitOps import + [`cub-scout`](https://github.com/confighub/cub-scout) + then `cub-gen` | trace one chosen field back to source | live cluster state first, source provenance second | Best when the cluster is already the urgent source of truth |

## What ends with something live today

| Path | Live thing you can inspect | Command | Truth today |
|---|---|---|---|
| [`springboot-paas`](./springboot-paas/) | real `inventory-api` app on a kind cluster | `./examples/springboot-paas/verify-e2e.sh` | Standalone live app proof is real |
| [`helm-paas`](./helm-paas/) + [`live-reconcile`](./live-reconcile/) | Flux and Argo reconciliation of rendered Helm output | `RECONCILER=both ./examples/live-reconcile/demo-local.sh` | Runtime proof is paired, not standalone in `helm-paas` |
| [`scoredev-paas`](./scoredev-paas/) | connected governed output plus rendered runtime fields | `./examples/scoredev-paas/demo-connected.sh` | Standalone Score live proof is still open |

## Two audiences, one set of wrappers

Every flagship example supports the same two entry points:

- Local first: `./examples/<example>/demo-local.sh`
- Connected next: `cub auth login && ./examples/<example>/demo-connected.sh`

If you already use ConfigHub, start with the connected wrapper for one example.
If you already use Helm, Argo, Flux, Score, or Spring, start with local mode so
you see value before backend setup.

## What `--space` means

`--space` is the ConfigHub space where cub-gen records bundle metadata and
decision-related context.

- In local examples, `platform` is the default teaching value.
- In connected runs, use the space from your current ConfigHub context.
- If you only want source-side proof, you can treat `--space` as descriptive
  context rather than a deployment target.

## Source-first vs cluster-first

Use the `cub-gen` examples when your starting point is a source repo and your
question is "what rendered this field?" Use ConfigHub GitOps import plus
[`cub-scout`](https://github.com/confighub/cub-scout) when your starting point
is a live Argo or Flux workload and your question is "what is running right
now?"

That split is especially important for:

- Helm plus Argo/Flux teams using chart values and overlays,
- Score teams keeping `score.yaml` as the app-team contract,
- platform teams tracing one cluster field back to the DRY source that produced it.

## Flagship truth right now

| Example | Best current answer for | Current trust level |
|---|---|---|
| [`helm-paas`](./helm-paas/) | Helm + Argo/Flux + values ownership | Strongest platform-first source-side path; pair with [`live-reconcile`](./live-reconcile/) for runtime proof |
| [`springboot-paas`](./springboot-paas/) | Real app-team + platform-team config ownership | Strongest standalone end-to-end example in the repo today |
| [`scoredev-paas`](./scoredev-paas/) | Score intent to rendered runtime fields | Strongest Score source-side path today; runtime proof still needs its own standalone finish |

## What happens after import? (Day-2 stories)

Import is day 1. The real value shows on day 2:

| Day | What you do | What you gain |
|---|---|---|
| **Day 1** | Import and explain | Field-origin tracing, ownership clarity, inverse-edit guidance |
| **Day 2** | Governed change, promotion, or live-origin proposal | ALLOW/BLOCK decisions, cross-repo promotion, live→DRY proposals |
| **Day 3** | Optional AI-assisted change lane | Prompt-as-DRY, mutation-ledger evidence, verification boundary |

Every example should answer: "I imported my config. Now what?" The answer is
governed change, promotion, or live-origin proposal, not "wait for the next
feature."

## Current truth matrix

Use the generated [Example Truth Matrix](../docs/testing/example-truth-matrix.md) when you need an exact answer for:

- which examples are first-class generator fixtures,
- which ones are CI-proven through the full source-side `cub-gen` chain,
- which ones are in the connected smoke lane,
- which ones have real WET->LIVE proof today,
- which ones are explicitly AI-first versus only adjacent.

If you are about to spend real time on one example, check the matrix first.
It is the repo's source of truth for what is source-side only, what is connected,
and what has real live proof today.

For workflow and AI examples specifically, use the
[AI Example Hygiene Checklist](../docs/workflows/ai-example-hygiene-checklist.md)
as the repo's current bar for "what good looks like."

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

## How this differs from ConfigHub GitOps Import

If you have also seen ConfigHub's GitOps Import wizard or `cub gitops import`, the split is:

- ConfigHub GitOps import starts from ArgoCD/Flux resources already represented in a cluster.
- `cub-gen` starts from source repos and generator inputs such as `Chart.yaml`, `score.yaml`, `application.yaml`, or workflow files.

That means:

- use ConfigHub import for brownfield cluster/app discovery,
- use these `cub-gen` examples for source-side provenance, ownership routing, and governed edits.

If confidence scores are new to your team, use:
- [Confidence score guide](../docs/workflows/confidence-scores.md)

## Run modes

## Local mode (fastest, no login required)

```bash
go build -o ./cub-gen ./cmd/cub-gen

# Platform-first
./examples/helm-paas/demo-local.sh

# App-first
./examples/springboot-paas/demo-local.sh

# Score-first
./examples/scoredev-paas/demo-local.sh
```

## Connected mode (ConfigHub smoke first)

```bash
cub auth login
cub info

# Repo-wide smoke check for the current connected surface
./examples/demo/run-connected-smoke.sh

# Platform-first
./examples/helm-paas/demo-connected.sh

# App-first
./examples/springboot-paas/demo-connected.sh

# Score-first
./examples/scoredev-paas/demo-connected.sh
```

Use local mode for first value. Use connected smoke to confirm your ConfigHub
environment and the flagship wrappers. Use the deeper bridge/promotion scripts
only when you specifically need backend decision-query or promotion flows.

After that, use [`live-reconcile`](./live-reconcile/) for WET to LIVE proof and
[`cub-scout`](https://github.com/confighub/cub-scout) for cluster-side inspection.

## Use your own repo quickly

```bash
REPO=/path/to/your/repo
./cub-gen change preview --space platform "$REPO"
./cub-gen change run --mode local --space platform "$REPO"
./cub-gen change explain --space platform --owner app-team "$REPO"
```

Advanced connected decision path for the same repo:

```bash
cub auth login
BASE_URL="${CONFIGHUB_BASE_URL:-$(cub context get --json | jq -r '.coordinate.serverURL')}"
TOKEN="$(cub auth get-token)"
./cub-gen change run --mode connected --base-url "$BASE_URL" --token "$TOKEN" --space platform "$REPO"
```

Use that path when your backend exposes the bridge endpoints used by
`change run --mode connected`. For a simpler connected first run, prefer the
example wrappers and `./examples/demo/run-connected-smoke.sh`.

Deep connected full-entrypoint runner:

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
./cub-gen gitops import --space platform --json ./examples/ops-workflow \
  | jq '.provenance[0].ops_workflow_analysis'

# Swamp workflows: model/method/required-step structural governance
./examples/swamp-automation/demo-local.sh
./cub-gen gitops import --space platform --json ./examples/swamp-automation \
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

### AI prompt-as-DRY (first-class product lane)

For AI and workflow examples, the DRY/WET model extends to prompts and context:

| Concept | How it maps |
|---------|-------------|
| **Prompt + context** | DRY input (what the team authors) |
| **LLM/agent layer** | Non-deterministic generator (produces WET output) |
| **Verification + attestation** | Safety boundary (makes non-determinism governable) |
| **Mutation ledger** | Compliance and forensics proof |

This means human-authored changes and AI-assisted changes can run through the same
governed ConfigHub MR path. The mutation ledger is what makes this auditable.

See the [AI lane workflow](../docs/workflows/prompt-as-dry.md) for details.

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

## Add your own generator

If your platform has its own config format (not Helm, Score, Spring Boot, etc.),
you can add `cub-gen` support for it.

See [Custom Generator Onboarding](../docs/workflows/custom-generator-onboarding.md) for:

- When you need a custom generator
- Fork-and-extend path for internal platforms
- Request-inclusion path for community-relevant generators
- Kubara-like layered platform requirements

## How to read each example

Every example README satisfies the [Universal Example Contract](../docs/workflows/example-checklist.md):

| Section | What it tells you |
|---------|-------------------|
| 1. Who this is for | Two-audience paths (ConfigHub-first + tool-first) |
| 2. What runs | Real app/runtime, real cluster objects, real inspection target |
| 3. Why ConfigHub + cub-gen helps | One pain, one answer, one governed change win |
| 4. Run from ConfigHub | Connected mode instructions |
| 5. Run from platform tool | Local mode instructions |
| 6. Inspect the result | URL/pods/evidence artifacts |
| 7. Governed change | ALLOW + ESCALATE/BLOCK examples |
| 8. Generation chain | (Layered only) Multi-hop tracing |
| 9. Ownership boundary | (Layered only) Platform vs downstream guidance |

For full contract details, see the [Universal Example Contract](../docs/workflows/example-checklist.md).
