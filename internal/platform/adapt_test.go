package platform

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestBuildAdaptationPlanDetectsPlaceholdersAndApplyGate(t *testing.T) {
	manifest, absPath, err := LoadManifest(adaptationManifestPath(t))
	if err != nil {
		t.Fatalf("load adaptation manifest: %v", err)
	}
	plan, err := BuildAdaptationPlan(absPath, manifest, AdaptationOptions{GeneratedAt: "2026-05-02T12:00:00Z"})
	if err != nil {
		t.Fatalf("build adaptation plan: %v", err)
	}
	firstJSON := mustMarshalAdaptation(t, plan)
	second, err := BuildAdaptationPlan(absPath, manifest, AdaptationOptions{GeneratedAt: "2026-05-02T12:00:00Z"})
	if err != nil {
		t.Fatalf("build second adaptation plan: %v", err)
	}
	secondJSON := mustMarshalAdaptation(t, second)
	if firstJSON != secondJSON {
		t.Fatalf("adaptation plan is not stable\n--- first ---\n%s\n--- second ---\n%s", firstJSON, secondJSON)
	}
	if plan.Summary.DeploymentCount != 1 || plan.Summary.PlaceholderCount != 3 || plan.Summary.ProposedReplacementCount != 3 || plan.Summary.ApplyGateCount != 1 {
		t.Fatalf("unexpected adaptation summary: %+v", plan.Summary)
	}
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", plan.Diagnostics)
	}

	deployment := plan.Deployments[0]
	if deployment.ID != "checkout-api/prod-us" || deployment.VariantKind != "deployment" || deployment.Target != "prod-us" {
		t.Fatalf("unexpected deployment adaptation: %+v", deployment)
	}
	if deployment.BaseVariant != "base" {
		t.Fatalf("expected base variant, got %+v", deployment)
	}
	if deployment.ApplyGate.Name != "vet-placeholders" || deployment.ApplyGate.State != "blocked-before-adaptation" || deployment.ApplyGate.UnresolvedCount != 3 {
		t.Fatalf("unexpected apply gate: %+v", deployment.ApplyGate)
	}

	imageTag := findAdaptationPlaceholder(t, deployment, "{{IMAGE_TAG}}")
	if imageTag.Value != "2026.05.02" || imageTag.ValueFrom != "image_tag" || imageTag.Owner != "app-team" || imageTag.Route != "apply-here" {
		t.Fatalf("unexpected image tag placeholder: %+v", imageTag)
	}
	if imageTag.Status != "proposed" || imageTag.OccurrenceCount != 1 {
		t.Fatalf("unexpected image tag placeholder status: %+v", imageTag)
	}
	if len(imageTag.Files) != 1 || imageTag.Files[0].Path != "values.yaml" || len(imageTag.Files[0].Occurrences) != 1 {
		t.Fatalf("unexpected image tag occurrences: %+v", imageTag.Files)
	}

	database := findAdaptationPlaceholder(t, deployment, "{{DATABASE_URL}}")
	if database.Route != "lift-upstream" || database.Value != "secretref://checkout-prod-us/db-url" {
		t.Fatalf("unexpected database placeholder: %+v", database)
	}
}

func TestBuildAdaptationPlanReportsMissingContext(t *testing.T) {
	manifest, absPath, err := LoadManifest(adaptationManifestPath(t))
	if err != nil {
		t.Fatalf("load adaptation manifest: %v", err)
	}
	delete(manifest.Adaptations[0].Context, "database_url")
	plan, err := BuildAdaptationPlan(absPath, manifest, AdaptationOptions{GeneratedAt: "2026-05-02T12:00:00Z"})
	if err != nil {
		t.Fatalf("build adaptation plan: %v", err)
	}
	assertAdaptationDiagnostic(t, plan, "missing_adaptation_context", "checkout-prod-us")
	database := findAdaptationPlaceholder(t, plan.Deployments[0], "{{DATABASE_URL}}")
	if database.Status != "blocked" || database.Value != "" {
		t.Fatalf("expected database placeholder to block without context, got %+v", database)
	}
	if plan.Summary.ProposedReplacementCount != 2 || plan.Summary.DiagnosticCount != 1 {
		t.Fatalf("unexpected summary after missing context: %+v", plan.Summary)
	}
}

func TestBuildAdaptationPlanRequiresDeploymentVariant(t *testing.T) {
	manifest, absPath, err := LoadManifest(adaptationManifestPath(t))
	if err != nil {
		t.Fatalf("load adaptation manifest: %v", err)
	}
	manifest.Adaptations[0].Repo = "checkout-base"
	manifest.Adaptations[0].Variant = "base"
	manifest.Adaptations[0].ID = "checkout-api/base"
	plan, err := BuildAdaptationPlan(absPath, manifest, AdaptationOptions{GeneratedAt: "2026-05-02T12:00:00Z"})
	if err != nil {
		t.Fatalf("build adaptation plan: %v", err)
	}
	assertAdaptationDiagnostic(t, plan, "not_deployment_variant", "checkout-base")
	if len(plan.Deployments) != 1 {
		t.Fatalf("expected deployment record with blocking diagnostic, got %+v", plan.Deployments)
	}
	if plan.Deployments[0].VariantKind != "base" || plan.Deployments[0].Target != "" {
		t.Fatalf("expected base variant to remain classified as base, got %+v", plan.Deployments[0])
	}
}

func adaptationManifestPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "deployment-adaptation", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve adaptation fixture path: %v", err)
	}
	return path
}

func mustMarshalAdaptation(t *testing.T, plan AdaptationResult) string {
	t.Helper()
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatalf("marshal adaptation plan: %v", err)
	}
	return string(raw)
}

func findAdaptationPlaceholder(t *testing.T, deployment AdaptationDeployment, token string) AdaptationPlaceholder {
	t.Helper()
	for _, placeholder := range deployment.Placeholders {
		if placeholder.Token == token {
			return placeholder
		}
	}
	t.Fatalf("missing placeholder %s in %+v", token, deployment.Placeholders)
	return AdaptationPlaceholder{}
}

func assertAdaptationDiagnostic(t *testing.T, plan AdaptationResult, code, repoID string) {
	t.Helper()
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Code == code && diagnostic.RepoID == repoID {
			return
		}
	}
	t.Fatalf("missing diagnostic code=%s repo=%s in %+v", code, repoID, plan.Diagnostics)
}
