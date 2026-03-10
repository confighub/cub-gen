# Examples — Governed Config for Every Stack

You already have a deployment pipeline. Git, Helm, Flux, Argo, Spring Boot, Score —
whatever you use to get code running in production. That pipeline works.

What it can't answer: *who changed this field, what generator produced it,
why was it allowed, and what would break if I change it back?*

**cub-gen** is a local CLI that scans your existing repo, classifies every config
file by owner and purpose, and produces a traceable evidence chain — without
changing your workflow.

Each example in this directory is a complete, runnable scenario for a specific
technology stack. Pick the one closest to your world and try it.

## What is cub-gen?

`cub-gen` is a deterministic, Git-native generator importer. It reads your
existing config files (Helm charts, Spring Boot properties, Score workloads,
AI agent fleet configs, operational workflows) and produces:

1. **Generator detection** — recognizes what kind of project you have
2. **DRY/WET classification** — separates human-authored intent (DRY) from
   rendered deployment artifacts (WET)
3. **Field-origin tracing** — maps every deployed field back to its source file,
   line, and owner
4. **Inverse-edit guidance** — tells you *where to edit* to change a deployed value
5. **Evidence bundles** — publish, verify, and attest changes for governance

It runs locally. No server required. No vendor lock-in at the CLI layer.
When you're ready for cross-repo queries, policy evaluation, and governed
decisions, connect to [ConfigHub](https://confighub.com).

## How DRY → WET → LIVE works

```
 YOU EDIT (DRY)              cub-gen TRACES (WET)           RECONCILER DEPLOYS (LIVE)
┌──────────────┐           ┌───────────────────┐          ┌──────────────────┐
│ values.yaml  │──detect──▶│ Deployment.yaml   │──Flux───▶│ Running pod      │
│ score.yaml   │  import   │ Service.yaml      │  Argo    │ Live config      │
│ app.yaml     │  publish  │ ConfigMap.yaml     │          │ Cluster state    │
│ c3agent.yaml │           │ HelmRelease.yaml  │          │                  │
└──────────────┘           └───────────────────┘          └──────────────────┘
       │                          │                              │
   Your files.               What generators               What's actually
   You own these.            produce from them.             running.
```

- **DRY** = the files you edit in Git. Human-authored intent.
- **WET** = rendered manifests that generators produce from DRY. Machine-readable,
  queryable, governable.
- **LIVE** = what's actually running in your cluster. Reconciled by Flux/Argo.

cub-gen works at the DRY→WET boundary. It doesn't replace your reconciler — it
makes the link between "what you wrote" and "what got deployed" explicit and
traceable.

## Verifier identity

When you run `cub-gen attest --verifier <name>`, the verifier name identifies
*who or what* vouches for the evidence bundle. Common verifier identities:

| Verifier | When to use |
|----------|-------------|
| `ci-bot` | CI pipeline attestation (GitHub Actions, GitLab CI, Jenkins) |
| `platform-lead` | Human platform engineer sign-off |
| `security-review` | Security team review attestation |
| `deploy-agent` | Automated deployment agent |

The verifier is a string label — not a cryptographic identity (yet). It records
*intent*: "this actor reviewed and approved this evidence bundle." Future versions
will support signature-based verification.

## Pick your starting point

### Platform + App patterns (Kubernetes workloads)

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**helm-paas**](helm-paas/) | Helm charts + values overlays | Chart contract tracing, values ownership, ALLOW/BLOCK governance |
| [**scoredev-paas**](scoredev-paas/) | Score.dev workload specs | Platform-agnostic DRY with full field-origin mapping |
| [**springboot-paas**](springboot-paas/) | Spring Boot + application.yaml | Framework config ownership, datasource vs app-config boundaries |

### Integration patterns (external services + developer portals)

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**backstage-idp**](backstage-idp/) | Backstage software catalog | Catalog entity governance with ownership standards |
| [**just-apps-no-platform-config**](just-apps-no-platform-config/) | SaaS provider config (Ably, etc.) | App-only config without platform layer — simplest possible example |

### AI + automation patterns

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**c3agent**](c3agent/) | AI agent fleets (Claude, GPT) | Fleet config governance, model policy, token budgets |
| [**ai-ops-paas**](ai-ops-paas/) | Full AI platform with registry + constraints | Enterprise AI fleet with constraint enforcement |
| [**swamp-automation**](swamp-automation/) | Swamp AI workflow orchestration | DAG workflows with model binding governance |
| [**swamp-project**](swamp-project/) | Helm chart for AI runtime | Helm-based Swamp deployment with model gateway policy |

### Operations patterns

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**ops-workflow**](ops-workflow/) | Scheduled maintenance + rollouts | Execution policy, approval gates, deploy windows |
| [**confighub-actions**](confighub-actions/) | ConfigHub lifecycle automation | Recursive governance — ConfigHub governing itself |

### Infrastructure

| Example | Purpose |
|---------|---------|
| [**live-reconcile**](live-reconcile/) | Flux e2e test fixture — proves WET→LIVE reconciliation |
| [**demo**](demo/) | Runnable demo scripts for all examples |

## Quick start

```bash
# Build once
go build -o ./cub-gen ./cmd/cub-gen

# Pick any example — here's helm-paas
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas

# Full evidence chain (works with any example)
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json
./cub-gen verify-attestation --in attestation.json --bundle bundle.json

# Bridge to ConfigHub (governance decisions)
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest.json
./cub-gen bridge decision create --ingest ingest.json > decision.json
./cub-gen bridge decision apply --decision decision.json --state ALLOW \
  --approved-by platform-lead --reason "reviewed and approved"
```

## Next steps

- **Run the E2E demo**: `./examples/demo/run-all-modules.sh`
- **Try the AI work platform demos**: `./examples/demo/ai-work-platform/run-all.sh`
- **Read the design docs**: `docs/agentic-gitops/`
- **Contribute a new generator**: see [CONTRIBUTING.md](../CONTRIBUTING.md)
