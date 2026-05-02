# Platform Generator Product Worklist

Status: major product slices landed; remaining work sequenced
Date: 2026-05-01

This worklist turns the platform-generator manifesto into product work. It
combines the current open roadmap items with the gaps exposed by the
OpenChoreo, ApplicationSet, and app-of-apps discussion.

For the sequenced roadmap that organizes this issue pack into workstreams, see
[v0.4 Working Roadmap: Component -> Variant -> Proof](2026-04-30-v0.4-obvious-value-roadmap.md)
and GitHub release tracker [#302](https://github.com/confighub/cub-gen/issues/302).

## Current Open Roadmap Touchpoints

Open issues checked on 2026-05-01:

| Issue | Title | Why it matters for the manifesto |
|---|---|---|
| [#287](https://github.com/confighub/cub-gen/issues/287) | make cub-gen role and value obvious | parent roadmap issue keeping the product story coherent |
| [#283](https://github.com/confighub/cub-gen/issues/283) | enforce generator route metadata server-side | route ownership must become an authoritative backend decision |
| [#236](https://github.com/confighub/cub-gen/issues/236) | ship cub-gen as `cub gen` plugin | product workflow should feel like one platform; local plugin proof now works, release artifact remains |
| [#213](https://github.com/confighub/cub-gen/issues/213) | GUI provenance trace | click-field-to-source is the core teaching moment |
| [#212](https://github.com/confighub/cub-gen/issues/212) | GUI regeneration/refresh preview | users need to see generated impact before merge/apply |
| [#211](https://github.com/confighub/cub-gen/issues/211) | GUI mutation history/activity log | overlay, lift, and block decisions need visible history |
| [#210](https://github.com/confighub/cub-gen/issues/210) | GUI multi-environment comparison | variant fanout is not useful unless differences are obvious |
| [#209](https://github.com/confighub/cub-gen/issues/209) | GUI field-level route badges | route classification must be visible at the field |

Earlier roadmap refs still relevant as themes:

| Ref | Theme | Current interpretation |
|---|---|---|
| `#185` | custom generator onboarding | extend from "how to fork" to "how to import a platform estate" |
| `#187` | Kubara-like layered provenance | broaden to ApplicationSet/app-of-apps/OC style chains |
| `#188` | Git PR <-> ConfigHub MR pairing | turn contracts and demos into a product workflow |
| `#189` | promotion and live->wet->dry | make overlay-to-source promotion first-class |
| `#190` | clarify DRY->WET thesis | use the platform generator manifesto language |
| `#192` | prompt-as-DRY | keep AI-authored changes inside the same generator/provenance model |

## Issue Pack

GitHub issues created on 2026-04-30:

| Work item | Issue | Scope |
|---|---|---|
| PG-01 | [#276](https://github.com/confighub/cub-gen/issues/276) | generic multi-repo platform import graph |
| PG-02 | [#277](https://github.com/confighub/cub-gen/issues/277) | OpenChoreo adapter |
| PG-03 | [#278](https://github.com/confighub/cub-gen/issues/278) | Argo app-of-apps generator boundary |
| PG-04 | [#279](https://github.com/confighub/cub-gen/issues/279) | multi-env/tenant variant fanout |
| PG-05 | [#280](https://github.com/confighub/cub-gen/issues/280) | provenance enrichment proposals |
| PG-06 | [#281](https://github.com/confighub/cub-gen/issues/281) | governed config rewrite proposals |
| PG-07 | [#282](https://github.com/confighub/cub-gen/issues/282) | GitHub PR to ConfigHub MR linkage |
| PG-08 | [#283](https://github.com/confighub/cub-gen/issues/283) | server-side route enforcement |
| PG-09 | [#284](https://github.com/confighub/cub-gen/issues/284) | platform generator teaching-pack QA |
| Spring-platform follow-up | [#285](https://github.com/confighub/cub-gen/issues/285) | cross-repo docs consistency; `confighub/examples` has issues disabled |
| OC-QA-01 | [#288](https://github.com/confighub/cub-gen/issues/288) | make README OpenChoreo claims honest and explicit |
| OC-QA-02 | [#289](https://github.com/confighub/cub-gen/issues/289) | add an OpenChoreo-shaped field trace example |
| OC-QA-03 | [#290](https://github.com/confighub/cub-gen/issues/290) | produce read-only existing-platform adoption report before rewrites |
| OC-QA-04 | [#291](https://github.com/confighub/cub-gen/issues/291) | model generated-resource ownership and edit routing |
| OC-QA-05 | [#292](https://github.com/confighub/cub-gen/issues/292) | add an OpenChoreo credibility-gate fixture |

### PG-01: Generic Platform Import Graph

Problem: `cub-gen` import is still repo/generator oriented. Platform estates
often spread app source config, platform contracts, environment bindings, and rendered
output across multiple repos.

Status: landed in PR #286 via `cub-gen platform import --json <manifest>`.
The command reads a local manifest, imports each existing repo read-only, emits
Components, Variants, Deployment Variants, Targets, generator inputs, WET targets, and
connections, and reports missing repo, missing owner, and unsupported generator
diagnostics explicitly.

Deterministic success criteria:

1. Given a fixture with `apps/`, `platform/`, `envs/`, and `rendered/` repos,
   `cub-gen platform import --json <manifest>` emits one stable graph.
2. The graph includes apps, variants, generator inputs, WET targets, owners,
   and missing-link diagnostics.
3. Re-running on the same inputs produces byte-stable JSON after timestamp
   normalization.

Proof matrix:

| Proof | Required |
|---|---|
| unit | parser for platform manifest and graph normalization |
| golden | multi-repo fixture graph JSON |
| example | docs/demo that imports a tiny platform estate |
| degradation | missing repo, missing owner, unsupported generator all emit explicit diagnostics |

Definition of done:

1. command docs and help text exist,
2. golden tests pass,
3. example README explains read-only import first,
4. no existing `gitops import` contract drift.

### PG-02: OpenChoreo Adapter

Status: implemented for the fixture-backed v1alpha1 hardgate shape. Broader
upstream estate validation and multi-repo OpenChoreo import remain follow-on
adoption work.

Problem: OpenChoreo is a clean candidate generator. `cub-gen` now reads the
hard platform case from an OpenChoreo-shaped repo: app source config, deployable
variant bindings, platform contracts, secret references, rendered releases, and
generated Kubernetes resources with ownership proof.

Quality gate: the adapter is not credible until the README is honest about
support status ([#288](https://github.com/confighub/cub-gen/issues/288)), the
docs include an OpenChoreo-shaped trace
([#289](https://github.com/confighub/cub-gen/issues/289)), adoption starts with
a read-only platform report
([#290](https://github.com/confighub/cub-gen/issues/290)), generated-resource
ownership and edit routing are modeled
([#291](https://github.com/confighub/cub-gen/issues/291)), and the hard fixture
exists ([#292](https://github.com/confighub/cub-gen/issues/292)).

Deterministic success criteria:

1. Given Workload, ReleaseBinding, SecretReference, ComponentType, and
   RenderedRelease fixtures, `cub-gen gitops import` detects `openchoreo`.
2. Import emits DRY roles for app source config, variant binding, secret reference,
   component type, rendered release, and generated manifests.
3. Field-origin map routes env vars, mounted files, secret refs, image,
   service port, resource limits, and platform defaults.
4. Generated Kubernetes resources can be identified as platform/controller
   owned where applicable.
5. At least two Deployment Variants are represented.
6. `gitops discover --adoption-report` emits a read-only platform adoption
   report before any rewrite.

Proof matrix:

| Proof | Required |
|---|---|
| unit | CRD parsing and role classification |
| golden | discover/import/publish OpenChoreo parity outputs |
| example | worked example fixture with prod override, platform default, generated resources, mounted-file ConfigMap, and secret reference |
| route | generated Deployment edit routes to apply-here, overlay, lift-upstream, or block/escalate |
| degradation | unknown CRD version, missing ComponentType, unresolved SecretReference, or unknown owner degrade explicitly |

Definition of done:

1. registry family exists,
2. example satisfies universal example contract,
3. docs link to the OpenChoreo worked example,
4. unsupported CRD shapes do not produce guessed lineage.

### PG-03: App-of-Apps Generator Boundary

Status: landed in PR #286 as a fixture-backed initial adapter.

Problem: ApplicationSet has bounded support, and app-of-apps needed the same
root/child generator boundary rather than remaining only a documented pattern.

Deterministic success criteria:

1. Given a root Argo `Application` pointing at a child app catalog,
   `cub-gen` classifies the root as an app-of-apps generator candidate.
2. Import emits child `Application` lineage back to catalog files.
3. Downstream Helm/Kustomize sources remain separate generator layers.

Proof matrix:

| Proof | Required |
|---|---|
| unit | root Application detection and child catalog discovery |
| golden | app-of-apps discover/import/publish outputs |
| example | root app plus three child apps, one Helm, one Kustomize, one plain YAML |
| degradation | remote-only root path or missing child catalog reports observed-only mode |

Definition of done:

1. app-of-apps boundary doc exists,
2. worked example becomes runnable fixture,
3. no confusion with plain Argo transport-only repos,
4. child app edits route to catalog or downstream source.

### PG-04: Variant Fanout Command

Problem: the docs explicitly say there is no one-command multi-env/tenant
fanout.

Status: landed in PR #286 via `cub-gen platform fanout`. The command reads
explicit `variants:` entries from a platform manifest, emits one standard
`change-bundle/v1` per variant, supports `--variant` scoping, and lets
`change explain` select a variant bundle from fanout JSON. It remains
manifest-driven; implicit glob discovery is intentionally not implemented.

Deterministic success criteria:

1. Given a repo with `dev`, `stage`, and `prod` inputs, one command emits one
   bundle per variant.
2. Each bundle has a distinct variant identity and stable `change_id`.
3. Shared base inputs and variant-specific inputs are separately attributed.

Proof matrix:

| Proof | Required |
|---|---|
| unit | variant discovery and ordering |
| golden | Helm, Spring, and Score variant fanout JSON |
| example | one demo showing cross-env differences |
| degradation | ambiguous variant source reports explicit error |

Definition of done:

1. CLI docs describe `--variant` and/or manifest-driven fanout,
2. `change explain` can scope by variant,
3. ConfigHub bundle shape is accepted by bridge ingest,
4. no globbing surprises or implicit deploys.

### PG-05: Annotation and Enrichment Proposals

Problem: users may want provenance annotations on platform and app artifacts,
but automatic in-place edits would be risky.

Status: landed in PR #286 via `cub-gen enrich preview` and
`cub-gen enrich write`. The command proposes sidecar provenance under
`.cub-gen/enrichment/`, emits JSON or patch preview, and treats existing
sidecars as `review-required` instead of overwriting them.

Deterministic success criteria:

1. `cub-gen enrich preview` emits proposed annotations or sidecar files without
   mutating the repo.
2. `cub-gen enrich write` writes only explicit, reviewed enrichment artifacts.
3. The preview explains which annotations are source links, ownership labels,
   PR/MR links, or route badges.

Proof matrix:

| Proof | Required |
|---|---|
| unit | annotation rendering and stable sorting |
| golden | preview JSON and patch output |
| example | OpenChoreo or Argo fixture gets sidecar provenance |
| degradation | existing conflicting annotation becomes review-required, not overwritten |

Definition of done:

1. read-only preview is default,
2. writes are PR-friendly and small,
3. docs explain sidecar-first over mass mutation,
4. no secrets or high-volume evidence written to Git.

### PG-06: Config Rewrite / Normalize Proposals

Problem: `springboot init` scaffolds starter material, but there is no general
"make this config better" proposal engine.

Status: implemented by `cub-gen normalize preview`. The Spring Boot example now
produces one review-only patch set with route policy annotations,
lift-upstream source routing, Deployment Variant inventory, owner annotations,
and explicit secret-reference proposals.

Deterministic success criteria:

1. Given a known anti-pattern fixture, `cub-gen normalize preview` proposes
   one reviewable patch set.
2. Supported transforms include lift generated patch to source, split
   environment values into variants, add missing owners, and replace implicit
   secret wiring with explicit references.
3. Every proposed patch includes source, owner, risk, and rendered impact.

Proof matrix:

| Proof | Required |
|---|---|
| unit | transform planner |
| golden | patch bundle and rendered impact |
| example | one Spring or OpenChoreo-style normalization |
| degradation | unknown pattern reports no-op with explanation |

Definition of done:

1. preview-only default,
2. all writes are branch/PR oriented,
3. no direct platform source mutation without review,
4. docs explain this is optional refactoring after import.

### PG-07: Seamless PR/MR Linkage Command

Problem: PR/MR linkage exists as contracts, bridge promote commands, and demo
scripts, but not as one easy product workflow.

Status: landed in PR #297 via `cub-gen bridge link`. The command derives
`change_id` from a verified publish bundle, emits canonical review-link JSON for
local evidence, and can POST the linkage record to ConfigHub with a deterministic
idempotency key.

Deterministic success criteria:

1. Given a GitHub PR and a `cub-gen publish` bundle, one command creates or
   updates the ConfigHub MR linkage record.
2. The same `change_id` appears in GitHub status/check output, ConfigHub MR,
   bundle, attestation, and promotion flow.
3. Re-running is idempotent.

Proof matrix:

| Proof | Required |
|---|---|
| unit | linkage state machine |
| integration | mocked GitHub and ConfigHub endpoints |
| connected | optional live smoke with safe repo |
| degradation | missing token, missing PR, or backend 404 reports actionable error |

Definition of done:

1. command docs show GitHub-first and ConfigHub-first flows,
2. status checks reflect ALLOW/ESCALATE/BLOCK,
3. linked evidence includes DRY inputs and WET impact,
4. no deploy happens without ConfigHub decision gate.

### PG-08: Server-Side Route Enforcement

Problem: Spring proves client-side route validation, but backend enforcement is
still thinner than the model wants.

Deterministic success criteria:

1. ConfigHub rejects or escalates mutations to generator-owned fields using
   route metadata from imported provenance.
2. App-owned fields can still be mutated through approved apply-here paths.
3. Lift-upstream fields produce source proposal instructions, not direct WET
   mutation.

Proof matrix:

| Proof | Required |
|---|---|
| unit | route policy evaluation |
| integration | ConfigHub mutation endpoint rejects blocked field |
| example | Spring Boot apply-here and datasource block against backend |
| degradation | missing provenance downgrades to review-required, not allow |

Definition of done:

1. backend decision state is authoritative,
2. client-side validation remains a convenience, not the only gate,
3. docs update Spring caveat,
4. connected smoke includes one backend route decision.

### PG-09: Platform Generator Teaching Pack

Problem: Jesper-style skepticism is evidence that the thesis needs teaching
artifacts, not only commands.

Status: landed in PR #296. The teaching pack now has conservative
OpenChoreo-support language, stricter docs checks, and fixture-backed diagrams
and examples. The `spring-platform` teaching repo was aligned in
`confighub/examples` PR #138.

Deterministic success criteria:

1. Docs include diagram-heavy worked examples for OpenChoreo, ApplicationSet,
   and app-of-apps.
2. Main docs explain "generator, not replacement" within the first minute.
3. Example docs distinguish implemented proof from future adapter work.

Proof matrix:

| Proof | Required |
|---|---|
| docs | manifesto, worked examples, examples index, mkdocs nav |
| review | links render, claims are caveated |
| example | at least one ApplicationSet proof command points at a real fixture |
| degradation | future work marked as not implemented |

Definition of done:

1. docs build,
2. no example truth matrix is manually edited,
3. roadmap links to next product gaps,
4. product claims remain conservative.
