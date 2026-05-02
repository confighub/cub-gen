package main

import (
	"encoding/json"
	"os"
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

func TestPlatformFanoutGolden(t *testing.T) {
	manifest, err := filepath.Abs(filepath.Join("..", "..", "testdata", "variant-fanout", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve variant fanout fixture: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "fanout.json")
	stdout, stderr, err := runWithCapturedIO([]string{"platform", "fanout", "--json", "--out", outPath, manifest})
	if err != nil {
		t.Fatalf("platform fanout failed: %v\nstderr=%s", err, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout when --out is used, got %q", stdout)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read fanout output: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("parse platform fanout: %v\n%s", err, string(raw))
	}
	normalizeFanout(got)
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "platform-fanout.golden.json"), got)
}

func TestPlatformAdaptCommand(t *testing.T) {
	manifest, err := filepath.Abs(filepath.Join("..", "..", "testdata", "deployment-adaptation", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve adaptation fixture: %v", err)
	}
	stdout, stderr, err := runWithCapturedIO([]string{"platform", "adapt", "--json", manifest})
	if err != nil {
		t.Fatalf("platform adapt failed: %v\nstderr=%s", err, stderr)
	}
	var plan platformflow.AdaptationResult
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("parse platform adapt output: %v\n%s", err, stdout)
	}
	if plan.Summary.DeploymentCount != 1 || plan.Summary.PlaceholderCount != 3 || plan.Summary.ProposedReplacementCount != 3 {
		t.Fatalf("unexpected adaptation summary: %+v", plan.Summary)
	}
	if len(plan.Deployments) != 1 || plan.Deployments[0].ApplyGate.State != "blocked-before-adaptation" {
		t.Fatalf("unexpected adaptation deployments: %+v", plan.Deployments)
	}
}

func TestChangeExplainCanScopeFanoutBundleByVariant(t *testing.T) {
	manifest, err := filepath.Abs(filepath.Join("..", "..", "testdata", "variant-fanout", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve variant fanout fixture: %v", err)
	}
	fanoutPath := filepath.Join(t.TempDir(), "fanout.json")
	stdout, stderr, err := runWithCapturedIO([]string{"platform", "fanout", "--json", "--out", fanoutPath, manifest})
	if err != nil {
		t.Fatalf("platform fanout failed: %v\nstderr=%s", err, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout when --out is used, got %q", stdout)
	}
	raw, err := os.ReadFile(fanoutPath)
	if err != nil {
		t.Fatalf("read fanout output: %v", err)
	}
	var fanout platformflow.FanoutResult
	if err := json.Unmarshal(raw, &fanout); err != nil {
		t.Fatalf("parse fanout: %v", err)
	}
	changeID := ""
	for _, variant := range fanout.Variants {
		if variant.VariantID == "checkout-api/dev" {
			changeID = variant.ChangeID
			break
		}
	}
	if changeID == "" {
		t.Fatalf("did not find checkout-api/dev change_id")
	}
	explainOut, explainErr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--bundle", fanoutPath,
		"--change-id", changeID,
		"--variant", "checkout-api/dev",
	})
	if err != nil {
		t.Fatalf("change explain failed: %v\nstderr=%s", err, explainErr)
	}
	var explain changeExplainResult
	if err := json.Unmarshal([]byte(explainOut), &explain); err != nil {
		t.Fatalf("parse change explain: %v\n%s", err, explainOut)
	}
	if explain.Input.TargetSlug != "checkout-api/dev" {
		t.Fatalf("expected variant-scoped target slug, got %s", explain.Input.TargetSlug)
	}
	if explain.Query.VariantFilter != "checkout-api/dev" {
		t.Fatalf("expected variant filter in query, got %+v", explain.Query)
	}
}

func normalizeFanout(m map[string]any) {
	replaceString(m, "manifest_path", "<manifest_path>")
	replaceString(m, "generated_at", "<timestamp>")
	for _, variant := range asSlice(m["variants"]) {
		replaceString(variant, "change_id", "<change_id>")
		replaceString(variant, "bundle_digest", "<bundle_digest>")
		if bundle, ok := variant["bundle"].(map[string]any); ok {
			normalizePublish(bundle)
		}
	}
}
