package publish

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
)

func TestBuildBundleAtFromHelmImport(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")
	imported, err := gitopsflow.Import(repo, repo, "HEAD", "platform", "")
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}

	at := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
	bundle := BuildBundleAt(imported, at)

	if bundle.SchemaVersion != changeBundleSchema {
		t.Fatalf("unexpected schema version: %q", bundle.SchemaVersion)
	}
	if bundle.Source != changeBundleSource {
		t.Fatalf("unexpected source: %q", bundle.Source)
	}
	if bundle.GeneratedAt != "2026-03-06T00:00:00Z" {
		t.Fatalf("unexpected generated_at: %q", bundle.GeneratedAt)
	}
	if bundle.DigestAlgorithm != "sha256" {
		t.Fatalf("unexpected digest_algorithm: %q", bundle.DigestAlgorithm)
	}
	if !strings.HasPrefix(bundle.BundleDigest, "sha256:") {
		t.Fatalf("expected bundle_digest with sha256 prefix, got %q", bundle.BundleDigest)
	}
	if bundle.Space != "platform" {
		t.Fatalf("expected space=platform, got %q", bundle.Space)
	}
	if bundle.ChangeID == "" {
		t.Fatal("expected non-empty change_id")
	}
	if bundle.Summary.DiscoveredResources != len(imported.Discovered) {
		t.Fatalf("summary discovered mismatch: %d vs %d", bundle.Summary.DiscoveredResources, len(imported.Discovered))
	}
	if bundle.Summary.DryInputs != len(imported.DryInputs) {
		t.Fatalf("summary dry_inputs mismatch: %d vs %d", bundle.Summary.DryInputs, len(imported.DryInputs))
	}
	if len(bundle.Summary.GeneratorProfiles) != 1 || bundle.Summary.GeneratorProfiles[0] != "helm-paas" {
		t.Fatalf("unexpected generator profiles: %+v", bundle.Summary.GeneratorProfiles)
	}

	again := BuildBundleAt(imported, at)
	if bundle.BundleDigest != again.BundleDigest {
		t.Fatalf("expected deterministic bundle_digest, got %q and %q", bundle.BundleDigest, again.BundleDigest)
	}
}

func TestBuildBundleAtChangeIDFallbackToInversePlans(t *testing.T) {
	imported := gitopsflow.ImportFlowResult{
		Provenance: nil,
		InversePlans: []model.InverseTransformPlan{{
			ChangeID: "chg_fallback",
		}},
	}

	bundle := BuildBundleAt(imported, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC))
	if bundle.ChangeID != "chg_fallback" {
		t.Fatalf("expected fallback change id chg_fallback, got %q", bundle.ChangeID)
	}
}
