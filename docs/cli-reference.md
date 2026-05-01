# CLI Reference

All commands are deterministic: same input produces same output.

`cub-gen` starts from a repo, shows what it renders, and traces each rendered
field back to the file you should edit. It is for repo-side GitOps questions,
not cluster/runtime ones.

## Start By Question

| If you want to know... | Start with |
|---|---|
| What does this repo render, and where did it come from? | `gitops import` |
| What changed between two rendered git refs? | `change diff` |
| What changed between two release refs, and which DRY edits caused it? | `change revision-diff` |
| Which DRY file/path should I edit for a rendered field? | `change explain` |
| Which rendered fields would this DRY path affect? | `change impact` |
| What evidence bundle and safe next step should I prepare? | `change preview` |
| How do I add provenance to a PR without editing manifests? | `enrich preview`, then optionally `enrich write` |
| How do I turn implicit platform rules into reviewable config proposals? | `normalize preview` |
| How do I see a multi-repo platform estate? | `platform import` |
| How do I emit one proof bundle per environment or tenant? | `platform fanout` |
| What evidence bundle should I verify or ship? | `publish -> verify -> attest` |
| How do I use the deeper ConfigHub API flow? | `bridge` |

## Boundaries

- `cub-gen` reads repos and produces provenance/evidence.
- `cub-gen` does not deploy and does not read live cluster state.
- Use `cub-scout` for runtime/cluster questions.
- Use `cub gitops` for cluster-side import and ConfigHub management.

## Path model

`cub-gen` uses the `target` / `render-target` names from the wider ConfigHub
GitOps vocabulary, but in day-to-day local use they are usually just repo
paths.

- `<target-path>`: the repo path to inspect and classify
- `<render-target-path>`: optional local render target path; if omitted, `cub-gen` reuses `<target-path>`
- For cluster-side import against a real render target, use `cub gitops`, not `cub-gen`

## Platform Import

### `platform import`

Read a local manifest for a multi-repo platform estate and emit one stable
Component/Deployable Variant graph.

```
cub-gen platform import --json <manifest>
```

The manifest names app, platform, environment, and rendered repos. The command
imports each existing local repo read-only and emits:

- `repos`: manifest entries with owner, role, component, variant, and target metadata
- `components` and `variants`: the small product model users should see first
- `generators`: detected generator families per repo
- `dry_inputs` and `wet_targets`: source inputs and rendered targets from each generator
- `connections`: repo -> generator -> input/target and component -> variant -> target links
- `diagnostics`: explicit gaps such as `missing_repo`, `missing_owner`, or `unsupported_generator`

Example:

```
cub-gen platform import --json ./testdata/platform-estate/platform.yaml
```

This command does not rewrite repos, deploy, or contact ConfigHub. Use it before
adoption reports, enrichment, fanout, or rewrite proposals when the first job is
to understand the estate shape.

### `platform fanout`

Read the same platform manifest and emit one ConfigHub-ready change bundle per
Deployable Variant.

```
cub-gen platform fanout --json <manifest>
cub-gen platform fanout --variant <name-or-component/variant> --json <manifest>
```

Use this when one Component has concrete deployable copies such as dev, stage,
prod, tenant-a, tenant-b, or per-region targets. The manifest declares which
repo/input set owns each variant. The command emits:

| Output | Meaning |
|---|---|
| `variants[].variant_id` | Stable identity such as `checkout-api/prod` |
| `variants[].shared_inputs` | Base inputs shared by the Component |
| `variants[].variant_inputs` | Variant-specific values, files, or invocation overrides |
| `variants[].change_id` | Stable change identity for that variant bundle |
| `variants[].bundle` | Standard `change-bundle/v1`, accepted by publish/bridge proof flows |

Example:

```
cub-gen platform fanout --json ./testdata/variant-fanout/platform.yaml \
  | jq '.summary'

cub-gen platform fanout --variant dev --json ./testdata/variant-fanout/platform.yaml \
  | jq '.variants[] | {variant_id, change_id, profiles: .generator_profiles}'
```

Ambiguous repo metadata is rejected. If two repos claim the same
`component/variant`, declare the `variants:` list explicitly instead of relying
on inference.

### `gitops discover`

Scan a repo path and classify generator roots.

```
cub-gen gitops discover --space <space> [--json] [--where-resource <expr>] <target-path>
```

| Flag | Description |
|------|-------------|
| `--space` | Space label for discover state partitioning |
| `--json` | Emit JSON output (default: table) |
| `--where-resource` | Filter resources (`kind`, `name`, `root`, `id`, `LIKE`, `IN`, `AND`) |

If a repo declares a generator chain, `gitops discover` now reports that chain
directly as a compact pipeline summary such as `score -> helm`, instead of
making you infer it from separate stage detections.

### `gitops import`

Import DRY/WET classification with provenance and inverse-edit guidance.

```
cub-gen gitops import --space <space> [--json] [--wait] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] <target-path> [<render-target-path>]
```

| Flag | Description |
|------|-------------|
| `--space` | Space label (must match discover) |
| `--json` | Emit JSON output with full provenance |
| `--wait` | Accepted for connected-shape compatibility; no-op in local mode |
| `--set` / `--set-string` / `--set-file` | Helm-style invocation overrides captured in provenance when Helm is the active generator |

Arguments:

- `<target-path>`: local repo path to discover and import
- `<render-target-path>`: optional local render target path; omit it when the render target is the same repo path

Output includes: `generator_profile`, `dry_inputs`, `wet_manifest_targets`, `provenance` (field-origin map, inverse-edit pointers).

For layered Helm repos, `provenance[].helm_layered_analysis` also explains how
the chart is composed:

- `dependency_charts`: subcharts, aliases, vendored chart paths, and any
  `condition:` expression that controls whether the subchart is active
- `field_origin_map[].source_layer`: which Helm layer won for a field
  (`umbrella` vs a subchart alias/name)
- `crd_paths`: CRDs under `crds/`, kept separate from templated resources
- `hook_templates`: hook template paths plus their `helm.sh/hook` annotations
- `values_schema_path`, `schema_validation_state`,
  `schema_validation_violations`: pre-render values-schema findings

### `gitops cleanup`

Remove local discover state.

```
cub-gen gitops cleanup --space <space> <target-path>
```

---

## Enrichment

### `enrich preview`

Preview PR-friendly sidecar provenance without mutating the repo.

```
cub-gen enrich preview --space <space> [--json|--patch] [--where-resource <expr>] <target-path> [<render-target-path>]
```

The JSON preview proposes `.cub-gen/enrichment/*.provenance.json` artifacts and
explains four kinds of metadata:

- `source-link`: which source files feed the generator
- `ownership-label`: who owns source and rendered artifacts
- `route-badge`: whether edits are `apply-here`, `lift-upstream`, `overlay`, or `block/escalate`
- `pr-mr-link`: the `change_id` used to correlate a GitHub PR with a ConfigHub MR

`--patch` emits a unified diff for the same sidecar files, suitable for review
or a small PR. Existing sidecars are marked `review-required`; preview remains
read-only.

### `enrich write`

Write the reviewed sidecar provenance files.

```
cub-gen enrich write --space <space> [--where-resource <expr>] <target-path> [<render-target-path>]
```

`enrich write` creates only sidecar files under `.cub-gen/enrichment/`. It does
not annotate or rewrite app manifests. It refuses to overwrite an existing
sidecar so provenance updates stay explicit and reviewable.

---

## Normalize

### `normalize preview`

Preview governed config rewrite proposals without applying them.

```
cub-gen normalize preview --space <space> [--json|--patch] [--where-resource <expr>] <target-path> [<render-target-path>]
```

Use this after import/enrichment when the generator works but important product
boundaries are still implicit. The command emits one reviewable patch set under
`.cub-gen/normalize/`; it does not edit app source, platform source, rendered
manifests, or ConfigHub Units.

Current proposal types:

| Transform | What it makes explicit |
|---|---|
| `add-route-policy-annotation` | ConfigHub Unit route policy annotations from field-route metadata |
| `lift-generated-patch-to-source` | source files that should receive durable changes instead of generated YAML |
| `split-env-values-into-variants` | Deployable Variant inventory from environment/profile files |
| `add-missing-owners` | owner annotations from generator provenance and rendered resources |
| `replace-implicit-secret-wiring` | explicit secret-reference proposals for literal secret-shaped env values |

Example:

```
cub-gen normalize preview --patch --space platform ./examples/springboot-paas
```

The Spring Boot example produces proposals for route enforcement,
lift-upstream cache changes, dev/stage/prod variants, owner annotations, and a
datasource SecretReference candidate. Unknown repos degrade to a no-op preview
with a `NO_KNOWN_PATTERNS` diagnostic.

---

## Bridge artifacts

### `publish`

Generate a ConfigHub-ready change bundle from import output.

```
# Pipe mode (from import)
cub-gen gitops import ... | cub-gen publish --in - --out -

# Direct mode (import + bundle in one step)
cub-gen publish --space <space> [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] <target-path> [<render-target-path>]
```

Output includes `digest_algorithm` (sha256) and `bundle_digest` for verification.

## Change Commands

### `change diff`

Render two git refs of the same Helm repo path and show which rendered fields
changed, with before/after provenance attached to each changed field.

```
cub-gen change diff --before-ref <ref> --after-ref <ref> [--space <space>] [--where-resource <expr>] [--dry-path <path>] [--wet-path <path>] [--owner <owner>] <target-path> [<render-target-path>]
```

Current scope:

- today this is a Helm-first command
- it expects `<target-path>` to live in a git repo
- it renders each side with the Helm value files `cub-gen` already selected for import/provenance
- output includes `before.value`, `after.value`, and before/after provenance metadata for each changed field

### `change revision-diff`

Compare two git refs and pair each rendered field change with the DRY edit route
that produced it.

```
cub-gen change revision-diff --from <ref> --to <ref> [--space <space>] [--where-resource <expr>] [--dry-path <path>] [--wet-path <path>] [--owner <owner>] <target-path> [<render-target-path>]
```

Current scope:

- today this reuses the Helm-first field-level diff engine
- output groups each changed rendered field under a `cause` block with before/after DRY paths, source paths, origin types, edit hint, and warning metadata
- use this when you want release-note style "what DRY edit caused what deployed-field change?" output instead of just a raw before/after render diff

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

## Bridge flow (advanced ConfigHub API path)

These are the deeper connected commands. For the repo's default connected
first run, use `./examples/demo/run-connected-smoke.sh` first. Use bridge
commands only for deeper bridge-backed walkthroughs or bridge-only capability
gaps.

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

### `bridge link`

One-command GitHub PR to ConfigHub MR correlation. It derives `change_id` from a
`cub-gen publish` bundle, builds the canonical review-link record, and can POST
it to ConfigHub with an idempotency key.

```bash
cub-gen bridge link \
  --bundle bundle.json \
  --github-pr-repo github.com/confighub/apps \
  --github-pr-number 42 \
  --github-pr-url https://github.com/confighub/apps/pull/42 \
  --confighub-mr-id mr_123 \
  --confighub-mr-url https://confighub.example/mr/123 \
  --base-url https://confighub.example \
  --token "$CONFIGHUB_TOKEN"
```

Omit `--base-url` when you only want the local review-link JSON for a PR body,
status payload, or changeset label set.

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

## Generator helper commands

These helper groups are narrower than the main repo-first flow. Use them when a
generator-specific example wants a local contract check or scaffold step after
you already know the repo path.

### `score validate-workload`

Check whether a `score.yaml` resource set stays inside the platform-approved
workload class.

```bash
cub-gen score validate-workload --score <score.yaml> --contract <workload-class.yaml>
```

Typical example path:

```bash
./cub-gen score validate-workload \
  --score ./examples/scoredev-paas/score.yaml \
  --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml
```

Output:

- `ALLOW` when all declared Score resource types are approved
- `ESCALATE` when a new resource type needs platform review

### `springboot validate-mutation`

Check whether a Spring field path stays inside the app-owned route map.

```bash
cub-gen springboot validate-mutation --routes <field-routes.yaml> <field-path>
```

Typical example path:

```bash
./cub-gen springboot validate-mutation \
  --routes ./examples/springboot-paas/operational/field-routes.yaml \
  feature.inventory.reservationMode
```

### `springboot set-embedded-config`

Set a field directly inside `ConfigMap.data["application.yaml"]` for a Spring
ConfigHub payload or rendered ConfigMap file.

```bash
cub-gen springboot set-embedded-config --file <payload.yaml> [--configmap <name>] [--config-key application.yaml] [--routes <field-routes.yaml>] <field-path> <value>
```

Typical example path:

```bash
./cub-gen springboot set-embedded-config \
  --routes ./examples/springboot-paas/operational/field-routes.yaml \
  --file ./examples/springboot-paas/confighub/inventory-api-prod.yaml \
  --configmap inventory-api-config \
  feature.inventory.reservationMode optimistic
```

Use `--routes` when you want the command to enforce the same app-owned versus
platform-owned boundary as `validate-mutation` before writing the file.

### `springboot init`

Bootstrap the Spring example's operational route files for a repo.

```bash
cub-gen springboot init [flags] <source-path>
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

## Contract status

The current preview keeps its command and JSON contracts locked so first-run
docs, examples, and automation stay reliable. See [Command Parity](parity.md)
for the full contributor-facing drift matrix.

| Status | Meaning |
|--------|---------|
| `matched` | Behavior intentionally mirrored from `cub gitops` |
| `partial` | Same contract shape, simplified implementation |
| `deferred` | Intentionally not implemented yet |

---

## Variants and overlays

Current status: explicit variant fanout plus generator-specific overlays.

What works today:

- One `gitops import` / `publish` invocation works on one repo path pair.
- `platform fanout` reads a platform manifest and emits one ConfigHub-ready bundle per declared deployable variant.
- `platform fanout --variant dev` scopes output to one environment name across Components; `--variant checkout-api/prod` scopes to one concrete variant.
- `change explain --change-id ID --bundle fanout.json --variant checkout-api/prod` can explain a selected variant bundle from fanout output.
- Supported generators can pick up overlay files that already live in that repo, such as Helm `values-prod.yaml`, Spring `application-dev.yaml`, and generator-specific overlay files.
- Helm flows also capture invocation-time `--set`, `--set-string`, and `--set-file` overrides and rank them above values files in provenance and `change explain`.
- Helm `change explain` also calls out when a field is currently coming from `.Chart.AppVersion`, `.Files.Get`, a helper chain, `lookup`/other render-time logic, an external ref declared in values, or an unresolved chart-default path instead of an observed values file.
- Provenance records include those generator inputs in `dry_inputs`, `values_paths`, and `field_origin_map` when the generator emits separate overlay transforms.
- When a repo declares a generator chain, `change explain` includes `hops[]` so you can see the upstream DRY stage and the downstream generator stage together instead of stopping at the last transform boundary.
- `gitops discover` and `gitops import` also surface declared generator chains directly in table output so the Score -> Helm style pipeline is obvious on first run.
- `change explain` can point to overlay-specific edit locations. For Spring Boot, the current edit hint routes `server.port` changes to `application-dev.yaml` for environment overrides while keeping `application.yaml` as the base.

What does not work today:

- No repeated `--values`, `--overlay`, or `--variant` flag surface
- No glob-based fan-out
- No generic Kustomize overlay workflow beyond the generator-specific support already implemented

Reality-check example:

```bash
./cub-gen gitops import --space platform --json ./examples/springboot-paas \
  | jq '.provenance[0].field_origin_map[] | select(.dry_path=="server.port")'

./cub-gen change impact --space platform \
  --dry-path "values.image.tag" \
  ./examples/helm-paas

./cub-gen change explain --space platform \
  --wet-path "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort" \
  ./examples/springboot-paas

./cub-gen platform fanout --json --out fanout.json \
  ./testdata/variant-fanout/platform.yaml

CHANGE_ID="$(jq -r '.variants[] | select(.variant_id=="checkout-api/dev") | .change_id' fanout.json)"
./cub-gen change explain --change-id "$CHANGE_ID" --bundle fanout.json \
  --variant checkout-api/dev
```

Use this section as the answer for "can cub-gen render N explicit variants from one generator in one command?":
yes, when the variants are explicit in a platform manifest. It does not yet
perform implicit glob fanout or guess environment semantics from directory
names.

Helm precedence today is:

1. `--set` / `--set-string` / `--set-file`
2. overlay files such as `values-prod.yaml`
3. base `values.yaml`
4. chart defaults

---

## Generator quickstart recipes

Build once, then try any generator:

```bash
go build -o ./cub-gen ./cmd/cub-gen
```

### Helm

```bash
./cub-gen gitops discover --space platform ./examples/helm-paas
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets}'
./cub-gen change impact --space platform --dry-path values.image.tag \
  ./examples/helm-paas
./cub-gen change explain --space platform --set image.tag=v1.2.4 \
  ./examples/helm-paas
./cub-gen gitops cleanup --space platform ./examples/helm-paas
```

### Argo ApplicationSet

```bash
./cub-gen gitops discover --space platform ./testdata/applicationset-standalone
./cub-gen gitops import --space platform --json ./testdata/applicationset-standalone \
  | jq '{profile: .discovered[0].generator_profile, analysis: .provenance[0].application_set_analysis}'
```

### Argo app-of-apps

```bash
./cub-gen gitops discover --space platform ./testdata/app-of-apps-standalone
./cub-gen gitops import --space platform --json ./testdata/app-of-apps-standalone \
  | jq '{profile: .discovered[0].generator_profile, analysis: .provenance[0].app_of_apps_analysis}'
```

### Score.dev

```bash
./cub-gen gitops discover --space platform ./examples/scoredev-paas
./cub-gen gitops import --space platform --json ./examples/scoredev-paas \
  | jq '{profile: .discovered[0].generator_profile, field_origin_map: .provenance[0].field_origin_map, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/scoredev-paas
```

### Spring Boot

```bash
./cub-gen gitops discover --space platform ./examples/springboot-paas
./cub-gen gitops import --space platform --json ./examples/springboot-paas \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/springboot-paas
```

### Backstage IDP

```bash
./cub-gen gitops discover --space platform ./examples/backstage-idp
./cub-gen gitops import --space platform --json ./examples/backstage-idp \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/backstage-idp
```

### No-config-platform (app-only)

```bash
./cub-gen gitops discover --space platform ./examples/just-apps-no-platform-config
./cub-gen gitops import --space platform --json ./examples/just-apps-no-platform-config \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/just-apps-no-platform-config
```

### Ops workflow

```bash
./cub-gen gitops discover --space platform ./examples/ops-workflow
./cub-gen gitops import --space platform --json ./examples/ops-workflow \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/ops-workflow
```

### C3 Agent

```bash
./cub-gen gitops discover --space platform ./examples/c3agent
./cub-gen gitops import --space platform --json ./examples/c3agent \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets_count: (.wet_manifest_targets|length), inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/c3agent
```

### Swamp automation

```bash
./cub-gen gitops discover --space platform ./examples/swamp-automation
./cub-gen gitops import --space platform --json ./examples/swamp-automation \
  | jq '{profile: .discovered[0].generator_profile, dry_inputs, wet_manifest_targets, inverse_edit_pointers: .provenance[0].inverse_edit_pointers}'
./cub-gen gitops cleanup --space platform ./examples/swamp-automation
```

---

## Bridge artifact examples

### Publish + verify (pipe mode)

```bash
./cub-gen gitops import --space platform --json ./examples/helm-paas \
  | ./cub-gen publish --in - --out - \
  | jq '{schema_version, source, change_id, summary}'
```

### Publish + verify + attest (file mode)

```bash
./cub-gen publish --space platform ./examples/helm-paas > bundle.json
./cub-gen verify --in bundle.json
./cub-gen attest --in bundle.json --verifier ci-bot > attestation.json
./cub-gen verify-attestation --in attestation.json --bundle bundle.json
```

### Bridge flow (advanced bridge-only ConfigHub API path)

Most users should start with the standard connected environment check:

```bash
cub auth login
./examples/demo/run-connected-smoke.sh
```

Use the commands below only when you explicitly need the deeper bridge-backed
connected path.

```bash
# 1) Build bundle and attestation artifacts
./cub-gen publish --space platform ./examples/helm-paas > bundle.json
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

# 5) Link the GitHub PR and ConfigHub MR with the same change_id
./cub-gen bridge link \
  --bundle bundle.json \
  --github-pr-repo github.com/confighub/apps \
  --github-pr-number 42 \
  --github-pr-url https://github.com/confighub/apps/pull/42 \
  --confighub-mr-id "$(jq -r .artifact_id ingest-result.json)" \
  --confighub-mr-url https://confighub.example/mr/123 \
  --base-url https://confighub.example \
  --token "$CONFIGHUB_TOKEN"
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
./cub-gen publish --space platform ./examples/helm-paas
./cub-gen publish --space platform ./examples/scoredev-paas
./cub-gen publish --space platform ./examples/springboot-paas
./cub-gen publish --space platform ./examples/backstage-idp
./cub-gen publish --space platform ./examples/just-apps-no-platform-config
./cub-gen publish --space platform ./examples/ops-workflow
./cub-gen publish --space platform ./examples/c3agent
./cub-gen publish --space platform ./examples/swamp-automation
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
