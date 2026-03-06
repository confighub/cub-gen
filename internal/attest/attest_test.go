package attest

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
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
