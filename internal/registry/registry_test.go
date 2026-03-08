package registry

import (
	"reflect"
	"testing"

	"github.com/confighub/cub-gen/internal/model"
)

func TestRegistryHasSpecForAllKinds(t *testing.T) {
	expected := []model.GeneratorKind{
		model.GeneratorAbly,
		model.GeneratorBackstage,
		model.GeneratorC3Agent,
		model.GeneratorHelm,
		model.GeneratorOpsFlow,
		model.GeneratorScore,
		model.GeneratorSpringBoot,
		model.GeneratorSwamp,
	}

	if got := Kinds(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected kinds %+v, got %+v", expected, got)
	}

	for _, kind := range expected {
		spec, ok := Spec(kind)
		if !ok {
			t.Fatalf("expected spec for kind %q", kind)
		}
		if spec.Profile == "" {
			t.Fatalf("expected non-empty profile for kind %q", kind)
		}
		if spec.ResourceKind == "" || spec.ResourceType == "" {
			t.Fatalf("expected resource mapping for kind %q, got %+v", kind, spec)
		}
		if len(spec.Capabilities) == 0 {
			t.Fatalf("expected capabilities for kind %q", kind)
		}
	}

	expectedResourceKinds := []string{
		"Application",
		"Component",
		"ConfigMap",
		"HelmRelease",
		"Kustomization",
		"Workflow",
	}
	if got := SupportedResourceKinds(); !reflect.DeepEqual(got, expectedResourceKinds) {
		t.Fatalf("expected supported resource kinds %+v, got %+v", expectedResourceKinds, got)
	}
}

func TestRegistryFallbacks(t *testing.T) {
	unknown := model.GeneratorKind("unknown")

	if got := Profile(unknown); got != "generator" {
		t.Fatalf("expected profile fallback generator, got %q", got)
	}
	if got := ResourceKind(unknown); got != "Resource" {
		t.Fatalf("expected resource kind fallback Resource, got %q", got)
	}
	if got := ResourceType(unknown); got != "v1/Resource" {
		t.Fatalf("expected resource type fallback v1/Resource, got %q", got)
	}
	if got := Capabilities(unknown); !reflect.DeepEqual(got, []string{"render-manifests"}) {
		t.Fatalf("expected capabilities fallback render-manifests, got %+v", got)
	}
	if got := HintDefault(unknown, "source_path", "default.yaml"); got != "default.yaml" {
		t.Fatalf("expected hint default fallback default.yaml, got %q", got)
	}
	if got := FieldOriginTransform(unknown); got != "generator-transform" {
		t.Fatalf("expected field origin transform fallback generator-transform, got %q", got)
	}
	if got := FieldOriginOverlayTransform(unknown); got != "generator-transform" {
		t.Fatalf("expected field origin overlay transform fallback generator-transform, got %q", got)
	}
	if got := FieldOriginConfidenceFor(unknown, "image_tag", 0.42); got != 0.42 {
		t.Fatalf("expected field origin confidence fallback 0.42, got %v", got)
	}
	if got := InversePatchReason(unknown, "image_tag", "fallback-reason"); got != "fallback-reason" {
		t.Fatalf("expected inverse patch reason fallback fallback-reason, got %q", got)
	}
	if got := InversePatchTemplateFor(unknown, "image_tag", InversePatchTemplate{
		EditableBy: "fallback-owner", Confidence: 1.0, RequiresReview: true,
	}); got.EditableBy != "fallback-owner" || got.Confidence != 1.0 || !got.RequiresReview {
		t.Fatalf("expected inverse patch template fallback to be returned, got %+v", got)
	}
	if got := InversePointerTemplateFor(unknown, "image_tag", InversePointerTemplate{
		Owner: "fallback-owner", Confidence: 0.77,
	}); got.Owner != "fallback-owner" || got.Confidence != 0.77 {
		t.Fatalf("expected inverse pointer template fallback to be returned, got %+v", got)
	}
	if got := InverseEditHint(unknown, "image_tag", "fallback-hint"); got != "fallback-hint" {
		t.Fatalf("expected inverse edit hint fallback fallback-hint, got %q", got)
	}
	if got := RenderedLineageTemplates(unknown); got != nil {
		t.Fatalf("expected rendered lineage templates fallback nil, got %+v", got)
	}
	if got := InputRole(unknown, "any.yaml"); got != "input" {
		t.Fatalf("expected input role fallback input, got %q", got)
	}
	if got := OwnerForRole(unknown, "any"); got != "platform-engineer" {
		t.Fatalf("expected owner fallback platform-engineer, got %q", got)
	}
}

func TestRegistryInputRoleAndOwnerClassification(t *testing.T) {
	tests := []struct {
		name          string
		kind          model.GeneratorKind
		path          string
		expectedRole  string
		expectedOwner string
	}{
		{name: "helm-chart", kind: model.GeneratorHelm, path: "Chart.yaml", expectedRole: "chart", expectedOwner: "platform-engineer"},
		{name: "helm-values", kind: model.GeneratorHelm, path: "values-prod.yaml", expectedRole: "values", expectedOwner: "app-team"},
		{name: "score-spec", kind: model.GeneratorScore, path: "score.yaml", expectedRole: "score-spec", expectedOwner: "app-team"},
		{name: "spring-build", kind: model.GeneratorSpringBoot, path: "pom.xml", expectedRole: "build-config", expectedOwner: "platform-engineer"},
		{name: "spring-profile", kind: model.GeneratorSpringBoot, path: "application-prod.yml", expectedRole: "app-config-profile", expectedOwner: "app-team"},
		{name: "backstage-catalog", kind: model.GeneratorBackstage, path: "catalog-info.yaml", expectedRole: "catalog-spec", expectedOwner: "platform-engineer"},
		{name: "backstage-app-config", kind: model.GeneratorBackstage, path: "app-config.yaml", expectedRole: "app-config", expectedOwner: "app-team"},
		{name: "ably-base", kind: model.GeneratorAbly, path: "ably.yaml", expectedRole: "provider-config-base", expectedOwner: "app-team"},
		{name: "ably-overlay", kind: model.GeneratorAbly, path: "ably-prod.json", expectedRole: "provider-config-overlay", expectedOwner: "app-team"},
		{name: "ops-base", kind: model.GeneratorOpsFlow, path: "operations.yaml", expectedRole: "operations-base", expectedOwner: "platform-engineer"},
		{name: "ops-overlay", kind: model.GeneratorOpsFlow, path: "workflow-prod.yml", expectedRole: "operations-overlay", expectedOwner: "platform-engineer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := InputRole(tt.kind, tt.path)
			if role != tt.expectedRole {
				t.Fatalf("expected role %q, got %q", tt.expectedRole, role)
			}
			owner := OwnerForRole(tt.kind, role)
			if owner != tt.expectedOwner {
				t.Fatalf("expected owner %q, got %q", tt.expectedOwner, owner)
			}
		})
	}
}

func TestRegistrySchemaRef(t *testing.T) {
	tests := []struct {
		name     string
		kind     model.GeneratorKind
		path     string
		expected string
	}{
		{name: "helm-chart", kind: model.GeneratorHelm, path: "Chart.yaml", expected: "https://json.schemastore.org/chart"},
		{name: "score", kind: model.GeneratorScore, path: "score.yaml", expected: "https://docs.score.dev/schemas/score-v1b1.json"},
		{name: "spring-app", kind: model.GeneratorSpringBoot, path: "application.yaml", expected: "https://json.schemastore.org/spring-configuration-metadata"},
		{name: "backstage-catalog", kind: model.GeneratorBackstage, path: "catalog-info.yaml", expected: "https://json.schemastore.org/backstage-catalog-info"},
		{name: "backstage-app-config", kind: model.GeneratorBackstage, path: "app-config.yaml", expected: "https://json.schemastore.org/backstage-app-config"},
		{name: "ably-yaml", kind: model.GeneratorAbly, path: "ably.yaml", expected: "https://schema.confighub.dev/generators/ably-config-v1"},
		{name: "ably-json", kind: model.GeneratorAbly, path: "ably-prod.json", expected: "https://schema.confighub.dev/generators/ably-config-v1"},
		{name: "ops", kind: model.GeneratorOpsFlow, path: "operations.yaml", expected: "https://schema.confighub.dev/generators/ops-workflow-v1"},
		{name: "xml-maven", kind: model.GeneratorSpringBoot, path: "pom.xml", expected: "https://maven.apache.org/xsd/maven-4.0.0.xsd"},
		{name: "default", kind: model.GeneratorSpringBoot, path: "README.md", expected: "https://json-schema.org/draft/2020-12/schema"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SchemaRef(tt.kind, tt.path); got != tt.expected {
				t.Fatalf("expected schema %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRegistryWetTargetTemplates(t *testing.T) {
	score := WetTargetTemplates(model.GeneratorScore)
	if len(score) != 3 {
		t.Fatalf("expected 3 score wet target templates, got %d", len(score))
	}
	if score[0].Kind != "Application" || score[0].NameTemplate != "{{name}}" {
		t.Fatalf("unexpected score template[0]: %+v", score[0])
	}
	if score[1].SourceDryPathTemplate != "containers.{{container}}.image" {
		t.Fatalf("unexpected score deployment source path template: %q", score[1].SourceDryPathTemplate)
	}
	if score[2].SourceDryPathTemplate != "service.ports.{{service_port}}.port" {
		t.Fatalf("unexpected score service source path template: %q", score[2].SourceDryPathTemplate)
	}

	// Ensure returned templates are copies, not direct registry storage.
	score[0].Kind = "Mutated"
	again := WetTargetTemplates(model.GeneratorScore)
	if again[0].Kind != "Application" {
		t.Fatalf("expected immutable template copy, got %+v", again[0])
	}

	springLineage := RenderedLineageTemplates(model.GeneratorSpringBoot)
	if len(springLineage) != 4 {
		t.Fatalf("expected 4 spring rendered lineage templates, got %d", len(springLineage))
	}
	if springLineage[3].SourcePathHint != "profile_config_path" || springLineage[3].SourcePathHintFallback != "base_config_path" {
		t.Fatalf("unexpected spring profile lineage template: %+v", springLineage[3])
	}
	springLineage[0].Kind = "Mutated"
	againLineage := RenderedLineageTemplates(model.GeneratorSpringBoot)
	if againLineage[0].Kind != "Kustomization" {
		t.Fatalf("expected immutable rendered lineage template copy, got %+v", againLineage[0])
	}
}

func TestRegistryHintDefaults(t *testing.T) {
	if got := HintDefault(model.GeneratorHelm, "chart_role", "fallback-role"); got != "chart" {
		t.Fatalf("expected helm chart_role hint default chart, got %q", got)
	}
	if got := HintDefault(model.GeneratorHelm, "values_role", "fallback-role"); got != "values" {
		t.Fatalf("expected helm values_role hint default values, got %q", got)
	}
	if got := HintDefault(model.GeneratorHelm, "primary_values_path", "fallback.yaml"); got != "values.yaml" {
		t.Fatalf("expected helm primary_values_path hint default values.yaml, got %q", got)
	}
	if got := HintDefault(model.GeneratorScore, "source_path", "fallback.yaml"); got != "score.yaml" {
		t.Fatalf("expected score source_path hint default score.yaml, got %q", got)
	}
	if got := HintDefault(model.GeneratorSpringBoot, "base_config_path", "fallback.yaml"); got != "src/main/resources/application.yaml" {
		t.Fatalf("expected spring base_config_path hint default, got %q", got)
	}
	if got := FieldOriginTransform(model.GeneratorBackstage); got != "backstage-component-to-application" {
		t.Fatalf("expected backstage field origin transform, got %q", got)
	}
	if got := FieldOriginOverlayTransform(model.GeneratorSpringBoot); got != "spring-profile-overlay" {
		t.Fatalf("expected spring overlay field origin transform, got %q", got)
	}
	if got := FieldOriginOverlayTransform(model.GeneratorHelm); got != "helm-template" {
		t.Fatalf("expected helm overlay transform to fall back to base transform, got %q", got)
	}
	if got := InversePatchReason(model.GeneratorBackstage, "identity", "fallback"); got != "Backstage component identity is sourced from {{catalog_path}}." {
		t.Fatalf("expected backstage inverse patch reason template, got %q", got)
	}
	if got := InversePatchReason(model.GeneratorAbly, "channels", "fallback"); got != "Channel mapping is app-level runtime behavior." {
		t.Fatalf("expected ably channels inverse patch reason, got %q", got)
	}
	if got := FieldOriginConfidenceFor(model.GeneratorOpsFlow, "schedule_overlay", 0.0); got != 0.80 {
		t.Fatalf("expected ops schedule overlay field origin confidence 0.80, got %v", got)
	}
	if got := InversePatchTemplateFor(model.GeneratorOpsFlow, "schedule", InversePatchTemplate{}); got.EditableBy != "platform-engineer" || got.Confidence != 0.84 || !got.RequiresReview {
		t.Fatalf("expected ops schedule inverse patch template, got %+v", got)
	}
	if got := InversePointerTemplateFor(model.GeneratorBackstage, "name", InversePointerTemplate{}); got.Owner != "platform-engineer" || got.Confidence != 0.90 {
		t.Fatalf("expected backstage name inverse pointer template, got %+v", got)
	}
	if got := InverseEditHint(model.GeneratorScore, "env_var", "fallback"); got != "Edit {{variable_name}} under containers.{{container_name}}.variables in {{source_path}}." {
		t.Fatalf("expected score env_var inverse edit hint template, got %q", got)
	}
	if got := InverseEditHint(model.GeneratorSpringBoot, "server_port_overlay", "fallback"); got != "Edit server.port in {{profile_config_path}} for environment overrides; use {{base_config_path}} for the default." {
		t.Fatalf("expected spring server_port_overlay inverse edit hint template, got %q", got)
	}

	// Ensure returned specs are copies.
	spec, ok := Spec(model.GeneratorScore)
	if !ok {
		t.Fatalf("expected score spec")
	}
	spec.HintDefaults["source_path"] = "mutated.yaml"
	if got := HintDefault(model.GeneratorScore, "source_path", "fallback.yaml"); got != "score.yaml" {
		t.Fatalf("expected immutable hint defaults, got %q", got)
	}
	spec.InversePatchReasons["env_var"] = "mutated reason"
	if got := InversePatchReason(model.GeneratorScore, "env_var", "fallback"); got != "Score variable maps to a single Kubernetes env var." {
		t.Fatalf("expected immutable inverse patch reasons, got %q", got)
	}
	spec.InversePatchTemplates["env_var"] = InversePatchTemplate{EditableBy: "mutated", Confidence: 0.01, RequiresReview: true}
	if got := InversePatchTemplateFor(model.GeneratorScore, "env_var", InversePatchTemplate{}); got.EditableBy != "app-team" || got.Confidence != 0.90 || got.RequiresReview {
		t.Fatalf("expected immutable inverse patch templates, got %+v", got)
	}
	spec.InversePointerTemplates["env_var"] = InversePointerTemplate{Owner: "mutated", Confidence: 0.01}
	if got := InversePointerTemplateFor(model.GeneratorScore, "env_var", InversePointerTemplate{}); got.Owner != "app-team" || got.Confidence != 0.90 {
		t.Fatalf("expected immutable inverse pointer templates, got %+v", got)
	}
	spec.FieldOriginConfidences["env_var"] = 0.01
	if got := FieldOriginConfidenceFor(model.GeneratorScore, "env_var", 0.0); got != 0.90 {
		t.Fatalf("expected immutable field origin confidences, got %v", got)
	}
	spec.InverseEditHints["env_var"] = "mutated hint"
	if got := InverseEditHint(model.GeneratorScore, "env_var", "fallback"); got != "Edit {{variable_name}} under containers.{{container_name}}.variables in {{source_path}}." {
		t.Fatalf("expected immutable inverse edit hints, got %q", got)
	}
}
