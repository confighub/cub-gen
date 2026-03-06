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
		model.GeneratorHelm,
		model.GeneratorOpsFlow,
		model.GeneratorScore,
		model.GeneratorSpringBoot,
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
