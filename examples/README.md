# Examples — Governed Config for Every Stack

You already have a deployment pipeline: Git, Helm, Flux, Argo, Spring Boot, Score, or internal workflows.

`cub-gen` adds the missing answers:

- who changed this field,
- what source produced this deployed value,
- who owns it,
- what file to edit safely.

Each example in this directory is runnable and maps to a real platform/app pattern.

## What `cub-gen` does

`cub-gen` is a deterministic, Git-native generator importer. It reads existing config and emits:

1. generator detection (what type of project this is),
2. DRY/WET classification (authoring intent vs rendered targets),
3. field-origin tracing (WET field -> DRY source path),
4. inverse-edit guidance (where to edit),
5. evidence bundles (`publish`, `verify`, `attest`).

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
BASE_URL="https://confighub.example"

./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > /tmp/bundle.json
./cub-gen verify --in /tmp/bundle.json
./cub-gen attest --in /tmp/bundle.json --verifier ci-bot > /tmp/attestation.json
./cub-gen bridge ingest --in /tmp/bundle.json --base-url "$BASE_URL" --token "$TOKEN" > /tmp/ingest.json
```

Use local mode for first value. Use connected mode for centralized governance state and cross-repo visibility.

## Verifier identity

When you run `cub-gen attest --verifier <name>`, the verifier name records who/what attested the bundle.

| Verifier | When to use |
|----------|-------------|
| `ci-bot` | CI pipeline attestation |
| `platform-lead` | Human platform sign-off |
| `security-review` | Security approval |
| `deploy-agent` | Automated deployment agent |

## Pick your starting point

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
| [**swamp-automation**](swamp-automation/) | Swamp workflow orchestration | DAG/model binding governance |
| [**swamp-project**](swamp-project/) | Helm chart for AI runtime | Helm-based runtime policy mapping |

## Operations patterns

| Example | You use... | cub-gen shows you... |
|---------|-----------|---------------------|
| [**ops-workflow**](ops-workflow/) | Scheduled maintenance workflows | Approval and execution policy mapping |
| [**confighub-actions**](confighub-actions/) | ConfigHub lifecycle automation | Recursive governance (ConfigHub governing itself) |

## Infrastructure

| Example | Purpose |
|---------|---------|
| [**live-reconcile**](live-reconcile/) | Flux e2e fixture proving WET->LIVE reconciliation |
| [**demo**](demo/) | Runnable demo script index |

## How to read each example

Every example README should answer:

1. What scenario it models.
2. Who owns which fields.
3. How to run local mode.
4. How to run connected mode.
5. What proof artifacts to inspect.

For acceptance criteria across examples, see [Example Checklist](../docs/workflows/example-checklist.md).
