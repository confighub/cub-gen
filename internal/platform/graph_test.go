package platform

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestLoadManifestParsesPlatformEstate(t *testing.T) {
	manifest, _, err := LoadManifest(fixtureManifestPath(t))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if manifest.Name != "checkout-platform" {
		t.Fatalf("unexpected manifest name: %s", manifest.Name)
	}
	if manifest.Space != "platform" || manifest.Ref != "HEAD" {
		t.Fatalf("unexpected manifest scope: space=%s ref=%s", manifest.Space, manifest.Ref)
	}
	if len(manifest.Repos) != 6 {
		t.Fatalf("expected 6 repos, got %d", len(manifest.Repos))
	}
}

func TestBuildGraphIsStableAndReportsDiagnostics(t *testing.T) {
	manifest, absPath, err := LoadManifest(fixtureManifestPath(t))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	first, err := BuildGraph(absPath, manifest, ImportOptions{GeneratedAt: "2026-04-30T12:00:00Z"})
	if err != nil {
		t.Fatalf("build first graph: %v", err)
	}
	second, err := BuildGraph(absPath, manifest, ImportOptions{GeneratedAt: "2026-04-30T12:00:00Z"})
	if err != nil {
		t.Fatalf("build second graph: %v", err)
	}
	firstJSON := mustMarshalGraph(t, first)
	secondJSON := mustMarshalGraph(t, second)
	if firstJSON != secondJSON {
		t.Fatalf("graph is not stable\n--- first ---\n%s\n--- second ---\n%s", firstJSON, secondJSON)
	}

	if first.Summary.RepoCount != 6 || first.Summary.ComponentCount != 3 || first.Summary.VariantCount != 3 || first.Summary.GeneratorCount != 2 {
		t.Fatalf("unexpected graph summary: %+v", first.Summary)
	}
	assertDiagnostic(t, first, "missing_owner", "orders-api")
	assertDiagnostic(t, first, "missing_repo", "ghost-api")
	assertDiagnostic(t, first, "unsupported_generator", "platform-base")
}

func TestBuildGraphClassifiesBaseAndDeploymentVariants(t *testing.T) {
	manifest, absPath, err := LoadManifest(variantTopologyManifestPath(t))
	if err != nil {
		t.Fatalf("load topology manifest: %v", err)
	}
	graph, err := BuildGraph(absPath, manifest, ImportOptions{GeneratedAt: "2026-05-02T12:00:00Z"})
	if err != nil {
		t.Fatalf("build topology graph: %v", err)
	}
	if graph.Summary.ComponentCount != 1 || graph.Summary.VariantCount != 2 || graph.Summary.TargetCount != 1 {
		t.Fatalf("unexpected topology summary: %+v", graph.Summary)
	}
	base := findVariant(t, graph, "checkout-api/base")
	if base.VariantKind != "base" {
		t.Fatalf("expected checkout-api/base to be a base variant, got %+v", base)
	}
	if base.Target != "" {
		t.Fatalf("base variant should not have a target, got %+v", base)
	}
	deployment := findVariant(t, graph, "checkout-api/prod-us")
	if deployment.VariantKind != "deployment" || deployment.Target != "prod-us" {
		t.Fatalf("expected checkout-api/prod-us to be a deployment variant with target, got %+v", deployment)
	}
	assertConnection(t, graph, "component", "checkout-api", "variant", "checkout-api/base", "has-variant")
	assertConnection(t, graph, "component", "checkout-api", "variant", "checkout-api/prod-us", "has-variant")
	assertConnection(t, graph, "variant", "checkout-api/prod-us", "target", "prod-us", "deploys-to")
	assertNoConnection(t, graph, "variant", "checkout-api/base", "target", "prod-us", "deploys-to")
}

func TestBuildGraphReportsInvalidVariantKind(t *testing.T) {
	manifest, absPath, err := LoadManifest(variantTopologyManifestPath(t))
	if err != nil {
		t.Fatalf("load topology manifest: %v", err)
	}
	manifest.Repos[0].VariantKind = "shared"
	graph, err := BuildGraph(absPath, manifest, ImportOptions{GeneratedAt: "2026-05-02T12:00:00Z"})
	if err != nil {
		t.Fatalf("build topology graph: %v", err)
	}
	base := findVariant(t, graph, "checkout-api/base")
	if base.VariantKind != "unknown" {
		t.Fatalf("invalid variant kind should degrade to unknown, got %+v", base)
	}
	assertDiagnostic(t, graph, "invalid_variant_kind", "checkout-base")
}

func TestBuildGraphReportsContradictoryVariantKindAndTarget(t *testing.T) {
	manifest, absPath, err := LoadManifest(variantTopologyManifestPath(t))
	if err != nil {
		t.Fatalf("load topology manifest: %v", err)
	}
	manifest.Repos[0].VariantKind = "base"
	manifest.Repos[0].Target = "prod-us"
	manifest.Repos[1].VariantKind = "deployment"
	manifest.Repos[1].Target = ""
	graph, err := BuildGraph(absPath, manifest, ImportOptions{GeneratedAt: "2026-05-02T12:00:00Z"})
	if err != nil {
		t.Fatalf("build topology graph: %v", err)
	}
	base := findVariant(t, graph, "checkout-api/base")
	if base.VariantKind != "unknown" {
		t.Fatalf("base with a target should degrade to unknown, got %+v", base)
	}
	deployment := findVariant(t, graph, "checkout-api/prod-us")
	if deployment.VariantKind != "unknown" {
		t.Fatalf("deployment with no target should degrade to unknown, got %+v", deployment)
	}
	assertDiagnostic(t, graph, "invalid_variant_kind", "checkout-base")
	assertDiagnostic(t, graph, "invalid_variant_kind", "checkout-prod-us")
}

func fixtureManifestPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "platform-estate", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	return path
}

func variantTopologyManifestPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "variant-topology", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve variant topology fixture path: %v", err)
	}
	return path
}

func mustMarshalGraph(t *testing.T, graph Graph) string {
	t.Helper()
	raw, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	return string(raw)
}

func assertDiagnostic(t *testing.T, graph Graph, code, repoID string) {
	t.Helper()
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code == code && diagnostic.RepoID == repoID {
			return
		}
	}
	t.Fatalf("missing diagnostic code=%s repo=%s in %+v", code, repoID, graph.Diagnostics)
}

func findVariant(t *testing.T, graph Graph, id string) VariantNode {
	t.Helper()
	for _, variant := range graph.Variants {
		if variant.ID == id {
			return variant
		}
	}
	t.Fatalf("missing variant %s in %+v", id, graph.Variants)
	return VariantNode{}
}

func assertConnection(t *testing.T, graph Graph, fromType, fromID, toType, toID, kind string) {
	t.Helper()
	if !hasConnection(graph, fromType, fromID, toType, toID, kind) {
		t.Fatalf("missing connection %s/%s --%s--> %s/%s in %+v", fromType, fromID, kind, toType, toID, graph.Connections)
	}
}

func assertNoConnection(t *testing.T, graph Graph, fromType, fromID, toType, toID, kind string) {
	t.Helper()
	if hasConnection(graph, fromType, fromID, toType, toID, kind) {
		t.Fatalf("unexpected connection %s/%s --%s--> %s/%s in %+v", fromType, fromID, kind, toType, toID, graph.Connections)
	}
}

func hasConnection(graph Graph, fromType, fromID, toType, toID, kind string) bool {
	for _, conn := range graph.Connections {
		if conn.FromType == fromType &&
			conn.FromID == fromID &&
			conn.ToType == toType &&
			conn.ToID == toID &&
			conn.Kind == kind {
			return true
		}
	}
	return false
}
