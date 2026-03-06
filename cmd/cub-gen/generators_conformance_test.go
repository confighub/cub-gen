package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/registry"
)

type generatorsJSONEnvelope struct {
	Count    int                     `json:"count"`
	Families []generatorFamilyRecord `json:"families"`
}

func TestGeneratorMetadataConformanceRegistryAndList(t *testing.T) {
	kinds := registry.Kinds()
	base := listGeneratorFamilies("", "", "", false)
	details := listGeneratorFamilies("", "", "", true)

	if len(base) != len(kinds) {
		t.Fatalf("expected %d generator families, got %d", len(kinds), len(base))
	}
	if len(details) != len(kinds) {
		t.Fatalf("expected %d detailed generator families, got %d", len(kinds), len(details))
	}

	for i, kind := range kinds {
		spec, ok := registry.Spec(kind)
		if !ok {
			t.Fatalf("expected registry spec for kind %q", kind)
		}
		if strings.TrimSpace(spec.Profile) == "" {
			t.Fatalf("kind %q has empty profile", kind)
		}
		if strings.TrimSpace(spec.ResourceKind) == "" || strings.TrimSpace(spec.ResourceType) == "" {
			t.Fatalf("kind %q has incomplete resource mapping: kind=%q type=%q", kind, spec.ResourceKind, spec.ResourceType)
		}
		if len(spec.Capabilities) == 0 {
			t.Fatalf("kind %q has empty capability set", kind)
		}
		if len(spec.InversePatchTemplates) == 0 || len(spec.InversePointerTemplates) == 0 || len(spec.FieldOriginConfidences) == 0 {
			t.Fatalf("kind %q has incomplete policy/provenance metadata templates", kind)
		}

		baseRecord := base[i]
		if baseRecord.Kind != string(spec.Kind) {
			t.Fatalf("base record[%d] kind mismatch: expected %q, got %q", i, spec.Kind, baseRecord.Kind)
		}
		if baseRecord.Profile != spec.Profile || baseRecord.ResourceKind != spec.ResourceKind || baseRecord.ResourceType != spec.ResourceType {
			t.Fatalf("base record[%d] metadata mismatch: %+v vs spec %+v", i, baseRecord, spec)
		}
		if !reflect.DeepEqual(baseRecord.Capabilities, spec.Capabilities) {
			t.Fatalf("base record[%d] capabilities mismatch: expected %+v, got %+v", i, spec.Capabilities, baseRecord.Capabilities)
		}
		if baseRecord.Policies != nil {
			t.Fatalf("base record[%d] unexpectedly includes policies", i)
		}

		detailRecord := details[i]
		if detailRecord.Kind != string(spec.Kind) {
			t.Fatalf("detail record[%d] kind mismatch: expected %q, got %q", i, spec.Kind, detailRecord.Kind)
		}
		if detailRecord.Policies == nil {
			t.Fatalf("detail record[%d] missing policies", i)
		}
		expectedPolicies := generatorPolicyRecord(spec)
		if !reflect.DeepEqual(detailRecord.Policies, expectedPolicies) {
			t.Fatalf("detail record[%d] policies mismatch: expected %+v, got %+v", i, expectedPolicies, detailRecord.Policies)
		}
	}
}

func TestGeneratorMetadataConformanceCLIJSONSurfaces(t *testing.T) {
	expectedBase := listGeneratorFamilies("", "", "", false)
	baseEnvelope := runGeneratorsJSON(t, []string{"generators", "--json"})
	if baseEnvelope.Count != len(expectedBase) {
		t.Fatalf("base json count mismatch: expected %d, got %d", len(expectedBase), baseEnvelope.Count)
	}
	if !reflect.DeepEqual(baseEnvelope.Families, expectedBase) {
		t.Fatalf("base json families mismatch\nexpected=%+v\ngot=%+v", expectedBase, baseEnvelope.Families)
	}

	expectedDetails := listGeneratorFamilies("", "", "", true)
	detailsEnvelope := runGeneratorsJSON(t, []string{"generators", "--json", "--details"})
	if detailsEnvelope.Count != len(expectedDetails) {
		t.Fatalf("details json count mismatch: expected %d, got %d", len(expectedDetails), detailsEnvelope.Count)
	}
	if !reflect.DeepEqual(detailsEnvelope.Families, expectedDetails) {
		t.Fatalf("details json families mismatch\nexpected=%+v\ngot=%+v", expectedDetails, detailsEnvelope.Families)
	}
}

func runGeneratorsJSON(t *testing.T, args []string) generatorsJSONEnvelope {
	t.Helper()

	stdout, stderr, err := runWithCapturedIO(args)
	if err != nil {
		t.Fatalf("run %v returned error: %v\nstderr=%s", args, err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("run %v expected empty stderr, got %q", args, stderr)
	}

	var envelope generatorsJSONEnvelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("unmarshal output for %v: %v\noutput=%s", args, err, stdout)
	}
	return envelope
}
