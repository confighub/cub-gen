package attest

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/proof"
	"github.com/confighub/cub-gen/internal/publish"
)

func TestBuildAt(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")
	imported, err := gitopsflow.Import(repo, repo, "HEAD", "platform", "")
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	bundle := publish.BuildBundleAt(imported, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC))

	rec, err := BuildAt(bundle, time.Date(2026, 3, 6, 1, 0, 0, 0, time.UTC), "ci-bot")
	if err != nil {
		t.Fatalf("BuildAt returned error: %v", err)
	}

	if rec.SchemaVersion != attestationSchema {
		t.Fatalf("unexpected schema_version: %q", rec.SchemaVersion)
	}
	if rec.Source != attestationSource {
		t.Fatalf("unexpected source: %q", rec.Source)
	}
	if rec.Verifier != "ci-bot" {
		t.Fatalf("unexpected verifier: %q", rec.Verifier)
	}
	if rec.TargetPath != bundle.TargetPath {
		t.Fatalf("expected target_path %q, got %q", bundle.TargetPath, rec.TargetPath)
	}
	if rec.RenderTargetPath != bundle.RenderTargetPath {
		t.Fatalf("expected render_target_path %q, got %q", bundle.RenderTargetPath, rec.RenderTargetPath)
	}
	if rec.Status != "verified" {
		t.Fatalf("unexpected status: %q", rec.Status)
	}
	if rec.DigestAlgorithm != "sha256" {
		t.Fatalf("unexpected digest_algorithm: %q", rec.DigestAlgorithm)
	}
	if !strings.HasPrefix(rec.BundleDigest, "sha256:") {
		t.Fatalf("expected bundle digest sha256 prefix, got %q", rec.BundleDigest)
	}
	if !strings.HasPrefix(rec.AttestationDigest, "sha256:") {
		t.Fatalf("expected attestation digest sha256 prefix, got %q", rec.AttestationDigest)
	}
	if rec.TraceID != bundle.TraceID {
		t.Fatalf("expected trace_id %q, got %q", bundle.TraceID, rec.TraceID)
	}
	if len(rec.ProofEvents) != 1 {
		t.Fatalf("expected one proof event, got %d", len(rec.ProofEvents))
	}
	event := rec.ProofEvents[0]
	if event.EventType != proof.EventTypeAttestationVerified {
		t.Fatalf("unexpected proof event type: %q", event.EventType)
	}
	if event.TraceID != rec.TraceID {
		t.Fatalf("expected proof event trace_id %q, got %q", rec.TraceID, event.TraceID)
	}
	if event.ArtifactKind != proof.ArtifactKindAttestation {
		t.Fatalf("unexpected proof artifact kind: %q", event.ArtifactKind)
	}
	if event.ArtifactDigest != rec.AttestationDigest {
		t.Fatalf("expected proof event artifact digest %q, got %q", rec.AttestationDigest, event.ArtifactDigest)
	}
	if event.ParentArtifactDigest != bundle.BundleDigest {
		t.Fatalf("expected parent artifact digest %q, got %q", bundle.BundleDigest, event.ParentArtifactDigest)
	}
	if event.ParentEventID != bundle.ProofEvents[0].EventID {
		t.Fatalf("expected parent event id %q, got %q", bundle.ProofEvents[0].EventID, event.ParentEventID)
	}

	again, err := BuildAt(bundle, time.Date(2026, 3, 6, 1, 0, 0, 0, time.UTC), "ci-bot")
	if err != nil {
		t.Fatalf("BuildAt second run error: %v", err)
	}
	if rec.AttestationDigest != again.AttestationDigest {
		t.Fatalf("expected deterministic attestation digest, got %q and %q", rec.AttestationDigest, again.AttestationDigest)
	}
}

func TestBuildAtRejectsInvalidBundle(t *testing.T) {
	bundle := publish.ChangeBundle{}
	if _, err := BuildAt(bundle, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), ""); err == nil {
		t.Fatal("expected BuildAt to fail for invalid bundle")
	}
}

func TestVerifyRecord(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")
	imported, err := gitopsflow.Import(repo, repo, "HEAD", "platform", "")
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	bundle := publish.BuildBundleAt(imported, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC))
	rec, err := BuildAt(bundle, time.Date(2026, 3, 6, 1, 0, 0, 0, time.UTC), "ci-bot")
	if err != nil {
		t.Fatalf("BuildAt returned error: %v", err)
	}

	if err := VerifyRecord(rec); err != nil {
		t.Fatalf("VerifyRecord returned error for valid record: %v", err)
	}

	tampered := rec
	tampered.Status = "tampered"
	if err := VerifyRecord(tampered); err == nil {
		t.Fatal("expected VerifyRecord to fail for tampered status")
	}

	missingDigest := rec
	missingDigest.AttestationDigest = ""
	if err := VerifyRecord(missingDigest); err == nil || !strings.Contains(err.Error(), "missing attestation_digest") {
		t.Fatalf("expected missing attestation digest error, got %v", err)
	}

	missingProof := rec
	missingProof.ProofEvents = nil
	missingProof.AttestationDigest = computeAttestationDigest(missingProof)
	if err := VerifyRecord(missingProof); err == nil || !strings.Contains(err.Error(), "missing proof_events") {
		t.Fatalf("expected missing proof_events error, got %v", err)
	}

	tamperedProof := rec
	tamperedProof.ProofEvents = append([]proof.Event(nil), rec.ProofEvents...)
	tamperedProof.ProofEvents[0].ParentArtifactDigest = "sha256:wrong"
	tamperedProof.ProofEvents[0].EventID = proof.EventID(tamperedProof.ProofEvents[0])
	tamperedProof.AttestationDigest = computeAttestationDigest(tamperedProof)
	tamperedProof.ProofEvents = proof.SetArtifactDigest(tamperedProof.ProofEvents, proof.ArtifactKindAttestation, tamperedProof.AttestationDigest)
	if err := VerifyRecord(tamperedProof); err == nil || !strings.Contains(err.Error(), "parent_artifact_digest mismatch") {
		t.Fatalf("expected parent artifact digest mismatch, got %v", err)
	}
}

func TestVerifyRecordAgainstBundle(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")
	imported, err := gitopsflow.Import(repo, repo, "HEAD", "platform", "")
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	bundle := publish.BuildBundleAt(imported, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC))
	rec, err := BuildAt(bundle, time.Date(2026, 3, 6, 1, 0, 0, 0, time.UTC), "ci-bot")
	if err != nil {
		t.Fatalf("BuildAt returned error: %v", err)
	}

	if err := VerifyRecordAgainstBundle(rec, bundle); err != nil {
		t.Fatalf("VerifyRecordAgainstBundle returned error for valid link: %v", err)
	}

	// Use a different generated_at to produce a different valid bundle digest.
	mismatched := publish.BuildBundleAt(imported, time.Date(2026, 3, 6, 2, 0, 0, 0, time.UTC))
	if err := VerifyRecordAgainstBundle(rec, mismatched); err == nil || !strings.Contains(err.Error(), "bundle digest link mismatch") {
		t.Fatalf("expected bundle digest link mismatch, got %v", err)
	}
}
