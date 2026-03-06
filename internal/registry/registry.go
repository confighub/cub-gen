package registry

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
)

type InputRoleRule struct {
	Role           string
	ExactBasenames []string
	Prefixes       []string
	Extensions     []string
}

type WetTargetTemplate struct {
	Kind                  string
	NameTemplate          string
	Owner                 string
	Namespace             string
	SourceDryPathTemplate string
}

// FamilySpec captures cross-cutting generator-family metadata used by multiple
// command/runtime layers.
type FamilySpec struct {
	Kind             model.GeneratorKind
	Profile          string
	ResourceKind     string
	ResourceType     string
	Capabilities     []string
	RoleSchemaRefs   map[string]string
	InputRoleRules   []InputRoleRule
	DefaultInputRole string
	RoleOwners       map[string]string
	DefaultOwner     string
	WetTargets       []WetTargetTemplate
}

var familySpecs = map[model.GeneratorKind]FamilySpec{
	model.GeneratorHelm: {
		Kind:         model.GeneratorHelm,
		Profile:      "helm-paas",
		ResourceKind: "HelmRelease",
		ResourceType: "helm.toolkit.fluxcd.io/v2/HelmRelease",
		Capabilities: []string{"render-manifests", "values-overrides", "inverse-values-patch"},
		RoleSchemaRefs: map[string]string{
			"chart": "https://json.schemastore.org/chart",
		},
		InputRoleRules: []InputRoleRule{
			{Role: "chart", ExactBasenames: []string{"chart.yaml"}},
			{Role: "values", Prefixes: []string{"values"}, Extensions: []string{".yaml", ".yml"}},
		},
		DefaultInputRole: "helm-input",
		RoleOwners:       map[string]string{"values": "app-team"},
		DefaultOwner:     "platform-engineer",
		WetTargets: []WetTargetTemplate{
			{Kind: "HelmRelease", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps"},
			{Kind: "Deployment", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "values.image.tag"},
			{Kind: "Service", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "values.service.port"},
		},
	},
	model.GeneratorScore: {
		Kind:             model.GeneratorScore,
		Profile:          "scoredev-paas",
		ResourceKind:     "Application",
		ResourceType:     "argoproj.io/v1alpha1/Application",
		Capabilities:     []string{"render-manifests", "workload-spec", "inverse-score-patch"},
		RoleSchemaRefs:   map[string]string{"score-spec": "https://docs.score.dev/schemas/score-v1b1.json"},
		InputRoleRules:   []InputRoleRule{{Role: "score-spec", ExactBasenames: []string{"score.yaml", "score.yml"}}},
		DefaultInputRole: "score-input",
		DefaultOwner:     "app-team",
		WetTargets: []WetTargetTemplate{
			{Kind: "Application", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps"},
			{Kind: "Deployment", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "containers.{{container}}.image"},
			{Kind: "Service", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "service.ports.{{service_port}}.port"},
		},
	},
	model.GeneratorSpringBoot: {
		Kind:         model.GeneratorSpringBoot,
		Profile:      "springboot-paas",
		ResourceKind: "Kustomization",
		ResourceType: "kustomize.toolkit.fluxcd.io/v1/Kustomization",
		Capabilities: []string{"render-app-config", "profile-overrides", "inverse-app-config-patch"},
		RoleSchemaRefs: map[string]string{
			"app-config-base":    "https://json.schemastore.org/spring-configuration-metadata",
			"app-config-profile": "https://json.schemastore.org/spring-configuration-metadata",
		},
		InputRoleRules: []InputRoleRule{
			{Role: "build-config", ExactBasenames: []string{"pom.xml", "build.gradle", "build.gradle.kts"}},
			{Role: "app-config-base", ExactBasenames: []string{"application.yaml", "application.yml"}},
			{Role: "app-config-profile", Prefixes: []string{"application-"}, Extensions: []string{".yaml", ".yml"}},
		},
		DefaultInputRole: "spring-input",
		RoleOwners: map[string]string{
			"app-config-base":    "app-team",
			"app-config-profile": "app-team",
		},
		DefaultOwner: "platform-engineer",
		WetTargets: []WetTargetTemplate{
			{Kind: "Kustomization", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps"},
			{Kind: "Deployment", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "server.port"},
			{Kind: "ConfigMap", NameTemplate: "{{name}}-config", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "spring.datasource.url"},
		},
	},
	model.GeneratorBackstage: {
		Kind:         model.GeneratorBackstage,
		Profile:      "backstage-idp",
		ResourceKind: "Component",
		ResourceType: "backstage.io/v1alpha1/Component",
		Capabilities: []string{"catalog-metadata", "render-manifests", "inverse-catalog-patch"},
		RoleSchemaRefs: map[string]string{
			"catalog-spec": "https://json.schemastore.org/backstage-catalog-info",
			"app-config":   "https://json.schemastore.org/backstage-app-config",
		},
		InputRoleRules: []InputRoleRule{
			{Role: "catalog-spec", ExactBasenames: []string{"catalog-info.yaml", "catalog-info.yml"}},
			{Role: "app-config", ExactBasenames: []string{"app-config.yaml", "app-config.yml"}},
		},
		DefaultInputRole: "backstage-input",
		RoleOwners:       map[string]string{"app-config": "app-team"},
		DefaultOwner:     "platform-engineer",
		WetTargets: []WetTargetTemplate{
			{Kind: "Application", NameTemplate: "{{name}}", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "metadata.name"},
			{Kind: "ConfigMap", NameTemplate: "{{name}}-catalog", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "spec.lifecycle"},
		},
	},
	model.GeneratorAbly: {
		Kind:         model.GeneratorAbly,
		Profile:      "ably-config",
		ResourceKind: "ConfigMap",
		ResourceType: "v1/ConfigMap",
		Capabilities: []string{"app-config-only", "provider-config", "inverse-provider-config-patch"},
		RoleSchemaRefs: map[string]string{
			"provider-config-base":    "https://schema.confighub.dev/generators/ably-config-v1",
			"provider-config-overlay": "https://schema.confighub.dev/generators/ably-config-v1",
		},
		InputRoleRules: []InputRoleRule{
			{Role: "provider-config-base", ExactBasenames: []string{"ably.yaml", "ably.yml", "ably.json"}},
			{Role: "provider-config-overlay", Prefixes: []string{"ably-"}, Extensions: []string{".yaml", ".yml", ".json"}},
		},
		DefaultInputRole: "provider-config",
		DefaultOwner:     "app-team",
		WetTargets: []WetTargetTemplate{
			{Kind: "ConfigMap", NameTemplate: "{{name}}-ably", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "app.environment"},
			{Kind: "Secret", NameTemplate: "{{name}}-ably-credentials", Owner: "platform-runtime", Namespace: "apps", SourceDryPathTemplate: "credentials.api_key_ref"},
		},
	},
	model.GeneratorOpsFlow: {
		Kind:         model.GeneratorOpsFlow,
		Profile:      "ops-workflow",
		ResourceKind: "Workflow",
		ResourceType: "argoproj.io/v1alpha1/Workflow",
		Capabilities: []string{"workflow-plan", "governed-execution-intent", "inverse-workflow-patch"},
		RoleSchemaRefs: map[string]string{
			"operations-base":    "https://schema.confighub.dev/generators/ops-workflow-v1",
			"operations-overlay": "https://schema.confighub.dev/generators/ops-workflow-v1",
		},
		InputRoleRules: []InputRoleRule{
			{Role: "operations-base", ExactBasenames: []string{"operations.yaml", "operations.yml", "workflow.yaml", "workflow.yml"}},
			{Role: "operations-overlay", Prefixes: []string{"operations-", "workflow-"}, Extensions: []string{".yaml", ".yml"}},
		},
		DefaultInputRole: "operations-input",
		DefaultOwner:     "platform-engineer",
		WetTargets: []WetTargetTemplate{
			{Kind: "Workflow", NameTemplate: "{{name}}-workflow", Owner: "platform-runtime", Namespace: "ops", SourceDryPathTemplate: "actions.deploy.image_tag"},
			{Kind: "Job", NameTemplate: "{{name}}-dry-run", Owner: "platform-runtime", Namespace: "ops", SourceDryPathTemplate: "triggers.schedule"},
		},
	},
}

func Spec(kind model.GeneratorKind) (FamilySpec, bool) {
	spec, ok := familySpecs[kind]
	if !ok {
		return FamilySpec{}, false
	}
	return FamilySpec{
		Kind:             spec.Kind,
		Profile:          spec.Profile,
		ResourceKind:     spec.ResourceKind,
		ResourceType:     spec.ResourceType,
		Capabilities:     append([]string(nil), spec.Capabilities...),
		RoleSchemaRefs:   copyRoleSchemaRefs(spec.RoleSchemaRefs),
		InputRoleRules:   copyInputRoleRules(spec.InputRoleRules),
		DefaultInputRole: spec.DefaultInputRole,
		RoleOwners:       copyRoleOwners(spec.RoleOwners),
		DefaultOwner:     spec.DefaultOwner,
		WetTargets:       copyWetTargets(spec.WetTargets),
	}, true
}

func Profile(kind model.GeneratorKind) string {
	spec, ok := Spec(kind)
	if !ok {
		return "generator"
	}
	return spec.Profile
}

func ResourceKind(kind model.GeneratorKind) string {
	spec, ok := Spec(kind)
	if !ok {
		return "Resource"
	}
	return spec.ResourceKind
}

func ResourceType(kind model.GeneratorKind) string {
	spec, ok := Spec(kind)
	if !ok {
		return "v1/Resource"
	}
	return spec.ResourceType
}

func Capabilities(kind model.GeneratorKind) []string {
	spec, ok := Spec(kind)
	if !ok {
		return []string{"render-manifests"}
	}
	return append([]string(nil), spec.Capabilities...)
}

func SchemaRef(kind model.GeneratorKind, inputPath string) string {
	base := strings.ToLower(filepath.Base(inputPath))
	ext := strings.ToLower(filepath.Ext(base))

	switch ext {
	case ".xml":
		return "https://maven.apache.org/xsd/maven-4.0.0.xsd"
	default:
		spec, ok := Spec(kind)
		if ok {
			role := InputRole(kind, inputPath)
			if schemaRef, found := spec.RoleSchemaRefs[role]; found && strings.TrimSpace(schemaRef) != "" {
				return schemaRef
			}
		}
	}

	return "https://json-schema.org/draft/2020-12/schema"
}

func InputRole(kind model.GeneratorKind, inputPath string) string {
	spec, ok := Spec(kind)
	if !ok {
		return "input"
	}
	base := strings.ToLower(filepath.Base(inputPath))
	ext := strings.ToLower(filepath.Ext(base))
	for _, rule := range spec.InputRoleRules {
		if matchesInputRule(rule, base, ext) {
			return rule.Role
		}
	}
	if spec.DefaultInputRole != "" {
		return spec.DefaultInputRole
	}
	return "input"
}

func OwnerForRole(kind model.GeneratorKind, role string) string {
	spec, ok := Spec(kind)
	if !ok {
		return "platform-engineer"
	}
	if owner, ok := spec.RoleOwners[role]; ok {
		return owner
	}
	if spec.DefaultOwner != "" {
		return spec.DefaultOwner
	}
	return "platform-engineer"
}

func Kinds() []model.GeneratorKind {
	out := make([]model.GeneratorKind, 0, len(familySpecs))
	for kind := range familySpecs {
		out = append(out, kind)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

func SupportedResourceKinds() []string {
	kinds := make([]string, 0, len(familySpecs))
	seen := map[string]struct{}{}
	for _, kind := range Kinds() {
		mapped := ResourceKind(kind)
		if _, ok := seen[mapped]; ok {
			continue
		}
		seen[mapped] = struct{}{}
		kinds = append(kinds, mapped)
	}
	sort.Strings(kinds)
	return kinds
}

func WetTargetTemplates(kind model.GeneratorKind) []WetTargetTemplate {
	spec, ok := Spec(kind)
	if !ok {
		return nil
	}
	return copyWetTargets(spec.WetTargets)
}

func matchesInputRule(rule InputRoleRule, base, ext string) bool {
	for _, exact := range rule.ExactBasenames {
		if base == strings.ToLower(exact) {
			return true
		}
	}

	for _, prefix := range rule.Prefixes {
		if !strings.HasPrefix(base, strings.ToLower(prefix)) {
			continue
		}
		if len(rule.Extensions) == 0 {
			return true
		}
		for _, allowed := range rule.Extensions {
			if ext == strings.ToLower(allowed) {
				return true
			}
		}
	}
	return false
}

func copyInputRoleRules(in []InputRoleRule) []InputRoleRule {
	out := make([]InputRoleRule, 0, len(in))
	for _, rule := range in {
		out = append(out, InputRoleRule{
			Role:           rule.Role,
			ExactBasenames: append([]string(nil), rule.ExactBasenames...),
			Prefixes:       append([]string(nil), rule.Prefixes...),
			Extensions:     append([]string(nil), rule.Extensions...),
		})
	}
	return out
}

func copyRoleOwners(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyRoleSchemaRefs(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyWetTargets(in []WetTargetTemplate) []WetTargetTemplate {
	out := make([]WetTargetTemplate, 0, len(in))
	for _, target := range in {
		out = append(out, WetTargetTemplate{
			Kind:                  target.Kind,
			NameTemplate:          target.NameTemplate,
			Owner:                 target.Owner,
			Namespace:             target.Namespace,
			SourceDryPathTemplate: target.SourceDryPathTemplate,
		})
	}
	return out
}
