# cub-gen

`cub-gen` is a local-first prototype for deterministic DRY -> WET generator import, modeled on `cub gitops` command flow.

## What cub-gen does today

- Detects generator-style app sources in Git repos (`helm`, `score.dev`, `springboot`, `backstage`, `ably-config`, `ops-workflow`).
- Runs the same staged flow shape as `cub gitops`:
  - `gitops discover`
  - `gitops import`
  - `gitops cleanup`
- Emits provenance and inverse-edit guidance ("what rendered field came from which DRY field").
- Stays local and pre-sync in v0.1 and v0.2 preview (no cluster deploys, no ConfigHub backend required).
- Exposes supported generator families via `cub-gen generators`.

## Where Flux/Argo/OCI fit

`cub-gen` is the import/provenance step, not the reconciler.

1. DRY app intent lives in Git (`Chart.yaml`, `score.yaml`, `application.yaml`, etc.).
2. `cub-gen gitops import` classifies DRY inputs + WET targets and emits provenance/inverse map data.
3. WET artifacts move through Git/OCI transport.
4. Flux/Argo continue to reconcile WET -> LIVE.

This means teams can add `cub-gen` to existing Flux/Argo repos today without changing runtime controllers.

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
- `partial`: local state/artifacts stand in for server-side units during this phase.
- `deferred`: ConfigHub API bridge coupling and runtime reconcile execution.

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
```

Or run direct mode (import + bundle in one command):

```bash
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp
./cub-gen publish --space platform ./examples/ably-config ./examples/ably-config
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow
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
```

Verify an attestation (optionally linked against a bundle file):

```bash
./cub-gen verify-attestation --in attestation.json --bundle bundle.json
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
