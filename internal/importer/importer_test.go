package importer

import (
	"os"
	"path/filepath"
	"reflect"
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
		{name: "backstage-idp", repoDir: "backstage-idp", expectedKind: model.GeneratorBackstage, expectedProfile: "backstage-idp"},
		{name: "ably-config", repoDir: "ably-config", expectedKind: model.GeneratorAbly, expectedProfile: "ably-config"},
		{name: "ops-workflow", repoDir: "ops-workflow", expectedKind: model.GeneratorOpsFlow, expectedProfile: "ops-workflow"},
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
			if len(result.DryInputs) == 0 {
				t.Fatalf("expected dry input refs, got %+v", result.DryInputs)
			}
			if len(result.WetManifestTargets) == 0 {
				t.Fatalf("expected wet manifest targets, got %+v", result.WetManifestTargets)
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
			if len(prov.RenderedLineage) == 0 {
				t.Fatalf("expected rendered object lineage entries; got %+v", prov)
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

func TestImportRepoGeneratorCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		repoDir              string
		expectedCapabilities []string
	}{
		{
			name:                 "helm-paas",
			repoDir:              "helm-paas",
			expectedCapabilities: []string{"render-manifests", "values-overrides", "inverse-values-patch"},
		},
		{
			name:                 "scoredev-paas",
			repoDir:              "scoredev-paas",
			expectedCapabilities: []string{"render-manifests", "workload-spec", "inverse-score-patch"},
		},
		{
			name:                 "springboot-paas",
			repoDir:              "springboot-paas",
			expectedCapabilities: []string{"render-app-config", "profile-overrides", "inverse-app-config-patch"},
		},
		{
			name:                 "backstage-idp",
			repoDir:              "backstage-idp",
			expectedCapabilities: []string{"catalog-metadata", "render-manifests", "inverse-catalog-patch"},
		},
		{
			name:                 "ably-config",
			repoDir:              "ably-config",
			expectedCapabilities: []string{"app-config-only", "provider-config", "inverse-provider-config-patch"},
		},
		{
			name:                 "ops-workflow",
			repoDir:              "ops-workflow",
			expectedCapabilities: []string{"workflow-plan", "governed-execution-intent", "inverse-workflow-patch"},
		},
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
			if len(result.GeneratorContracts) != 1 {
				t.Fatalf("expected 1 generator contract, got %d", len(result.GeneratorContracts))
			}

			got := result.GeneratorContracts[0].Capabilities
			if !reflect.DeepEqual(got, tt.expectedCapabilities) {
				t.Fatalf("expected capabilities %+v, got %+v", tt.expectedCapabilities, got)
			}
		})
	}
}

func TestImportRepoHelmDryWetContract(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")
	result, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo returned error: %v", err)
	}

	if len(result.Provenance) != 1 {
		t.Fatalf("expected single provenance record, got %d", len(result.Provenance))
	}
	prov := result.Provenance[0]
	if prov.ChartPath != "Chart.yaml" {
		t.Fatalf("expected chart_path Chart.yaml, got %q", prov.ChartPath)
	}
	if !containsString(prov.ValuesPaths, "values.yaml") || !containsString(prov.ValuesPaths, "values-prod.yaml") {
		t.Fatalf("expected values paths to include values.yaml and values-prod.yaml, got %+v", prov.ValuesPaths)
	}
	if !renderedLineageHasKind(prov.RenderedLineage, "Deployment") || !renderedLineageHasKind(prov.RenderedLineage, "Service") {
		t.Fatalf("expected rendered lineage to include Deployment and Service, got %+v", prov.RenderedLineage)
	}
	if !renderedLineageHasSourcePath(prov.RenderedLineage, "values.yaml") || !renderedLineageHasSourcePath(prov.RenderedLineage, "values-prod.yaml") {
		t.Fatalf("expected rendered lineage to include both Helm values source paths, got %+v", prov.RenderedLineage)
	}

	if !dryInputHasRolePath(result.DryInputs, "chart", "Chart.yaml") {
		t.Fatalf("expected chart dry input, got %+v", result.DryInputs)
	}
	if !dryInputHasRolePath(result.DryInputs, "values", "values.yaml") {
		t.Fatalf("expected values.yaml dry input, got %+v", result.DryInputs)
	}
	if !dryInputHasRolePath(result.DryInputs, "values", "values-prod.yaml") {
		t.Fatalf("expected values-prod.yaml dry input, got %+v", result.DryInputs)
	}
	if !dryInputHasRoleOwnerPath(result.DryInputs, "chart", "platform-engineer", "Chart.yaml") {
		t.Fatalf("expected chart owner to be platform-engineer, got %+v", result.DryInputs)
	}
	if !dryInputHasRoleOwnerPath(result.DryInputs, "values", "app-team", "values.yaml") {
		t.Fatalf("expected values.yaml owner to be app-team, got %+v", result.DryInputs)
	}

	if !wetTargetHasKind(result.WetManifestTargets, "HelmRelease") {
		t.Fatalf("expected HelmRelease wet target, got %+v", result.WetManifestTargets)
	}
	if !wetTargetHasKind(result.WetManifestTargets, "Deployment") {
		t.Fatalf("expected Deployment wet target, got %+v", result.WetManifestTargets)
	}
	if !wetTargetHasKindOwner(result.WetManifestTargets, "HelmRelease", "platform-runtime") {
		t.Fatalf("expected HelmRelease owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
}

func TestImportRepoSpringBootDryWetContract(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "springboot-paas")
	result, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo returned error: %v", err)
	}

	if len(result.Provenance) != 1 {
		t.Fatalf("expected single provenance record, got %d", len(result.Provenance))
	}
	prov := result.Provenance[0]
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "spring.application.name") {
		t.Fatalf("expected spring.application.name field origin, got %+v", prov.FieldOriginMap)
	}
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "server.port") {
		t.Fatalf("expected server.port field origin, got %+v", prov.FieldOriginMap)
	}
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "spring.datasource.url") {
		t.Fatalf("expected spring.datasource.url field origin, got %+v", prov.FieldOriginMap)
	}
	if !inversePointerHasDryPath(prov.InverseEditPointers, "spring.application.name") {
		t.Fatalf("expected spring.application.name inverse pointer, got %+v", prov.InverseEditPointers)
	}
	if !inversePointerHasDryPath(prov.InverseEditPointers, "server.port") {
		t.Fatalf("expected server.port inverse pointer, got %+v", prov.InverseEditPointers)
	}

	if !dryInputHasRoleOwnerPath(result.DryInputs, "app-config-base", "app-team", "src/main/resources/application.yaml") {
		t.Fatalf("expected base app config owned by app-team, got %+v", result.DryInputs)
	}
	if !dryInputHasRoleOwnerPath(result.DryInputs, "app-config-profile", "app-team", "src/main/resources/application-prod.yaml") {
		t.Fatalf("expected profile app config owned by app-team, got %+v", result.DryInputs)
	}
	if !dryInputHasRoleOwnerPath(result.DryInputs, "build-config", "platform-engineer", "pom.xml") {
		t.Fatalf("expected build config owned by platform-engineer, got %+v", result.DryInputs)
	}

	if !wetTargetHasKindOwner(result.WetManifestTargets, "Kustomization", "platform-runtime") {
		t.Fatalf("expected Kustomization owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
	if !wetTargetHasKindOwner(result.WetManifestTargets, "Deployment", "platform-runtime") {
		t.Fatalf("expected Deployment owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
	if !wetTargetHasKindOwner(result.WetManifestTargets, "ConfigMap", "platform-runtime") {
		t.Fatalf("expected ConfigMap owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}

	if len(result.InversePlans) != 1 {
		t.Fatalf("expected 1 inverse plan, got %d", len(result.InversePlans))
	}
	if len(result.InversePlans[0].Patches) < 2 {
		t.Fatalf("expected at least 2 spring inverse patches, got %+v", result.InversePlans[0].Patches)
	}
}

func TestImportRepoBackstageDryWetContract(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "backstage-idp")
	result, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo returned error: %v", err)
	}

	if len(result.Provenance) != 1 {
		t.Fatalf("expected single provenance record, got %d", len(result.Provenance))
	}
	prov := result.Provenance[0]
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "metadata.name") {
		t.Fatalf("expected metadata.name field origin, got %+v", prov.FieldOriginMap)
	}
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "spec.lifecycle") {
		t.Fatalf("expected spec.lifecycle field origin, got %+v", prov.FieldOriginMap)
	}
	if !inversePointerHasDryPath(prov.InverseEditPointers, "metadata.name") {
		t.Fatalf("expected metadata.name inverse pointer, got %+v", prov.InverseEditPointers)
	}

	if !dryInputHasRoleOwnerPath(result.DryInputs, "catalog-spec", "platform-engineer", "catalog-info.yaml") {
		t.Fatalf("expected catalog-spec owner to be platform-engineer, got %+v", result.DryInputs)
	}
	if !dryInputHasRoleOwnerPath(result.DryInputs, "app-config", "app-team", "app-config.yaml") {
		t.Fatalf("expected app-config owner to be app-team, got %+v", result.DryInputs)
	}

	if !wetTargetHasKindOwner(result.WetManifestTargets, "Application", "platform-runtime") {
		t.Fatalf("expected Application owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
	if !wetTargetHasKindOwner(result.WetManifestTargets, "ConfigMap", "platform-runtime") {
		t.Fatalf("expected ConfigMap owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
}

func TestImportRepoAblyDryWetContract(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "ably-config")
	result, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo returned error: %v", err)
	}

	if len(result.Provenance) != 1 {
		t.Fatalf("expected single provenance record, got %d", len(result.Provenance))
	}
	prov := result.Provenance[0]
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "app.environment") {
		t.Fatalf("expected app.environment field origin, got %+v", prov.FieldOriginMap)
	}
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "channels.inbound") {
		t.Fatalf("expected channels.inbound field origin, got %+v", prov.FieldOriginMap)
	}
	if !inversePointerHasDryPath(prov.InverseEditPointers, "channels.inbound") {
		t.Fatalf("expected channels.inbound inverse pointer, got %+v", prov.InverseEditPointers)
	}

	if !dryInputHasRoleOwnerPath(result.DryInputs, "provider-config-base", "app-team", "ably.yaml") {
		t.Fatalf("expected provider-config-base owner to be app-team, got %+v", result.DryInputs)
	}
	if !dryInputHasRoleOwnerPath(result.DryInputs, "provider-config-overlay", "app-team", "ably-prod.yaml") {
		t.Fatalf("expected provider-config-overlay owner to be app-team, got %+v", result.DryInputs)
	}

	if !wetTargetHasKindOwner(result.WetManifestTargets, "ConfigMap", "platform-runtime") {
		t.Fatalf("expected ConfigMap owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
	if !wetTargetHasKindOwner(result.WetManifestTargets, "Secret", "platform-runtime") {
		t.Fatalf("expected Secret owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
}

func TestImportRepoOpsWorkflowDryWetContract(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "ops-workflow")
	result, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo returned error: %v", err)
	}

	if len(result.Provenance) != 1 {
		t.Fatalf("expected single provenance record, got %d", len(result.Provenance))
	}
	prov := result.Provenance[0]
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "actions.deploy.image_tag") {
		t.Fatalf("expected actions.deploy.image_tag field origin, got %+v", prov.FieldOriginMap)
	}
	if !fieldOriginHasDryPath(prov.FieldOriginMap, "triggers.schedule") {
		t.Fatalf("expected triggers.schedule field origin, got %+v", prov.FieldOriginMap)
	}
	if !inversePointerHasDryPath(prov.InverseEditPointers, "triggers.schedule") {
		t.Fatalf("expected triggers.schedule inverse pointer, got %+v", prov.InverseEditPointers)
	}

	if !dryInputHasRoleOwnerPath(result.DryInputs, "operations-base", "platform-engineer", "operations.yaml") {
		t.Fatalf("expected operations-base owner to be platform-engineer, got %+v", result.DryInputs)
	}
	if !dryInputHasRoleOwnerPath(result.DryInputs, "operations-overlay", "platform-engineer", "operations-prod.yaml") {
		t.Fatalf("expected operations-overlay owner to be platform-engineer, got %+v", result.DryInputs)
	}

	if !wetTargetHasKindOwner(result.WetManifestTargets, "Workflow", "platform-runtime") {
		t.Fatalf("expected Workflow owner to be platform-runtime, got %+v", result.WetManifestTargets)
	}
	if !wetTargetHasKindOwner(result.WetManifestTargets, "Job", "platform-runtime") {
		t.Fatalf("expected Job owner to be platform-runtime, got %+v", result.WetManifestTargets)
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

func containsString(v []string, needle string) bool {
	for _, item := range v {
		if item == needle {
			return true
		}
	}
	return false
}

func renderedLineageHasKind(v []model.RenderedObjectLineage, kind string) bool {
	for _, item := range v {
		if item.Kind == kind {
			return true
		}
	}
	return false
}

func renderedLineageHasSourcePath(v []model.RenderedObjectLineage, sourcePath string) bool {
	for _, item := range v {
		if item.SourcePath == sourcePath {
			return true
		}
	}
	return false
}

func dryInputHasRolePath(v []model.DryInputRef, role, path string) bool {
	for _, item := range v {
		if item.Role == role && item.Path == path {
			return true
		}
	}
	return false
}

func dryInputHasRoleOwnerPath(v []model.DryInputRef, role, owner, path string) bool {
	for _, item := range v {
		if item.Role == role && item.Owner == owner && item.Path == path {
			return true
		}
	}
	return false
}

func wetTargetHasKind(v []model.WetManifestTarget, kind string) bool {
	for _, item := range v {
		if item.Kind == kind {
			return true
		}
	}
	return false
}

func wetTargetHasKindOwner(v []model.WetManifestTarget, kind, owner string) bool {
	for _, item := range v {
		if item.Kind == kind && item.Owner == owner {
			return true
		}
	}
	return false
}
