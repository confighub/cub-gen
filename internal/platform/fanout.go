package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
	"github.com/confighub/cub-gen/internal/registry"
)

const FanoutSchemaVersion = "cub.confighub.io/variant-fanout/v1"

type FanoutOptions struct {
	GeneratedAt   string
	VariantFilter string
}

type FanoutResult struct {
	SchemaVersion string          `json:"schema_version"`
	ManifestPath  string          `json:"manifest_path"`
	Name          string          `json:"name"`
	Space         string          `json:"space"`
	Ref           string          `json:"ref"`
	GeneratedAt   string          `json:"generated_at"`
	Summary       FanoutSummary   `json:"summary"`
	Variants      []FanoutVariant `json:"variants,omitempty"`
	Diagnostics   []Diagnostic    `json:"diagnostics,omitempty"`
}

type FanoutSummary struct {
	VariantCount    int `json:"variant_count"`
	BundleCount     int `json:"bundle_count"`
	ComponentCount  int `json:"component_count"`
	TargetCount     int `json:"target_count"`
	DiagnosticCount int `json:"diagnostic_count"`
}

type FanoutVariant struct {
	VariantID         string               `json:"variant_id"`
	Component         string               `json:"component"`
	Variant           string               `json:"variant"`
	Target            string               `json:"target,omitempty"`
	RepoID            string               `json:"repo_id"`
	RepoPath          string               `json:"repo_path"`
	SharedInputs      []string             `json:"shared_inputs,omitempty"`
	VariantInputs     []string             `json:"variant_inputs,omitempty"`
	GeneratorProfiles []string             `json:"generator_profiles,omitempty"`
	ChangeID          string               `json:"change_id,omitempty"`
	BundleDigest      string               `json:"bundle_digest,omitempty"`
	BundleValid       bool                 `json:"bundle_valid"`
	Bundle            publish.ChangeBundle `json:"bundle"`
}

func FanoutManifest(manifestPath string, opts FanoutOptions) (FanoutResult, error) {
	manifest, absManifestPath, err := LoadManifest(manifestPath)
	if err != nil {
		return FanoutResult{}, err
	}
	return BuildFanout(absManifestPath, manifest, opts)
}

func BuildFanout(absManifestPath string, manifest Manifest, opts FanoutOptions) (FanoutResult, error) {
	at, err := fanoutGeneratedAt(opts.GeneratedAt)
	if err != nil {
		return FanoutResult{}, err
	}

	variants, err := ResolveFanoutVariants(manifest)
	if err != nil {
		return FanoutResult{}, err
	}
	if filter := strings.TrimSpace(opts.VariantFilter); filter != "" {
		variants = filterFanoutVariants(variants, filter)
		if len(variants) == 0 {
			return FanoutResult{}, fmt.Errorf("variant filter %q matched no variants", filter)
		}
	}

	baseDir := filepath.Dir(absManifestPath)
	repos := normalizeRepos(manifest.Repos)
	repoByID := map[string]ManifestRepo{}
	for _, repo := range repos {
		repoByID[repo.ID] = repo
	}

	result := FanoutResult{
		SchemaVersion: FanoutSchemaVersion,
		ManifestPath:  filepath.ToSlash(filepath.Clean(absManifestPath)),
		Name:          manifest.Name,
		Space:         manifest.Space,
		Ref:           manifest.Ref,
		GeneratedAt:   at.UTC().Format(time.RFC3339),
	}
	componentSet := map[string]struct{}{}
	targetSet := map[string]struct{}{}

	for _, spec := range variants {
		repo, ok := repoByID[spec.Repo]
		if !ok {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "missing_variant_repo",
				RepoID:   spec.Repo,
				Message:  fmt.Sprintf("variant %s references repo %q that is not declared", spec.ID, spec.Repo),
			})
			continue
		}
		target := strings.TrimSpace(spec.Target)
		if target == "" {
			target = strings.TrimSpace(repo.Target)
		}
		if spec.Component != "" {
			componentSet[spec.Component] = struct{}{}
		}
		if target != "" {
			targetSet[target] = struct{}{}
		}
		if strings.TrimSpace(repo.Path) == "" {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "missing_repo_path",
				RepoID:   repo.ID,
				Message:  fmt.Sprintf("variant %s repo has no path", spec.ID),
			})
			continue
		}
		repoPath := normalizeRelPath(repo.Path)
		absRepo := filepath.Join(baseDir, filepath.FromSlash(repoPath))
		info, statErr := os.Stat(absRepo)
		if statErr != nil || !info.IsDir() {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "missing_repo",
				RepoID:   repo.ID,
				Path:     repoPath,
				Message:  fmt.Sprintf("variant %s repo path does not exist or is not a directory", spec.ID),
			})
			continue
		}

		helmOverrides, parseErr := importer.ParseHelmCLIOverrides(spec.HelmSet, spec.HelmSetString, spec.HelmSetFile)
		if parseErr != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "invalid_variant_override",
				RepoID:   repo.ID,
				Path:     repoPath,
				Message:  fmt.Sprintf("variant %s Helm override is invalid: %v", spec.ID, parseErr),
			})
			continue
		}
		imported, importErr := importer.ImportRepoWithOptions(absRepo, manifest.Ref, manifest.Space, importer.ImportOptions{
			HelmCLIOverrides: helmOverrides,
		})
		if importErr != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "import_failed",
				RepoID:   repo.ID,
				Path:     repoPath,
				Message:  fmt.Sprintf("variant %s import failed: %v", spec.ID, importErr),
			})
			continue
		}
		if len(imported.Detection.Generators) == 0 {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "warning",
				Code:     "unsupported_generator",
				RepoID:   repo.ID,
				Path:     repoPath,
				Message:  fmt.Sprintf("variant %s repo exists but no supported generator was detected", spec.ID),
			})
			continue
		}
		normalizeFanoutImportTimestamps(&imported, result.GeneratedAt)

		flow := fanoutImportFlow(spec, absRepo, imported, helmOverrides)
		bundle := publish.BuildBundleAt(flow, at)
		bundleValid := publish.VerifyBundle(bundle) == nil
		result.Variants = append(result.Variants, FanoutVariant{
			VariantID:         spec.ID,
			Component:         spec.Component,
			Variant:           spec.Variant,
			Target:            target,
			RepoID:            repo.ID,
			RepoPath:          repoPath,
			SharedInputs:      normalizedStrings(spec.SharedInputs),
			VariantInputs:     normalizedStrings(spec.VariantInputs),
			GeneratorProfiles: append([]string(nil), bundle.Summary.GeneratorProfiles...),
			ChangeID:          bundle.ChangeID,
			BundleDigest:      bundle.BundleDigest,
			BundleValid:       bundleValid,
			Bundle:            bundle,
		})
		if !bundleValid {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "invalid_bundle",
				RepoID:   repo.ID,
				Path:     repoPath,
				Message:  fmt.Sprintf("variant %s emitted a bundle that failed verification", spec.ID),
			})
		}
	}

	sortFanout(&result)
	result.Summary = FanoutSummary{
		VariantCount:    len(variants),
		BundleCount:     len(result.Variants),
		ComponentCount:  len(componentSet),
		TargetCount:     len(targetSet),
		DiagnosticCount: len(result.Diagnostics),
	}
	return result, nil
}

func normalizeFanoutImportTimestamps(imported *model.ImportResult, at string) {
	imported.ImportedAt = at
	imported.Detection.DetectedAt = at
	for i := range imported.Provenance {
		imported.Provenance[i].RenderedAt = at
	}
	for i := range imported.InversePlans {
		imported.InversePlans[i].CreatedAt = at
	}
}

func ResolveFanoutVariants(manifest Manifest) ([]ManifestVariant, error) {
	if len(manifest.Variants) > 0 {
		return normalizeExplicitFanoutVariants(manifest.Variants)
	}

	repos := normalizeRepos(manifest.Repos)
	byKey := map[string][]ManifestRepo{}
	for _, repo := range repos {
		component := strings.TrimSpace(repo.Component)
		variant := strings.TrimSpace(repo.Variant)
		if component == "" || variant == "" {
			continue
		}
		key := component + "/" + variant
		byKey[key] = append(byKey[key], repo)
	}
	if len(byKey) == 0 {
		return nil, errors.New("no fanout variants declared; add manifest variants or repo component/variant metadata")
	}

	var variants []ManifestVariant
	for key, entries := range byKey {
		if len(entries) > 1 {
			ids := make([]string, 0, len(entries))
			for _, entry := range entries {
				ids = append(ids, entry.ID)
			}
			sort.Strings(ids)
			return nil, fmt.Errorf("ambiguous variant source for %s: repos %s declare the same component/variant; add manifest variants explicitly", key, strings.Join(ids, ", "))
		}
		repo := entries[0]
		variants = append(variants, ManifestVariant{
			ID:        key,
			Component: strings.TrimSpace(repo.Component),
			Variant:   strings.TrimSpace(repo.Variant),
			Target:    strings.TrimSpace(repo.Target),
			Repo:      strings.TrimSpace(repo.ID),
		})
	}
	return sortedFanoutVariants(variants), nil
}

func normalizeExplicitFanoutVariants(in []ManifestVariant) ([]ManifestVariant, error) {
	out := make([]ManifestVariant, 0, len(in))
	seen := map[string]struct{}{}
	for i, raw := range in {
		spec := raw
		spec.ID = strings.TrimSpace(spec.ID)
		spec.Component = strings.TrimSpace(spec.Component)
		spec.Variant = strings.TrimSpace(spec.Variant)
		spec.Target = strings.TrimSpace(spec.Target)
		spec.Repo = strings.TrimSpace(spec.Repo)
		if spec.Component == "" {
			return nil, fmt.Errorf("variant %d component is required", i+1)
		}
		if spec.Variant == "" {
			return nil, fmt.Errorf("variant %d variant is required", i+1)
		}
		if spec.Repo == "" {
			return nil, fmt.Errorf("variant %s/%s repo is required", spec.Component, spec.Variant)
		}
		if spec.ID == "" {
			spec.ID = spec.Component + "/" + spec.Variant
		}
		if _, ok := seen[spec.ID]; ok {
			return nil, fmt.Errorf("duplicate variant id %q", spec.ID)
		}
		seen[spec.ID] = struct{}{}
		spec.SharedInputs = normalizedStrings(spec.SharedInputs)
		spec.VariantInputs = normalizedStrings(spec.VariantInputs)
		spec.HelmSet = normalizedStrings(spec.HelmSet)
		spec.HelmSetString = normalizedStrings(spec.HelmSetString)
		spec.HelmSetFile = normalizedStrings(spec.HelmSetFile)
		out = append(out, spec)
	}
	return sortedFanoutVariants(out), nil
}

func sortedFanoutVariants(in []ManifestVariant) []ManifestVariant {
	out := append([]ManifestVariant(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Component != out[j].Component {
			return out[i].Component < out[j].Component
		}
		if out[i].Variant != out[j].Variant {
			return out[i].Variant < out[j].Variant
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func filterFanoutVariants(in []ManifestVariant, filter string) []ManifestVariant {
	var out []ManifestVariant
	for _, spec := range in {
		if spec.ID == filter || spec.Variant == filter || spec.Component+"/"+spec.Variant == filter {
			out = append(out, spec)
		}
	}
	return out
}

func fanoutGeneratedAt(raw string) (time.Time, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return time.Now().UTC(), nil
	}
	at, err := time.Parse(time.RFC3339, clean)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse generated timestamp: %w", err)
	}
	return at.UTC(), nil
}

func fanoutImportFlow(
	spec ManifestVariant,
	absRepo string,
	imported model.ImportResult,
	helmOverrides []model.HelmCLIOverride,
) gitopsflow.ImportFlowResult {
	dryUnits, wetUnits, generatorUnits := fanoutSplitUnits(imported.Units)
	discovered := fanoutDiscoveredResources(imported.Detection.Generators)
	return gitopsflow.ImportFlowResult{
		Space:              imported.Space,
		TargetSlug:         spec.ID,
		TargetPath:         filepath.ToSlash(filepath.Clean(absRepo)),
		RenderTargetSlug:   spec.ID,
		RenderTargetPath:   filepath.ToSlash(filepath.Clean(absRepo)),
		Ref:                imported.Ref,
		HelmCLIOverrides:   append([]model.HelmCLIOverride(nil), helmOverrides...),
		DiscoverUnitSlug:   "fanout-" + sanitizeID(spec.ID),
		ImportedAt:         imported.ImportedAt,
		Discovered:         discovered,
		Chains:             append([]model.GeneratorChain(nil), imported.Detection.Chains...),
		ChainSummaries:     append([]model.GeneratorChainSummary(nil), imported.Detection.ChainSummaries...),
		DryUnits:           dryUnits,
		WetUnits:           wetUnits,
		GeneratorUnits:     generatorUnits,
		Links:              append([]model.UnitLink(nil), imported.Links...),
		Contracts:          append([]model.GeneratorContract(nil), imported.GeneratorContracts...),
		Provenance:         append([]model.ProvenanceRecord(nil), imported.Provenance...),
		InversePlans:       append([]model.InverseTransformPlan(nil), imported.InversePlans...),
		DryInputs:          append([]model.DryInputRef(nil), imported.DryInputs...),
		WetManifestTargets: append([]model.WetManifestTarget(nil), imported.WetManifestTargets...),
	}
}

func fanoutDiscoveredResources(detections []model.GeneratorDetection) []gitopsflow.DiscoveredResource {
	resources := make([]gitopsflow.DiscoveredResource, 0, len(detections))
	for _, g := range detections {
		kind := registry.ResourceKind(g.Kind)
		resourceType := registry.ResourceType(g.Kind)
		body := fmt.Sprintf("kind: %s\nmetadata:\n  name: %s\nspec:\n  root: %s\n", kind, g.Name, g.Root)
		resources = append(resources, gitopsflow.DiscoveredResource{
			GeneratorID:      g.ID,
			GeneratorProfile: g.Profile,
			ResourceName:     g.Name,
			ResourceKind:     kind,
			ResourceType:     resourceType,
			ResourceBody:     body,
			GeneratorKind:    string(g.Kind),
			Root:             g.Root,
			Inputs:           append([]string(nil), g.Inputs...),
		})
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].ResourceType != resources[j].ResourceType {
			return resources[i].ResourceType < resources[j].ResourceType
		}
		return resources[i].ResourceName < resources[j].ResourceName
	})
	return resources
}

func fanoutSplitUnits(units []model.UnitRef) (dry, wet, generator []model.UnitRef) {
	for _, u := range units {
		switch u.Layer {
		case "dry":
			dry = append(dry, u)
		case "wet":
			wet = append(wet, u)
		case "generator":
			generator = append(generator, u)
		}
	}
	return dry, wet, generator
}

func sortFanout(result *FanoutResult) {
	sort.Slice(result.Variants, func(i, j int) bool {
		return result.Variants[i].VariantID < result.Variants[j].VariantID
	})
	sort.Slice(result.Diagnostics, func(i, j int) bool {
		left := result.Diagnostics[i]
		right := result.Diagnostics[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.RepoID != right.RepoID {
			return left.RepoID < right.RepoID
		}
		return left.Message < right.Message
	})
}

func normalizedStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		clean := strings.TrimSpace(value)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}
