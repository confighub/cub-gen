package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const AdaptationSchemaVersion = "cub.confighub.io/deployment-adaptation-plan/v1"

type AdaptationOptions struct {
	GeneratedAt   string
	VariantFilter string
}

type AdaptationResult struct {
	SchemaVersion string                 `json:"schema_version"`
	ManifestPath  string                 `json:"manifest_path"`
	Name          string                 `json:"name"`
	Space         string                 `json:"space"`
	Ref           string                 `json:"ref"`
	GeneratedAt   string                 `json:"generated_at"`
	Summary       AdaptationSummary      `json:"summary"`
	Deployments   []AdaptationDeployment `json:"deployments,omitempty"`
	Diagnostics   []Diagnostic           `json:"diagnostics,omitempty"`
}

type AdaptationSummary struct {
	DeploymentCount          int `json:"deployment_count"`
	PlaceholderCount         int `json:"placeholder_count"`
	ProposedReplacementCount int `json:"proposed_replacement_count"`
	ApplyGateCount           int `json:"apply_gate_count"`
	DiagnosticCount          int `json:"diagnostic_count"`
}

type AdaptationDeployment struct {
	ID           string                  `json:"id"`
	Component    string                  `json:"component"`
	BaseVariant  string                  `json:"base_variant,omitempty"`
	Variant      string                  `json:"variant"`
	VariantKind  string                  `json:"variant_kind"`
	Target       string                  `json:"target"`
	RepoID       string                  `json:"repo_id"`
	RepoPath     string                  `json:"repo_path"`
	ApplyGate    ApplyGate               `json:"apply_gate"`
	Context      map[string]string       `json:"context,omitempty"`
	Placeholders []AdaptationPlaceholder `json:"placeholders,omitempty"`
	Proposals    []AdaptationProposal    `json:"proposals,omitempty"`
}

type ApplyGate struct {
	Name            string `json:"name"`
	State           string `json:"state"`
	Reason          string `json:"reason"`
	UnresolvedCount int    `json:"unresolved_count"`
}

type AdaptationPlaceholder struct {
	Token           string            `json:"token"`
	Value           string            `json:"value,omitempty"`
	ValueFrom       string            `json:"value_from,omitempty"`
	Owner           string            `json:"owner,omitempty"`
	Route           string            `json:"route,omitempty"`
	Status          string            `json:"status"`
	Reason          string            `json:"reason,omitempty"`
	OccurrenceCount int               `json:"occurrence_count"`
	Files           []PlaceholderFile `json:"files,omitempty"`
}

type PlaceholderFile struct {
	Path        string                  `json:"path"`
	Occurrences []PlaceholderOccurrence `json:"occurrences,omitempty"`
}

type PlaceholderOccurrence struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type AdaptationProposal struct {
	ID     string          `json:"id"`
	Title  string          `json:"title"`
	Action string          `json:"action"`
	Owner  string          `json:"owner,omitempty"`
	Route  string          `json:"route,omitempty"`
	Reason string          `json:"reason,omitempty"`
	Patch  AdaptationPatch `json:"patch"`
}

type AdaptationPatch struct {
	Path         string             `json:"path"`
	Action       string             `json:"action"`
	ContentType  string             `json:"content_type"`
	Replacements []PatchReplacement `json:"replacements"`
}

type PatchReplacement struct {
	Token       string `json:"token"`
	Value       string `json:"value"`
	Occurrences int    `json:"occurrences"`
}

func AdaptManifest(manifestPath string, opts AdaptationOptions) (AdaptationResult, error) {
	manifest, absManifestPath, err := LoadManifest(manifestPath)
	if err != nil {
		return AdaptationResult{}, err
	}
	return BuildAdaptationPlan(absManifestPath, manifest, opts)
}

func BuildAdaptationPlan(absManifestPath string, manifest Manifest, opts AdaptationOptions) (AdaptationResult, error) {
	generatedAt := strings.TrimSpace(opts.GeneratedAt)
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	baseDir := filepath.Dir(absManifestPath)
	result := AdaptationResult{
		SchemaVersion: AdaptationSchemaVersion,
		ManifestPath:  filepath.ToSlash(filepath.Clean(absManifestPath)),
		Name:          manifest.Name,
		Space:         manifest.Space,
		Ref:           manifest.Ref,
		GeneratedAt:   generatedAt,
	}

	repoByID := map[string]ManifestRepo{}
	for _, repo := range normalizeRepos(manifest.Repos) {
		repoByID[repo.ID] = repo
	}
	adaptations := normalizeAdaptations(manifest.Adaptations)
	if filter := strings.TrimSpace(opts.VariantFilter); filter != "" {
		adaptations = filterAdaptations(adaptations, filter)
		if len(adaptations) == 0 {
			return AdaptationResult{}, fmt.Errorf("variant filter %q matched no adaptations", filter)
		}
	}
	if len(adaptations) == 0 {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "warning",
			Code:     "no_adaptations_declared",
			Message:  "platform manifest declares no deployment adaptations",
		})
		result.refreshAdaptationSummary()
		return result, nil
	}

	for _, spec := range adaptations {
		deployment, diagnostics := buildDeploymentAdaptation(baseDir, repoByID, spec)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if deployment.ID != "" {
			result.Deployments = append(result.Deployments, deployment)
		}
	}
	sortAdaptationResult(&result)
	result.refreshAdaptationSummary()
	return result, nil
}

func buildDeploymentAdaptation(baseDir string, repoByID map[string]ManifestRepo, spec ManifestAdaptation) (AdaptationDeployment, []Diagnostic) {
	var diagnostics []Diagnostic
	repo, ok := repoByID[spec.Repo]
	if !ok {
		return AdaptationDeployment{}, []Diagnostic{{
			Severity: "error",
			Code:     "missing_adaptation_repo",
			RepoID:   spec.Repo,
			Message:  fmt.Sprintf("adaptation %s references repo %q that is not declared", adaptationID(spec), spec.Repo),
		}}
	}

	component := firstNonEmpty(spec.Component, repo.Component)
	variant := firstNonEmpty(spec.Variant, repo.Variant)
	target := strings.TrimSpace(repo.Target)
	variantKind, variantKindInvalid := resolveVariantKind(repo.VariantKind, target)
	repoPath := normalizeRelPath(repo.Path)
	deployment := AdaptationDeployment{
		ID:          firstNonEmpty(spec.ID, component+"/"+variant),
		Component:   component,
		BaseVariant: strings.TrimSpace(spec.BaseVariant),
		Variant:     variant,
		VariantKind: variantKind,
		Target:      target,
		RepoID:      repo.ID,
		RepoPath:    repoPath,
		Context:     normalizedContext(spec.Context),
	}
	if variantKindInvalid {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "warning",
			Code:     "invalid_variant_kind",
			RepoID:   repo.ID,
			Path:     repoPath,
			Message:  fmt.Sprintf("adaptation %s repo variant_kind %q is not supported or is inconsistent with target %q", deployment.ID, strings.TrimSpace(repo.VariantKind), target),
		})
	}
	if component == "" || variant == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "missing_adaptation_variant",
			RepoID:   repo.ID,
			Path:     repoPath,
			Message:  "adaptation repo must name a component and variant",
		})
		return AdaptationDeployment{}, diagnostics
	}
	if variantKind != "deployment" {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "not_deployment_variant",
			RepoID:   repo.ID,
			Path:     repoPath,
			Message:  fmt.Sprintf("adaptation %s requires a Deployment Variant; current ConfigHub rule requires a Target", deployment.ID),
		})
		return deployment, diagnostics
	}
	if deployment.BaseVariant != "" {
		diagnostics = append(diagnostics, validateBaseVariant(repoByID, component, deployment.BaseVariant)...)
	}
	if repoPath == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "missing_repo_path",
			RepoID:   repo.ID,
			Message:  fmt.Sprintf("adaptation %s repo has no path", deployment.ID),
		})
		return deployment, diagnostics
	}
	absRepo := filepath.Join(baseDir, filepath.FromSlash(repoPath))
	info, err := os.Stat(absRepo)
	if err != nil || !info.IsDir() {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "missing_repo",
			RepoID:   repo.ID,
			Path:     repoPath,
			Message:  fmt.Sprintf("adaptation %s repo path does not exist or is not a directory", deployment.ID),
		})
		return deployment, diagnostics
	}

	for i, placeholderSpec := range spec.Placeholders {
		placeholder, proposal, placeholderDiagnostics := buildPlaceholderPlan(absRepo, deployment, placeholderSpec, i)
		diagnostics = append(diagnostics, placeholderDiagnostics...)
		if placeholder.Token != "" {
			deployment.Placeholders = append(deployment.Placeholders, placeholder)
		}
		if proposal.ID != "" {
			deployment.Proposals = append(deployment.Proposals, proposal)
		}
	}
	sortDeploymentAdaptation(&deployment)
	deployment.ApplyGate = applyGateFor(spec.ApplyGate, deployment)
	return deployment, diagnostics
}

func buildPlaceholderPlan(absRepo string, deployment AdaptationDeployment, spec ManifestAdaptationPlaceholder, index int) (AdaptationPlaceholder, AdaptationProposal, []Diagnostic) {
	var diagnostics []Diagnostic
	token := strings.TrimSpace(spec.Token)
	placeholder := AdaptationPlaceholder{
		Token:     token,
		Value:     strings.TrimSpace(spec.Value),
		ValueFrom: strings.TrimSpace(spec.ValueFrom),
		Owner:     strings.TrimSpace(spec.Owner),
		Route:     strings.TrimSpace(spec.Route),
		Reason:    strings.TrimSpace(spec.Reason),
		Status:    "proposed",
	}
	if token == "" {
		return AdaptationPlaceholder{}, AdaptationProposal{}, []Diagnostic{{
			Severity: "error",
			Code:     "invalid_adaptation_placeholder",
			RepoID:   deployment.RepoID,
			Path:     deployment.RepoPath,
			Message:  fmt.Sprintf("adaptation %s placeholder %d has no token", deployment.ID, index+1),
		}}
	}
	if placeholder.Value == "" && placeholder.ValueFrom != "" {
		placeholder.Value = strings.TrimSpace(deployment.Context[placeholder.ValueFrom])
		if placeholder.Value == "" {
			placeholder.Status = "blocked"
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "missing_adaptation_context",
				RepoID:   deployment.RepoID,
				Path:     deployment.RepoPath,
				Message:  fmt.Sprintf("adaptation %s placeholder %s references missing context key %q", deployment.ID, token, placeholder.ValueFrom),
			})
		}
	}
	if placeholder.Value == "" && placeholder.ValueFrom == "" {
		placeholder.Status = "blocked"
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "missing_adaptation_value",
			RepoID:   deployment.RepoID,
			Path:     deployment.RepoPath,
			Message:  fmt.Sprintf("adaptation %s placeholder %s has no value or value_from", deployment.ID, token),
		})
	}

	files := normalizedStrings(spec.Files)
	if len(files) == 0 {
		placeholder.Status = "blocked"
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "missing_placeholder_files",
			RepoID:   deployment.RepoID,
			Path:     deployment.RepoPath,
			Message:  fmt.Sprintf("adaptation %s placeholder %s declares no files", deployment.ID, token),
		})
	}
	for _, file := range files {
		rel := normalizeRelPath(file)
		occurrences, readErr := findTokenOccurrences(filepath.Join(absRepo, filepath.FromSlash(rel)), token)
		if readErr != nil {
			placeholder.Status = "blocked"
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "missing_placeholder_file",
				RepoID:   deployment.RepoID,
				Path:     rel,
				Message:  fmt.Sprintf("adaptation %s placeholder %s file cannot be read: %v", deployment.ID, token, readErr),
			})
			continue
		}
		placeholder.Files = append(placeholder.Files, PlaceholderFile{
			Path:        rel,
			Occurrences: occurrences,
		})
		placeholder.OccurrenceCount += len(occurrences)
	}
	if placeholder.OccurrenceCount == 0 && placeholder.Status == "proposed" {
		placeholder.Status = "not_found"
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "warning",
			Code:     "placeholder_not_found",
			RepoID:   deployment.RepoID,
			Path:     deployment.RepoPath,
			Message:  fmt.Sprintf("adaptation %s placeholder %s was not found in declared files", deployment.ID, token),
		})
	}
	if placeholder.Status != "proposed" {
		return placeholder, AdaptationProposal{}, diagnostics
	}
	return placeholder, AdaptationProposal{
		ID:     fmt.Sprintf("replace-%02d-%s", index+1, sanitizeID(token)),
		Title:  fmt.Sprintf("Replace %s for %s", token, deployment.ID),
		Action: "review-patch",
		Owner:  placeholder.Owner,
		Route:  firstNonEmpty(placeholder.Route, "apply-here"),
		Reason: placeholder.Reason,
		Patch: AdaptationPatch{
			Path:        ".cub-gen/adapt/" + sanitizeID(deployment.ID) + "/" + sanitizeID(token) + ".replacement.json",
			Action:      "create-sidecar",
			ContentType: "application/json",
			Replacements: []PatchReplacement{{
				Token:       token,
				Value:       placeholder.Value,
				Occurrences: placeholder.OccurrenceCount,
			}},
		},
	}, diagnostics
}

func findTokenOccurrences(path, token string) ([]PlaceholderOccurrence, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("placeholder token is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	var occurrences []PlaceholderOccurrence
	for lineIndex, line := range lines {
		searchFrom := 0
		for {
			idx := strings.Index(line[searchFrom:], token)
			if idx < 0 {
				break
			}
			column := searchFrom + idx + 1
			occurrences = append(occurrences, PlaceholderOccurrence{
				Line:   lineIndex + 1,
				Column: column,
			})
			searchFrom = searchFrom + idx + len(token)
		}
	}
	return occurrences, nil
}

func applyGateFor(name string, deployment AdaptationDeployment) ApplyGate {
	gateName := strings.TrimSpace(name)
	if gateName == "" {
		gateName = "vet-placeholders"
	}
	unresolved := 0
	for _, placeholder := range deployment.Placeholders {
		if placeholder.Status != "proposed" || placeholder.OccurrenceCount > 0 {
			unresolved++
		}
	}
	if unresolved == 0 {
		return ApplyGate{
			Name:   gateName,
			State:  "clear",
			Reason: "no declared placeholders remain in this Deployment Variant",
		}
	}
	return ApplyGate{
		Name:            gateName,
		State:           "blocked-before-adaptation",
		Reason:          "declared placeholders are still present; review and apply the adaptation proposals before deployment",
		UnresolvedCount: unresolved,
	}
}

func validateBaseVariant(repoByID map[string]ManifestRepo, component, baseVariant string) []Diagnostic {
	for _, repo := range repoByID {
		if strings.TrimSpace(repo.Component) != component || strings.TrimSpace(repo.Variant) != baseVariant {
			continue
		}
		kind, invalid := resolveVariantKind(repo.VariantKind, repo.Target)
		if invalid || kind != "base" {
			return []Diagnostic{{
				Severity: "warning",
				Code:     "base_variant_has_target",
				RepoID:   repo.ID,
				Path:     normalizeRelPath(repo.Path),
				Message:  fmt.Sprintf("base_variant %s/%s should have no Target under the current ConfigHub rule", component, baseVariant),
			}}
		}
		return nil
	}
	return []Diagnostic{{
		Severity: "warning",
		Code:     "missing_base_variant",
		Message:  fmt.Sprintf("base_variant %s/%s was not found in manifest repos", component, baseVariant),
	}}
}

func normalizeAdaptations(adaptations []ManifestAdaptation) []ManifestAdaptation {
	out := make([]ManifestAdaptation, 0, len(adaptations))
	for _, raw := range adaptations {
		spec := raw
		spec.ID = strings.TrimSpace(spec.ID)
		spec.Component = strings.TrimSpace(spec.Component)
		spec.Variant = strings.TrimSpace(spec.Variant)
		spec.Repo = strings.TrimSpace(spec.Repo)
		spec.BaseVariant = strings.TrimSpace(spec.BaseVariant)
		spec.ApplyGate = strings.TrimSpace(spec.ApplyGate)
		if spec.ID == "" && spec.Component != "" && spec.Variant != "" {
			spec.ID = spec.Component + "/" + spec.Variant
		}
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool {
		return adaptationID(out[i]) < adaptationID(out[j])
	})
	return out
}

func filterAdaptations(in []ManifestAdaptation, filter string) []ManifestAdaptation {
	var out []ManifestAdaptation
	for _, spec := range in {
		id := adaptationID(spec)
		if id == filter || spec.Variant == filter || spec.Component+"/"+spec.Variant == filter {
			out = append(out, spec)
		}
	}
	return out
}

func adaptationID(spec ManifestAdaptation) string {
	if strings.TrimSpace(spec.ID) != "" {
		return strings.TrimSpace(spec.ID)
	}
	if strings.TrimSpace(spec.Component) != "" && strings.TrimSpace(spec.Variant) != "" {
		return strings.TrimSpace(spec.Component) + "/" + strings.TrimSpace(spec.Variant)
	}
	if strings.TrimSpace(spec.Repo) != "" {
		return strings.TrimSpace(spec.Repo)
	}
	return "adaptation"
}

func normalizedContext(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range in {
		cleanKey := strings.TrimSpace(key)
		cleanValue := strings.TrimSpace(value)
		if cleanKey == "" || cleanValue == "" {
			continue
		}
		out[cleanKey] = cleanValue
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sortDeploymentAdaptation(deployment *AdaptationDeployment) {
	sort.Slice(deployment.Placeholders, func(i, j int) bool {
		return deployment.Placeholders[i].Token < deployment.Placeholders[j].Token
	})
	sort.Slice(deployment.Proposals, func(i, j int) bool {
		return deployment.Proposals[i].ID < deployment.Proposals[j].ID
	})
	for i := range deployment.Placeholders {
		sort.Slice(deployment.Placeholders[i].Files, func(left, right int) bool {
			return deployment.Placeholders[i].Files[left].Path < deployment.Placeholders[i].Files[right].Path
		})
	}
}

func sortAdaptationResult(result *AdaptationResult) {
	sort.Slice(result.Deployments, func(i, j int) bool {
		return result.Deployments[i].ID < result.Deployments[j].ID
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
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		return left.Message < right.Message
	})
}

func (result *AdaptationResult) refreshAdaptationSummary() {
	summary := AdaptationSummary{
		DeploymentCount: len(result.Deployments),
		DiagnosticCount: len(result.Diagnostics),
	}
	for _, deployment := range result.Deployments {
		if deployment.ApplyGate.State != "" && deployment.ApplyGate.State != "clear" {
			summary.ApplyGateCount++
		}
		summary.PlaceholderCount += len(deployment.Placeholders)
		for _, proposal := range deployment.Proposals {
			if len(proposal.Patch.Replacements) > 0 {
				summary.ProposedReplacementCount++
			}
		}
	}
	result.Summary = summary
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if clean := strings.TrimSpace(value); clean != "" {
			return clean
		}
	}
	return ""
}
