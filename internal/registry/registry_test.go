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
}
