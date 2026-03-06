package registry

import (
	"sort"

	"github.com/confighub/cub-gen/internal/model"
)

// FamilySpec captures cross-cutting generator-family metadata used by multiple
// command/runtime layers.
type FamilySpec struct {
	Kind         model.GeneratorKind
	Profile      string
	ResourceKind string
	ResourceType string
	Capabilities []string
}

var familySpecs = map[model.GeneratorKind]FamilySpec{
	model.GeneratorHelm: {
		Kind:         model.GeneratorHelm,
		Profile:      "helm-paas",
		ResourceKind: "HelmRelease",
		ResourceType: "helm.toolkit.fluxcd.io/v2/HelmRelease",
		Capabilities: []string{"render-manifests", "values-overrides", "inverse-values-patch"},
	},
	model.GeneratorScore: {
		Kind:         model.GeneratorScore,
		Profile:      "scoredev-paas",
		ResourceKind: "Application",
		ResourceType: "argoproj.io/v1alpha1/Application",
		Capabilities: []string{"render-manifests", "workload-spec", "inverse-score-patch"},
	},
	model.GeneratorSpringBoot: {
		Kind:         model.GeneratorSpringBoot,
		Profile:      "springboot-paas",
		ResourceKind: "Kustomization",
		ResourceType: "kustomize.toolkit.fluxcd.io/v1/Kustomization",
		Capabilities: []string{"render-app-config", "profile-overrides", "inverse-app-config-patch"},
	},
	model.GeneratorBackstage: {
		Kind:         model.GeneratorBackstage,
		Profile:      "backstage-idp",
		ResourceKind: "Component",
		ResourceType: "backstage.io/v1alpha1/Component",
		Capabilities: []string{"catalog-metadata", "render-manifests", "inverse-catalog-patch"},
	},
	model.GeneratorAbly: {
		Kind:         model.GeneratorAbly,
		Profile:      "ably-config",
		ResourceKind: "ConfigMap",
		ResourceType: "v1/ConfigMap",
		Capabilities: []string{"app-config-only", "provider-config", "inverse-provider-config-patch"},
	},
	model.GeneratorOpsFlow: {
		Kind:         model.GeneratorOpsFlow,
		Profile:      "ops-workflow",
		ResourceKind: "Workflow",
		ResourceType: "argoproj.io/v1alpha1/Workflow",
		Capabilities: []string{"workflow-plan", "governed-execution-intent", "inverse-workflow-patch"},
	},
}

func Spec(kind model.GeneratorKind) (FamilySpec, bool) {
	spec, ok := familySpecs[kind]
	if !ok {
		return FamilySpec{}, false
	}
	return FamilySpec{
		Kind:         spec.Kind,
		Profile:      spec.Profile,
		ResourceKind: spec.ResourceKind,
		ResourceType: spec.ResourceType,
		Capabilities: append([]string(nil), spec.Capabilities...),
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
