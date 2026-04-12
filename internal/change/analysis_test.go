package change

import (
	"testing"

	"github.com/confighub/cub-gen/internal/model"
)

func TestPickInverseSuggestionIncludesGeneratorChainHops(t *testing.T) {
	t.Parallel()

	provenance := []model.ProvenanceRecord{{
		GeneratorName:    "checkout-api",
		GeneratorProfile: "helm-paas",
		FieldOriginMap: []model.FieldOrigin{{
			DryPath:    "containers.api.image",
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			SourcePath: "score.yaml",
			Transform:  "score-to-helm",
			Confidence: 0.80,
			Hops: []model.FieldOriginHop{
				{
					GeneratorKind:    "score",
					GeneratorProfile: "scoredev-paas",
					DryPath:          "containers.api.image",
					SourcePath:       "score.yaml",
					Transform:        "score-to-helm",
					Confidence:       0.94,
				},
				{
					GeneratorKind:    "helm",
					GeneratorProfile: "helm-paas",
					DryPath:          "values.image.tag",
					SourcePath:       "chart/values.yaml",
					Transform:        "helm-values",
					Confidence:       0.86,
				},
			},
		}},
		InverseEditPointers: []model.InverseEditPointer{{
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			DryPath:    "containers.api.image",
			Owner:      "app-team",
			EditHint:   "Edit containers.api.image in score.yaml.",
			Confidence: 0.80,
		}},
	}}

	suggestion, matchCount, ok := PickInverseSuggestion(provenance, "Deployment/spec/template/spec/containers[0]/image", "", "")
	if !ok {
		t.Fatal("expected inverse suggestion")
	}
	if matchCount != 1 {
		t.Fatalf("expected 1 match, got %d", matchCount)
	}
	if len(suggestion.Hops) != 2 {
		t.Fatalf("expected 2 provenance hops, got %+v", suggestion.Hops)
	}
	if suggestion.Hops[0].GeneratorKind != "score" || suggestion.Hops[1].GeneratorKind != "helm" {
		t.Fatalf("unexpected hop order: %+v", suggestion.Hops)
	}
}

func TestBestFieldOriginRequiresExactMatchWhenBothFiltersProvided(t *testing.T) {
	t.Parallel()

	origins := []model.FieldOrigin{
		{
			DryPath:    "values.image.tag",
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			SourcePath: "values.yaml",
			Transform:  "helm-template",
			Confidence: 0.90,
		},
		{
			DryPath:    "values.service.port",
			WetPath:    "Service/spec/ports[name=web]/port",
			SourcePath: "values.yaml",
			Transform:  "helm-template",
			Confidence: 0.95,
		},
	}

	if _, ok := BestFieldOrigin(origins, "Deployment/spec/template/spec/containers[0]/image", "values.service.port"); ok {
		t.Fatal("expected no exact match when wet_path and dry_path point to different origins")
	}
}
