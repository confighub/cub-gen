package gitops

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/detect"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
)

const (
	discoverDirName = ".cub-gen/discover"
	targetsFileName = ".cub-gen/targets.json"

	toolchainKubernetesYAML = "kubernetes/yaml"
	providerKubernetes      = "kubernetes"
	providerFluxRenderer    = "fluxrenderer"
	providerArgoCDRenderer  = "argocdrenderer"
)

var (
	eqClauseRe = regexp.MustCompile(`(?i)^(kind|name|resource_name|root|id)\s*=\s*'([^']+)'$`)
	inClauseRe = regexp.MustCompile(`(?i)^kind\s+IN\s*\(([^)]+)\)$`)
	likeRe     = regexp.MustCompile(`(?i)^(name|resource_name|root)\s+LIKE\s+'([^']+)'$`)
	quotedRe   = regexp.MustCompile(`'([^']+)'`)
	andSplitRe = regexp.MustCompile(`(?i)\s+AND\s+`)
)

// DiscoveredResource is the discover-phase resource abstraction, modeled after
// cub gitops discover output entries.
type DiscoveredResource struct {
	GeneratorID      string   `json:"generator_id"`
	GeneratorProfile string   `json:"generator_profile"`
	ResourceName     string   `json:"resource_name"`
	ResourceKind     string   `json:"resource_kind"`
	ResourceType     string   `json:"resource_type"`
	ResourceBody     string   `json:"resource_body"`
	GeneratorKind    string   `json:"generator_kind"`
	Root             string   `json:"root"`
	Inputs           []string `json:"inputs"`
}

// DiscoverResult is the local discover-unit state used by import and cleanup.
type DiscoverResult struct {
	Space            string                     `json:"space"`
	TargetSlug       string                     `json:"target_slug"`
	TargetPath       string                     `json:"target_path"`
	Ref              string                     `json:"ref"`
	WhereResource    string                     `json:"where_resource,omitempty"`
	DiscoverUnitSlug string                     `json:"discover_unit_slug"`
	DiscoverFile     string                     `json:"discover_file"`
	DiscoveredAt     string                     `json:"discovered_at"`
	Resources        []DiscoveredResource       `json:"resources"`
	Detections       []model.GeneratorDetection `json:"detections"`
}

// ImportFlowResult models the staged import output in the same conceptual shape
// as cub gitops import: discover -> dry units -> rendered wet units + links.
type ImportFlowResult struct {
	Space              string                       `json:"space"`
	TargetSlug         string                       `json:"target_slug"`
	TargetPath         string                       `json:"target_path"`
	RenderTargetSlug   string                       `json:"render_target_slug"`
	Ref                string                       `json:"ref"`
	WhereResource      string                       `json:"where_resource,omitempty"`
	DiscoverUnitSlug   string                       `json:"discover_unit_slug"`
	ImportedAt         string                       `json:"imported_at"`
	Discovered         []DiscoveredResource         `json:"discovered"`
	DryUnits           []model.UnitRef              `json:"dry_units"`
	WetUnits           []model.UnitRef              `json:"wet_units"`
	GeneratorUnits     []model.UnitRef              `json:"generator_units"`
	Links              []model.UnitLink             `json:"links"`
	Contracts          []model.GeneratorContract    `json:"contracts"`
	Provenance         []model.ProvenanceRecord     `json:"provenance"`
	InversePlans       []model.InverseTransformPlan `json:"inverse_transform_plans"`
	DryInputs          []model.DryInputRef          `json:"dry_inputs"`
	WetManifestTargets []model.WetManifestTarget    `json:"wet_manifest_targets"`
}

// targetRef is a locally-resolved target identifier.
// In this prototype, target slugs can be direct repo paths or alias names.
type targetRef struct {
	Slug      string
	Path      string
	Toolchain string
	Providers []string
}

type targetMode string

const (
	targetModeDiscover targetMode = "discover"
	targetModeRender   targetMode = "render"
)

type targetConfig struct {
	Path      string
	Toolchain string
	Providers []string
}

// Discover scans a local repo target, applies where-resource filtering, and
// writes discover-unit state to .cub-gen/discover/<discover-slug>.json.
func Discover(targetPath, ref, space, whereResource string) (DiscoverResult, error) {
	if strings.TrimSpace(targetPath) == "" {
		return DiscoverResult{}, errors.New("target slug is required")
	}
	if strings.TrimSpace(space) == "" {
		space = "default"
	}
	if strings.TrimSpace(ref) == "" {
		ref = "HEAD"
	}

	resolved, err := resolveTarget(targetPath, targetModeDiscover)
	if err != nil {
		return DiscoverResult{}, err
	}
	if err := ensureTargetCapabilities(resolved, toolchainKubernetesYAML, []string{providerKubernetes}, targetModeDiscover); err != nil {
		return DiscoverResult{}, err
	}

	detection, err := detect.ScanRepo(resolved.Path, ref)
	if err != nil {
		return DiscoverResult{}, err
	}

	filtered, err := filterDetections(detection.Generators, whereResource)
	if err != nil {
		return DiscoverResult{}, err
	}
	resources := toDiscoveredResources(filtered)

	discoverSlug := discoverUnitSlug(resolved.Slug, space, resolved.Path)
	discoverFile := filepath.Join(resolved.Path, discoverDirName, discoverSlug+".json")

	result := DiscoverResult{
		Space:            space,
		TargetSlug:       resolved.Slug,
		TargetPath:       resolved.Path,
		Ref:              ref,
		WhereResource:    strings.TrimSpace(whereResource),
		DiscoverUnitSlug: discoverSlug,
		DiscoverFile:     discoverFile,
		DiscoveredAt:     time.Now().UTC().Format(time.RFC3339),
		Resources:        resources,
		Detections:       filtered,
	}

	if err := persistDiscoverResult(result); err != nil {
		return DiscoverResult{}, err
	}

	return result, nil
}

// Import runs discover and then creates local import artifacts from discovered resources.
func Import(targetPath, renderTargetSlug, ref, space, whereResource string) (ImportFlowResult, error) {
	renderTarget, err := resolveTarget(renderTargetSlug, targetModeRender)
	if err != nil {
		return ImportFlowResult{}, fmt.Errorf("resolve render target: %w", err)
	}
	if err := ensureTargetCapabilities(renderTarget, toolchainKubernetesYAML, nil, targetModeRender); err != nil {
		return ImportFlowResult{}, err
	}

	discovered, err := Discover(targetPath, ref, space, whereResource)
	if err != nil {
		return ImportFlowResult{}, err
	}

	if len(discovered.Detections) == 0 {
		return ImportFlowResult{
			Space:            discovered.Space,
			TargetSlug:       discovered.TargetSlug,
			TargetPath:       discovered.TargetPath,
			RenderTargetSlug: renderTarget.Slug,
			Ref:              discovered.Ref,
			WhereResource:    discovered.WhereResource,
			DiscoverUnitSlug: discovered.DiscoverUnitSlug,
			ImportedAt:       time.Now().UTC().Format(time.RFC3339),
			Discovered:       discovered.Resources,
		}, nil
	}
	requiredProviders := requiredProvidersForDiscovered(discovered.Resources)
	if err := ensureTargetCapabilities(renderTarget, toolchainKubernetesYAML, requiredProviders, targetModeRender); err != nil {
		return ImportFlowResult{}, err
	}

	importResult, err := importer.ImportDetection(model.DetectionResult{
		Repo:       discovered.TargetPath,
		Ref:        discovered.Ref,
		DetectedAt: discovered.DiscoveredAt,
		Generators: discovered.Detections,
	}, discovered.Space)
	if err != nil {
		return ImportFlowResult{}, err
	}

	dryUnits, wetUnits, generatorUnits := splitUnits(importResult.Units)

	return ImportFlowResult{
		Space:              discovered.Space,
		TargetSlug:         discovered.TargetSlug,
		TargetPath:         discovered.TargetPath,
		RenderTargetSlug:   renderTarget.Slug,
		Ref:                discovered.Ref,
		WhereResource:      discovered.WhereResource,
		DiscoverUnitSlug:   discovered.DiscoverUnitSlug,
		ImportedAt:         time.Now().UTC().Format(time.RFC3339),
		Discovered:         discovered.Resources,
		DryUnits:           dryUnits,
		WetUnits:           wetUnits,
		GeneratorUnits:     generatorUnits,
		Links:              importResult.Links,
		Contracts:          importResult.GeneratorContracts,
		Provenance:         importResult.Provenance,
		InversePlans:       importResult.InversePlans,
		DryInputs:          importResult.DryInputs,
		WetManifestTargets: importResult.WetManifestTargets,
	}, nil
}

// Cleanup deletes the persisted discover-unit state for a local target.
func Cleanup(targetPath, space string) (bool, string, error) {
	if strings.TrimSpace(targetPath) == "" {
		return false, "", errors.New("target slug is required")
	}
	if strings.TrimSpace(space) == "" {
		space = "default"
	}

	resolved, err := resolveTarget(targetPath, targetModeDiscover)
	if err != nil {
		return false, "", err
	}
	discoverSlug := discoverUnitSlug(resolved.Slug, space, resolved.Path)
	discoverFile := filepath.Join(resolved.Path, discoverDirName, discoverSlug+".json")

	if _, err := os.Stat(discoverFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, discoverFile, nil
		}
		return false, discoverFile, fmt.Errorf("stat discover file: %w", err)
	}
	if err := os.Remove(discoverFile); err != nil {
		return false, discoverFile, fmt.Errorf("remove discover file: %w", err)
	}
	return true, discoverFile, nil
}

func resolveTarget(targetArg string, mode targetMode) (targetRef, error) {
	raw := strings.TrimSpace(targetArg)
	if raw == "" {
		return targetRef{}, errors.New("target slug is required")
	}

	// Direct repo path input.
	if abs, ok, err := toAbsDirIfExists(raw); err != nil {
		return targetRef{}, err
	} else if ok {
		return defaultPathTarget(raw, abs, mode), nil
	}

	aliases, err := loadTargetAliases()
	if err != nil {
		return targetRef{}, err
	}

	cfg, ok := aliases[raw]
	if !ok {
		return targetRef{}, fmt.Errorf("target %q is not a directory and not found in aliases", raw)
	}

	ref := targetRef{
		Slug:      raw,
		Toolchain: normalizeToolchain(cfg.Toolchain),
		Providers: normalizeProviders(cfg.Providers),
	}

	path := strings.TrimSpace(cfg.Path)
	if path != "" {
		abs, exists, pathErr := toAbsDirIfExists(path)
		if pathErr != nil {
			return targetRef{}, pathErr
		}
		if !exists {
			return targetRef{}, fmt.Errorf("target alias %q resolves to non-directory path %q", raw, path)
		}
		ref.Path = abs
	}

	applyDefaultCapabilities(&ref, mode)

	if mode == targetModeDiscover && ref.Path == "" {
		return targetRef{}, fmt.Errorf("target alias %q has no path; discover/cleanup require a repo path", raw)
	}

	return ref, nil
}

func loadTargetAliases() (map[string]targetConfig, error) {
	cfgPath := strings.TrimSpace(os.Getenv("CUB_GEN_TARGETS_FILE"))
	if cfgPath == "" {
		cfgPath = targetsFileName
	}
	cfgPath = filepath.Clean(cfgPath)

	b, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]targetConfig{}, nil
		}
		return nil, fmt.Errorf("read targets file %s: %w", cfgPath, err)
	}

	parseMap := func(raw map[string]json.RawMessage) (map[string]targetConfig, error) {
		out := make(map[string]targetConfig, len(raw))
		for alias, rv := range raw {
			cfg, parseErr := parseTargetConfigRaw(rv, filepath.Dir(cfgPath))
			if parseErr != nil {
				return nil, fmt.Errorf("parse target alias %q: %w", alias, parseErr)
			}
			out[strings.TrimSpace(alias)] = cfg
		}
		return out, nil
	}

	type wrapped struct {
		Targets map[string]json.RawMessage `json:"targets"`
	}
	var w wrapped
	if err := json.Unmarshal(b, &w); err == nil && len(w.Targets) > 0 {
		return parseMap(w.Targets)
	}

	var flat map[string]json.RawMessage
	if err := json.Unmarshal(b, &flat); err == nil && len(flat) > 0 {
		return parseMap(flat)
	}

	return nil, fmt.Errorf("parse targets file %s: expected {\"targets\":{...}} or {\"alias\":\"path\"}", cfgPath)
}

func parseTargetConfigRaw(raw json.RawMessage, baseDir string) (targetConfig, error) {
	var pathOnly string
	if err := json.Unmarshal(raw, &pathOnly); err == nil {
		return targetConfig{
			Path: absolutizePath(pathOnly, baseDir),
		}, nil
	}

	var obj struct {
		Path       string   `json:"path"`
		Toolchain  string   `json:"toolchain"`
		Toolchains []string `json:"toolchains"`
		Provider   string   `json:"provider"`
		Providers  []string `json:"providers"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return targetConfig{}, fmt.Errorf("expected string or object target config: %w", err)
	}

	toolchain := obj.Toolchain
	if toolchain == "" && len(obj.Toolchains) > 0 {
		toolchain = obj.Toolchains[0]
	}
	providers := append([]string{}, obj.Providers...)
	if strings.TrimSpace(obj.Provider) != "" {
		providers = append(providers, obj.Provider)
	}

	return targetConfig{
		Path:      absolutizePath(obj.Path, baseDir),
		Toolchain: toolchain,
		Providers: providers,
	}, nil
}

func absolutizePath(path, baseDir string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	return filepath.Clean(p)
}

func defaultPathTarget(raw, absPath string, mode targetMode) targetRef {
	ref := targetRef{
		Slug:      filepath.Base(absPath),
		Path:      absPath,
		Toolchain: toolchainKubernetesYAML,
	}
	if mode == targetModeDiscover {
		ref.Providers = []string{providerKubernetes}
	} else {
		ref.Providers = []string{providerFluxRenderer, providerArgoCDRenderer}
	}
	return ref
}

func applyDefaultCapabilities(ref *targetRef, mode targetMode) {
	if ref == nil {
		return
	}
	if strings.TrimSpace(ref.Toolchain) == "" {
		ref.Toolchain = toolchainKubernetesYAML
	} else {
		ref.Toolchain = normalizeToolchain(ref.Toolchain)
	}
	ref.Providers = normalizeProviders(ref.Providers)
	if len(ref.Providers) > 0 {
		return
	}
	if mode == targetModeDiscover {
		ref.Providers = []string{providerKubernetes}
		return
	}
	ref.Providers = []string{providerFluxRenderer, providerArgoCDRenderer}
}

func ensureTargetCapabilities(ref targetRef, requiredToolchain string, requiredProviders []string, mode targetMode) error {
	wantToolchain := normalizeToolchain(requiredToolchain)
	haveToolchain := normalizeToolchain(ref.Toolchain)
	if wantToolchain != "" && haveToolchain != wantToolchain {
		return fmt.Errorf("%s target %q must use toolchain %q (have %q)", mode, ref.Slug, requiredToolchain, ref.Toolchain)
	}

	requiredProviders = normalizeProviders(requiredProviders)
	if len(requiredProviders) == 0 {
		return nil
	}
	haveSet := make(map[string]struct{}, len(ref.Providers))
	for _, p := range normalizeProviders(ref.Providers) {
		haveSet[p] = struct{}{}
	}
	var missing []string
	for _, p := range requiredProviders {
		if _, ok := haveSet[p]; !ok {
			missing = append(missing, p)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	have := normalizeProviders(ref.Providers)
	sort.Strings(have)
	return fmt.Errorf("%s target %q missing providers: %s (have: %s)",
		mode, ref.Slug, strings.Join(missing, ","), strings.Join(have, ","))
}

func normalizeToolchain(in string) string {
	s := strings.ToLower(strings.TrimSpace(in))
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	switch s {
	case "kubernetes/yaml", "kubernetesyaml":
		return toolchainKubernetesYAML
	default:
		return strings.ToLower(strings.TrimSpace(in))
	}
}

func normalizeProviders(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, p := range in {
		n := normalizeProvider(p)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func normalizeProvider(in string) string {
	s := strings.ToLower(strings.TrimSpace(in))
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	switch s {
	case "kubernetes", "k8s":
		return providerKubernetes
	case "flux", "fluxrenderer":
		return providerFluxRenderer
	case "argocd", "argocdrenderer":
		return providerArgoCDRenderer
	default:
		return s
	}
}

func requiredProvidersForDiscovered(resources []DiscoveredResource) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 2)
	for _, r := range resources {
		p := providerForResourceType(r.ResourceType)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func providerForResourceType(resourceType string) string {
	s := strings.ToLower(strings.TrimSpace(resourceType))
	switch {
	case strings.Contains(s, "argoproj.io/"):
		return providerArgoCDRenderer
	case strings.Contains(s, "fluxcd.io/"):
		return providerFluxRenderer
	default:
		return ""
	}
}

func toAbsDirIfExists(path string) (string, bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("resolve path %s: %w", path, err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("stat path %s: %w", abs, err)
	}
	if !st.IsDir() {
		return "", false, nil
	}
	return abs, true, nil
}

func persistDiscoverResult(result DiscoverResult) error {
	dir := filepath.Dir(result.DiscoverFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create discover dir: %w", err)
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal discover result: %w", err)
	}
	if err := os.WriteFile(result.DiscoverFile, b, 0o644); err != nil {
		return fmt.Errorf("write discover file: %w", err)
	}
	return nil
}

func toDiscoveredResources(detections []model.GeneratorDetection) []DiscoveredResource {
	resources := make([]DiscoveredResource, 0, len(detections))
	for _, g := range detections {
		kind := mappedResourceKind(g.Kind)
		resourceType := mappedResourceType(g.Kind)
		body := fmt.Sprintf("kind: %s\nmetadata:\n  name: %s\nspec:\n  root: %s\n", kind, g.Name, g.Root)
		resources = append(resources, DiscoveredResource{
			GeneratorID:      g.ID,
			GeneratorProfile: g.Profile,
			ResourceName:     g.Name,
			ResourceKind:     kind,
			ResourceType:     resourceType,
			ResourceBody:     body,
			GeneratorKind:    string(g.Kind),
			Root:             g.Root,
			Inputs:           append([]string{}, g.Inputs...),
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

func splitUnits(units []model.UnitRef) (dry, wet, generator []model.UnitRef) {
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

func discoverUnitSlug(targetSlug, space, targetPath string) string {
	raw := fmt.Sprintf("discover-%s-%s-%s", slugify(targetSlug), shortHash(space), shortHash(targetPath))
	if len(raw) <= 63 {
		return raw
	}
	return raw[:63]
}

func shortHash(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	h := 2166136261
	for i := 0; i < len(s); i++ {
		h ^= int(s[i])
		h *= 16777619
	}
	if h < 0 {
		h = -h
	}
	return fmt.Sprintf("%08x", h)
}

func slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "/", "-")
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			out = append(out, r)
			continue
		}
		out = append(out, '-')
	}
	clean := strings.Trim(strings.ReplaceAll(string(out), "--", "-"), "-")
	if clean == "" {
		return "target"
	}
	return clean
}

func filterDetections(in []model.GeneratorDetection, where string) ([]model.GeneratorDetection, error) {
	where = strings.TrimSpace(where)
	if where == "" {
		return append([]model.GeneratorDetection{}, in...), nil
	}

	clauses := andSplitRe.Split(where, -1)
	out := make([]model.GeneratorDetection, 0, len(in))
	for _, g := range in {
		match, err := matchesAllClauses(g, clauses)
		if err != nil {
			return nil, err
		}
		if match {
			out = append(out, g)
		}
	}
	return out, nil
}

func matchesAllClauses(g model.GeneratorDetection, clauses []string) (bool, error) {
	for _, clause := range clauses {
		c := strings.TrimSpace(clause)
		if c == "" {
			continue
		}

		if m := eqClauseRe.FindStringSubmatch(c); len(m) == 3 {
			field := strings.ToLower(m[1])
			value := strings.ToLower(strings.TrimSpace(m[2]))
			if !matchesField(g, field, value) {
				return false, nil
			}
			continue
		}

		if m := inClauseRe.FindStringSubmatch(c); len(m) == 2 {
			vals := quotedRe.FindAllStringSubmatch(m[1], -1)
			if len(vals) == 0 {
				return false, fmt.Errorf("invalid kind IN clause: %s", c)
			}
			ok := false
			for _, v := range vals {
				if matchesField(g, "kind", strings.ToLower(strings.TrimSpace(v[1]))) {
					ok = true
					break
				}
			}
			if !ok {
				return false, nil
			}
			continue
		}

		if m := likeRe.FindStringSubmatch(c); len(m) == 3 {
			field := strings.ToLower(m[1])
			pattern := strings.ToLower(strings.TrimSpace(m[2]))
			if !matchesLike(fieldValue(g, field), pattern) {
				return false, nil
			}
			continue
		}

		return false, fmt.Errorf("unsupported where-resource clause: %s", c)
	}
	return true, nil
}

func matchesField(g model.GeneratorDetection, field, value string) bool {
	field = strings.ToLower(field)
	value = strings.ToLower(value)
	if field == "kind" {
		kinds := []string{
			strings.ToLower(string(g.Kind)),
			strings.ToLower(mappedResourceKind(g.Kind)),
		}
		for _, k := range kinds {
			if k == value {
				return true
			}
		}
		return false
	}
	return strings.EqualFold(fieldValue(g, field), value)
}

func fieldValue(g model.GeneratorDetection, field string) string {
	switch strings.ToLower(field) {
	case "name", "resource_name":
		return strings.ToLower(g.Name)
	case "root":
		return strings.ToLower(g.Root)
	case "id":
		return strings.ToLower(g.ID)
	default:
		return ""
	}
}

func matchesLike(value, pattern string) bool {
	value = strings.ToLower(value)
	pattern = strings.ToLower(pattern)
	if !strings.Contains(pattern, "%") {
		return value == pattern
	}
	if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
		needle := strings.Trim(pattern, "%")
		return strings.Contains(value, needle)
	}
	if strings.HasPrefix(pattern, "%") {
		suffix := strings.TrimPrefix(pattern, "%")
		return strings.HasSuffix(value, suffix)
	}
	if strings.HasSuffix(pattern, "%") {
		prefix := strings.TrimSuffix(pattern, "%")
		return strings.HasPrefix(value, prefix)
	}
	parts := strings.Split(pattern, "%")
	idx := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		n := strings.Index(value[idx:], part)
		if n < 0 {
			return false
		}
		idx += n + len(part)
	}
	return true
}

func mappedResourceKind(kind model.GeneratorKind) string {
	switch kind {
	case model.GeneratorHelm:
		return "HelmRelease"
	case model.GeneratorScore:
		return "Application"
	case model.GeneratorSpringBoot:
		return "Kustomization"
	default:
		return "Resource"
	}
}

func mappedResourceType(kind model.GeneratorKind) string {
	switch kind {
	case model.GeneratorHelm:
		return "helm.toolkit.fluxcd.io/v2/HelmRelease"
	case model.GeneratorScore:
		return "argoproj.io/v1alpha1/Application"
	case model.GeneratorSpringBoot:
		return "kustomize.toolkit.fluxcd.io/v1/Kustomization"
	default:
		return "v1/Resource"
	}
}
