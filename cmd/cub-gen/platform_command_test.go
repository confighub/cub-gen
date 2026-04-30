package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	platformflow "github.com/confighub/cub-gen/internal/platform"
)

func TestPlatformImportGolden(t *testing.T) {
	manifest, err := filepath.Abs(filepath.Join("..", "..", "testdata", "platform-estate", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve platform fixture: %v", err)
	}
	stdout, stderr, err := runWithCapturedIO([]string{"platform", "import", "--json", manifest})
	if err != nil {
		t.Fatalf("platform import failed: %v\nstderr=%s", err, stderr)
	}
	var graph platformflow.Graph
	if err := json.Unmarshal([]byte(stdout), &graph); err != nil {
		t.Fatalf("parse platform graph: %v\n%s", err, stdout)
	}
	graph.ManifestPath = "<manifest_path>"
	graph.GeneratedAt = "<timestamp>"
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "platform-import.golden.json"), graph)
}
