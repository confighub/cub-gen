# Custom Generator Onboarding

This guide is for platform owners who want to add `cub-gen` support for their
own platform framework, internal DSL, or custom toolchain.

## When you need a custom generator

You need a custom generator when:

- Your platform has its own config format (not Helm, Score, Spring Boot, etc.)
- Your internal DSL compiles to Kubernetes manifests
- Your platform framework uses proprietary overlays or templating
- You want field-origin tracing and governance for your custom stack

## Two paths to custom generators

| Path | Who it's for | Effort |
|------|--------------|--------|
| **Fork and extend** | Teams with Go expertise who can maintain a fork | Medium |
| **Request inclusion** | Teams whose generator would benefit the community | Low (if accepted) |

Currently, generators must be compiled into `cub-gen`. A YAML-based extension
mechanism is planned but not yet available.

## Fork and extend (recommended for internal platforms)

### Step 1: Understand the model

Every generator in `cub-gen` has three parts:

| Part | Purpose | Location |
|------|---------|----------|
| **Detector** | Finds your config files and returns confidence | `internal/detect/detect.go` |
| **FamilySpec** | Declares metadata, capabilities, field mappings | `internal/registry/registry.go` |
| **Example** | Proves the generator works, documents usage | `examples/<your-generator>/` |

### Step 2: Add detection logic

Your detector must:

- Walk the input directory for signature files
- Validate file content (not just filename)
- Return a `DetectionResult` with kind, confidence, inputs list, and root path
- Be deterministic — same input always produces same result

Example detector structure:

```go
func detectMyPlatform(root string) ([]DetectionResult, error) {
    // 1. Walk for signature file (e.g., "my-platform.yaml")
    // 2. Validate content (e.g., contains "apiVersion: myplatform/v1")
    // 3. Return DetectionResult with confidence 0.85-0.98
}
```

### Step 3: Register the generator family

Add a `FamilySpec` in `internal/registry/registry.go`:

| Field | Purpose |
|-------|---------|
| `Profile` | User-facing name (e.g., `"my-platform-paas"`) |
| `Kind` | Internal key (e.g., `"myplatform"`) |
| `ResourceKind`, `ResourceType` | Primary Kubernetes resource |
| `Capabilities` | What it can do (`render-manifests`, etc.) |
| `InversePatchTemplates` | Editable fields with ownership and confidence |
| `FieldOriginConfidences` | Confidence levels for field-origin tracing |
| `RenderedLineageTemplates` | Expected WET output targets |

### Step 4: Create an example

Create `examples/my-platform/` with:

```
examples/my-platform/
  my-platform.yaml           # DRY source
  my-platform-prod.yaml      # Production overlay
  platform/
    base/
      runtime-policy.yaml    # Platform contracts
  docs/
    user-stories.md          # 3-4 narrative turns
  README.md                  # Follows universal contract
```

Your README must satisfy the [Universal Example Contract](example-checklist.md).

### Step 5: Add tests and validate

```bash
# Generate golden files
UPDATE_GOLDEN=1 go test ./cmd/cub-gen -run 'TestGitOpsParityGolden.*MyPlatform' -count=1 -v

# Run full validation
go build ./cmd/cub-gen
go test ./...
go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v
```

### Step 6: Maintain your fork

- Rebase periodically on upstream `main`
- Watch for registry contract changes
- Run `make ci` after each rebase

## Request inclusion (for community-relevant generators)

If your generator would benefit others (not just your org), consider contributing upstream.

Requirements for inclusion:

1. The platform/tool has public documentation
2. The generator is useful beyond your organization
3. You can provide ongoing maintenance support
4. The example meets the universal contract standard

Open an issue at [github.com/confighub/cub-gen](https://github.com/confighub/cub-gen)
with:

- Platform/tool name and public docs link
- Why it benefits the community
- Your maintenance commitment

## Kubara-like / layered platform frameworks

If your platform has multiple generation hops (overlays, ApplicationSets,
umbrella charts), your generator needs to:

1. **Trace the full chain**: From cluster labels through overlays to deployed resources
2. **Declare layer ownership**: Which invariants are enforced at which layer
3. **Surface provenance**: How to answer "why does this cluster have this value?"

See `examples/helm-paas/` for layered provenance patterns. Your example README
must include sections 8 and 9 from the universal contract:

- **Section 8**: Show the generation chain
- **Section 9**: Explain the ownership boundary

## What's not yet available

| Feature | Status |
|---------|--------|
| YAML-based generator definitions | Planned, decision pending |
| Runtime plugin loading | Not planned |
| External registry server | Not planned |

For now, custom generators require Go code and recompilation.

## Getting help

- [CONTRIBUTING.md](../../CONTRIBUTING.md) — full technical details
- [Example checklist](example-checklist.md) — universal contract requirements
- [Generator PRD](../agentic-gitops/02-design/10-generators-prd.md) — design context
