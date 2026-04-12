package change

import (
	"testing"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
)

func TestQueryContextFromBundleBuildsExplainAndImpactResults(t *testing.T) {
	t.Parallel()

	bundle := publish.ChangeBundle{
		ChangeID:         "chg_123",
		BundleDigest:     "sha256:test",
		TargetSlug:       "helm-paas",
		TargetPath:       "/repo/helm-paas",
		RenderTargetSlug: "helm-paas",
		RenderTargetPath: "/repo/helm-paas",
		Space:            "platform",
		Ref:              "HEAD",
		Provenance: []model.ProvenanceRecord{{
			GeneratorName:    "checkout-api",
			GeneratorProfile: "helm-paas",
			FieldOriginMap: []model.FieldOrigin{{
				DryPath:    "values.image.tag",
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				SourcePath: "values.yaml",
				Transform:  "helm-template",
				Confidence: 0.91,
			}},
			InverseEditPointers: []model.InverseEditPointer{{
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				DryPath:    "values.image.tag",
				Owner:      "app-team",
				EditHint:   "Edit values.image.tag in values.yaml.",
				Confidence: 0.91,
			}},
		}},
	}

	ctx := QueryContextFromBundle(bundle)
	explain, err := BuildExplainResult(ctx, "Deployment/spec/template/spec/containers[0]/image", "", "")
	if err != nil {
		t.Fatalf("BuildExplainResult returned error: %v", err)
	}
	if explain.Change.ChangeID != "chg_123" {
		t.Fatalf("expected change_id chg_123, got %+v", explain.Change)
	}
	if explain.Explanation.DryPath != "values.image.tag" {
		t.Fatalf("expected dry path values.image.tag, got %+v", explain.Explanation)
	}

	impact, err := BuildImpactResult(ctx, "values.image.tag", "", "")
	if err != nil {
		t.Fatalf("BuildImpactResult returned error: %v", err)
	}
	if len(impact.Impacts) != 1 {
		t.Fatalf("expected one impact, got %+v", impact.Impacts)
	}
	if impact.Impacts[0].WetPath != "Deployment/spec/template/spec/containers[0]/image" {
		t.Fatalf("unexpected impact wet path: %+v", impact.Impacts[0])
	}
}
