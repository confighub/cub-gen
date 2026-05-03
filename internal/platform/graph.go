package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	ManifestSchemaVersion = "cub.confighub.io/platform-import-manifest/v1"
	GraphSchemaVersion    = "cub.confighub.io/platform-import-graph/v1"
)

type Manifest struct {
	SchemaVersion string               `json:"schema_version" yaml:"schema_version"`
	Name          string               `json:"name" yaml:"name"`
	Space         string               `json:"space,omitempty" yaml:"space,omitempty"`
	Ref           string               `json:"ref,omitempty" yaml:"ref,omitempty"`
	Repos         []ManifestRepo       `json:"repos" yaml:"repos"`
	Variants      []ManifestVariant    `json:"variants,omitempty" yaml:"variants,omitempty"`
	Adaptations   []ManifestAdaptation `json:"adaptations,omitempty" yaml:"adaptations,omitempty"`
}

type ManifestRepo struct {
	ID          string `json:"id" yaml:"id"`
	Role        string `json:"role" yaml:"role"`
	Path        string `json:"path" yaml:"path"`
	Owner       string `json:"owner,omitempty" yaml:"owner,omitempty"`
	Component   string `json:"component,omitempty" yaml:"component,omitempty"`
	Variant     string `json:"variant,omitempty" yaml:"variant,omitempty"`
	VariantKind string `json:"variant_kind,omitempty" yaml:"variant_kind,omitempty"`
	Target      string `json:"target,omitempty" yaml:"target,omitempty"`
}

type ManifestVariant struct {
	ID            string   `json:"id,omitempty" yaml:"id,omitempty"`
	Component     string   `json:"component" yaml:"component"`
	Variant       string   `json:"variant" yaml:"variant"`
	VariantKind   string   `json:"variant_kind,omitempty" yaml:"variant_kind,omitempty"`
	Target        string   `json:"target,omitempty" yaml:"target,omitempty"`
	Repo          string   `json:"repo" yaml:"repo"`
	SharedInputs  []string `json:"shared_inputs,omitempty" yaml:"shared_inputs,omitempty"`
	VariantInputs []string `json:"variant_inputs,omitempty" yaml:"variant_inputs,omitempty"`
	HelmSet       []string `json:"helm_set,omitempty" yaml:"helm_set,omitempty"`
	HelmSetString []string `json:"helm_set_string,omitempty" yaml:"helm_set_string,omitempty"`
	HelmSetFile   []string `json:"helm_set_file,omitempty" yaml:"helm_set_file,omitempty"`
}

type ManifestAdaptation struct {
	ID           string                          `json:"id,omitempty" yaml:"id,omitempty"`
	Component    string                          `json:"component,omitempty" yaml:"component,omitempty"`
	Variant      string                          `json:"variant,omitempty" yaml:"variant,omitempty"`
	Repo         string                          `json:"repo" yaml:"repo"`
	BaseVariant  string                          `json:"base_variant,omitempty" yaml:"base_variant,omitempty"`
	ApplyGate    string                          `json:"apply_gate,omitempty" yaml:"apply_gate,omitempty"`
	Context      map[string]string               `json:"context,omitempty" yaml:"context,omitempty"`
	Placeholders []ManifestAdaptationPlaceholder `json:"placeholders,omitempty" yaml:"placeholders,omitempty"`
}

type ManifestAdaptationPlaceholder struct {
	Token     string   `json:"token" yaml:"token"`
	Value     string   `json:"value,omitempty" yaml:"value,omitempty"`
	ValueFrom string   `json:"value_from,omitempty" yaml:"value_from,omitempty"`
	Files     []string `json:"files,omitempty" yaml:"files,omitempty"`
	Owner     string   `json:"owner,omitempty" yaml:"owner,omitempty"`
	Route     string   `json:"route,omitempty" yaml:"route,omitempty"`
	Reason    string   `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type ImportOptions struct {
	GeneratedAt string
}

type Graph struct {
	SchemaVersion string          `json:"schema_version"`
	ManifestPath  string          `json:"manifest_path"`
	Name          string          `json:"name"`
	Space         string          `json:"space"`
	Ref           string          `json:"ref"`
	GeneratedAt   string          `json:"generated_at"`
	Summary       GraphSummary    `json:"summary"`
	Repos         []RepoNode      `json:"repos"`
	Components    []ComponentNode `json:"components,omitempty"`
	Variants      []VariantNode   `json:"variants,omitempty"`
	Targets       []TargetNode    `json:"targets,omitempty"`
	Generators    []GeneratorNode `json:"generators,omitempty"`
	DryInputs     []DryInputNode  `json:"dry_inputs,omitempty"`
	WetTargets    []WetTargetNode `json:"wet_targets,omitempty"`
	Connections   []Connection    `json:"connections,omitempty"`
	Diagnostics   []Diagnostic    `json:"diagnostics,omitempty"`
}

type GraphSummary struct {
	RepoCount       int `json:"repo_count"`
	ComponentCount  int `json:"component_count"`
	VariantCount    int `json:"variant_count"`
	TargetCount     int `json:"target_count"`
	GeneratorCount  int `json:"generator_count"`
	DryInputCount   int `json:"dry_input_count"`
	WetTargetCount  int `json:"wet_target_count"`
	ConnectionCount int `json:"connection_count"`
	DiagnosticCount int `json:"diagnostic_count"`
}

type RepoNode struct {
	ID             string `json:"id"`
	Role           string `json:"role"`
	Path           string `json:"path"`
	Owner          string `json:"owner,omitempty"`
	Component      string `json:"component,omitempty"`
	Variant        string `json:"variant,omitempty"`
	VariantKind    string `json:"variant_kind,omitempty"`
	Target         string `json:"target,omitempty"`
	Exists         bool   `json:"exists"`
	GeneratorCount int    `json:"generator_count"`
}

type ComponentNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Owner  string `json:"owner,omitempty"`
	RepoID string `json:"repo_id,omitempty"`
}

type VariantNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Component   string `json:"component"`
	VariantKind string `json:"variant_kind"`
	Target      string `json:"target,omitempty"`
	Owner       string `json:"owner,omitempty"`
	RepoID      string `json:"repo_id"`
}

type TargetNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Owner  string `json:"owner,omitempty"`
	RepoID string `json:"repo_id,omitempty"`
}

type GeneratorNode struct {
	ID       string `json:"id"`
	RepoID   string `json:"repo_id"`
	Kind     string `json:"kind"`
	Profile  string `json:"profile,omitempty"`
	Name     string `json:"name"`
	Root     string `json:"root,omitempty"`
	Owner    string `json:"owner,omitempty"`
	DryCount int    `json:"dry_count"`
	WetCount int    `json:"wet_count"`
}

type DryInputNode struct {
	ID          string `json:"id"`
	GeneratorID string `json:"generator_id"`
	RepoID      string `json:"repo_id"`
	Role        string `json:"role"`
	Owner       string `json:"owner,omitempty"`
	Path        string `json:"path"`
	Required    bool   `json:"required"`
}

type WetTargetNode struct {
	ID            string `json:"id"`
	GeneratorID   string `json:"generator_id"`
	RepoID        string `json:"repo_id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace,omitempty"`
	Owner         string `json:"owner,omitempty"`
	SourceDryPath string `json:"source_dry_path,omitempty"`
}

type Connection struct {
	FromType string `json:"from_type"`
	FromID   string `json:"from_id"`
	ToType   string `json:"to_type"`
	ToID     string `json:"to_id"`
	Kind     string `json:"kind"`
}

type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	RepoID   string `json:"repo_id,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

func ImportManifest(manifestPath string, opts ImportOptions) (Graph, error) {
	manifest, absManifestPath, err := LoadManifest(manifestPath)
	if err != nil {
		return Graph{}, err
	}
	return BuildGraph(absManifestPath, manifest, opts)
}

func LoadManifest(manifestPath string) (Manifest, string, error) {
	if strings.TrimSpace(manifestPath) == "" {
		return Manifest{}, "", errors.New("platform manifest path is required")
	}
	abs, err := filepath.Abs(manifestPath)
	if err != nil {
		return Manifest{}, "", fmt.Errorf("resolve platform manifest: %w", err)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return Manifest{}, "", fmt.Errorf("read platform manifest: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, "", fmt.Errorf("parse platform manifest: %w", err)
	}
	if len(manifest.Repos) == 0 {
		return Manifest{}, "", errors.New("platform manifest must declare at least one repo")
	}
	if strings.TrimSpace(manifest.SchemaVersion) == "" {
		manifest.SchemaVersion = ManifestSchemaVersion
	}
	if manifest.SchemaVersion != ManifestSchemaVersion {
		return Manifest{}, "", fmt.Errorf("unsupported platform manifest schema_version %q", manifest.SchemaVersion)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		manifest.Name = strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	}
	if strings.TrimSpace(manifest.Space) == "" {
		manifest.Space = "default"
	}
	if strings.TrimSpace(manifest.Ref) == "" {
		manifest.Ref = "HEAD"
	}
	return manifest, abs, nil
}

func BuildGraph(absManifestPath string, manifest Manifest, opts ImportOptions) (Graph, error) {
	generatedAt := strings.TrimSpace(opts.GeneratedAt)
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	baseDir := filepath.Dir(absManifestPath)
	graph := Graph{
		SchemaVersion: GraphSchemaVersion,
		ManifestPath:  filepath.ToSlash(filepath.Clean(absManifestPath)),
		Name:          manifest.Name,
		Space:         manifest.Space,
		Ref:           manifest.Ref,
		GeneratedAt:   generatedAt,
	}

	componentByID := map[string]ComponentNode{}
	variantByID := map[string]VariantNode{}
	targetByID := map[string]TargetNode{}
	targetRankByID := map[string]int{}

	repos := normalizeRepos(manifest.Repos)
	for _, repo := range repos {
		variantName := strings.TrimSpace(repo.Variant)
		variantKind := ""
		variantKindInvalid := false
		if variantName != "" {
			variantKind, variantKindInvalid = resolveVariantKind(repo.VariantKind, repo.Target)
		}
		repoNode := RepoNode{
			ID:          repo.ID,
			Role:        normalizeToken(repo.Role, "repo"),
			Path:        normalizeRelPath(repo.Path),
			Owner:       strings.TrimSpace(repo.Owner),
			Component:   strings.TrimSpace(repo.Component),
			Variant:     variantName,
			VariantKind: variantKind,
			Target:      strings.TrimSpace(repo.Target),
		}
		if variantKindInvalid {
			graph.Diagnostics = append(graph.Diagnostics, Diagnostic{
				Severity: "warning",
				Code:     "invalid_variant_kind",
				RepoID:   repoNode.ID,
				Path:     repoNode.Path,
				Message:  fmt.Sprintf("repo variant_kind %q is not supported or is inconsistent with target %q; expected base without a target or deployment with a target", strings.TrimSpace(repo.VariantKind), repoNode.Target),
			})
		}
		if repoNode.Owner == "" {
			graph.Diagnostics = append(graph.Diagnostics, Diagnostic{
				Severity: "warning",
				Code:     "missing_owner",
				RepoID:   repoNode.ID,
				Path:     repoNode.Path,
				Message:  "repo has no owner; route decisions will require review",
			})
		}
		if repoNode.Component != "" {
			component := componentByID[repoNode.Component]
			component.ID = repoNode.Component
			component.Name = repoNode.Component
			if component.Owner == "" {
				component.Owner = repoNode.Owner
			}
			if component.RepoID == "" {
				component.RepoID = repoNode.ID
			}
			componentByID[repoNode.Component] = component
			graph.Connections = append(graph.Connections, Connection{
				FromType: "repo",
				FromID:   repoNode.ID,
				ToType:   "component",
				ToID:     repoNode.Component,
				Kind:     "defines",
			})
		}
		if repoNode.Variant != "" {
			if repoNode.Component == "" {
				graph.Diagnostics = append(graph.Diagnostics, Diagnostic{
					Severity: "error",
					Code:     "missing_component",
					RepoID:   repoNode.ID,
					Path:     repoNode.Path,
					Message:  "variant repo must name a component",
				})
			} else {
				variantID := repoNode.Component + "/" + repoNode.Variant
				variantByID[variantID] = VariantNode{
					ID:          variantID,
					Name:        repoNode.Variant,
					Component:   repoNode.Component,
					VariantKind: repoNode.VariantKind,
					Target:      repoNode.Target,
					Owner:       repoNode.Owner,
					RepoID:      repoNode.ID,
				}
				graph.Connections = append(graph.Connections, Connection{
					FromType: "component",
					FromID:   repoNode.Component,
					ToType:   "variant",
					ToID:     variantID,
					Kind:     "has-variant",
				})
			}
		}
		if repoNode.Target != "" {
			candidateRank := targetOwnerRank(repoNode.Role)
			target, exists := targetByID[repoNode.Target]
			if !exists || candidateRank > targetRankByID[repoNode.Target] || (target.Owner == "" && repoNode.Owner != "") {
				target.ID = repoNode.Target
				target.Name = repoNode.Target
				target.Owner = repoNode.Owner
				target.RepoID = repoNode.ID
				targetByID[repoNode.Target] = target
				targetRankByID[repoNode.Target] = candidateRank
			}
			if repoNode.Variant != "" && repoNode.Component != "" {
				graph.Connections = append(graph.Connections, Connection{
					FromType: "variant",
					FromID:   repoNode.Component + "/" + repoNode.Variant,
					ToType:   "target",
					ToID:     repoNode.Target,
					Kind:     "deploys-to",
				})
			}
		}
		if repoNode.Path == "" {
			graph.Diagnostics = append(graph.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "missing_repo_path",
				RepoID:   repoNode.ID,
				Message:  "repo has no path",
			})
			graph.Repos = append(graph.Repos, repoNode)
			continue
		}

		absRepo := filepath.Join(baseDir, filepath.FromSlash(repoNode.Path))
		info, err := os.Stat(absRepo)
		if err != nil || !info.IsDir() {
			graph.Diagnostics = append(graph.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "missing_repo",
				RepoID:   repoNode.ID,
				Path:     repoNode.Path,
				Message:  "repo path does not exist or is not a directory",
			})
			graph.Repos = append(graph.Repos, repoNode)
			continue
		}
		repoNode.Exists = true
		imported, err := importer.ImportRepo(absRepo, manifest.Ref, manifest.Space)
		if err != nil {
			graph.Diagnostics = append(graph.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "import_failed",
				RepoID:   repoNode.ID,
				Path:     repoNode.Path,
				Message:  err.Error(),
			})
			graph.Repos = append(graph.Repos, repoNode)
			continue
		}
		if len(imported.Detection.Generators) == 0 {
			graph.Diagnostics = append(graph.Diagnostics, Diagnostic{
				Severity: "warning",
				Code:     "unsupported_generator",
				RepoID:   repoNode.ID,
				Path:     repoNode.Path,
				Message:  "repo exists but no supported generator was detected",
			})
			graph.Repos = append(graph.Repos, repoNode)
			continue
		}

		repoNode.GeneratorCount = len(imported.Detection.Generators)
		graph.Repos = append(graph.Repos, repoNode)
		appendImport(&graph, repoNode, imported)
		if repoNode.Component != "" && repoNode.Variant != "" {
			variantID := repoNode.Component + "/" + repoNode.Variant
			for _, gen := range imported.Detection.Generators {
				graph.Connections = append(graph.Connections, Connection{
					FromType: "variant",
					FromID:   variantID,
					ToType:   "generator",
					ToID:     gen.ID,
					Kind:     "generated-by",
				})
			}
		}
	}

	for _, component := range componentByID {
		graph.Components = append(graph.Components, component)
	}
	for _, variant := range variantByID {
		graph.Variants = append(graph.Variants, variant)
	}
	for _, target := range targetByID {
		graph.Targets = append(graph.Targets, target)
	}
	sortGraph(&graph)
	graph.Summary = GraphSummary{
		RepoCount:       len(graph.Repos),
		ComponentCount:  len(graph.Components),
		VariantCount:    len(graph.Variants),
		TargetCount:     len(graph.Targets),
		GeneratorCount:  len(graph.Generators),
		DryInputCount:   len(graph.DryInputs),
		WetTargetCount:  len(graph.WetTargets),
		ConnectionCount: len(graph.Connections),
		DiagnosticCount: len(graph.Diagnostics),
	}
	return graph, nil
}

func appendImport(graph *Graph, repo RepoNode, imported model.ImportResult) {
	dryByGenerator := map[string][]model.DryInputRef{}
	for _, dry := range imported.DryInputs {
		dryByGenerator[dry.GeneratorID] = append(dryByGenerator[dry.GeneratorID], dry)
		owner := dry.Owner
		if owner == "" {
			owner = repo.Owner
		}
		node := DryInputNode{
			ID:          dryInputID(dry.GeneratorID, dry.Path),
			GeneratorID: dry.GeneratorID,
			RepoID:      repo.ID,
			Role:        dry.Role,
			Owner:       owner,
			Path:        dry.Path,
			Required:    dry.Required,
		}
		graph.DryInputs = append(graph.DryInputs, node)
		graph.Connections = append(graph.Connections, Connection{
			FromType: "generator",
			FromID:   dry.GeneratorID,
			ToType:   "dry-input",
			ToID:     node.ID,
			Kind:     "reads",
		})
	}
	wetByGenerator := map[string][]model.WetManifestTarget{}
	for _, wet := range imported.WetManifestTargets {
		wetByGenerator[wet.GeneratorID] = append(wetByGenerator[wet.GeneratorID], wet)
		owner := wet.Owner
		if owner == "" {
			owner = repo.Owner
		}
		node := WetTargetNode{
			ID:            wetTargetID(wet.GeneratorID, wet),
			GeneratorID:   wet.GeneratorID,
			RepoID:        repo.ID,
			Kind:          wet.Kind,
			Name:          wet.Name,
			Namespace:     wet.Namespace,
			Owner:         owner,
			SourceDryPath: wet.SourceDryPath,
		}
		graph.WetTargets = append(graph.WetTargets, node)
		graph.Connections = append(graph.Connections, Connection{
			FromType: "generator",
			FromID:   wet.GeneratorID,
			ToType:   "wet-target",
			ToID:     node.ID,
			Kind:     "renders",
		})
	}
	for _, gen := range imported.Detection.Generators {
		graph.Generators = append(graph.Generators, GeneratorNode{
			ID:       gen.ID,
			RepoID:   repo.ID,
			Kind:     string(gen.Kind),
			Profile:  gen.Profile,
			Name:     gen.Name,
			Root:     gen.Root,
			Owner:    repo.Owner,
			DryCount: len(dryByGenerator[gen.ID]),
			WetCount: len(wetByGenerator[gen.ID]),
		})
		graph.Connections = append(graph.Connections, Connection{
			FromType: "repo",
			FromID:   repo.ID,
			ToType:   "generator",
			ToID:     gen.ID,
			Kind:     "contains-generator",
		})
	}
}

func normalizeRepos(repos []ManifestRepo) []ManifestRepo {
	out := make([]ManifestRepo, 0, len(repos))
	seen := map[string]int{}
	for i, repo := range repos {
		normalized := repo
		normalized.ID = strings.TrimSpace(normalized.ID)
		if normalized.ID == "" {
			normalized.ID = fmt.Sprintf("%s-%02d", normalizeToken(normalized.Role, "repo"), i+1)
		}
		if count := seen[normalized.ID]; count > 0 {
			normalized.ID = fmt.Sprintf("%s-%02d", normalized.ID, count+1)
		}
		seen[normalized.ID]++
		out = append(out, normalized)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func sortGraph(graph *Graph) {
	sort.Slice(graph.Repos, func(i, j int) bool { return graph.Repos[i].ID < graph.Repos[j].ID })
	sort.Slice(graph.Components, func(i, j int) bool { return graph.Components[i].ID < graph.Components[j].ID })
	sort.Slice(graph.Variants, func(i, j int) bool { return graph.Variants[i].ID < graph.Variants[j].ID })
	sort.Slice(graph.Targets, func(i, j int) bool { return graph.Targets[i].ID < graph.Targets[j].ID })
	sort.Slice(graph.Generators, func(i, j int) bool { return graph.Generators[i].ID < graph.Generators[j].ID })
	sort.Slice(graph.DryInputs, func(i, j int) bool { return graph.DryInputs[i].ID < graph.DryInputs[j].ID })
	sort.Slice(graph.WetTargets, func(i, j int) bool { return graph.WetTargets[i].ID < graph.WetTargets[j].ID })
	sort.Slice(graph.Connections, func(i, j int) bool {
		left := graph.Connections[i]
		right := graph.Connections[j]
		if left.FromType != right.FromType {
			return left.FromType < right.FromType
		}
		if left.FromID != right.FromID {
			return left.FromID < right.FromID
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.ToType != right.ToType {
			return left.ToType < right.ToType
		}
		return left.ToID < right.ToID
	})
	sort.Slice(graph.Diagnostics, func(i, j int) bool {
		left := graph.Diagnostics[i]
		right := graph.Diagnostics[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.RepoID != right.RepoID {
			return left.RepoID < right.RepoID
		}
		return left.Path < right.Path
	})
}

func dryInputID(generatorID, path string) string {
	return "dry_" + sanitizeID(generatorID+"_"+path)
}

func wetTargetID(generatorID string, wet model.WetManifestTarget) string {
	return "wet_" + sanitizeID(strings.Join([]string{generatorID, wet.Namespace, wet.Kind, wet.Name}, "_"))
}

func sanitizeID(value string) string {
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func normalizeToken(value, fallback string) string {
	clean := sanitizeID(value)
	if clean == "" {
		return fallback
	}
	return clean
}

func targetOwnerRank(role string) int {
	switch normalizeToken(role, "repo") {
	case "env":
		return 4
	case "rendered":
		return 3
	case "platform":
		return 2
	case "app":
		return 1
	default:
		return 0
	}
}

func resolveVariantKind(explicitKind, target string) (string, bool) {
	clean := normalizeToken(explicitKind, "")
	hasTarget := strings.TrimSpace(target) != ""
	switch clean {
	case "base":
		if hasTarget {
			return "unknown", true
		}
		return clean, false
	case "deployment":
		if !hasTarget {
			return "unknown", true
		}
		return clean, false
	case "":
		if !hasTarget {
			return "base", false
		}
		return "deployment", false
	}
	return "unknown", true
}

func normalizeRelPath(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(clean)))
}
