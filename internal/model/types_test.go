package model

import (
	"encoding/json"
	"testing"
)

func TestProvenanceRecordJSONIncludesRecentSchemaFields(t *testing.T) {
	t.Parallel()

	record := ProvenanceRecord{
		SchemaVersion:    "cub.confighub.io/provenance/v1",
		ProvenanceID:     "prov_123",
		ChangeID:         "chg_123",
		GeneratorID:      "gen_123",
		GeneratorName:    "payments-api",
		GeneratorProfile: "helm-paas",
		Version:          "0.3.0",
		InputDigest:      "sha256:abc",
		HelmCLIOverrides: []HelmCLIOverride{{
			Flag:  "--set",
			Key:   "image.tag",
			Value: "v1.2.3",
		}},
		FieldOriginMap: []FieldOrigin{{
			DryPath:     "values.image.tag",
			WetPath:     "Deployment/spec/template/spec/containers[0]/image",
			SourcePath:  "values-prod.yaml",
			SourceLayer: "overlay",
			Transform:   "helm-template",
			Confidence:  0.97,
			Hops: []FieldOriginHop{{
				GeneratorKind:    "helm",
				GeneratorProfile: "helm-paas",
				DryPath:          "values.image.tag",
				SourcePath:       "values-prod.yaml",
				Transform:        "helm-template",
				Confidence:       0.97,
			}},
		}},
		InverseEditPointers: []InverseEditPointer{{
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			DryPath:    "values.image.tag",
			Owner:      "app-team",
			EditHint:   "Edit values-prod.yaml",
			Confidence: 0.97,
		}},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal provenance record: %v", err)
	}

	var got ProvenanceRecord
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal provenance record: %v", err)
	}

	if len(got.HelmCLIOverrides) != 1 || got.HelmCLIOverrides[0].Key != "image.tag" {
		t.Fatalf("expected helm CLI override to round-trip, got %+v", got.HelmCLIOverrides)
	}
	if len(got.FieldOriginMap) != 1 {
		t.Fatalf("expected one field origin, got %+v", got.FieldOriginMap)
	}
	origin := got.FieldOriginMap[0]
	if origin.SourceLayer != "overlay" {
		t.Fatalf("expected source_layer overlay, got %+v", origin)
	}
	if len(origin.Hops) != 1 || origin.Hops[0].GeneratorKind != "helm" {
		t.Fatalf("expected field origin hops to round-trip, got %+v", origin.Hops)
	}
}

func TestDetectionResultJSONIncludesChainSummaries(t *testing.T) {
	t.Parallel()

	result := DetectionResult{
		Repo: "examples/incubator/score-helm-chain",
		Ref:  "main",
		ChainSummaries: []GeneratorChainSummary{{
			ID:           "chain_123",
			Name:         "score-to-helm",
			Display:      "score -> helm",
			StageCount:   2,
			Stages:       []string{"score", "helm"},
			MappingCount: 1,
		}},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal detection result: %v", err)
	}

	var got DetectionResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal detection result: %v", err)
	}

	if len(got.ChainSummaries) != 1 {
		t.Fatalf("expected one chain summary, got %+v", got.ChainSummaries)
	}
	if got.ChainSummaries[0].Display != "score -> helm" {
		t.Fatalf("expected chain summary display to round-trip, got %+v", got.ChainSummaries[0])
	}
}
