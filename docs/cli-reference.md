# CLI Reference

All commands are deterministic: same input produces same output.

## Core flow

### `gitops discover`

Scan a repo path and classify generator roots.

```
cub-gen gitops discover --space <space> [--json] [--where-resource <expr>] <target-slug>
```

| Flag | Description |
|------|-------------|
| `--space` | Space label for discover state partitioning |
| `--json` | Emit JSON output (default: table) |
| `--where-resource` | Filter resources (`kind`, `name`, `root`, `id`, `LIKE`, `IN`, `AND`) |

### `gitops import`

Import DRY/WET classification with provenance and inverse-edit guidance.

```
cub-gen gitops import --space <space> [--json] [--wait] <target-slug> <render-target-slug>
```

| Flag | Description |
|------|-------------|
| `--space` | Space label (must match discover) |
| `--json` | Emit JSON output with full provenance |
| `--wait` | Accepted for CLI compatibility (no-op in local mode) |

Output includes: `generator_profile`, `dry_inputs`, `wet_manifest_targets`, `provenance` (field-origin map, inverse-edit pointers).

### `gitops cleanup`

Remove local discover state.

```
cub-gen gitops cleanup --space <space> <target-slug>
```

---

## Bridge artifacts

### `publish`

Generate a ConfigHub-ready change bundle from import output.

```
# Pipe mode (from import)
cub-gen gitops import ... | cub-gen publish --in - --out -

# Direct mode (import + bundle in one step)
cub-gen publish --space <space> <target-slug> <render-target-slug>
```

Output includes `digest_algorithm` (sha256) and `bundle_digest` for verification.

### `verify`

Verify bundle schema and digest integrity.

```
cub-gen verify --in <bundle.json>
cub-gen verify --in -          # stdin
```

Non-zero exit on integrity mismatch.

### `attest`

Emit an attestation record from a verified bundle.

```
cub-gen attest --in <bundle.json> --verifier <verifier-id>
cub-gen attest --in - --verifier ci-bot
```

### `verify-attestation`

Verify attestation integrity, optionally linked against a bundle.

```
cub-gen verify-attestation --in <attestation.json>
cub-gen verify-attestation --in <attestation.json> --bundle <bundle.json>
```

---

## Bridge flow (ConfigHub API path)

### `bridge ingest`

Submit a bundle to the ConfigHub bridge endpoint.

```
cub-gen bridge ingest --in <bundle.json> --base-url <url>
```

### `bridge decision`

Decision commands support two modes:

- Connected authoritative lookup via `query` against ConfigHub.
- Local/offline contract simulation via `create`, `attach`, and `apply`.

```
cub-gen bridge decision query --base-url <url> --change-id <id>
cub-gen bridge decision create --ingest <ingest-result.json>
cub-gen bridge decision attach --decision <decision.json> --attestation <attestation.json>
cub-gen bridge decision apply --decision <decision.json> --state ALLOW --approved-by <who> --reason <why>
```

### `bridge promote`

Promotion guardrail flow (app PR -> CH MR -> platform DRY PR).

```
cub-gen bridge promote init --change-id <id> --app-pr-repo <repo> --app-pr-number <n> ...
cub-gen bridge promote govern --flow <flow.json> --state ALLOW --decision-ref <ref>
cub-gen bridge promote verify --flow <flow.json>
cub-gen bridge promote open --flow <flow.json> --repo <repo> --number <n> --url <url>
cub-gen bridge promote approve --flow <flow.json> --by <who>
cub-gen bridge promote merge --flow <flow.json> --by <who>
```

---

## Generator catalog

### `generators`

List supported generator families from the registry.

```
cub-gen generators [--json] [--kind <kinds>] [--capability <caps>] [--profile <profiles>]
                   [--strict-filters] [--details] [--markdown]
```

| Flag | Description |
|------|-------------|
| `--json` | JSON output |
| `--kind` | Filter by kind(s), comma-separated |
| `--capability` | Filter by capability(s), comma-separated |
| `--profile` | Filter by profile(s), comma-separated |
| `--strict-filters` | Require all filters to match (AND logic) |
| `--details` | Include full family policy/provenance templates |
| `--markdown` | Emit markdown-formatted output |

Examples:

```bash
# List all generators
./cub-gen generators

# JSON with details for Helm
./cub-gen generators --json --details --kind helm

# Filter by capability
./cub-gen generators --capability render-manifests

# Markdown output for documentation
./cub-gen generators --markdown --details
```

---

## Parity status

Command contracts are **frozen** at `v0.2-preview-parity-locked` (2026-03-06). See [Command Parity](parity.md) for the full contract matrix.

| Status | Meaning |
|--------|---------|
| `matched` | Behavior intentionally mirrored from `cub gitops` |
| `partial` | Same contract shape, simplified implementation |
| `deferred` | Intentionally not implemented yet |

---

## Generator quickstart recipes

Build once, then try any generator:

```bash
go build -o ./cub-gen ./cmd/cub-gen
```

### Helm

```bash
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

### Score.dev

```bash
./cub-gen gitops discover --space platform ./examples/scoredev-paas
./cub-gen gitops import --space platform --json ./examples/scoredev-paas ./examples/scoredev-paas \
  | jq '{profile: .discovered[0].generator_profile, field_origin_map: .provenance[0].field_origin_map, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/scoredev-paas
```

### Spring Boot

```bash
./cub-gen gitops discover --space platform ./examples/springboot-paas
./cub-gen gitops import --space platform --json ./examples/springboot-paas ./examples/springboot-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/springboot-paas
```

### Backstage IDP

```bash
./cub-gen gitops discover --space platform ./examples/backstage-idp
./cub-gen gitops import --space platform --json ./examples/backstage-idp ./examples/backstage-idp \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/backstage-idp
```

### Ably app-config

```bash
./cub-gen gitops discover --space platform ./examples/just-apps-no-platform-config
./cub-gen gitops import --space platform --json ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/just-apps-no-platform-config
```

### Ops workflow

```bash
./cub-gen gitops discover --space platform ./examples/ops-workflow
./cub-gen gitops import --space platform --json ./examples/ops-workflow ./examples/ops-workflow \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/ops-workflow
```

### C3 Agent

```bash
./cub-gen gitops discover --space platform ./examples/c3agent
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets_count: (.wet_manifest_targets|length), inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/c3agent
```

### Swamp automation

```bash
./cub-gen gitops discover --space platform ./examples/swamp-automation
./cub-gen gitops import --space platform --json ./examples/swamp-automation ./examples/swamp-automation \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/swamp-automation
```

---

## Bridge artifact examples

### Publish + verify (pipe mode)

```bash
./cub-gen gitops import --space platform --json ./examples/helm-paas ./examples/helm-paas \
  | ./cub-gen publish --in - --out - \
  | jq '{schema_version, source, change_id, summary}'
```

### Publish + verify + attest (file mode)

```bash
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json
./cub-gen verify-attestation --in attestation.json --bundle bundle.json
```

### Bridge flow (ConfigHub API path)

```bash
# 1) Build bundle and attestation artifacts
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas > bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json

# 2) Ingest to ConfigHub bridge endpoint
./cub-gen bridge ingest --in bundle.json --base-url https://confighub.example > ingest-result.json

# 3) Build decision state, attach attestation, apply explicit decision
./cub-gen bridge decision create --ingest ingest-result.json > decision.json
./cub-gen bridge decision attach --decision decision.json --attestation attestation.json > decision-attested.json
./cub-gen bridge decision apply --decision decision-attested.json --state ALLOW \
  --approved-by platform-owner --reason "policy checks passed" > decision-allow.json

# 4) Query decision state by change_id
./cub-gen bridge decision query --base-url https://confighub.example \
  --change-id "$(jq -r .change_id decision-allow.json)"
```

### Promotion guardrail flow

```bash
./cub-gen bridge promote init --change-id chg_123 \
  --app-pr-repo github.com/confighub/apps --app-pr-number 42 \
  --app-pr-url https://github.com/confighub/apps/pull/42 \
  --mr-id mr_123 --mr-url https://confighub.example/mr/123 > flow.json
./cub-gen bridge promote govern --flow flow.json --state ALLOW --decision-ref decision_123 > flow-allow.json
./cub-gen bridge promote verify --flow flow-allow.json > flow-verified.json
./cub-gen bridge promote open --flow flow-verified.json \
  --repo github.com/confighub/platform-dry --number 7 \
  --url https://github.com/confighub/platform-dry/pull/7 > flow-open.json
./cub-gen bridge promote approve --flow flow-open.json --by platform-owner > flow-approved.json
./cub-gen bridge promote merge --flow flow-approved.json --by platform-owner > flow-promoted.json
```

### Direct publish for all generators

```bash
./cub-gen publish --space platform ./examples/helm-paas ./examples/helm-paas
./cub-gen publish --space platform ./examples/scoredev-paas ./examples/scoredev-paas
./cub-gen publish --space platform ./examples/springboot-paas ./examples/springboot-paas
./cub-gen publish --space platform ./examples/backstage-idp ./examples/backstage-idp
./cub-gen publish --space platform ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config
./cub-gen publish --space platform ./examples/ops-workflow ./examples/ops-workflow
./cub-gen publish --space platform ./examples/c3agent ./examples/c3agent
./cub-gen publish --space platform ./examples/swamp-automation ./examples/swamp-automation
```

---

## Triple expression styles

The repo includes three full style projections for every generator kind:

1. **Style A** YAML: `docs/triple-styles/style-a-yaml/*.yaml`
2. **Style B** Markdown: `docs/triple-styles/style-b-markdown/*.md`
3. **Style C** YAML+Markdown pair: `docs/triple-styles/style-c-yaml-plus-docs/<kind>/`

Index: `docs/triple-styles/README.md`

Regenerate style projections:

```bash
make sync-triple-styles
# or
go run ./cmd/cub-gen-style-sync
```
