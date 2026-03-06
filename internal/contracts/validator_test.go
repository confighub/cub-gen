package contracts

import (
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/model"
)

func TestValidateTripleSetValid(t *testing.T) {
	contract, provenance, inversePlan := validTripleFixture()
	if err := ValidateTriple(contract, provenance, inversePlan); err != nil {
		t.Fatalf("expected valid triple, got error: %v", err)
	}

	if err := ValidateTripleSet(
		[]model.GeneratorContract{contract},
		[]model.ProvenanceRecord{provenance},
		[]model.InverseTransformPlan{inversePlan},
	); err != nil {
		t.Fatalf("expected valid triple set, got error: %v", err)
	}
}

func TestValidateTripleSetCardinalityMismatch(t *testing.T) {
	contract, provenance, _ := validTripleFixture()
	err := ValidateTripleSet(
		[]model.GeneratorContract{contract},
		[]model.ProvenanceRecord{provenance},
		nil,
	)
	if err == nil {
		t.Fatal("expected cardinality mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "contract triple cardinality mismatch") {
		t.Fatalf("expected cardinality mismatch message, got: %v", err)
	}
}

func TestValidateGovernedImportTriplesMissing(t *testing.T) {
	err := ValidateGovernedImportTriples(1, nil, nil, nil)
	if err == nil {
		t.Fatal("expected governed import missing triple error, got nil")
	}

	expected := "governed import blocked: required contract triple missing (detected=1 contracts=0 provenance=0 inverse_plans=0)"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestValidateGovernedImportTriplesCardinalityMismatch(t *testing.T) {
	contract, provenance, inversePlan := validTripleFixture()
	err := ValidateGovernedImportTriples(
		2,
		[]model.GeneratorContract{contract},
		[]model.ProvenanceRecord{provenance},
		[]model.InverseTransformPlan{inversePlan},
	)
	if err == nil {
		t.Fatal("expected governed import cardinality mismatch error, got nil")
	}

	expected := "governed import blocked: contract triple cardinality mismatch (detected=2 contracts=1 provenance=1 inverse_plans=1)"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestValidateGovernedImportTriplesZeroDetectedAllowed(t *testing.T) {
	if err := ValidateGovernedImportTriples(0, nil, nil, nil); err != nil {
		t.Fatalf("expected nil error for zero-detected import, got: %v", err)
	}
}

func TestValidateGeneratorContractInvalid(t *testing.T) {
	contract, _, _ := validTripleFixture()
	contract.SchemaVersion = "invalid/schema"

	err := ValidateGeneratorContract(contract)
	if err == nil {
		t.Fatal("expected generator contract validation error, got nil")
	}
	if !strings.Contains(err.Error(), "generator_contract schema validation failed") {
		t.Fatalf("expected generator contract validation message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "/schema_version") {
		t.Fatalf("expected deterministic schema path in error, got: %v", err)
	}
}

func TestValidateProvenanceRecordInvalid(t *testing.T) {
	_, provenance, _ := validTripleFixture()
	provenance.FieldOriginMap[0].Confidence = 1.3

	err := ValidateProvenanceRecord(provenance)
	if err == nil {
		t.Fatal("expected provenance validation error, got nil")
	}
	if !strings.Contains(err.Error(), "provenance_record schema validation failed") {
		t.Fatalf("expected provenance validation message, got: %v", err)
	}
}

func TestValidateInverseTransformPlanInvalid(t *testing.T) {
	_, _, inversePlan := validTripleFixture()
	inversePlan.Patches[0].Confidence = -0.1

	err := ValidateInverseTransformPlan(inversePlan)
	if err == nil {
		t.Fatal("expected inverse plan validation error, got nil")
	}
	if !strings.Contains(err.Error(), "inverse_transform_plan schema validation failed") {
		t.Fatalf("expected inverse plan validation message, got: %v", err)
	}
}

func validTripleFixture() (model.GeneratorContract, model.ProvenanceRecord, model.InverseTransformPlan) {
	contract := model.GeneratorContract{
		SchemaVersion: "cub.confighub.io/generator-contract/v1",
		GeneratorID:   "gen_123",
		Name:          "payments-api",
		Kind:          "helm",
		Profile:       "helm-paas",
		Version:       "0.2.0",
		SourceRepo:    "/tmp/repo",
		SourceRef:     "HEAD",
		SourcePath:    "",
		Inputs: []model.GeneratorInput{
			{Name: "input_01", SchemaRef: "https://json.schemastore.org/chart", Required: true},
		},
		OutputFormat:  "kubernetes/yaml",
		Transport:     "oci+git",
		Capabilities:  []string{"render-manifests"},
		Deterministic: true,
	}

	provenance := model.ProvenanceRecord{
		SchemaVersion:    "cub.confighub.io/provenance/v1",
		ProvenanceID:     "prov_123",
		ChangeID:         "chg_123",
		GeneratorID:      "gen_123",
		GeneratorName:    "payments-api",
		GeneratorProfile: "helm-paas",
		Version:          "0.2.0",
		InputDigest:      "sha256:abc",
		Sources: []model.SourceRef{
			{Role: "generator-input", URI: "git+file:///tmp/repo#HEAD:Chart.yaml", Revision: "HEAD", Path: "Chart.yaml"},
		},
		Outputs: []model.OutputRef{
			{Role: "rendered-manifests", URI: "oci://example.local/platform/payments-api:latest", Digest: "sha256:def"},
		},
		FieldOriginMap: []model.FieldOrigin{
			{
				DryPath:    "values.image.tag",
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				SourcePath: "values.yaml",
				Transform:  "helm-template",
				Confidence: 0.86,
			},
		},
		InverseEditPointers: []model.InverseEditPointer{
			{
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				DryPath:    "values.image.tag",
				Owner:      "app-team",
				EditHint:   "Edit chart values file and keep chart template unchanged.",
				Confidence: 0.86,
			},
		},
		RenderedAt: "2026-03-06T00:00:00Z",
	}

	inversePlan := model.InverseTransformPlan{
		SchemaVersion: "cub.confighub.io/inverse-transform-plan/v1",
		PlanID:        "inv_123",
		ChangeID:      "chg_123",
		SourceKind:    "helm",
		SourceRef:     "",
		TargetUnitID:  "dry_123",
		Status:        "draft",
		Patches: []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "values.image.tag",
				WetPath:        "Deployment/spec/template/spec/containers[0]/image",
				EditableBy:     "app-team",
				Confidence:     0.86,
				RequiresReview: false,
				Reason:         "Container image tag maps cleanly to helm values.",
			},
		},
		CreatedAt: "2026-03-06T00:00:00Z",
	}

	return contract, provenance, inversePlan
}
