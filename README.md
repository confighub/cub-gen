# cub-gen

`cub-gen` is the local-first entry point to the [ConfigHub](https://github.com/confighubai/confighub) platform — a deterministic DRY → WET generator importer with command-shape parity to `cub gitops`.

## Start here (intro + demos)

If you are new, use this path first:

1. Plain-English platform story:
   [`docs/workflows/build-your-own-heroku-in-a-weekend.md`](docs/workflows/build-your-own-heroku-in-a-weekend.md)
2. Example catalog and narratives:
   [`examples/README.md`](examples/README.md)
3. Demo track index:
   [`examples/demo/README.md`](examples/demo/README.md)

Fast demo entry points:

- Core module walkthrough (all modules): `./examples/demo/run-all-modules.sh`
- AI work platform scenarios: `./examples/demo/ai-work-platform/run-all.sh`
- AI Ops PaaS narrative demo: `./examples/ai-ops-paas/demo.sh`
- Full lifecycle matrix (create -> govern -> update): `./examples/demo/run-all-confighub-lifecycles.sh`
- Live Flux reconciliation proof (kind + Flux): `./examples/demo/e2e-live-reconcile-flux.sh`

## What cub-gen does today

- Detects generator-style app sources in Git repos (`helm`, `score.dev`, `springboot`, `backstage`, `ably-config`, `ops-workflow`, `c3agent`, `swamp`).
- Runs the same staged flow shape as `cub gitops`:
  - `gitops discover`
  - `gitops import`
  - `gitops cleanup`
- Emits provenance and inverse-edit guidance ("what rendered field came from which DRY field").
- Works standalone (no backend required) and produces [ConfigHub-ready change bundles](https://confighub.github.io/cub-gen/platform/) for governed execution when connected.
- Exposes supported generator families via `cub-gen generators`.

## Documentation - FIXME **

Full documentation is published at **https://confighub.github.io/cub-gen/**

- [Getting Started](https://confighub.github.io/cub-gen/getting-started/) — build and run your first import in 10 minutes
- [The ConfigHub Platform](https://confighub.github.io/cub-gen/platform/) — how cub-gen connects to ConfigHub, bridge workers, and Flux/ArgoCD
- [Architecture](https://confighub.github.io/cub-gen/architecture/) — DRY/WET model, field-origin maps, governed execution
- [Generator Reference](https://confighub.github.io/cub-gen/generators/) — contract triples for all 8 generator kinds
- [Architecture](https://confighub.github.io/cub-gen/agentic-gitops/02-design/00-agentic-gitops-design/) — DRY/WET model, field-origin maps, governed execution
- [Generator Reference](https://confighub.github.io/cub-gen/triple-styles/) — contract triples for all 8 generator kinds
- [CLI Reference](https://confighub.github.io/cub-gen/cli-reference/) — full command and flag documentation
- [Contributing](https://confighub.github.io/cub-gen/contributing-guide/) — proof-first delivery, test-backed PRs

## Part of the ConfigHub platform

`cub-gen` is the local-first on-ramp. The full stack:

1. **DRY** app intent lives in Git (`Chart.yaml`, `score.yaml`, `application.yaml`, etc.)
2. **cub-gen** classifies DRY inputs + WET targets and emits provenance with field-origin tracing
3. **cub-gen publish** produces change bundles with SHA-256 digest verification
4. **[ConfigHub](https://github.com/confighubai/confighub)** ingests bundles, enforces governed decision state (ALLOW | ESCALATE | BLOCK), manages units with revision history
5. **Bridge workers** connect ConfigHub to clusters via HTTP/2 SSE (Kubernetes, Flux, ArgoCD)
6. **Flux/Argo** continue to reconcile WET → LIVE — unchanged

Teams can start with `cub-gen` locally today and connect to ConfigHub when they need cross-repo queries, policy at write time, and governed execution. See the [platform docs](https://confighub.github.io/cub-gen/platform/) for the full story.

## One platform, different workload adapters

`cub-gen` models one platform pattern that supports many workload types.

It is not "one PaaS for Spring Boot" and a different one for Helm or AI agents.
It is one governance layer with workload adapters:

- Spring Boot adapter reads `application.yaml` conventions.
- Helm adapter reads `Chart.yaml` + `values.yaml`.
- score.dev adapter reads `score.yaml`.
- c3agent adapter reads `c3agent.yaml`.

Same flow every time:

1. Team pushes workload config to Git.
2. `cub-gen` discovers/imports and emits provenance + inverse edit guidance.
3. Flux/Argo reconciles rendered manifests.

### Who does what

- Platform team: defines adapters, defaults, and guardrails once.
- App team: writes app config and code (the files they already use).

### Which comes first in practice

Most orgs start with existing repos and drifted manifests.

1. Path 1 (most common): import what already exists to get visibility and governance first.
2. Path 2 (next step): standardize clean self-service contracts after visibility is in place.

For a plain-English narrative and team responsibilities, see
[`docs/workflows/build-your-own-heroku-in-a-weekend.md`](docs/workflows/build-your-own-heroku-in-a-weekend.md).

## Jump-in demo modules

Run any module independently:

```bash
./examples/demo/module-1-helm-import.sh
./examples/demo/module-2-score-field-map.sh
./examples/demo/module-3-spring-ownership.sh
./examples/demo/module-4-bridge-governance.sh
./examples/demo/module-5-ably-platform.sh
```

Or run all modules in one pass:

```bash
./examples/demo/run-all-modules.sh
```

Example repo narratives and ownership maps are documented in `/Users/alexis/Public/github-repos/cub-gen/examples/README.md`.

## Wizard Simulation (Repo-First)

Simulate a future GUI discover/import wizard against any repo fixture:

```bash
./examples/demo/simulate-repo-wizard.sh ./examples/helm-paas ./examples/helm-paas auto
./examples/demo/simulate-repo-wizard.sh ./examples/springboot-paas ./examples/springboot-paas springboot-paas
```

This script walks the same step sequence planned for the GUI:
1. source selection
2. discover preview
3. import graph preview (DRY -> GEN -> WET)
4. provenance/inverse hint preview
5. import confirmation + bundle/verify/attest summary

## ConfigHub Lifecycle Demo (Create -> Deploy -> Update)

Run one example through the full governance lifecycle with surface views:

```bash
./examples/demo/simulate-confighub-lifecycle.sh ./examples/c3agent ./examples/c3agent c3agent
```

Run all current platform examples (10 fixtures):

```bash
./examples/demo/run-all-confighub-lifecycles.sh
```

Each run shows:

1. Create path (`discover -> import -> publish -> verify -> attest`)
2. Decision/promotion path (`bridge decision` + `bridge promote`)
3. Update path (source config mutation + re-run chain)
4. Visibility surfaces:
   - OCI (bundle digest + output URIs)
   - Flux fixtures (`gitops/flux/*` when present)
   - Argo fixtures (`gitops/argo/*` when present)
   - cub-scout watchlist (derived from wet targets)

Optional full live reconciliation proof (kind + Flux):

```bash
./examples/demo/e2e-live-reconcile-flux.sh
```

This script creates a local cluster, installs Flux controllers, and proves:
create reconciliation, update reconciliation, and drift correction against LIVE resources.

## Complete Inventory And Coverage Check

App demos:

- `helm-paas` (`./examples/demo/module-1-helm-import.sh`)
- `scoredev-paas` (`./examples/demo/module-2-score-field-map.sh`)
- `springboot-paas` (`./examples/demo/module-3-spring-ownership.sh`)
- `ably-config` (`./examples/demo/module-5-ably-platform.sh`)
- `backstage-idp` (included in `./examples/demo/run-all-confighub-lifecycles.sh`)

Platform demos:

- Governance bridge path (`./examples/demo/module-4-bridge-governance.sh`)
- AI work platform track (`./examples/demo/ai-work-platform/run-all.sh`)
- AI Ops PaaS narrative demo (`./examples/ai-ops-paas/demo.sh`)
- Full create -> governance/deploy path -> update matrix (`./examples/demo/run-all-confighub-lifecycles.sh`)

Other things users may want to see:

- Repo-first wizard simulation (`./examples/demo/simulate-repo-wizard.sh`)
- Core module aggregator (`./examples/demo/run-all-modules.sh`)
- Live reconciler e2e (`./examples/demo/e2e-live-reconcile-flux.sh`)
- Demo index and track entrypoints (`./examples/demo/README.md`)
- Story-card matrix (`docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md`)

Qualification caveat:
Without live `WET -> LIVE` reconciler evidence, classify the flow as `governed config automation`, not full `Agentic GitOps` (see `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md` and `docs/agentic-gitops/02-design/10-generators-prd.md`). Running `./examples/demo/e2e-live-reconcile-flux.sh` provides that evidence for the Flux path.

Have we solved all user stories?

| Status | User stories | Notes |
|---|---|---|
| Met/strong in current demos | 2, 3, 4, 5, 6, 13 | Proven by current local-first examples and lifecycle flows. |
| Partial (simulated/local-first, not full backend/runtime integration) | 1, 7, 9, 12 | Command shape and evidence model are present; backend/runtime coupling is still simulated. |
| Deferred | 8, 10, 11 | Requires additional platform features and runtime-connected workflows. |

## AI Work Platform Demo Track

Second demo track focused on AI-work platform scenarios:

```bash
./examples/demo/ai-work-platform/run-all.sh
```

Or run scenarios individually:

```bash
./examples/demo/ai-work-platform/scenario-1-c3agent.sh
./examples/demo/ai-work-platform/scenario-2-swamp.sh
./examples/demo/ai-work-platform/scenario-3-confighub-actions.sh
./examples/demo/ai-work-platform/scenario-4-operations.sh
```

## 10-minute adoption path (Flux/Argo/Helm)

Start with a Helm-based repo and keep your existing runtime model intact.

What stays unchanged:

- Flux/Argo remains the reconciler for WET -> LIVE.
- Git/OCI remains the transport path.
- Existing cluster/controller permissions and PR workflow stay in place.

What you add:

- `cub-gen gitops discover` to classify generator roots.
- `cub-gen gitops import` to emit DRY/WET contracts + provenance/inverse pointers.
- `cub-gen gitops cleanup` to clear local discover state.

Copy/paste this path:

```bash
go build -o cub-gen ./cmd/cub-gen
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, provenance: .provenance[0] | {chart_path, values_paths, rendered_object_lineage}}'
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

Boundary language (aligned with `PARITY.md`):

- `matched`: `gitops discover|import|cleanup` command shape and output contracts.
- `matched`: bridge artifacts (`publish`, `verify`, `attest`, `verify-attestation`) symmetric across all 8 generators.
- `partial`: local state/artifacts stand in for server-side units during this phase.
- `partial`: bridge flow commands (`ingest`, `decision`, `promote`) produce correct contract shapes; [ConfigHub backend integration](https://confighub.github.io/cub-gen/platform/) is the next step.

## Terminology (locked for v0.1)

| Term | Meaning in cub-gen |
|---|---|
| DRY source | Human-editable app/platform intent (`values.yaml`, `score.yaml`, `application.yaml`) |
| WET rendered units | Explicit rendered deployment-facing units/manifests |
| Provenance | Record of DRY inputs, rendered outputs, field-origin map, inverse-edit pointers |
| Inverse map | Guidance from changed WET field -> where to edit DRY safely |
| Pre-sync | `cub-gen` stops before WET->LIVE; Flux/Argo own reconciliation |

## Full quickstart examples (copy/paste)

```bash
go build ./cmd/cub-gen
```

### Helm example

```bash
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

### score.dev example

```bash
./cub-gen gitops discover --space platform ./examples/scoredev-paas
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas | jq '{profile: .discovered[0].generator_profile, field_origin_map: .provenance[0].field_origin_map, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/scoredev-paas
```

### Spring Boot example

```bash
./cub-gen gitops discover --space platform ./examples/springboot-paas
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/springboot-paas
```

### Backstage IDP example

```bash
./cub-gen gitops discover --space platform ./examples/backstage-idp
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/backstage-idp
```

### Ably app-config example

```bash
./cub-gen gitops discover --space platform ./examples/ably-config
./cub-gen gitops import --space platform --json ./examples/ably-config ./examples/ably-config | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/ably-config
```

### Ops workflow example

```bash
./cub-gen gitops discover --space platform ./examples/ops-workflow
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/ops-workflow
```

### C3 Agent example

```bash
./cub-gen gitops discover --space platform ./examples/c3agent
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets_count: (.wet_manifest_targets|length), inverse_patches_count: (.inverse_transform_plans[0].patches|length), inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/c3agent
```

### Swamp automation example

```bash
./cub-gen gitops discover --space platform ./examples/swamp-automation
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/swamp-automation
```

### Optional bridge artifact (local, no backend)

Generate a ConfigHub-ready change bundle from import output:

```bash
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | ./cub-gen publish --in - --out - \
  | jq '{schema_version,source,change_id,summary}'
```

This emits a deterministic `change-bundle` JSON envelope you can upload later,
without coupling the core flow to a running ConfigHub backend.

### List supported generator families

```bash
./cub-gen generators
./cub-gen generators --json | jq '.families[] | {kind, profile, resource_kind}'
./cub-gen generators --kind helm
./cub-gen generators --kind helm,score
./cub-gen generators --capability render-manifests
./cub-gen generators --capability inverse-values-patch,inverse-score-patch
./cub-gen generators --strict-filters --kind helm,score
./cub-gen generators --json --details | jq '.families[] | {kind, profile, policies}'
./cub-gen generators --markdown
./cub-gen generators --markdown --details
```

`--details` exposes full family policy/provenance templates, including:
`inverse_patch_templates`, `inverse_pointer_templates`,
`field_origin_confidences`, `hint_defaults`, `inverse_patch_reasons`,
`inverse_edit_hints`, `input_role_rules`, `default_input_role`,
`role_owners`, `default_owner`, `wet_targets`, `rendered_lineage_templates`,
`field_origin_transform`, and `field_origin_overlay_transform`.

### Compare triple expression styles (all 8 generators)

The repo also includes three full style projections for every generator kind:

1. Style A YAML: `/docs/triple-styles/style-a-yaml/*.yaml`
2. Style B Markdown: `/docs/triple-styles/style-b-markdown/*.md`
3. Style C YAML+Markdown pair: `/docs/triple-styles/style-c-yaml-plus-docs/<kind>/`

Index:

1. `/docs/triple-styles/README.md`

Regenerate these style projections:

```bash
make sync-triple-styles
# or
go run ./cmd/cub-gen-style-sync
```

Or run direct mode (import + bundle in one command):

```bash
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp
./cub-gen publish --space platform ./examples/ably-config ./examples/ably-config
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow
./cub-gen publish --space platform ./examples/c3agent ./examples/c3agent
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation
```

Bundle output includes:

- `digest_algorithm` (currently `sha256`)
- `bundle_digest` (deterministic digest over bundle content excluding digest fields)

This gives you a simple verification handle for attestation pipelines.

Verify a bundle (file or stdin):

```bash
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas | ./cub-gen verify --in -
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas | ./cub-gen verify --in -
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas | ./cub-gen verify --in -
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp | ./cub-gen verify --in -
./cub-gen publish --space platform ./examples/ably-config ./examples/ably-config | ./cub-gen verify --in -
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow | ./cub-gen verify --in -
./cub-gen publish --space platform ./examples/c3agent ./examples/c3agent | ./cub-gen verify --in -
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation | ./cub-gen verify --in -
```

Emit an attestation record from a verified bundle:

```bash
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
./cub-gen publish --space platform ./examples/ably-config ./examples/ably-config \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
./cub-gen publish --space platform ./examples/c3agent ./examples/c3agent \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation \
  | ./cub-gen attest --in - --verifier ci-bot \
  | jq '{schema_version,status,verifier,bundle_digest,attestation_digest}'
```

Verify an attestation (optionally linked against a bundle file):

```bash
./cub-gen verify-attestation --in attestation.json --bundle bundle.json
```

### Bridge flow quickstart (ConfigHub API path)

Generate bundle + attestation, then run bridge flow commands:

```bash
# 1) Build bundle and attestation artifacts
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 2) Ingest to ConfigHub bridge endpoint
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest-result.json

# 3) Build decision state, attach attestation, apply explicit decision
./cub-gen bridge decision create --ingest ingest-result.json > decision.json
./cub-gen bridge decision attach --decision decision.json --attestation attestation.json > decision-attested.json
./cub-gen bridge decision apply --decision decision-attested.json --state ALLOW --approved-by platform-owner --reason "policy checks passed" > decision-allow.json

# 4) Query decision state by change_id from API
./cub-gen bridge decision query --base-url https://confighub.example --change-id "$(jq -r .change_id decision-allow.json)"
```

Promotion guardrail flow (app PR -> CH MR -> platform DRY PR):

```bash
./cub-gen bridge promote init --change-id chg_123 --app-pr-repo github.com/confighub/apps --app-pr-number 42 --app-pr-url https://github.com/confighub/apps/pull/42 --mr-id mr_123 --mr-url https://confighub.example/mr/123 > flow.json
./cub-gen bridge promote govern --flow flow.json --state ALLOW --decision-ref decision_123 > flow-allow.json
./cub-gen bridge promote verify --flow flow-allow.json > flow-verified.json
./cub-gen bridge promote open --flow flow-verified.json --repo github.com/confighub/platform-dry --number 7 --url https://github.com/confighub/platform-dry/pull/7 > flow-open.json
./cub-gen bridge promote approve --flow flow-open.json --by platform-owner > flow-approved.json
./cub-gen bridge promote merge --flow flow-approved.json --by platform-owner > flow-promoted.json
```

## Plain-English collaboration story

A practical app-team/platform-team path in a Spring Boot repo:

1. App team changes `server.port` in `application-prod.yaml` for a feature rollout.
2. Platform team runs `cub-gen gitops import --json` and sees:
   - app-owned DRY inputs (`app-config-base`, `app-config-profile`)
   - platform-owned WET targets (`wet_manifest_targets.owner = platform-runtime`)
   - inverse pointers showing app-editable fields (`spring.application.name`, `server.port`) vs platform-governed field (`spring.datasource.url`)
3. Flux/Argo reconciliation path stays unchanged for deployment.
4. Teams keep app velocity while preserving governance boundaries.

## Contract highlights by example

### score.dev (MVP-01)

- `generator_profile: "scoredev-paas"`
- provenance `field_origin_map`
- provenance `inverse_edit_pointers`

### Helm (MVP-02)

- top-level `dry_inputs` and `wet_manifest_targets`
- provenance `chart_path`, `values_paths`, `rendered_object_lineage`

### Spring Boot (MVP-03)

- `dry_inputs.owner` separates app-team vs platform-engineer edit ownership
- `wet_manifest_targets.owner` marks platform runtime ownership
- inverse-edit paths include app-team edits (`spring.application.name`, `server.port`) and platform-governed edits (`spring.datasource.url`)

### Backstage IDP (v0.2 preview)

- `generator_profile: "backstage-idp"`
- dry input ownership split (`catalog-spec` vs `app-config`)
- inverse-edit paths for component metadata (`metadata.name`, `spec.lifecycle`)

### Ably app-config (v0.2 preview)

- `generator_profile: "ably-config"`
- app-team DRY ownership (`provider-config-base`, `provider-config-overlay`)
- inverse-edit paths for app runtime provider config (`app.environment`, `channels.inbound`)

### Ops workflow (v0.2 preview)

- `generator_profile: "ops-workflow"`
- platform-engineer DRY ownership (`operations-base`, `operations-overlay`)
- inverse-edit paths for workflow execution intent (`actions.deploy.image_tag`, `triggers.schedule`)

### C3 Agent (v0.2 preview)

- `generator_profile: "c3agent"`
- app-team DRY ownership (`fleet-config-base`, `fleet-config-overlay`)
- manifest-set metadata expansion to 11 WET targets (Deployments, Services, RBAC, PVC, ConfigMap, Secret)
- inverse-edit coverage expanded to runtime/storage/replicas/rbac in addition to fleet and credentials

### Swamp automation (v0.2 preview)

- `generator_profile: "swamp"`
- app-team DRY ownership (`swamp-config-base`, `swamp-workflow`)
- inverse-edit paths for workflow automation (`workflow_definition`, `vault_config`, `model_binding`)

## Quality model (inherited from cub-scout, adapted)

- Deterministic behavior: same input => same output
- Contract parity tests for CLI outputs (JSON + table goldens)
- Proof-first delivery: define test matrix before implementation
- Example-backed validation for user-visible behavior

See:

- `CLAUDE.md`
- `CONTRIBUTING.md`
- `docs/contracts/canonical-triple-and-storage-boundary.md`
- `docs/contracts/decision-and-attestation-state.md`
- `docs/contracts/pr-mr-linkage-and-dry-promotion.md`
- `docs/testing/README.md`
- `docs/workflows/proof-first-delivery.md`
- `PARITY.md`

## Test

```bash
go test ./...
go test ./cmd/cub-gen -run '^(TestGitOpsParity|TestPublishGolden|TestVerifyGolden|TestAttestGolden|TestVerifyAttestationGolden|TestTopLevelCommand)' -count=1 -v
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
# or via make:
make test-contracts
make test-examples
```
