package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
	"gopkg.in/yaml.v3"
)

func helmInversePointerOwner(preferredImageTagPath, fallbackOwner string) string {
	if helmIsSubchartPath(preferredImageTagPath) {
		return "platform-engineer"
	}
	if strings.TrimSpace(fallbackOwner) == "" {
		return "platform-engineer"
	}
	return fallbackOwner
}

type helmProvenancePaths struct {
	ChartPath         string
	ValuesPaths       []string
	PrimaryValuesPath string
	OverlayValuesPath string
}

type helmValueSource struct {
	Path     string
	Layer    string
	External bool
}

type helmDependencyDoc struct {
	Name       string `yaml:"name"`
	Alias      string `yaml:"alias"`
	Condition  string `yaml:"condition"`
	Repository string `yaml:"repository"`
	Version    string `yaml:"version"`
	ChartPath  string
	ValuesPath string
}

func helmProvenancePathsForGenerator(repoPath string, g model.GeneratorDetection) helmProvenancePaths {
	if g.Kind != model.GeneratorHelm {
		return helmProvenancePaths{}
	}

	chartRole := strings.TrimSpace(registry.HintDefault(g.Kind, "chart_role", "chart"))
	if chartRole == "" {
		chartRole = "chart"
	}
	valuesRole := strings.TrimSpace(registry.HintDefault(g.Kind, "values_role", "values"))
	if valuesRole == "" {
		valuesRole = "values"
	}
	primaryValuesBase := strings.TrimSpace(registry.HintDefault(g.Kind, "primary_values_path", "values.yaml"))
	if primaryValuesBase == "" {
		primaryValuesBase = "values.yaml"
	}

	chartPath := firstInputPathForRole(g.Kind, g.Inputs, chartRole)
	valuesPaths := helmValuesPathsForInputs(g.Kind, g.Inputs, valuesRole)

	primaryValuesPath := selectPrimaryHelmValuesPath(valuesPaths, primaryValuesBase)
	imageTagSources := helmImageTagSources(repoPath, helmProvenancePaths{
		ChartPath:         chartPath,
		ValuesPaths:       valuesPaths,
		PrimaryValuesPath: primaryValuesPath,
	})
	overlayValuesPath := ""
	for _, source := range imageTagSources {
		if source.Path != primaryValuesPath {
			overlayValuesPath = source.Path
			break
		}
	}

	return helmProvenancePaths{
		ChartPath:         chartPath,
		ValuesPaths:       valuesPaths,
		PrimaryValuesPath: primaryValuesPath,
		OverlayValuesPath: overlayValuesPath,
	}
}

func selectPrimaryHelmValuesPath(valuesPaths []string, primaryValuesBase string) string {
	primaryValuesPath := primaryValuesBase
	if selected := selectPreferredPathByBase(valuesPaths, primaryValuesBase); selected != "" {
		return selected
	}
	if len(valuesPaths) > 0 {
		return valuesPaths[0]
	}
	return primaryValuesPath
}

func helmValuesPathsForInputs(kind model.GeneratorKind, inputs []string, valuesRole string) []string {
	out := make([]string, 0, len(inputs))
	for _, in := range inputs {
		role := registry.InputRole(kind, in)
		if role == valuesRole || role == "subchart-values" {
			out = append(out, in)
		}
	}
	sort.Strings(out)
	return out
}

func firstInputPathForRole(kind model.GeneratorKind, inputs []string, role string) string {
	for _, in := range inputs {
		if registry.InputRole(kind, in) == role {
			return in
		}
	}
	return ""
}

func inputPathsForRole(kind model.GeneratorKind, inputs []string, role string) []string {
	out := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if registry.InputRole(kind, in) == role {
			out = append(out, in)
		}
	}
	sort.Strings(out)
	return out
}

func selectPreferredPathByBase(paths []string, preferredBase string) string {
	for _, p := range paths {
		if strings.EqualFold(filepath.Base(p), preferredBase) {
			return p
		}
	}
	return ""
}

func helmImageTagSources(repoPath string, hints helmProvenancePaths) []helmValueSource {
	imageTagPaths := make([]string, 0, len(hints.ValuesPaths))
	for _, path := range hints.ValuesPaths {
		if helmValuesPathDefinesImageTag(repoPath, path) {
			imageTagPaths = append(imageTagPaths, path)
		}
	}
	if len(imageTagPaths) == 0 {
		return nil
	}

	sort.SliceStable(imageTagPaths, func(i, j int) bool {
		leftRank := helmValuesPathPrecedence(imageTagPaths[i], hints.PrimaryValuesPath, hints.ChartPath)
		rightRank := helmValuesPathPrecedence(imageTagPaths[j], hints.PrimaryValuesPath, hints.ChartPath)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return imageTagPaths[i] < imageTagPaths[j]
	})

	umbrella := helmHasDependencyCharts(repoPath, hints.ChartPath)
	ordered := make([]helmValueSource, 0, len(imageTagPaths))
	for _, path := range imageTagPaths {
		ordered = append(ordered, helmValueSource{
			Path:     path,
			Layer:    helmValuesSourceLayer(path, umbrella),
			External: helmValuesPathImageTagIsExternal(repoPath, path),
		})
	}
	return ordered
}

func helmValuesPathPrecedence(path, primaryValuesPath, chartPath string) int {
	pathDir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(path)))
	if pathDir == "." {
		pathDir = ""
	}
	switch {
	case path == primaryValuesPath:
		return 0
	case helmIsSubchartPath(path):
		return 2
	case helmChartDir(chartPath) == pathDir:
		return 1
	default:
		return 3
	}
}

func helmValuesSourceLayer(path string, umbrella bool) string {
	if helmIsSubchartPath(path) {
		parts := strings.Split(filepath.ToSlash(path), "/")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	if umbrella {
		return "umbrella"
	}
	return ""
}

func helmIsSubchartPath(path string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	return normalized == "charts" || strings.HasPrefix(normalized, "charts/")
}

func helmChartDir(chartPath string) string {
	dir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(chartPath)))
	if dir == "." {
		return ""
	}
	return dir
}

func helmHasDependencyCharts(repoPath, chartPath string) bool {
	return len(parseHelmDependencyDocs(repoPath, chartPath)) > 0
}

func helmImageTagBuiltinOrigin(repoPath string, hints helmProvenancePaths) (model.FieldOrigin, bool) {
	if strings.TrimSpace(repoPath) == "" {
		return model.FieldOrigin{}, false
	}
	if fileRef := helmTemplatesUseFilesGetForImageTag(repoPath); fileRef != "" {
		return model.FieldOrigin{
			DryPath:    "values.image.tag",
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			SourcePath: fmt.Sprintf("<helm-builtin>:.Files.Get(%s)", fileRef),
			Transform:  helmBuiltinTransform,
			Confidence: 1.0,
		}, true
	}
	if !helmTemplatesUseChartAppVersionFallback(repoPath) {
		return model.FieldOrigin{}, false
	}
	if appVersion := helmChartAppVersion(repoPath, hints.ChartPath); strings.TrimSpace(appVersion) != "" {
		return model.FieldOrigin{
			DryPath:    "values.image.tag",
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			SourcePath: "<helm-builtin>:.Chart.AppVersion",
			Transform:  helmBuiltinTransform,
			Confidence: 1.0,
		}, true
	}
	return model.FieldOrigin{}, false
}

func helmImageTagNonDeterministicOrigin(repoPath string) (model.FieldOrigin, bool) {
	fn := helmTemplatesUseNonDeterministicImageTagSource(repoPath)
	if fn == "" {
		return model.FieldOrigin{}, false
	}
	return model.FieldOrigin{
		DryPath:    "values.image.tag",
		WetPath:    "Deployment/spec/template/spec/containers[0]/image",
		SourcePath: "<helm-nondeterministic>:" + fn,
		Transform:  helmNonDeterministicTransform,
		Confidence: 1.0,
	}, true
}

func helmImageTagDefaultOrigin() model.FieldOrigin {
	return model.FieldOrigin{
		DryPath:    "values.image.tag",
		WetPath:    "Deployment/spec/template/spec/containers[0]/image",
		SourcePath: "<helm-default>:values.image.tag",
		Transform:  helmDefaultTransform,
		Confidence: 0.40,
	}
}

func helmTemplatesUseChartAppVersionFallback(repoPath string) bool {
	templatesDir := filepath.Join(repoPath, "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return false
	}

	patterns := []string{
		".Values.image.tag | default .Chart.AppVersion",
		"default .Chart.AppVersion .Values.image.tag",
	}
	found := false
	_ = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(content)
		for _, pattern := range patterns {
			if strings.Contains(text, pattern) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

func helmImageTagUsesHelperChain(repoPath string) bool {
	helpers := helmHelperDefinitions(repoPath)
	if len(helpers) == 0 {
		return false
	}
	for _, ref := range helmReferencedHelpers(repoPath) {
		if helmHelperUsesImageTag(helpers, ref, map[string]struct{}{}) {
			return true
		}
	}
	return false
}

func helmHelperDefinitions(repoPath string) map[string]string {
	templatesDir := filepath.Join(repoPath, "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	out := map[string]string{}
	_ = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".tpl" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, match := range helmHelperDefineRe.FindAllStringSubmatch(string(content), -1) {
			if len(match) < 3 {
				continue
			}
			out[match[1]] = match[2]
		}
		return nil
	})
	return out
}

func helmReferencedHelpers(repoPath string) []string {
	templatesDir := filepath.Join(repoPath, "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	refs := map[string]struct{}{}
	_ = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, match := range helmHelperRefRe.FindAllStringSubmatch(string(content), -1) {
			if len(match) < 2 {
				continue
			}
			refs[match[1]] = struct{}{}
		}
		return nil
	})
	out := make([]string, 0, len(refs))
	for ref := range refs {
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}

func helmHelperUsesImageTag(helpers map[string]string, helper string, seen map[string]struct{}) bool {
	if _, ok := seen[helper]; ok {
		return false
	}
	body, ok := helpers[helper]
	if !ok {
		return false
	}
	seen[helper] = struct{}{}
	if strings.Contains(body, ".Values.image.tag") {
		return true
	}
	for _, match := range helmHelperRefRe.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		if helmHelperUsesImageTag(helpers, match[1], seen) {
			return true
		}
	}
	return false
}

func helmTemplatesUseFilesGetForImageTag(repoPath string) string {
	return helmImageTagTemplateFunctionArg(repoPath, ".Files.Get")
}

func helmTemplatesUseNonDeterministicImageTagSource(repoPath string) string {
	for _, fn := range []string{"lookup", "now", "randAlphaNum", "uuidv4"} {
		if helmImageTagTemplateMentions(repoPath, fn) {
			return fn
		}
	}
	return ""
}

func helmImageTagTemplateFunctionArg(repoPath, fn string) string {
	templatesDir := filepath.Join(repoPath, "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return ""
	}

	pattern := regexp.MustCompile(regexp.QuoteMeta(fn) + `\s+"([^"]+)"`)
	ref := ""
	_ = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(content)
		if !strings.Contains(text, "image") || !strings.Contains(text, fn) {
			return nil
		}
		match := pattern.FindStringSubmatch(text)
		if len(match) < 2 {
			return nil
		}
		ref = match[1]
		return filepath.SkipAll
	})
	return ref
}

func helmImageTagTemplateMentions(repoPath, token string) bool {
	templatesDir := filepath.Join(repoPath, "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return false
	}
	found := false
	_ = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(content)
		if strings.Contains(text, "image") && strings.Contains(text, token) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func helmChartAppVersion(repoPath, chartPath string) string {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(chartPath) == "" {
		return ""
	}
	content, err := os.ReadFile(filepath.Join(repoPath, chartPath))
	if err != nil {
		return ""
	}

	var chartDoc struct {
		AppVersion string `yaml:"appVersion"`
	}
	if err := yaml.Unmarshal(content, &chartDoc); err != nil {
		return ""
	}
	return strings.TrimSpace(chartDoc.AppVersion)
}

func helmValuesPathDefinesImageTag(repoPath, relPath string) bool {
	_, ok := helmValuesPathImageTagValue(repoPath, relPath)
	return ok
}

func helmValuesPathImageTagIsExternal(repoPath, relPath string) bool {
	value, ok := helmValuesPathImageTagValue(repoPath, relPath)
	if !ok {
		return false
	}
	return helmValueLooksExternal(value)
}

func helmValuesPathImageTagValue(repoPath, relPath string) (string, bool) {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(relPath) == "" {
		return "", false
	}
	content, err := os.ReadFile(filepath.Join(repoPath, relPath))
	if err != nil {
		return "", false
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return "", false
	}
	node := yamlPathNode(&doc, []string{"image", "tag"})
	if node == nil {
		return "", false
	}
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node == nil {
		return "", false
	}
	if node.Kind == yaml.ScalarNode {
		return strings.TrimSpace(node.Value), true
	}
	return "", true
}

func yamlPathNode(node *yaml.Node, path []string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return yamlPathNode(node.Content[0], path)
	}
	if len(path) == 0 {
		return node
	}
	if node.Kind != yaml.MappingNode {
		if node.Kind == yaml.AliasNode {
			return yamlPathNode(node.Alias, path)
		}
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != path[0] {
			continue
		}
		return yamlPathNode(node.Content[i+1], path[1:])
	}
	return nil
}

func helmValueLooksExternal(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	switch {
	case strings.HasPrefix(value, "vault://"),
		strings.HasPrefix(value, "ref+vault://"),
		strings.HasPrefix(value, "secretref+"),
		strings.HasPrefix(value, "aws-secretsmanager://"),
		strings.HasPrefix(value, "ssm://"),
		strings.HasPrefix(value, "azurekeyvault://"),
		strings.HasPrefix(value, "gcpsecret://"),
		strings.HasPrefix(value, "external://"):
		return true
	}
	return false
}

func helmCLIOverridesForGenerator(g model.GeneratorDetection, overrides []model.HelmCLIOverride) []model.HelmCLIOverride {
	if g.Kind != model.GeneratorHelm || len(overrides) == 0 {
		return nil
	}
	out := make([]model.HelmCLIOverride, len(overrides))
	copy(out, overrides)
	return out
}

func helmCLIOverrideSourcesForGenerator(ref string, g model.GeneratorDetection, overrides []model.HelmCLIOverride) []model.SourceRef {
	if g.Kind != model.GeneratorHelm || len(overrides) == 0 {
		return nil
	}
	sources := make([]model.SourceRef, 0, len(overrides))
	for _, override := range overrides {
		sources = append(sources, model.SourceRef{
			Role:     "cli-override",
			URI:      fmt.Sprintf("helm-cli://%s/%s", override.Flag, override.Key),
			Revision: ref,
			Path:     HelmCLIOverrideDisplay(override),
		})
	}
	return sources
}

func helmCLIOverrideFieldOrigins(g model.GeneratorDetection, overrides []model.HelmCLIOverride) []model.FieldOrigin {
	if g.Kind != model.GeneratorHelm {
		return nil
	}

	override, ok := lastHelmCLIOverrideForKey(overrides, "image.tag")
	if !ok {
		return nil
	}

	return []model.FieldOrigin{{
		DryPath:    "values.image.tag",
		WetPath:    "Deployment/spec/template/spec/containers[0]/image",
		SourcePath: HelmCLIOverrideDisplay(override),
		Transform:  helmCLIOverrideTransform,
		Confidence: 1.0,
	}}
}

func lastHelmCLIOverrideForKey(overrides []model.HelmCLIOverride, key string) (model.HelmCLIOverride, bool) {
	for idx := len(overrides) - 1; idx >= 0; idx-- {
		if overrides[idx].Key == key {
			return overrides[idx], true
		}
	}
	return model.HelmCLIOverride{}, false
}
