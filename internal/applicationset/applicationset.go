package applicationset

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ModeAuthoritative                     = "authoritative"
	ModeObservedOnly                      = "observed-only"
	ModeAuthoritativeExpansionUnavailable = "authoritative-expansion-unavailable"
)

type ListElement struct {
	Name   string
	Values map[string]string
}

type Document struct {
	Path                      string
	Name                      string
	TemplateName              string
	TemplateSourcePath        string
	GeneratorTypes            []string
	UnsupportedGeneratorTypes []string
	ListElements              []ListElement
	ClusterSelector           map[string]string
}

type ClusterInventory struct {
	Path   string
	Name   string
	Server string
	Labels map[string]string
}

type GeneratedApplication struct {
	Name          string
	GeneratorType string
	Reason        string
	InventoryPath string
	Cluster       string
	ListElement   string
	SourcePath    string
}

type Analysis struct {
	ApplicationSetPath        string
	GeneratorTypes            []string
	UnsupportedGeneratorTypes []string
	ClusterInventoryPaths     []string
	MatchedClusters           []string
	ListElementNames          []string
	Mode                      string
	ModeReason                string
	GeneratedApplications     []GeneratedApplication
}

func IsApplicationSetFile(repo, path string) bool {
	doc, err := ParseFile(repo, path)
	return err == nil && strings.TrimSpace(doc.Path) != ""
}

func ParseFile(repo, path string) (Document, error) {
	type rawApplicationSet struct {
		Kind     string `yaml:"kind"`
		Metadata struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Spec struct {
			Generators []map[string]any `yaml:"generators"`
			Template   struct {
				Metadata struct {
					Name string `yaml:"name"`
				} `yaml:"metadata"`
				Spec struct {
					Source struct {
						Path string `yaml:"path"`
					} `yaml:"source"`
				} `yaml:"spec"`
			} `yaml:"template"`
		} `yaml:"spec"`
	}

	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return Document{}, err
	}

	var raw rawApplicationSet
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return Document{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(raw.Kind), "ApplicationSet") {
		return Document{}, fmt.Errorf("%s is not an ApplicationSet", path)
	}

	out := Document{
		Path:               filepath.ToSlash(path),
		Name:               strings.TrimSpace(raw.Metadata.Name),
		TemplateName:       strings.TrimSpace(raw.Spec.Template.Metadata.Name),
		TemplateSourcePath: strings.TrimSpace(raw.Spec.Template.Spec.Source.Path),
	}
	if out.Name == "" {
		out.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	generatorTypeSet := map[string]struct{}{}
	unsupportedSet := map[string]struct{}{}
	for _, generator := range raw.Spec.Generators {
		switch {
		case hasGeneratorKey(generator, "list"):
			generatorTypeSet["list"] = struct{}{}
			out.ListElements = append(out.ListElements, parseListElements(generator["list"])...)
		case hasGeneratorKey(generator, "clusters"):
			generatorTypeSet["clusters"] = struct{}{}
			if selector := parseClusterSelector(generator["clusters"]); len(selector) > 0 {
				out.ClusterSelector = selector
			}
		case hasGeneratorKey(generator, "git"):
			generatorTypeSet["git"] = struct{}{}
			unsupportedSet["git"] = struct{}{}
		case hasGeneratorKey(generator, "matrix"):
			generatorTypeSet["matrix"] = struct{}{}
			unsupportedSet["matrix"] = struct{}{}
		case hasGeneratorKey(generator, "merge"):
			generatorTypeSet["merge"] = struct{}{}
			unsupportedSet["merge"] = struct{}{}
		default:
			for key := range generator {
				key = strings.TrimSpace(strings.ToLower(key))
				if key == "" {
					continue
				}
				generatorTypeSet[key] = struct{}{}
				unsupportedSet[key] = struct{}{}
			}
		}
	}
	out.GeneratorTypes = sortedKeys(generatorTypeSet)
	out.UnsupportedGeneratorTypes = sortedKeys(unsupportedSet)
	return out, nil
}

func ParseClusterInventoryFile(repo, path string) (ClusterInventory, error) {
	type rawInventory struct {
		Metadata struct {
			Name   string            `yaml:"name"`
			Labels map[string]string `yaml:"labels"`
		} `yaml:"metadata"`
		Spec struct {
			Server string `yaml:"server"`
		} `yaml:"spec"`
	}

	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return ClusterInventory{}, err
	}

	var raw rawInventory
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return ClusterInventory{}, err
	}

	return ClusterInventory{
		Path:   filepath.ToSlash(path),
		Name:   strings.TrimSpace(raw.Metadata.Name),
		Server: strings.TrimSpace(raw.Spec.Server),
		Labels: raw.Metadata.Labels,
	}, nil
}

func Analyze(repo, appsetPath string, clusterPaths []string) (Analysis, error) {
	doc, err := ParseFile(repo, appsetPath)
	if err != nil {
		return Analysis{}, err
	}

	analysis := Analysis{
		ApplicationSetPath:        doc.Path,
		GeneratorTypes:            append([]string(nil), doc.GeneratorTypes...),
		UnsupportedGeneratorTypes: append([]string(nil), doc.UnsupportedGeneratorTypes...),
		ClusterInventoryPaths:     uniqueSorted(clusterPaths),
	}

	for _, element := range doc.ListElements {
		if element.Name != "" {
			analysis.ListElementNames = append(analysis.ListElementNames, element.Name)
		}
	}
	analysis.ListElementNames = uniqueSorted(analysis.ListElementNames)

	generated := make([]GeneratedApplication, 0, len(doc.ListElements)+len(clusterPaths))
	if len(doc.ListElements) > 0 {
		for _, element := range doc.ListElements {
			name := renderTemplate(doc.TemplateName, element.Values)
			if name == "" {
				continue
			}
			reason := "generated from list element"
			if element.Name != "" {
				reason = fmt.Sprintf("generated from list element %s", element.Name)
			}
			generated = append(generated, GeneratedApplication{
				Name:          name,
				GeneratorType: "list",
				Reason:        reason,
				ListElement:   element.Name,
				SourcePath:    doc.Path,
			})
		}
	}

	clusterGeneratorPresent := stringSliceContains(doc.GeneratorTypes, "clusters")
	if clusterGeneratorPresent {
		if len(clusterPaths) == 0 {
			analysis.Mode = ModeAuthoritativeExpansionUnavailable
			analysis.ModeReason = "ApplicationSet uses the clusters generator, but no pinned cluster inventory inputs were found in the repo."
		} else {
			clusters := loadClusterInventories(repo, clusterPaths)
			for _, cluster := range clusters {
				if cluster.Name == "" {
					continue
				}
				if len(doc.ClusterSelector) > 0 && !labelsMatch(cluster.Labels, doc.ClusterSelector) {
					continue
				}
				analysis.MatchedClusters = append(analysis.MatchedClusters, cluster.Name)
				values := map[string]string{
					"name":   cluster.Name,
					"server": cluster.Server,
				}
				name := renderTemplate(doc.TemplateName, values)
				if name == "" {
					name = cluster.Name
				}
				reason := "generated from pinned cluster inventory"
				if len(doc.ClusterSelector) > 0 {
					reason = fmt.Sprintf("generated from cluster inventory %s matching selector %s", cluster.Name, selectorString(doc.ClusterSelector))
				}
				generated = append(generated, GeneratedApplication{
					Name:          name,
					GeneratorType: "clusters",
					Reason:        reason,
					InventoryPath: cluster.Path,
					Cluster:       cluster.Name,
					SourcePath:    doc.Path,
				})
			}
			analysis.MatchedClusters = uniqueSorted(analysis.MatchedClusters)
			if analysis.Mode == "" {
				if len(analysis.MatchedClusters) == 0 {
					analysis.Mode = ModeAuthoritative
					if len(doc.ClusterSelector) == 0 {
						analysis.ModeReason = "Pinned cluster inventory is present, but no clusters were expanded into child Applications."
					} else {
						analysis.ModeReason = fmt.Sprintf("Pinned cluster inventory is present, but no clusters match selector %s.", selectorString(doc.ClusterSelector))
					}
				}
			}
		}
	}

	generated = dedupeGenerated(generated)
	analysis.GeneratedApplications = generated

	switch {
	case len(doc.UnsupportedGeneratorTypes) > 0:
		analysis.Mode = ModeObservedOnly
		analysis.ModeReason = fmt.Sprintf("ApplicationSet uses unsupported generator types %s, so cub-gen keeps the parent spec in observed-only mode.", strings.Join(doc.UnsupportedGeneratorTypes, ", "))
	case analysis.Mode != "":
		// Keep the earlier explicit degraded mode/reason.
	case len(generated) > 0:
		analysis.Mode = ModeAuthoritative
		analysis.ModeReason = fmt.Sprintf("Authoritative expansion produced %d child Application(s) from explicit repo inputs.", len(generated))
	case len(doc.GeneratorTypes) == 0:
		analysis.Mode = ModeObservedOnly
		analysis.ModeReason = "ApplicationSet generator inputs were not recognized deterministically, so cub-gen keeps the parent spec in observed-only mode."
	default:
		analysis.Mode = ModeObservedOnly
		analysis.ModeReason = "ApplicationSet did not include enough explicit repo inputs for authoritative child expansion."
	}

	return analysis, nil
}

func hasGeneratorKey(generator map[string]any, key string) bool {
	if generator == nil {
		return false
	}
	_, ok := generator[strings.TrimSpace(key)]
	return ok
}

func parseListElements(raw any) []ListElement {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	items, ok := obj["elements"].([]any)
	if !ok {
		return nil
	}
	out := make([]ListElement, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		values := make(map[string]string, len(m))
		name := ""
		for key, value := range m {
			str := scalarString(value)
			if strings.TrimSpace(str) == "" {
				continue
			}
			values[key] = str
			if strings.EqualFold(key, "name") {
				name = str
			}
		}
		out = append(out, ListElement{Name: name, Values: values})
	}
	return out
}

func parseClusterSelector(raw any) map[string]string {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	selector, ok := obj["selector"].(map[string]any)
	if !ok {
		return nil
	}
	matchLabels, ok := selector["matchLabels"].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(matchLabels))
	for key, value := range matchLabels {
		str := scalarString(value)
		if strings.TrimSpace(str) == "" {
			continue
		}
		out[key] = str
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case fmt.Stringer:
		return strings.TrimSpace(x.String())
	case int, int8, int16, int32, int64, float32, float64, bool:
		return fmt.Sprint(x)
	default:
		return ""
	}
}

func loadClusterInventories(repo string, paths []string) []ClusterInventory {
	out := make([]ClusterInventory, 0, len(paths))
	for _, path := range uniqueSorted(paths) {
		cluster, err := ParseClusterInventoryFile(repo, path)
		if err != nil {
			continue
		}
		out = append(out, cluster)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func renderTemplate(template string, values map[string]string) string {
	out := strings.TrimSpace(template)
	if out == "" {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out = strings.ReplaceAll(out, "{{"+key+"}}", values[key])
	}
	if strings.Contains(out, "{{") {
		return ""
	}
	return out
}

func labelsMatch(labels, selector map[string]string) bool {
	if len(selector) == 0 {
		return true
	}
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return true
}

func selectorString(selector map[string]string) string {
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

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(values []string) []string {
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
	sort.Strings(out)
	return out
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func dedupeGenerated(values []GeneratedApplication) []GeneratedApplication {
	seen := map[string]struct{}{}
	out := make([]GeneratedApplication, 0, len(values))
	for _, value := range values {
		key := strings.Join([]string{
			value.Name,
			value.GeneratorType,
			value.InventoryPath,
			value.Cluster,
			value.ListElement,
			value.SourcePath,
		}, "|")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].GeneratorType != out[j].GeneratorType {
			return out[i].GeneratorType < out[j].GeneratorType
		}
		return out[i].Reason < out[j].Reason
	})
	return out
}
