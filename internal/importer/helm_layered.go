package importer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

type helmApplicationSetDoc struct {
	Path       string
	Selector   map[string]string
	ValueFiles []string
}

type helmClusterInventoryDoc struct {
	Path   string
	Name   string
	Labels map[string]string
}

type helmManagedCatalogDoc struct {
	Path             string
	RequiredSecurity string
}

type helmCustomerCatalogDoc struct {
	Path             string
	ValuesFile       string
	OverrideSecurity string
	Enabled          *bool
}

func helmLayeredAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.HelmLayeredAnalysis {
	if g.Kind != model.GeneratorHelm {
		return nil
	}

	appsetPath := firstInputPathForRole(g.Kind, g.Inputs, "application-set")
	clusterPaths := inputPathsForRole(g.Kind, g.Inputs, "cluster-inventory")
	managedCatalogPaths := inputPathsForRole(g.Kind, g.Inputs, "managed-service-catalog")
	customerCatalogPaths := inputPathsForRole(g.Kind, g.Inputs, "customer-service-catalog")
	helmPaths := helmProvenancePathsForGenerator(detection.Repo, g)
	dependencyCharts := helmDependencyChartsForGenerator(detection.Repo, helmPaths)
	crdPaths := helmCRDPathsForChart(detection.Repo, helmPaths.ChartPath)
	hookTemplates := helmHookTemplates(detection.Repo, helmPaths.ChartPath)
	hookTemplatePaths := make([]string, 0, len(hookTemplates))
	for _, hookTemplate := range hookTemplates {
		hookTemplatePaths = append(hookTemplatePaths, hookTemplate.Path)
	}
	valuesSchemaPath := helmValuesSchemaPath(detection.Repo, helmPaths.ChartPath)
	schemaState, schemaViolations := helmValuesSchemaValidation(detection.Repo, helmPaths, valuesSchemaPath)

	if appsetPath == "" && len(clusterPaths) == 0 && len(managedCatalogPaths) == 0 && len(customerCatalogPaths) == 0 &&
		len(dependencyCharts) == 0 && len(crdPaths) == 0 && len(hookTemplatePaths) == 0 && valuesSchemaPath == "" {
		return nil
	}

	analysis := &model.HelmLayeredAnalysis{
		ApplicationSetPath:         appsetPath,
		ClusterInventoryPaths:      append([]string(nil), clusterPaths...),
		ManagedCatalogPaths:        append([]string(nil), managedCatalogPaths...),
		CustomerCatalogPaths:       append([]string(nil), customerCatalogPaths...),
		DependencyCharts:           append([]model.HelmDependencyChart(nil), dependencyCharts...),
		CRDPaths:                   append([]string(nil), crdPaths...),
		HookTemplates:              append([]model.HelmHookTemplate(nil), hookTemplates...),
		HookTemplatePaths:          append([]string(nil), hookTemplatePaths...),
		ValuesSchemaPath:           valuesSchemaPath,
		SchemaValidationState:      schemaState,
		SchemaValidationViolations: append([]string(nil), schemaViolations...),
		SelectedValueFiles:         append([]string(nil), helmPaths.ValuesPaths...),
	}

	if appsetPath != "" {
		appsetDoc, err := parseHelmApplicationSetFile(detection.Repo, appsetPath)
		if err == nil {
			analysis.ClusterSelector = selectorString(appsetDoc.Selector)
			if len(appsetDoc.ValueFiles) > 0 {
				analysis.SelectedValueFiles = append([]string(nil), appsetDoc.ValueFiles...)
			}
			switch {
			case len(appsetDoc.Selector) == 0:
				analysis.GenerationDecisionState = "unresolved"
				analysis.GenerationDecisionReason = "ApplicationSet input is present, but cub-gen could not find a cluster matchLabels selector to attribute yet."
			case len(clusterPaths) == 0:
				analysis.GenerationDecisionState = "unresolved"
				analysis.GenerationDecisionReason = "ApplicationSet selector is present, but no cluster inventory inputs were observed to prove which clusters match it."
			default:
				matched := matchedHelmClusterInventories(detection.Repo, clusterPaths, appsetDoc.Selector)
				if len(matched) == 0 {
					analysis.GenerationDecisionState = "unresolved"
					analysis.GenerationDecisionReason = fmt.Sprintf("ApplicationSet selector %s is present, but none of the observed cluster inventory inputs match it.", selectorString(appsetDoc.Selector))
				} else {
					analysis.MatchedClusters = matched
					analysis.GenerationDecisionState = "attributed"
					analysis.GenerationDecisionReason = fmt.Sprintf("ApplicationSet selector %s matches cluster inventory %s and selects value files %s.", selectorString(appsetDoc.Selector), strings.Join(matched, ", "), strings.Join(analysis.SelectedValueFiles, ", "))
				}
			}
		} else {
			analysis.GenerationDecisionState = "unresolved"
			analysis.GenerationDecisionReason = fmt.Sprintf("ApplicationSet input %s was observed, but cub-gen could not parse it deterministically yet.", appsetPath)
		}
	}

	managedDocs := parseHelmManagedCatalogs(detection.Repo, managedCatalogPaths)
	customerDocs := parseHelmCustomerCatalogs(detection.Repo, customerCatalogPaths)
	requiredControl, requiredPath := firstRequiredHelmSecurityControl(managedDocs)
	selectedOverlay := helmPaths.OverlayValuesPath
	customerOverride := matchingCustomerCatalog(customerDocs, selectedOverlay)
	if requiredControl != "" {
		analysis.SecurityControl = requiredControl
		analysis.SecurityControlPath = requiredPath
		if customerOverride.Path != "" {
			analysis.SecurityOverridePath = customerOverride.Path
		}
		switch {
		case customerOverride.Enabled != nil && !*customerOverride.Enabled:
			analysis.SecurityDecisionState = "blocked"
			analysis.SecurityDecisionReason = fmt.Sprintf("Customer service catalog %s weakens the platform-required %s control from %s.", customerOverride.Path, requiredControl, requiredPath)
		case customerOverride.Enabled != nil:
			analysis.SecurityDecisionState = "allow"
			analysis.SecurityDecisionReason = fmt.Sprintf("Customer service catalog %s keeps the platform-required %s control enabled.", customerOverride.Path, requiredControl)
		default:
			analysis.SecurityDecisionState = "allow"
			analysis.SecurityDecisionReason = fmt.Sprintf("No customer override weakens the platform-required %s control from %s.", requiredControl, requiredPath)
		}
	} else if len(managedCatalogPaths) > 0 || len(customerCatalogPaths) > 0 {
		analysis.SecurityDecisionState = "unresolved"
		analysis.SecurityDecisionReason = "Layered service catalog inputs were observed, but cub-gen could not classify a supported security control from them yet."
	}

	return analysis
}

func helmDependencyChartsForGenerator(repoPath string, hints helmProvenancePaths) []model.HelmDependencyChart {
	deps := parseHelmDependencyDocs(repoPath, hints.ChartPath)
	out := make([]model.HelmDependencyChart, 0, len(deps))
	for _, dep := range deps {
		layer := strings.TrimSpace(dep.Alias)
		if layer == "" {
			layer = strings.TrimSpace(dep.Name)
		}
		out = append(out, model.HelmDependencyChart{
			Name:       dep.Name,
			Alias:      dep.Alias,
			Layer:      layer,
			Condition:  dep.Condition,
			Repository: dep.Repository,
			Version:    dep.Version,
			ChartPath:  dep.ChartPath,
			ValuesPath: dep.ValuesPath,
		})
	}
	return out
}

func parseHelmDependencyDocs(repoPath, chartPath string) []helmDependencyDoc {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(chartPath) == "" {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(chartPath)))
	if err != nil {
		return nil
	}

	var chartDoc struct {
		Dependencies []helmDependencyDoc `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(content, &chartDoc); err != nil {
		return nil
	}

	chartDir := helmChartDir(chartPath)
	out := make([]helmDependencyDoc, 0, len(chartDoc.Dependencies))
	for _, dep := range chartDoc.Dependencies {
		layer := strings.TrimSpace(dep.Alias)
		if layer == "" {
			layer = strings.TrimSpace(dep.Name)
		}
		chartCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", layer, "Chart.yaml"))
		valuesCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", layer, "values.yaml"))
		if !fileExists(filepath.Join(repoPath, filepath.FromSlash(chartCandidate))) && strings.TrimSpace(dep.Name) != "" && dep.Name != layer {
			fallbackChartCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", dep.Name, "Chart.yaml"))
			fallbackValuesCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", dep.Name, "values.yaml"))
			if fileExists(filepath.Join(repoPath, filepath.FromSlash(fallbackChartCandidate))) {
				chartCandidate = fallbackChartCandidate
				valuesCandidate = fallbackValuesCandidate
			}
		}
		if !fileExists(filepath.Join(repoPath, filepath.FromSlash(chartCandidate))) {
			chartCandidate = ""
		}
		if !fileExists(filepath.Join(repoPath, filepath.FromSlash(valuesCandidate))) {
			valuesCandidate = ""
		}
		dep.ChartPath = chartCandidate
		dep.ValuesPath = valuesCandidate
		out = append(out, dep)
	}

	sort.Slice(out, func(i, j int) bool {
		left := out[i].Alias
		if strings.TrimSpace(left) == "" {
			left = out[i].Name
		}
		right := out[j].Alias
		if strings.TrimSpace(right) == "" {
			right = out[j].Name
		}
		return left < right
	})
	return out
}

func helmCRDPathsForChart(repoPath, chartPath string) []string {
	chartDir := helmChartDir(chartPath)
	crdDir := filepath.Join(repoPath, filepath.FromSlash(chartDir), "crds")
	info, err := os.Stat(crdDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	out := make([]string, 0)
	_ = filepath.WalkDir(crdDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out
}

func helmHookTemplates(repoPath, chartPath string) []model.HelmHookTemplate {
	chartDir := helmChartDir(chartPath)
	templatesDir := filepath.Join(repoPath, filepath.FromSlash(chartDir), "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	out := make([]model.HelmHookTemplate, 0)
	_ = filepath.WalkDir(templatesDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		hooks := helmHookAnnotations(string(content))
		if len(hooks) == 0 {
			return nil
		}
		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		out = append(out, model.HelmHookTemplate{
			Path:  filepath.ToSlash(rel),
			Hooks: hooks,
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func helmHookAnnotations(content string) []string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "helm.sh/hook:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "helm.sh/hook:"))
		raw = strings.Trim(raw, "\"'")
		for _, hook := range strings.Split(raw, ",") {
			hook = strings.TrimSpace(hook)
			if hook == "" {
				continue
			}
			if _, ok := seen[hook]; ok {
				continue
			}
			seen[hook] = struct{}{}
			out = append(out, hook)
		}
	}
	sort.Strings(out)
	return out
}

func helmValuesSchemaPath(repoPath, chartPath string) string {
	chartDir := helmChartDir(chartPath)
	candidate := filepath.Join(repoPath, filepath.FromSlash(chartDir), "values.schema.json")
	if !fileExists(candidate) {
		return ""
	}
	rel, err := filepath.Rel(repoPath, candidate)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func helmValuesSchemaValidation(repoPath string, hints helmProvenancePaths, valuesSchemaPath string) (string, []string) {
	if strings.TrimSpace(valuesSchemaPath) == "" {
		return "", nil
	}

	schemaContent, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(valuesSchemaPath)))
	if err != nil {
		return "unresolved", []string{fmt.Sprintf("%s: %v", valuesSchemaPath, err)}
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(valuesSchemaPath, strings.NewReader(string(schemaContent))); err != nil {
		return "unresolved", []string{fmt.Sprintf("%s: %v", valuesSchemaPath, err)}
	}
	schema, err := compiler.Compile(valuesSchemaPath)
	if err != nil {
		return "unresolved", []string{fmt.Sprintf("%s: %v", valuesSchemaPath, err)}
	}

	rootValuesPaths := make([]string, 0, len(hints.ValuesPaths))
	for _, path := range hints.ValuesPaths {
		if helmIsSubchartPath(path) {
			continue
		}
		rootValuesPaths = append(rootValuesPaths, path)
	}
	sort.SliceStable(rootValuesPaths, func(i, j int) bool {
		leftRank := helmValuesPathPrecedence(rootValuesPaths[i], hints.PrimaryValuesPath, hints.ChartPath)
		rightRank := helmValuesPathPrecedence(rootValuesPaths[j], hints.PrimaryValuesPath, hints.ChartPath)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return rootValuesPaths[i] < rootValuesPaths[j]
	})

	merged := map[string]any{}
	for i := len(rootValuesPaths) - 1; i >= 0; i-- {
		path := rootValuesPaths[i]
		payload, err := yamlFileToJSONCompatible(filepath.Join(repoPath, filepath.FromSlash(path)))
		if err != nil {
			return "unresolved", []string{fmt.Sprintf("%s: %v", path, err)}
		}
		merged = mergeJSONObjects(merged, payload)
	}

	if err := schema.Validate(merged); err != nil {
		return "invalid", jsonschemaValidationMessages(err)
	}
	return "valid", nil
}

func yamlFileToJSONCompatible(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, err
	}
	converted := yamlToJSONCompatible(raw)
	if converted == nil {
		return map[string]any{}, nil
	}
	if asMap, ok := converted.(map[string]any); ok {
		return asMap, nil
	}
	return nil, fmt.Errorf("top-level values document must be a mapping")
}

func yamlToJSONCompatible(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = yamlToJSONCompatible(value)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[fmt.Sprint(key)] = yamlToJSONCompatible(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, value := range typed {
			out = append(out, yamlToJSONCompatible(value))
		}
		return out
	default:
		return typed
	}
}

func mergeJSONObjects(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		if baseMap, ok := out[key].(map[string]any); ok {
			if overlayMap, ok := value.(map[string]any); ok {
				out[key] = mergeJSONObjects(baseMap, overlayMap)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func jsonschemaValidationMessages(err error) []string {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			return nil
		}
		return []string{msg}
	}
	leaves := flattenJSONSchemaValidationErrors(validationErr)
	out := make([]string, 0, len(leaves))
	for _, leaf := range leaves {
		location := strings.TrimSpace(leaf.InstanceLocation)
		if location == "" {
			location = "/"
		}
		out = append(out, fmt.Sprintf("%s: %s", location, strings.TrimSpace(leaf.Message)))
	}
	sort.Strings(out)
	return out
}

func flattenJSONSchemaValidationErrors(err *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if err == nil {
		return nil
	}
	if len(err.Causes) == 0 {
		return []*jsonschema.ValidationError{err}
	}
	out := make([]*jsonschema.ValidationError, 0, len(err.Causes))
	for _, cause := range err.Causes {
		out = append(out, flattenJSONSchemaValidationErrors(cause)...)
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func parseHelmApplicationSetFile(repo, path string) (helmApplicationSetDoc, error) {
	type applicationSet struct {
		Spec struct {
			Generators []struct {
				Clusters struct {
					Selector struct {
						MatchLabels map[string]string `yaml:"matchLabels"`
					} `yaml:"selector"`
				} `yaml:"clusters"`
			} `yaml:"generators"`
			Template struct {
				Spec struct {
					Source struct {
						Helm struct {
							ValueFiles []string `yaml:"valueFiles"`
						} `yaml:"helm"`
					} `yaml:"source"`
				} `yaml:"spec"`
			} `yaml:"template"`
		} `yaml:"spec"`
	}

	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return helmApplicationSetDoc{}, err
	}

	var doc applicationSet
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return helmApplicationSetDoc{}, err
	}

	out := helmApplicationSetDoc{Path: filepath.ToSlash(path)}
	for _, generator := range doc.Spec.Generators {
		if len(generator.Clusters.Selector.MatchLabels) == 0 {
			continue
		}
		out.Selector = map[string]string{}
		for k, v := range generator.Clusters.Selector.MatchLabels {
			out.Selector[k] = v
		}
		break
	}
	out.ValueFiles = uniqueStringsInOrder(doc.Spec.Template.Spec.Source.Helm.ValueFiles)
	return out, nil
}

func matchedHelmClusterInventories(repo string, paths []string, selector map[string]string) []string {
	matched := make([]string, 0, len(paths))
	for _, path := range paths {
		doc, err := parseHelmClusterInventoryFile(repo, path)
		if err != nil || doc.Name == "" {
			continue
		}
		if clusterLabelsMatch(doc.Labels, selector) {
			matched = append(matched, doc.Name)
		}
	}
	return uniqueSortedStrings(matched)
}

func parseHelmClusterInventoryFile(repo, path string) (helmClusterInventoryDoc, error) {
	type clusterInventory struct {
		Metadata struct {
			Name   string            `yaml:"name"`
			Labels map[string]string `yaml:"labels"`
		} `yaml:"metadata"`
	}

	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return helmClusterInventoryDoc{}, err
	}

	var doc clusterInventory
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return helmClusterInventoryDoc{}, err
	}
	return helmClusterInventoryDoc{
		Path:   filepath.ToSlash(path),
		Name:   strings.TrimSpace(doc.Metadata.Name),
		Labels: doc.Metadata.Labels,
	}, nil
}

func parseHelmManagedCatalogs(repo string, paths []string) []helmManagedCatalogDoc {
	type managedCatalog struct {
		Spec struct {
			SecurityControls struct {
				OAuth2Proxy struct {
					Required bool `yaml:"required"`
				} `yaml:"oauth2Proxy"`
			} `yaml:"securityControls"`
		} `yaml:"spec"`
	}

	out := make([]helmManagedCatalogDoc, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			continue
		}
		var doc managedCatalog
		if err := yaml.Unmarshal(content, &doc); err != nil {
			continue
		}
		entry := helmManagedCatalogDoc{Path: filepath.ToSlash(path)}
		if doc.Spec.SecurityControls.OAuth2Proxy.Required {
			entry.RequiredSecurity = "oauth2Proxy"
		}
		out = append(out, entry)
	}
	return out
}

func parseHelmCustomerCatalogs(repo string, paths []string) []helmCustomerCatalogDoc {
	type customerCatalog struct {
		Spec struct {
			Overlay struct {
				ValuesFile string `yaml:"valuesFile"`
			} `yaml:"overlay"`
			SecurityOverrides struct {
				OAuth2Proxy struct {
					Enabled *bool `yaml:"enabled"`
				} `yaml:"oauth2Proxy"`
			} `yaml:"securityOverrides"`
		} `yaml:"spec"`
	}

	out := make([]helmCustomerCatalogDoc, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			continue
		}
		var doc customerCatalog
		if err := yaml.Unmarshal(content, &doc); err != nil {
			continue
		}
		entry := helmCustomerCatalogDoc{
			Path:       filepath.ToSlash(path),
			ValuesFile: strings.TrimSpace(doc.Spec.Overlay.ValuesFile),
			Enabled:    doc.Spec.SecurityOverrides.OAuth2Proxy.Enabled,
		}
		if doc.Spec.SecurityOverrides.OAuth2Proxy.Enabled != nil {
			entry.OverrideSecurity = "oauth2Proxy"
		}
		out = append(out, entry)
	}
	return out
}

func firstRequiredHelmSecurityControl(docs []helmManagedCatalogDoc) (string, string) {
	for _, doc := range docs {
		if strings.TrimSpace(doc.RequiredSecurity) == "" {
			continue
		}
		return doc.RequiredSecurity, doc.Path
	}
	return "", ""
}

func matchingCustomerCatalog(docs []helmCustomerCatalogDoc, selectedOverlay string) helmCustomerCatalogDoc {
	for _, doc := range docs {
		if selectedOverlay != "" && doc.ValuesFile == selectedOverlay {
			return doc
		}
	}
	if len(docs) > 0 {
		return docs[0]
	}
	return helmCustomerCatalogDoc{}
}

func clusterLabelsMatch(labels, selector map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return true
}

func selectorString(selector map[string]string) string {
	if len(selector) == 0 {
		return ""
	}
	keys := make([]string, 0, len(selector))
	for key := range selector {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+selector[key])
	}
	return strings.Join(parts, ", ")
}

func uniqueStringsInOrder(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
