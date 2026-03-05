package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/model"
)

func TestImportRepoExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		repoDir         string
		expectedKind    model.GeneratorKind
		expectedProfile string
	}{
		{name: "helm-paas", repoDir: "helm-paas", expectedKind: model.GeneratorHelm, expectedProfile: "helm-paas"},
		{name: "scoredev-paas", repoDir: "scoredev-paas", expectedKind: model.GeneratorScore, expectedProfile: "scoredev-paas"},
		{name: "springboot-paas", repoDir: "springboot-paas", expectedKind: model.GeneratorSpringBoot, expectedProfile: "springboot-paas"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := filepath.Join("..", "..", "examples", tt.repoDir)
			result, err := ImportRepo(repo, "main", "platform")
			if err != nil {
				t.Fatalf("ImportRepo returned error: %v", err)
			}

			if !strings.HasPrefix(result.ChangeID, "chg_") {
				t.Fatalf("unexpected change ID: %q", result.ChangeID)
			}
			if result.Space != "platform" {
				t.Fatalf("expected space platform, got %q", result.Space)
			}

			if len(result.Detection.Generators) != 1 {
				t.Fatalf("expected 1 generator detection, got %d", len(result.Detection.Generators))
			}
			if result.Detection.Generators[0].Kind != tt.expectedKind {
				t.Fatalf("expected detected kind %q, got %q", tt.expectedKind, result.Detection.Generators[0].Kind)
			}
			if result.Detection.Generators[0].Profile != tt.expectedProfile {
				t.Fatalf("expected detected profile %q, got %q", tt.expectedProfile, result.Detection.Generators[0].Profile)
			}

			if len(result.Units) != 3 {
				t.Fatalf("expected 3 units, got %d", len(result.Units))
			}
			if len(result.Links) != 1 {
				t.Fatalf("expected 1 link, got %d", len(result.Links))
			}
			if len(result.GeneratorContracts) != 1 {
				t.Fatalf("expected 1 generator contract, got %d", len(result.GeneratorContracts))
			}
			if len(result.Provenance) != 1 {
				t.Fatalf("expected 1 provenance record, got %d", len(result.Provenance))
			}
			if len(result.InversePlans) != 1 {
				t.Fatalf("expected 1 inverse plan, got %d", len(result.InversePlans))
			}

			contract := result.GeneratorContracts[0]
			if contract.SchemaVersion == "" || contract.GeneratorID == "" || contract.Kind == "" {
				t.Fatalf("expected populated contract fields; got %+v", contract)
			}
			if contract.Kind != string(tt.expectedKind) {
				t.Fatalf("expected contract kind %q, got %q", tt.expectedKind, contract.Kind)
			}
			if contract.Profile != tt.expectedProfile {
				t.Fatalf("expected contract profile %q, got %q", tt.expectedProfile, contract.Profile)
			}

			prov := result.Provenance[0]
			if prov.InputDigest == "" || len(prov.Sources) == 0 || len(prov.Outputs) == 0 {
				t.Fatalf("expected populated provenance; got %+v", prov)
			}
			if prov.GeneratorProfile != tt.expectedProfile {
				t.Fatalf("expected provenance generator profile %q, got %q", tt.expectedProfile, prov.GeneratorProfile)
			}
			if len(prov.FieldOriginMap) == 0 {
				t.Fatalf("expected field_origin_map entries; got %+v", prov)
			}
			if len(prov.InverseEditPointers) == 0 {
				t.Fatalf("expected inverse_edit_pointers entries; got %+v", prov)
			}

			plan := result.InversePlans[0]
			if plan.Status != "draft" {
				t.Fatalf("expected inverse plan status draft, got %q", plan.Status)
			}
			if len(plan.Patches) == 0 {
				t.Fatalf("expected at least one inverse patch; got %+v", plan)
			}
		})
	}
}

func TestImportRepoDeterministicChangeIdentity(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")

	first, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo first run returned error: %v", err)
	}
	second, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo second run returned error: %v", err)
	}

	if first.ChangeID != second.ChangeID {
		t.Fatalf("expected stable change id, got %q and %q", first.ChangeID, second.ChangeID)
	}
	if len(first.Provenance) != len(second.Provenance) {
		t.Fatalf("expected same provenance count, got %d and %d", len(first.Provenance), len(second.Provenance))
	}
	for i := range first.Provenance {
		if first.Provenance[i].ProvenanceID != second.Provenance[i].ProvenanceID {
			t.Fatalf("expected stable provenance id at index %d, got %q and %q", i, first.Provenance[i].ProvenanceID, second.Provenance[i].ProvenanceID)
		}
	}
	for i := range first.InversePlans {
		if first.InversePlans[i].PlanID != second.InversePlans[i].PlanID {
			t.Fatalf("expected stable inverse plan id at index %d, got %q and %q", i, first.InversePlans[i].PlanID, second.InversePlans[i].PlanID)
		}
	}

	for i, prov := range first.Provenance {
		if prov.RenderedAt != first.ImportedAt {
			t.Fatalf("expected provenance[%d].rendered_at=%q to match imported_at=%q", i, prov.RenderedAt, first.ImportedAt)
		}
	}
	for i, plan := range first.InversePlans {
		if plan.CreatedAt != first.ImportedAt {
			t.Fatalf("expected inverse plan[%d].created_at=%q to match imported_at=%q", i, plan.CreatedAt, first.ImportedAt)
		}
	}
}

func TestImportDetectionScorePointersUseInputPaths(t *testing.T) {
	repo := t.TempDir()
	scorePath := filepath.Join(repo, "score.yaml")
	scoreYAML := `apiVersion: score.dev/v1b1
metadata:
  name: custom-api
containers:
  api:
    image: ghcr.io/example/custom-api:v1
    variables:
      DEBUG: "false"
service:
  ports:
    http:
      port: 9090
`
	if err := os.WriteFile(scorePath, []byte(scoreYAML), 0o644); err != nil {
		t.Fatalf("write score yaml: %v", err)
	}

	detection := model.DetectionResult{
		Repo: repo,
		Ref:  "main",
		Generators: []model.GeneratorDetection{
			{
				ID:      "gen_custom",
				Kind:    model.GeneratorScore,
				Profile: "scoredev-paas",
				Name:    "custom-api",
				Root:    "",
				Inputs:  []string{"score.yaml"},
			},
		},
	}

	result, err := ImportDetection(detection, "platform")
	if err != nil {
		t.Fatalf("ImportDetection returned error: %v", err)
	}
	if len(result.Provenance) != 1 {
		t.Fatalf("expected 1 provenance record, got %d", len(result.Provenance))
	}
	prov := result.Provenance[0]

	if !fieldOriginHasDryPath(prov.FieldOriginMap, "containers.api.image") {
		t.Fatalf("expected field origin for containers.api.image, got %+v", prov.FieldOriginMap)
	}
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "containers.api.variables.DEBUG") {
		t.Fatalf("expected field origin for containers.api.variables.DEBUG, got %+v", prov.FieldOriginMap)
	}
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "service.ports.http.port") {
		t.Fatalf("expected field origin for service.ports.http.port, got %+v", prov.FieldOriginMap)
	}

	if !inversePointerHasDryPath(prov.InverseEditPointers, "containers.api.image") {
		t.Fatalf("expected inverse pointer for containers.api.image, got %+v", prov.InverseEditPointers)
	}
	if !inversePointerHasDryPath(prov.InverseEditPointers, "containers.api.variables.DEBUG") {
		t.Fatalf("expected inverse pointer for containers.api.variables.DEBUG, got %+v", prov.InverseEditPointers)
	}
	if !inversePointerHasDryPath(prov.InverseEditPointers, "service.ports.http.port") {
		t.Fatalf("expected inverse pointer for service.ports.http.port, got %+v", prov.InverseEditPointers)
	}

	if len(result.InversePlans) != 1 {
		t.Fatalf("expected 1 inverse plan, got %d", len(result.InversePlans))
	}
	if len(result.InversePlans[0].Patches) != 1 {
		t.Fatalf("expected 1 inverse patch, got %+v", result.InversePlans[0].Patches)
	}
	patch := result.InversePlans[0].Patches[0]
	if patch.DryPath != "containers.api.variables.DEBUG" {
		t.Fatalf("expected dynamic score dry path, got %q", patch.DryPath)
	}
}

func fieldOriginHasDryPath(v []model.FieldOrigin, dryPath string) bool {
	for _, item := range v {
		if item.DryPath == dryPath {
			return true
		}
	}
	return false
}

func inversePointerHasDryPath(v []model.InverseEditPointer, dryPath string) bool {
	for _, item := range v {
		if item.DryPath == dryPath {
			return true
		}
	}
	return false
}
