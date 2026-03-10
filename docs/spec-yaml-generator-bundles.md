# Spec: YAML Generator Bundles

**Date**: 2026-03-08
**Status**: proposed — decision required before implementation (Go vs YAML)
**Supersedes**: `docs/decisions/2026-03-08-go-canonical-generator-triples.md`

---

## 1. Problem

Platform owners need to create, modify, and understand generator triples (contract + provenance + inverse transform plan) without reading or writing Go. Today, every new generator requires edits across 5 Go files and 32+ switch-case branches. This is an adoption blocker.

## 2. Solution

Each generator becomes a self-contained **YAML bundle**. AI writes the initial YAML from a platform's config files. The platform owner reviews and tweaks it. `cub-gen` loads YAML at runtime via `embed.FS`, replacing Go struct literals with a generic data-driven engine.

The Go is infrastructure, not interface. Nobody needs to see it unless they're debugging the engine.

## 2.1 Decision gate before implementation

No implementation starts until an architecture decision is made between:
1. **Go-canonical near-term:** keep Go as source of truth for now, defer YAML migration.
2. **YAML-canonical near-term:** execute this spec's migration phases.

Decision criteria:
1. Impact on helm/score.dev/springboot delivery in the next milestone.
2. Migration risk and parity-test complexity.
3. Authoring needs of non-Go platform owners vs maintainability for Go contributors.

Decision artifact:
1. An ADR/decision memo in `docs/decisions/` that records the chosen path and immediate scope.

## 3. Flow

```
Platform owner brings config files (helm chart, score.yaml, c3agent.yaml, etc.)
        │
        ▼
AI reads the config files and generates a generator bundle:
  generators/<kind>/triple.yaml   ← the full triple
  generators/<kind>/examples/     ← sample DRY input files
  generators/<kind>/README.md     ← auto-generated from triple.yaml
        │
        ▼
Platform owner reviews the YAML:
  - Does the input detection make sense?
  - Are the field mappings correct?
  - Is the ownership split right?
  - Do the confidence scores reflect reality?
        │
        ▼
cub-gen loads the YAML at startup.
No Go struct edits needed for bundle content.
No recompilation is only true when external bundle loading (for example `--generators-dir`) is available; `embed.FS`-only mode still requires rebuilding the binary.
```

## 4. Bundle structure

```
generators/
├── c3agent/
│   ├── triple.yaml          # Source of truth — loaded at runtime
│   ├── examples/
│   │   ├── c3agent.yaml     # Sample base DRY input
│   │   └── c3agent-prod.yaml # Sample overlay DRY input
│   └── README.md            # Auto-generated docs + Mermaid diagram
├── helm/
│   ├── triple.yaml
│   ├── examples/
│   │   ├── Chart.yaml
│   │   └── values.yaml
│   └── README.md
├── swamp/
│   ├── triple.yaml
│   ├── examples/
│   │   ├── .swamp.yaml
│   │   └── workflow-deploy.yaml
│   └── README.md
└── ... (one directory per generator kind)
```

## 5. YAML triple schema

The `triple.yaml` file contains everything needed to detect, import, map provenance, and plan inverse transforms for a generator. It unifies data currently spread across `registry.go` (FamilySpec), `detect.go` (detection functions), and `importer.go` (switch cases).

### 5.1 Complete schema

```yaml
# --- Identity ---
kind: "c3agent"                     # Generator kind (unique identifier)
profile: "c3agent"                  # Display profile name
resource_kind: "ConfigMap"          # Primary WET resource kind
resource_type: "v1/ConfigMap"       # Full API resource type

capabilities:
  - "fleet-config"
  - "agent-orchestration"
  - "inverse-fleet-config-patch"

# --- Detection ---
# Rules for how `cub-gen gitops discover` finds this generator in a repo.
# Replaces the per-generator detect<Kind>() functions in detect.go.
detection:
  # Trigger files: the filenames that indicate this generator might apply.
  # Walk the repo looking for files matching these names (case-insensitive).
  trigger_files:
    - "c3agent.yaml"
    - "c3agent.yml"
    - "c3agent.json"

  # Content keywords: after finding a trigger file, read it and check
  # that at least one of these strings appears (case-insensitive, OR logic).
  # If none match, skip. Set to [] to skip content validation.
  content_keywords:
    - "fleet:"
    - "agent"

  # Input globs: after confirming a trigger file, glob for related inputs
  # in the same directory. These become the detection's Inputs list.
  input_globs:
    - "c3agent*.yaml"
    - "c3agent*.yml"
    - "c3agent*.json"

  # Detection confidence: 0.0–1.0, how confident we are this is the right
  # generator when a trigger file matches.
  confidence: 0.88

# --- Contract ---
# What this generator consumes (DRY inputs) and produces (WET targets).
# Maps directly to the existing FamilySpec fields.
contract:
  default_input_role: "fleet-config"
  default_owner: "app-team"

  input_role_rules:
    - role: "fleet-config-base"
      exact_basenames:
        - "c3agent.yaml"
        - "c3agent.yml"
        - "c3agent.json"
    - role: "fleet-config-overlay"
      prefixes:
        - "c3agent-"
      extensions:
        - ".yaml"
        - ".yml"
        - ".json"

  role_owners: {}

  role_schema_refs:
    fleet-config-base: "https://schema.confighub.dev/generators/c3agent-v1"
    fleet-config-overlay: "https://schema.confighub.dev/generators/c3agent-v1"

  wet_targets:
    - kind: "ConfigMap"
      name_template: "{{name}}-fleet-config"
      owner: "platform-runtime"
      namespace: "apps"
      source_dry_path_template: "fleet.agent_model"
    - kind: "Secret"
      name_template: "{{name}}-fleet-credentials"
      owner: "platform-runtime"
      namespace: "apps"
      source_dry_path_template: "credentials.anthropic_key_ref"

# --- Provenance ---
# Field-level DRY→WET lineage with confidence scores.
provenance:
  field_origin_transform: "c3agent-config-to-runtime"
  field_origin_overlay_transform: "c3agent-overlay-merge"

  field_origin_confidences:
    fleet_config: 0.91
    credentials: 0.86
    component_ports_base: 0.84
    component_ports_overlay: 0.80
    max_concurrent_tasks_base: 0.84
    max_concurrent_tasks_overlay: 0.80

  rendered_lineage_templates:
    - kind: "ConfigMap"
      name_template: "{{name}}-fleet-config"
      namespace: "apps"
      source_path_hint: "base_config_path"
      source_dry_path_template: "fleet.agent_model"
    - kind: "Secret"
      name_template: "{{name}}-fleet-credentials"
      namespace: "apps"
      source_path_hint: "base_config_path"
      source_dry_path_template: "credentials.anthropic_key_ref"
    - kind: "ConfigMap"
      name_template: "{{name}}-fleet-config"
      namespace: "apps"
      source_path_hint: "overlay_config_path"
      source_dry_path_template: "fleet.max_concurrent_tasks"
      optional: true

  # Field origins: the concrete DRY→WET field mappings.
  # Replaces the per-generator cases in fieldOriginsForGenerator().
  # Each entry has a source_hint that resolves to an actual file path
  # via the path_hints system.
  field_origins:
    - dry_path: "fleet.agent_model"
      wet_path: "ConfigMap/data/AGENT_MODEL"
      source_hint: "base_config_path"
      confidence_key: "fleet_config"
      transform: "base"          # "base" → uses field_origin_transform
    - dry_path: "credentials.anthropic_key_ref"
      wet_path: "Secret/data/ANTHROPIC_API_KEY"
      source_hint: "base_config_path"
      confidence_key: "credentials"
      transform: "base"
    - dry_path: "fleet.max_concurrent_tasks"
      wet_path: "ConfigMap/data/MAX_CONCURRENT_TASKS"
      source_hint: "base_config_path"
      confidence_key: "max_concurrent_tasks_base"
      transform: "base"
    - dry_path: "fleet.max_concurrent_tasks"
      wet_path: "ConfigMap/data/MAX_CONCURRENT_TASKS"
      source_hint: "overlay_config_path"
      confidence_key: "max_concurrent_tasks_overlay"
      transform: "overlay"       # "overlay" → uses field_origin_overlay_transform
      optional: true             # Only emitted when overlay file exists

# --- Inverse ---
# WET→DRY edit guidance: who owns what, how to edit back.
inverse:
  inverse_patch_templates:
    fleet_config:
      editable_by: "app-team"
      confidence: 0.91
      requires_review: false
    credentials:
      editable_by: "platform-engineer"
      confidence: 0.86
      requires_review: true
    component_ports:
      editable_by: "platform-engineer"
      confidence: 0.84
      requires_review: true

  inverse_pointer_templates:
    fleet_config:
      owner: "app-team"
      confidence: 0.91
    credentials:
      owner: "platform-engineer"
      confidence: 0.86
    component_ports:
      owner: "platform-engineer"
      confidence: 0.84

  inverse_patch_reasons:
    fleet_config: "Fleet configuration (model, concurrency) is sourced from {{base_config_path}}."
    credentials: "Credential references impact platform secret management."
    component_ports: "Component port changes affect platform networking and service mesh."

  inverse_edit_hints:
    fleet_config: "Edit fleet.agent_model or fleet.max_concurrent_tasks in {{base_config_path}}."
    credentials: "Edit credentials section in {{base_config_path}} and coordinate with platform secret management."
    component_ports_base: "Edit components.controlplane.grpc_port in {{base_config_path}}."
    component_ports_overlay: "Edit component ports in {{overlay_config_path}} for environment-specific values; use {{base_config_path}} for defaults."

  # Inverse patches: the concrete WET→DRY field mappings with ownership.
  # Replaces the per-generator cases in defaultPatchesForGenerator().
  inverse_patches:
    - dry_path: "fleet.agent_model"
      wet_path: "ConfigMap/data/AGENT_MODEL"
      policy_key: "fleet_config"
      reason_key: "fleet_config"
      reason_hints:                 # Template vars for the reason string
        base_config_path: "{{base_config_path}}"
    - dry_path: "credentials.anthropic_key_ref"
      wet_path: "Secret/data/ANTHROPIC_API_KEY"
      policy_key: "credentials"
      reason_key: "credentials"
    - dry_path: "components.controlplane.grpc_port"
      wet_path: "ConfigMap/data/CP_GRPC_PORT"
      policy_key: "component_ports"
      reason_key: "component_ports"

  # Inverse edit pointers: the WET→DRY edit guidance list.
  # Replaces the per-generator cases in inversePointersForGenerator().
  inverse_pointers:
    - wet_path: "ConfigMap/data/AGENT_MODEL"
      dry_path: "fleet.agent_model"
      pointer_key: "fleet_config"
      hint_key: "fleet_config"
    - wet_path: "Secret/data/ANTHROPIC_API_KEY"
      dry_path: "credentials.anthropic_key_ref"
      pointer_key: "credentials"
      hint_key: "credentials"
    - wet_path: "ConfigMap/data/CP_GRPC_PORT"
      dry_path: "components.controlplane.grpc_port"
      pointer_key: "component_ports"
      hint_key: "component_ports_base"

# --- Path hints ---
# How to classify input files into base/overlay/workflow roles.
# Replaces the per-generator <kind>PathHintsFromInputs() functions.
# The engine uses input_role_rules to classify each input file, then
# assigns the first matching file to each hint key.
path_hints:
  defaults:
    base_config_path: "c3agent.yaml"
  mappings:
    - hint_key: "base_config_path"
      # Match inputs whose basename exactly matches one of these
      exact_basenames:
        - "c3agent.yaml"
        - "c3agent.yml"
        - "c3agent.json"
    - hint_key: "overlay_config_path"
      # Match inputs whose basename starts with this prefix
      prefix: "c3agent-"
```

### 5.2 Schema field reference

| Section | Field | Purpose | Currently lives in |
|---|---|---|---|
| Identity | `kind`, `profile`, `resource_kind`, `resource_type`, `capabilities` | Generator identity | `registry.FamilySpec` |
| Detection | `trigger_files`, `content_keywords`, `input_globs`, `confidence` | Filesystem scanning rules | `detect.go` per-generator functions |
| Contract | `input_role_rules`, `wet_targets`, `default_owner`, `role_schema_refs` | What is consumed and produced | `registry.FamilySpec` |
| Provenance | `field_origin_transform`, `field_origin_confidences`, `rendered_lineage_templates` | DRY→WET field-level lineage metadata | `registry.FamilySpec` |
| Provenance | `field_origins` | Concrete DRY→WET field mappings | `importer.go` `fieldOriginsForGenerator()` |
| Inverse | `inverse_patch_templates`, `inverse_pointer_templates`, `inverse_patch_reasons`, `inverse_edit_hints` | WET→DRY ownership and guidance metadata | `registry.FamilySpec` |
| Inverse | `inverse_patches` | Concrete WET→DRY patch entries | `importer.go` `defaultPatchesForGenerator()` |
| Inverse | `inverse_pointers` | Concrete WET→DRY edit pointer entries | `importer.go` `inversePointersForGenerator()` |
| Path hints | `defaults`, `mappings` | Base/overlay file classification | `importer.go` per-generator `<kind>PathHintsFromInputs()` |

### 5.3 What this schema unifies

Today, adding a new generator requires touching:

| File | What you add | Lines |
|---|---|---|
| `internal/model/types.go` | `GeneratorKind` constant | 1 |
| `internal/registry/registry.go` | `FamilySpec` struct literal | ~60 |
| `internal/detect/detect.go` | `detect<Kind>()` function + wire into `ScanRepo()` | ~70 |
| `internal/importer/importer.go` | 4 switch cases + path hints helper | ~120 |
| `cmd/cub-gen-style-sync/main.go` | (auto, but only if you know to run it) | 0 |
| **Total** | | **~250 lines of Go** |

With YAML bundles, adding a new generator requires:

| File | What you add |
|---|---|
| `generators/<kind>/triple.yaml` | One YAML file (see schema above) |
| `generators/<kind>/examples/` | Sample DRY input files |
| **Total** | **One YAML file + example files** |

## 6. Detection data for all 8 generators

Every existing detection function follows the same pattern. Here is the declarative data extracted from each:

| Kind | Trigger files | Content keywords | Input globs | Confidence |
|---|---|---|---|---|
| `helm` | `Chart.yaml` | *(none — presence is sufficient)* | `values*.yaml` | 0.98 |
| `score` | `score.yaml`, `score.yml` | `score.dev/`, `kind: workload` | *(trigger file only)* | 0.96 |
| `springboot` | `pom.xml`, `build.gradle`, `build.gradle.kts` + `src/main/resources/application.yaml\|yml` must exist | *(none — build file + app config is sufficient)* | `application-*.yaml`, `application-*.yml` | 0.93 |
| `backstage` | `catalog-info.yaml`, `catalog-info.yml` | `backstage.io/` AND `kind: component` | `app-config.yaml`, `app-config.yml` | 0.91 |
| `no-config-platform` | `no-config-platform.yaml`, `no-config-platform.yml`, `no-config-platform.json` | `no-config-platform` | `no-config-platform*.yaml`, `no-config-platform*.yml`, `no-config-platform*.json` | 0.90 |
| `opsworkflow` | `operations.yaml`, `operations.yml`, `workflow.yaml`, `workflow.yml` | `actions:`, `workflow:` | `operations*.yaml`, `operations*.yml`, `workflow*.yaml`, `workflow*.yml`, `actions*.yaml`, `actions*.yml` | 0.89 |
| `c3agent` | `c3agent.yaml`, `c3agent.yml`, `c3agent.json` | `fleet:`, `agent` | `c3agent*.yaml`, `c3agent*.yml`, `c3agent*.json` | 0.88 |
| `swamp` | `.swamp.yaml`, `.swamp.yml` | `swamp` | *(trigger file + `workflow-*.yaml\|yml` in same dir + children)* | 0.89 |

**Notes on special cases**:
- `helm` has no content validation (Chart.yaml is unambiguous)
- `springboot` requires a two-step detection: build file + application config in `src/main/resources/`
- `backstage` uses AND logic for content keywords (both must be present)
- `swamp` walks child directories for workflow files

These special cases need expressive detection rules in the YAML schema. The `detection` section handles this:

```yaml
# Helm: no content keywords needed
detection:
  trigger_files: ["Chart.yaml"]
  content_keywords: []            # Empty = skip content validation
  input_globs: ["values*.yaml"]
  confidence: 0.98

# Backstage: AND logic for content keywords
detection:
  trigger_files: ["catalog-info.yaml", "catalog-info.yml"]
  content_keywords_mode: "all"    # "any" (default) or "all"
  content_keywords:
    - "backstage.io/"
    - "kind: component"
  input_candidates:               # Named files to check for (not globs)
    - "app-config.yaml"
    - "app-config.yml"
  confidence: 0.91

# SpringBoot: two-step detection (build file + config file)
detection:
  trigger_files: ["pom.xml", "build.gradle", "build.gradle.kts"]
  content_keywords: []
  # Config file must exist relative to the trigger file's directory
  required_sibling: "src/main/resources/application.yaml|src/main/resources/application.yml"
  input_globs: ["src/main/resources/application-*.yaml", "src/main/resources/application-*.yml"]
  confidence: 0.93

# Swamp: recursive child walk for workflows
detection:
  trigger_files: [".swamp.yaml", ".swamp.yml"]
  content_keywords: ["swamp"]
  input_globs: ["workflow-*.yaml", "workflow-*.yml"]
  input_glob_recursive: true      # Walk child dirs for input globs
  confidence: 0.89
```

## 7. Go runtime changes

### 7.1 What changes

| Component | Current | After |
|---|---|---|
| `internal/registry/registry.go` | 510 lines of FamilySpec struct literals | YAML loader: `embed.FS` reads `generators/*/triple.yaml`, unmarshals into FamilySpec |
| `internal/detect/detect.go` | 8 per-generator functions (~580 lines) | 1 generic `detectFromSpec()` function (~80 lines) driven by `detection:` YAML section |
| `internal/importer/importer.go` | 32 switch cases (~800 lines) + 8 path hint helpers (~200 lines) | Generic functions reading from FamilySpec's new fields (field_origins, inverse_patches, inverse_pointers, path_hint_mappings) |
| `internal/model/types.go` | 8 hardcoded GeneratorKind constants | Auto-derived from loaded YAML `kind` fields (or keep constants for backward compat + type safety) |
| `cmd/cub-gen-style-sync/` | Reads Go registry, writes projections | Reads YAML bundles, writes README.md per generator |

### 7.2 What stays in Go

| Component | Why it stays |
|---|---|
| `internal/importer/importer.go` ImportRepo/ImportDetection | Core engine logic: unit creation, link creation, contract assembly |
| `internal/contracts/` | Conformance testing framework |
| `cmd/cub-gen/main.go` | CLI commands, output formatting |
| Rendering logic | How WET manifests are actually generated |
| `renderTargetTemplate()` | Template variable substitution (used by inverse reasons, edit hints) |

### 7.3 New types needed

```go
// Added to FamilySpec (or a wrapper) to hold YAML-only data:
type DetectionSpec struct {
    TriggerFiles         []string `yaml:"trigger_files"`
    ContentKeywords      []string `yaml:"content_keywords"`
    ContentKeywordsMode  string   `yaml:"content_keywords_mode"` // "any" or "all"
    InputGlobs           []string `yaml:"input_globs"`
    InputCandidates      []string `yaml:"input_candidates"`
    InputGlobRecursive   bool     `yaml:"input_glob_recursive"`
    RequiredSibling      string   `yaml:"required_sibling"`
    Confidence           float64  `yaml:"confidence"`
}

type FieldOriginSpec struct {
    DryPath       string `yaml:"dry_path"`
    WetPath       string `yaml:"wet_path"`
    SourceHint    string `yaml:"source_hint"`
    ConfidenceKey string `yaml:"confidence_key"`
    Transform     string `yaml:"transform"`     // "base" or "overlay"
    Optional      bool   `yaml:"optional"`
}

type InversePatchSpec struct {
    DryPath     string            `yaml:"dry_path"`
    WetPath     string            `yaml:"wet_path"`
    PolicyKey   string            `yaml:"policy_key"`
    ReasonKey   string            `yaml:"reason_key"`
    ReasonHints map[string]string `yaml:"reason_hints"`
}

type InversePointerSpec struct {
    WetPath    string `yaml:"wet_path"`
    DryPath    string `yaml:"dry_path"`
    PointerKey string `yaml:"pointer_key"`
    HintKey    string `yaml:"hint_key"`
}

type PathHintMapping struct {
    HintKey        string   `yaml:"hint_key"`
    ExactBasenames []string `yaml:"exact_basenames"`
    Prefix         string   `yaml:"prefix"`
}
```

### 7.4 Generic detection function (pseudocode)

```go
func detectFromSpec(repo string, spec GeneratorBundle) ([]model.GeneratorDetection, error) {
    detected := make(map[string]model.GeneratorDetection)
    err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
        // Skip dirs
        if d.IsDir() && shouldSkipDir(d.Name()) { return filepath.SkipDir }
        if d.IsDir() { return nil }

        // Check trigger files
        name := strings.ToLower(d.Name())
        if !matchesTriggerFiles(name, spec.Detection.TriggerFiles) { return nil }

        // Check required sibling (springboot case)
        if spec.Detection.RequiredSibling != "" {
            if !siblingExists(filepath.Dir(path), spec.Detection.RequiredSibling) { return nil }
        }

        // Content keyword validation
        if len(spec.Detection.ContentKeywords) > 0 {
            content := readFileLower(path)
            if !matchesKeywords(content, spec.Detection.ContentKeywords, spec.Detection.ContentKeywordsMode) {
                return nil
            }
        }

        // Gather inputs via globs
        root := filepath.Dir(path)
        inputs := gatherInputs(repo, root, spec.Detection.InputGlobs, spec.Detection.InputGlobRecursive)

        // Also check named candidates
        for _, candidate := range spec.Detection.InputCandidates {
            if fileExists(filepath.Join(root, candidate)) {
                inputs = append(inputs, relPath(repo, root, candidate))
            }
        }

        relRoot := relPath(repo, root, "")
        detected[spec.Kind+":"+relRoot] = model.GeneratorDetection{
            ID:         "gen_" + shortID(spec.Kind+":"+relRoot),
            Kind:       model.GeneratorKind(spec.Kind),
            Profile:    spec.Profile,
            Name:       filepath.Base(root),
            Root:       relRoot,
            Inputs:     sortedUnique(inputs),
            Confidence: spec.Detection.Confidence,
        }
        return nil
    })
    return mapValuesSorted(detected), err
}
```

### 7.5 Generic field origins function (pseudocode)

```go
func fieldOriginsFromSpec(spec GeneratorBundle, g model.GeneratorDetection) []model.FieldOrigin {
    hints := resolvePathHints(spec.PathHints, g.Inputs)
    var origins []model.FieldOrigin

    for _, fo := range spec.Provenance.FieldOrigins {
        sourcePath := hints[fo.SourceHint]
        if fo.Optional && sourcePath == "" { continue }

        transform := spec.Provenance.FieldOriginTransform
        if fo.Transform == "overlay" {
            transform = spec.Provenance.FieldOriginOverlayTransform
        }

        origins = append(origins, model.FieldOrigin{
            DryPath:    fo.DryPath,
            WetPath:    fo.WetPath,
            SourcePath: sourcePath,
            Transform:  transform,
            Confidence: spec.Provenance.FieldOriginConfidences[fo.ConfidenceKey],
        })
    }
    return origins
}
```

## 8. Migration path

### Phase 0: Decide (required precondition)

1. Choose Go-canonical or YAML-canonical near-term path.
2. Record decision in `docs/decisions/` with explicit scope for the next milestone.
3. Confirm helm/score.dev/springboot priority is preserved in that scope.

**Deliverable**: Approved decision memo. No implementation starts before this checkpoint.

### Phase 1: Extract (non-breaking)

1. Create `generators/` directory with one subdirectory per generator kind
2. Write `triple.yaml` for all 8 generators (the data already exists in Go — just serialize it)
3. Move example files from `examples/<kind>/` to `generators/<kind>/examples/`
4. Keep Go registry and detection functions unchanged
5. Add a conformance test: load each `triple.yaml`, compare against Go FamilySpec — must be identical
6. CI enforces: YAML ↔ Go parity

**Deliverable**: 8 YAML bundles that are verified-identical to the Go source of truth. This is a checkpoint — everything still works the old way.

### Phase 2: Load (switch source of truth)

1. Add YAML loader: `embed.FS` reads `generators/*/triple.yaml` at init
2. Registry populates FamilySpec from YAML instead of struct literals
3. Delete the 510 lines of Go struct literals
4. Add generic detection function, wire it into `ScanRepo()`
5. Delete the 8 per-generator detection functions
6. Add generic field-origins/inverse-patches/inverse-pointers functions
7. Delete the 32 switch cases and 8 path-hint helpers
8. Update `model/types.go` to derive GeneratorKind constants from loaded YAML (or keep constants with a validation check)
9. Add external bundle loading (`--generators-dir`) or explicitly defer it with documented recompilation constraints
10. All existing golden tests must pass unchanged

**Deliverable**: Go engine is now generic. YAML is the source of truth. Existing behavior is identical (verified by golden tests).

### Phase 3: Generate (auto-docs)

1. Adapt `cmd/cub-gen-style-sync` to read from `generators/*/triple.yaml` instead of Go registry
2. Generate `README.md` inside each bundle directory (Mermaid diagram + tables)
3. Keep or remove `docs/triple-styles/` (the bundle README.md replaces it)
4. CI check: `go run ./cmd/cub-gen-style-sync && git diff --exit-code generators/*/README.md`

**Deliverable**: Each bundle is self-documenting with auto-generated docs.

### Phase 4: Author (AI-assisted)

1. Add `cub-gen init <kind>` command that:
   - Takes a directory of sample config files
   - Calls an LLM (or uses heuristics) to generate a `triple.yaml`
   - Scaffolds the bundle directory structure
   - Runs validation against the YAML schema
2. Document the AI authoring prompt/workflow
3. Add a JSON Schema for `triple.yaml` so editors provide autocomplete

**Deliverable**: Platform owners can create new generators without knowing Go exists.

## 9. AI authoring prompt

When AI generates a `triple.yaml`, it needs:

### Inputs the AI receives

1. **Sample config files** from the platform (e.g., `c3agent.yaml`, `c3agent-prod.yaml`)
2. **Desired WET output description** — what Kubernetes resources should be generated
3. **Ownership model** — which team owns which fields

### What the AI produces

1. `triple.yaml` with all sections filled in
2. `examples/` directory with the sample files
3. Explanation of each section for the platform owner to review

### Key decisions the AI makes

| Decision | How |
|---|---|
| Trigger file names | From the sample file names |
| Content keywords | From distinctive strings in the sample files |
| Input role classification | Base = exact filename match, overlay = prefix match |
| Field mappings (DRY→WET) | From the platform owner's description of desired output |
| Confidence scores | Default 0.85 for base fields, 0.80 for overlays, adjustable by owner |
| Ownership split | From the ownership model description |
| Review requirements | Platform-engineer fields default to `requires_review: true` |

### What the platform owner reviews

The AI should flag these for human attention:

- [ ] Are the trigger file names complete? Any other file patterns?
- [ ] Are the content keywords specific enough? (Too broad = false positives)
- [ ] Is the base/overlay split correct?
- [ ] Are the field mappings accurate? Any missing fields?
- [ ] Is the ownership split right? (app-team vs platform-engineer)
- [ ] Do the confidence scores reflect actual certainty?
- [ ] Are there fields that need review gates?

## 10. Validation

### 10.1 Schema validation

A JSON Schema for `triple.yaml` enables:
- Editor autocomplete (VS Code, JetBrains)
- CI validation of YAML structure
- Clear error messages for malformed bundles

### 10.2 Runtime validation

When `cub-gen` loads a `triple.yaml`, it validates:
- `kind` is non-empty and unique across all bundles
- `detection.trigger_files` is non-empty
- `detection.confidence` is 0.0–1.0
- All `confidence_key` references in `field_origins` exist in `field_origin_confidences`
- All `policy_key` references in `inverse_patches` exist in `inverse_patch_templates`
- All `pointer_key` references in `inverse_pointers` exist in `inverse_pointer_templates`
- All `reason_key` references in `inverse_patches` exist in `inverse_patch_reasons`
- All `hint_key` references in `inverse_pointers` exist in `inverse_edit_hints`
- All `source_hint` references in `field_origins` exist in `path_hints.mappings[].hint_key`
- `wet_targets` is non-empty

### 10.3 Parity testing

Existing golden file tests verify the full pipeline output and must not change:
- `gitops-discover-*.golden.txt` — detection output
- `gitops-import-*.golden.json` — import output with contracts, provenance, inverse plans
- `generators-*.golden.json` — generator listing output

If any golden file changes, the migration has a bug.

## 11. Testing strategy

| Test | What it verifies | Phase |
|---|---|---|
| YAML ↔ Go parity | Each triple.yaml produces identical FamilySpec to Go struct literal | Phase 1 |
| Golden file stability | All existing golden tests pass with no output changes | Phase 2 |
| Schema validation | triple.yaml passes JSON Schema validation | Phase 2 |
| Cross-reference validation | All key references (confidence, policy, pointer, hint, source) resolve | Phase 2 |
| Detection parity | Generic detector produces same results as per-generator functions | Phase 2 |
| Import parity | Generic importer produces same results as switch-case importer | Phase 2 |
| Bundle completeness | Every generator has triple.yaml + examples/ + README.md | Phase 3 |
| Round-trip | Load YAML → serialize → compare with original (no data loss) | Phase 2 |

## 12. Relationship to existing work

### What ships already (PRs #112–#116)

| PR | What it delivered | Relationship to this spec |
|---|---|---|
| #112 | `--details` expanded payload | The payload structure stays the same; data source changes from Go → YAML |
| #113 | `--markdown --details` introspection | Rendering logic stays; data source changes |
| #114 | `docs/triple-styles/` with 3 style projections | Style A YAML is close to the `triple.yaml` schema; style B/C become the auto-generated README.md |
| #115 | `cmd/cub-gen-style-sync` tool | Adapted to read from YAML bundles instead of Go registry |
| #116 | Enhanced gitops discover/import text, per-generator Mermaid diagrams | Text output stays; Mermaid generation moves to README.md |

### Decision doc superseded

`docs/decisions/2026-03-08-go-canonical-generator-triples.md` declared Go as the canonical source. This spec supersedes that decision: YAML becomes canonical, Go becomes the loading engine.

A new decision doc should be written:
```
docs/decisions/2026-03-XX-yaml-canonical-generator-bundles.md
```

## 13. Open questions

1. **GeneratorKind constants**: Keep `model.GeneratorHelm` etc. as constants (validated against loaded YAML), or derive dynamically from YAML `kind` fields? Constants give compile-time safety but require Go changes for new generators. Dynamic gives full YAML-only authoring but loses type checking.

   **Recommendation**: Phase 2 keeps constants + validation. Phase 4 makes them dynamic with a registry lookup function.

2. **Score path hints**: The `score` generator reads actual file content to extract container names and variable names (`scorePathHintsFromInputs` reads YAML and inspects keys). This is runtime-dependent, not declarative. Options:
   - Keep a small Go hook for score's content-aware hints
   - Add a `path_hints.content_extraction` section to the YAML schema
   - Accept that score's hints are slightly less accurate without content inspection

   **Recommendation**: Keep a Go hook for score in Phase 2. Design content extraction in Phase 4 if needed.

3. **SpringBoot detection**: The two-step detection (build file in root, config file in `src/main/resources/`) is more complex than the other 7 generators. The `required_sibling` field handles this, but it's a one-off.

   **Recommendation**: `required_sibling` is fine. It's declarative and handles the springboot case cleanly.

4. **embed.FS vs external directory**: Should `generators/` be embedded in the binary (immutable, single-binary distribution) or loaded from disk (editable without recompilation)?

   **Recommendation**: `embed.FS` for built-in generators + optional `--generators-dir` flag for loading external bundles from disk. This allows both distribution models.

## 14. File changes summary

### New files

| File | Purpose |
|---|---|
| `generators/<kind>/triple.yaml` (×8) | Source of truth for each generator |
| `generators/<kind>/examples/` (×8) | Sample DRY input files (moved from `examples/`) |
| `generators/<kind>/README.md` (×8) | Auto-generated documentation |
| `internal/registry/loader.go` | YAML loader for generator bundles |
| `internal/detect/generic.go` | Generic detection function driven by YAML specs |
| `internal/importer/generic.go` | Generic field-origins/inverse-patches/inverse-pointers |
| `docs/decisions/2026-03-XX-yaml-canonical-generator-bundles.md` | New decision doc |

### Modified files

| File | Change |
|---|---|
| `internal/registry/registry.go` | Delete struct literals, add YAML init loading |
| `internal/detect/detect.go` | Delete 8 per-generator functions, call generic function per loaded spec |
| `internal/importer/importer.go` | Delete 32 switch cases + 8 path-hint helpers, call generic functions |
| `internal/model/types.go` | May keep constants or make dynamic |
| `cmd/cub-gen-style-sync/main.go` | Read from YAML bundles, generate README.md per bundle |
| `HANDOVER.md` | Updated with new direction |

### Deleted files

| File | Why |
|---|---|
| `docs/triple-styles/style-a-yaml/*.yaml` | Replaced by `generators/<kind>/triple.yaml` |
| `docs/triple-styles/style-b-markdown/*.md` | Replaced by `generators/<kind>/README.md` |
| `docs/triple-styles/style-c-yaml-plus-docs/` | Replaced by the bundle structure |
| `examples/<kind>/` (×8) | Moved to `generators/<kind>/examples/` |

## 15. Success criteria

1. **A platform owner can create a new generator by adding a YAML file** — no Go, no recompilation
2. **All existing tests pass unchanged** — golden file output is identical
3. **AI can generate a valid triple.yaml** from sample config files
4. **Each generator bundle is self-documenting** — README.md with Mermaid diagram, ownership tables, field maps
5. **CI catches invalid YAML** — schema validation, cross-reference validation, projection drift

---

*This spec was written on 2026-03-08 as a handoff for implementing the YAML generator bundle architecture.*
