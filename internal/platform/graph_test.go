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

func fixtureManifestPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "platform-estate", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
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
