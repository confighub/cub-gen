# Persona Value Playbook (User-Story Audience)

This playbook maximizes visibility of the model and practical value for the three v0.1 user-story audiences:

1. Platform engineer
2. App team
3. GitOps team

It is intentionally core-first: Helm, score.dev, and Spring Boot.

## Boundary (keep explicit)

`cub-gen` is import/provenance tooling, not a reconciler or deploy executor.

1. Flux/Argo remains WET -> LIVE reconciler.
2. `cub-gen` does not deploy to clusters.
3. Output here demonstrates deterministic metadata, lineage, and governance artifacts.

## Fast path (single command)

Run the bundled persona script:

```bash
./examples/demo/persona-value-bundles.sh
```

Optional output directory:

```bash
./examples/demo/persona-value-bundles.sh ./examples/demo/output/persona-value
```

## Command bundles by persona

### 1) Platform engineer: "Can I see and trust the model?"

```bash
go build -o ./cub-gen ./cmd/cub-gen
./cub-gen generators --markdown --details
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, rendered_lineage: .provenance[0].rendered_object_lineage}'
```

What value is visible:

1. Generator contract surface in readable form.
2. DRY/WET mapping, ownership, and lineage templates.
3. Deterministic, inspectable JSON contract.

### 2) App team: "If a field changes, where do I edit DRY?"

```bash
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '{profile: .discovered[0].generator_profile, field_origin_map: .provenance[0].field_origin_map, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas \
  | jq '{profile: .discovered[0].generator_profile, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
```

What value is visible:

1. Exact DRY ownership/edit hints for generated fields.
2. App vs platform boundary shown as data, not tribal knowledge.

### 3) GitOps team: "Does this preserve our runtime model?"

```bash
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > /tmp/helm-bundle.json
./cub-gen verify --in /tmp/helm-bundle.json
./cub-gen attest --in /tmp/helm-bundle.json --verifier ci-bot
```

What value is visible:

1. Existing Flux/Argo flow remains untouched.
2. Deterministic artifact/verification/attestation chain.
3. Governance evidence exists without adding runtime reconciliation logic to `cub-gen`.

## Proof gate for claims

Before presenting results, run:

```bash
make ci
```

## Core-first rule

Use this playbook as the default visibility path while core generators are priority:

1. Helm
2. score.dev
3. Spring Boot

c3agent and other families can be demonstrated after core parity/docs are green.
