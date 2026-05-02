package publish

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/proof"
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
	if bundle.TargetPath != imported.TargetPath {
		t.Fatalf("expected target_path %q, got %q", imported.TargetPath, bundle.TargetPath)
	}
	if bundle.RenderTargetPath != imported.RenderTargetPath {
		t.Fatalf("expected render_target_path %q, got %q", imported.RenderTargetPath, bundle.RenderTargetPath)
	}
	if bundle.ChangeID == "" {
		t.Fatal("expected non-empty change_id")
	}
	if bundle.TraceID != bundle.ChangeID {
		t.Fatalf("expected trace_id to match change_id, got %q", bundle.TraceID)
	}
	if len(bundle.ProofEvents) != 1 {
		t.Fatalf("expected one proof event, got %d", len(bundle.ProofEvents))
	}
	event := bundle.ProofEvents[0]
	if event.EventType != proof.EventTypeChangeBundlePublished {
		t.Fatalf("unexpected proof event type: %q", event.EventType)
	}
	if event.TraceID != bundle.TraceID {
		t.Fatalf("expected proof event trace_id %q, got %q", bundle.TraceID, event.TraceID)
	}
	if event.ChangeID != bundle.ChangeID {
		t.Fatalf("expected proof event change_id %q, got %q", bundle.ChangeID, event.ChangeID)
	}
	if event.ArtifactKind != proof.ArtifactKindChangeBundle {
		t.Fatalf("unexpected proof artifact kind: %q", event.ArtifactKind)
	}
	if event.ArtifactDigest != bundle.BundleDigest {
		t.Fatalf("expected proof event artifact digest %q, got %q", bundle.BundleDigest, event.ArtifactDigest)
	}
	if event.SummaryCounts["provenance_records"] != len(imported.Provenance) {
		t.Fatalf("unexpected proof summary counts: %+v", event.SummaryCounts)
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
	if bundle.TraceID != "chg_fallback" {
		t.Fatalf("expected fallback trace id chg_fallback, got %q", bundle.TraceID)
	}
}

func TestVerifyBundle(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")
	imported, err := gitopsflow.Import(repo, repo, "HEAD", "platform", "")
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	bundle := BuildBundleAt(imported, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC))

	if err := VerifyBundle(bundle); err != nil {
		t.Fatalf("VerifyBundle returned error for valid bundle: %v", err)
	}

	tampered := bundle
	tampered.Space = "tampered"
	if err := VerifyBundle(tampered); err == nil {
		t.Fatal("expected VerifyBundle to fail on tampered bundle")
	}

	missingDigest := bundle
	missingDigest.BundleDigest = ""
	if err := VerifyBundle(missingDigest); err == nil || !strings.Contains(err.Error(), "missing bundle_digest") {
		t.Fatalf("expected missing bundle_digest error, got %v", err)
	}

	missingProof := bundle
	missingProof.ProofEvents = nil
	missingProof.BundleDigest = computeBundleDigest(missingProof)
	if err := VerifyBundle(missingProof); err == nil || !strings.Contains(err.Error(), "missing proof_events") {
		t.Fatalf("expected missing proof_events error, got %v", err)
	}

	tamperedProof := bundle
	tamperedProof.ProofEvents = append([]proof.Event(nil), bundle.ProofEvents...)
	tamperedProof.ProofEvents[0].ArtifactDigest = "sha256:wrong"
	if err := VerifyBundle(tamperedProof); err == nil || !strings.Contains(err.Error(), "artifact_digest mismatch") {
		t.Fatalf("expected proof artifact digest mismatch, got %v", err)
	}

	tamperedTrace := bundle
	tamperedTrace.TraceID = "trace_custom"
	tamperedTrace.ProofEvents = append([]proof.Event(nil), bundle.ProofEvents...)
	tamperedTrace.ProofEvents[0].TraceID = tamperedTrace.TraceID
	tamperedTrace.ProofEvents[0].EventID = proof.EventID(tamperedTrace.ProofEvents[0])
	tamperedTrace.BundleDigest = computeBundleDigest(tamperedTrace)
	tamperedTrace.ProofEvents = proof.SetArtifactDigest(tamperedTrace.ProofEvents, proof.ArtifactKindChangeBundle, tamperedTrace.BundleDigest)
	if err := VerifyBundle(tamperedTrace); err == nil || !strings.Contains(err.Error(), "trace_id mismatch") {
		t.Fatalf("expected trace_id mismatch, got %v", err)
	}

	unsupported := bundle
	unsupported.DigestAlgorithm = "sha512"
	if err := VerifyBundle(unsupported); err == nil || !strings.Contains(err.Error(), "unsupported digest_algorithm") {
		t.Fatalf("expected unsupported digest_algorithm error, got %v", err)
	}
}
