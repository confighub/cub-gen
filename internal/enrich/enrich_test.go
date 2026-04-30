package enrich

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
)

func TestBuildPlanRendersStableSidecarProposal(t *testing.T) {
	plan := BuildPlan(sampleImportFlow(t.TempDir()))

	if plan.SchemaVersion != PlanSchemaVersion {
		t.Fatalf("unexpected schema version: %s", plan.SchemaVersion)
	}
	if len(plan.Artifacts) != 1 {
		t.Fatalf("expected one artifact, got %d", len(plan.Artifacts))
	}
	artifact := plan.Artifacts[0]
	if artifact.Path != ".cub-gen/enrichment/app-of-apps-root-app-app-of-apps.provenance.json" {
		t.Fatalf("unexpected artifact path: %s", artifact.Path)
	}
	if artifact.Action != ActionCreate {
		t.Fatalf("expected create action, got %s", artifact.Action)
	}
	if plan.Summary.SourceLinkCount != 2 || plan.Summary.OwnershipLabelCount != 3 || plan.Summary.RouteBadgeCount != 2 || plan.Summary.PRMRLinkCount != 1 {
		t.Fatalf("unexpected summary: %+v", plan.Summary)
	}

	assertSortedSourceLinks(t, artifact.Body.SourceLinks)
	assertSortedRouteBadges(t, artifact.Body.RouteBadges)
	assertAnnotationCategories(t, artifact.Body.ProposedAnnotations, []string{
		"ownership-label",
		"pr-mr-link",
		"route-badge",
		"source-link",
		"source-link",
		"source-link",
	})
}

func TestMarkExistingRequiresReviewAndWriteDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	plan := BuildPlan(sampleImportFlow(root))
	path := filepath.Join(root, filepath.FromSlash(plan.Artifacts[0].Path))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir sidecar dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write existing sidecar: %v", err)
	}

	if err := MarkExisting(root, &plan); err != nil {
		t.Fatalf("mark existing: %v", err)
	}
	if plan.Artifacts[0].Action != ActionReviewRequired {
		t.Fatalf("expected review-required action, got %s", plan.Artifacts[0].Action)
	}
	result, err := Write(root, plan)
	if err == nil {
		t.Fatal("expected write to be blocked")
	}
	if len(result.Blocked) != 1 || result.Blocked[0] != plan.Artifacts[0].Path {
		t.Fatalf("unexpected blocked result: %+v", result)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read existing sidecar: %v", readErr)
	}
	if string(got) != "existing\n" {
		t.Fatalf("write overwrote existing sidecar: %q", string(got))
	}
}

func TestWriteCreatesReviewableSidecar(t *testing.T) {
	root := t.TempDir()
	plan := BuildPlan(sampleImportFlow(root))
	if err := MarkExisting(root, &plan); err != nil {
		t.Fatalf("mark existing: %v", err)
	}
	result, err := Write(root, plan)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(result.Written) != 1 {
		t.Fatalf("expected one written artifact, got %+v", result)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.Written[0])))
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var body ArtifactBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("parse sidecar: %v", err)
	}
	if body.Generator.Kind != "app-of-apps" || len(body.RouteBadges) != 2 {
		t.Fatalf("unexpected sidecar body: %+v", body)
	}
}

func TestRenderPatchIncludesSidecarCreationOnly(t *testing.T) {
	plan := BuildPlan(sampleImportFlow(t.TempDir()))
	patch, err := RenderPatch(plan)
	if err != nil {
		t.Fatalf("render patch: %v", err)
	}
	for _, want := range []string{
		"diff --git a/.cub-gen/enrichment/app-of-apps-root-app-app-of-apps.provenance.json",
		"new file mode 100644",
		"+  \"schema_version\": \"cub.confighub.io/enrichment-artifact/v1\"",
		"+      \"category\": \"ownership-label\"",
		"+      \"key\": \"cub.confighub.io/routes\"",
	} {
		if !strings.Contains(patch, want) {
			t.Fatalf("patch missing %q\n%s", want, patch)
		}
	}
}

func sampleImportFlow(root string) gitopsflow.ImportFlowResult {
	return gitopsflow.ImportFlowResult{
		Space:            "platform",
		TargetSlug:       "app-of-apps",
		TargetPath:       root,
		RenderTargetSlug: "render-target",
		RenderTargetPath: root,
		Ref:              "HEAD",
		ImportedAt:       "2026-04-30T12:00:00Z",
		Discovered: []gitopsflow.DiscoveredResource{{
			GeneratorID:      "app-of-apps:test",
			GeneratorProfile: "argo-app-of-apps",
			ResourceName:     "root-app",
			ResourceKind:     "Application",
			GeneratorKind:    "app-of-apps",
			Root:             "root-application.yaml",
			Inputs:           []string{"root-application.yaml", "apps/catalog-api.yaml"},
		}},
		Contracts: []model.GeneratorContract{{
			GeneratorID: "app-of-apps:test",
			Kind:        "app-of-apps",
			Profile:     "argo-app-of-apps",
			Name:        "root-app",
			SourcePath:  "root-application.yaml",
		}},
		DryInputs: []model.DryInputRef{
			{GeneratorID: "app-of-apps:test", Profile: "argo-app-of-apps", Role: "child-application", Owner: "app-team", Path: "apps/catalog-api.yaml", Required: true},
			{GeneratorID: "app-of-apps:test", Profile: "argo-app-of-apps", Role: "root-application", Owner: "platform-team", Path: "root-application.yaml", Required: true},
		},
		WetManifestTargets: []model.WetManifestTarget{
			{GeneratorID: "app-of-apps:test", Kind: "Application", Name: "catalog-api", Owner: "app-team", Namespace: "argocd", SourceDryPath: "apps/catalog-api.yaml"},
		},
		Provenance: []model.ProvenanceRecord{{
			GeneratorID:      "app-of-apps:test",
			GeneratorName:    "root-app",
			GeneratorProfile: "argo-app-of-apps",
			ChangeID:         "chg_app_of_apps_test",
			InputDigest:      "sha256:abc123",
			Sources: []model.SourceRef{
				{Path: "apps/catalog-api.yaml", URI: "git://example/repo//apps/catalog-api.yaml"},
				{Path: "root-application.yaml", URI: "git://example/repo//root-application.yaml"},
			},
			FieldOriginMap: []model.FieldOrigin{
				{DryPath: "apps.catalog-api.spec.source.path", WetPath: "Application/spec/source/path", SourcePath: "apps/catalog-api.yaml"},
			},
			InverseEditPointers: []model.InverseEditPointer{
				{WetPath: "Application/spec/source/path", DryPath: "apps.catalog-api.spec.source.path", Owner: "app-team", Route: "apply-here", EditHint: "Edit the child Application.", Confidence: 0.92},
				{WetPath: "Application/spec.project", DryPath: "root.spec.project", Owner: "platform-team", Route: "lift-upstream", EditHint: "Route to the root Application owner.", Confidence: 0.87},
			},
		}},
	}
}

func assertSortedSourceLinks(t *testing.T, links []SourceLink) {
	t.Helper()
	if len(links) != 2 {
		t.Fatalf("expected 2 source links, got %d", len(links))
	}
	if links[0].Owner != "app-team" || links[0].Path != "apps/catalog-api.yaml" {
		t.Fatalf("source links are not sorted by owner/role/path: %+v", links)
	}
}

func assertSortedRouteBadges(t *testing.T, badges []RouteBadge) {
	t.Helper()
	if len(badges) != 2 {
		t.Fatalf("expected 2 route badges, got %d", len(badges))
	}
	if badges[0].Route != "apply-here" || badges[1].Route != "lift-upstream" {
		t.Fatalf("route badges are not sorted: %+v", badges)
	}
}

func assertAnnotationCategories(t *testing.T, annotations []ProposedAnnotation, want []string) {
	t.Helper()
	if len(annotations) != len(want) {
		t.Fatalf("annotation count mismatch: got %d want %d: %+v", len(annotations), len(want), annotations)
	}
	for i, annotation := range annotations {
		if annotation.Category != want[i] {
			t.Fatalf("annotation %d category = %s, want %s: %+v", i, annotation.Category, want[i], annotations)
		}
	}
}
