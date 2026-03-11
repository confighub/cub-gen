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
