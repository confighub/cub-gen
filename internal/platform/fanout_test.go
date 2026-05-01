package platform

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/publish"
)

func TestBuildFanoutEmitsStableBundlesPerVariant(t *testing.T) {
	manifest, absPath, err := LoadManifest(variantFanoutManifestPath(t))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	first, err := BuildFanout(absPath, manifest, FanoutOptions{GeneratedAt: "2026-04-30T12:00:00Z"})
	if err != nil {
		t.Fatalf("build first fanout: %v", err)
	}
	second, err := BuildFanout(absPath, manifest, FanoutOptions{GeneratedAt: "2026-04-30T12:00:00Z"})
	if err != nil {
		t.Fatalf("build second fanout: %v", err)
	}
	firstJSON := mustMarshalFanout(t, first)
	secondJSON := mustMarshalFanout(t, second)
	if firstJSON != secondJSON {
		t.Fatalf("fanout is not stable\n--- first ---\n%s\n--- second ---\n%s", firstJSON, secondJSON)
	}
	if first.Summary.VariantCount != 9 || first.Summary.BundleCount != 9 || first.Summary.ComponentCount != 3 || first.Summary.TargetCount != 3 {
		t.Fatalf("unexpected fanout summary: %+v", first.Summary)
	}
	if first.Summary.DiagnosticCount != 0 {
		t.Fatalf("expected no diagnostics, got %+v", first.Diagnostics)
	}

	seenChanges := map[string]struct{}{}
	seenProfiles := map[string]struct{}{}
	for _, variant := range first.Variants {
		if !variant.BundleValid {
			t.Fatalf("bundle for %s was not valid", variant.VariantID)
		}
		if err := publish.VerifyBundle(variant.Bundle); err != nil {
			t.Fatalf("bundle for %s failed bridge-shape verification: %v", variant.VariantID, err)
		}
		if variant.ChangeID == "" {
			t.Fatalf("variant %s has no change_id", variant.VariantID)
		}
		if _, ok := seenChanges[variant.ChangeID]; ok {
			t.Fatalf("duplicate change_id %s for variant %s", variant.ChangeID, variant.VariantID)
		}
		seenChanges[variant.ChangeID] = struct{}{}
		for _, profile := range variant.GeneratorProfiles {
			seenProfiles[profile] = struct{}{}
		}
	}
	for _, profile := range []string{"helm-paas", "scoredev-paas", "springboot-paas"} {
		if _, ok := seenProfiles[profile]; !ok {
			t.Fatalf("missing generator profile %s in fanout profiles %+v", profile, seenProfiles)
		}
	}
}

func TestBuildFanoutCanFilterByVariantName(t *testing.T) {
	manifest, absPath, err := LoadManifest(variantFanoutManifestPath(t))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	result, err := BuildFanout(absPath, manifest, FanoutOptions{
		GeneratedAt:   "2026-04-30T12:00:00Z",
		VariantFilter: "dev",
	})
	if err != nil {
		t.Fatalf("build filtered fanout: %v", err)
	}
	if result.Summary.VariantCount != 3 || result.Summary.BundleCount != 3 {
		t.Fatalf("unexpected filtered fanout summary: %+v", result.Summary)
	}
	for _, variant := range result.Variants {
		if variant.Variant != "dev" {
			t.Fatalf("expected only dev variants, got %+v", result.Variants)
		}
	}
}

func TestBuildFanoutSeparatesSharedAndVariantInputs(t *testing.T) {
	manifest, absPath, err := LoadManifest(variantFanoutManifestPath(t))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	result, err := BuildFanout(absPath, manifest, FanoutOptions{
		GeneratedAt:   "2026-04-30T12:00:00Z",
		VariantFilter: "checkout-api/dev",
	})
	if err != nil {
		t.Fatalf("build helm fanout: %v", err)
	}
	if len(result.Variants) != 1 {
		t.Fatalf("expected one variant, got %d", len(result.Variants))
	}
	variant := result.Variants[0]
	if !containsString(variant.SharedInputs, "Chart.yaml") || !containsString(variant.SharedInputs, "values.yaml") {
		t.Fatalf("expected helm shared inputs to name chart and values, got %+v", variant.SharedInputs)
	}
	if !containsString(variant.VariantInputs, "--set image.tag=dev") || !containsString(variant.VariantInputs, "--set replicaCount=1") {
		t.Fatalf("expected helm variant inputs to name CLI overrides, got %+v", variant.VariantInputs)
	}
}

func TestResolveFanoutVariantsRejectsAmbiguousInferredSource(t *testing.T) {
	_, err := ResolveFanoutVariants(Manifest{
		Repos: []ManifestRepo{
			{ID: "a", Path: "a", Component: "checkout", Variant: "dev"},
			{ID: "b", Path: "b", Component: "checkout", Variant: "dev"},
		},
	})
	if err == nil {
		t.Fatalf("expected ambiguous variant source error")
	}
	if !strings.Contains(err.Error(), "ambiguous variant source for checkout/dev") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustMarshalFanout(t *testing.T, fanout FanoutResult) string {
	t.Helper()
	raw, err := json.MarshalIndent(fanout, "", "  ")
	if err != nil {
		t.Fatalf("marshal fanout: %v", err)
	}
	return string(raw)
}

func variantFanoutManifestPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "variant-fanout", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve variant fanout fixture path: %v", err)
	}
	return path
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
