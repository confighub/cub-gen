package attest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/publish"
)

const (
	attestationSchema = "cub.confighub.io/attestation/v1"
	attestationSource = "cub-gen"
	digestAlgorithm   = "sha256"
)

// Record is an attestation envelope for a verified change bundle.
type Record struct {
	SchemaVersion     string `json:"schema_version"`
	Source            string `json:"source"`
	GeneratedAt       string `json:"generated_at"`
	Status            string `json:"status"`
	Verifier          string `json:"verifier"`
	DigestAlgorithm   string `json:"digest_algorithm"`
	BundleDigest      string `json:"bundle_digest"`
	ChangeID          string `json:"change_id,omitempty"`
	Space             string `json:"space,omitempty"`
	TargetSlug        string `json:"target_slug,omitempty"`
	RenderTargetSlug  string `json:"render_target_slug,omitempty"`
	AttestationDigest string `json:"attestation_digest"`
}

// BuildAt verifies a bundle and emits an attestation record.
func BuildAt(bundle publish.ChangeBundle, at time.Time, verifier string) (Record, error) {
	if err := publish.VerifyBundle(bundle); err != nil {
		return Record{}, err
	}
	if verifier == "" {
		verifier = attestationSource
	}

	rec := Record{
		SchemaVersion:    attestationSchema,
		Source:           attestationSource,
		GeneratedAt:      at.UTC().Format(time.RFC3339),
		Status:           "verified",
		Verifier:         verifier,
		DigestAlgorithm:  digestAlgorithm,
		BundleDigest:     bundle.BundleDigest,
		ChangeID:         bundle.ChangeID,
		Space:            bundle.Space,
		TargetSlug:       bundle.TargetSlug,
		RenderTargetSlug: bundle.RenderTargetSlug,
	}
	rec.AttestationDigest = computeAttestationDigest(rec)
	return rec, nil
}

// Build uses current UTC time for record generation.
func Build(bundle publish.ChangeBundle, verifier string) (Record, error) {
	return BuildAt(bundle, time.Now().UTC(), verifier)
}

func computeAttestationDigest(rec Record) string {
	input := rec
	input.AttestationDigest = ""
	input.DigestAlgorithm = ""

	b, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return digestAlgorithm + ":" + hex.EncodeToString(sum[:])
}

// VerifyRecord validates attestation schema and digest integrity.
func VerifyRecord(rec Record) error {
	if rec.SchemaVersion != attestationSchema {
		return fmt.Errorf("unsupported schema_version %q", rec.SchemaVersion)
	}
	if rec.Source != attestationSource {
		return fmt.Errorf("unsupported source %q", rec.Source)
	}
	if rec.Status != "verified" {
		return fmt.Errorf("unsupported status %q", rec.Status)
	}
	if rec.DigestAlgorithm == "" {
		return fmt.Errorf("missing digest_algorithm")
	}
	if rec.DigestAlgorithm != digestAlgorithm {
		return fmt.Errorf("unsupported digest_algorithm %q", rec.DigestAlgorithm)
	}
	if rec.BundleDigest == "" {
		return fmt.Errorf("missing bundle_digest")
	}
	if !strings.HasPrefix(rec.BundleDigest, digestAlgorithm+":") {
		return fmt.Errorf("bundle_digest must use %s prefix", digestAlgorithm)
	}
	if rec.AttestationDigest == "" {
		return fmt.Errorf("missing attestation_digest")
	}
	expected := computeAttestationDigest(rec)
	if rec.AttestationDigest != expected {
		return fmt.Errorf("attestation digest mismatch: expected %s, got %s", expected, rec.AttestationDigest)
	}
	return nil
}

// VerifyRecordAgainstBundle validates both attestation and bundle integrity and
// then checks their digest linkage.
func VerifyRecordAgainstBundle(rec Record, bundle publish.ChangeBundle) error {
	if err := VerifyRecord(rec); err != nil {
		return err
	}
	if err := publish.VerifyBundle(bundle); err != nil {
		return err
	}
	if rec.BundleDigest != bundle.BundleDigest {
		return fmt.Errorf("bundle digest link mismatch: attestation=%s bundle=%s", rec.BundleDigest, bundle.BundleDigest)
	}
	if rec.ChangeID != "" && bundle.ChangeID != "" && rec.ChangeID != bundle.ChangeID {
		return fmt.Errorf("change_id link mismatch: attestation=%s bundle=%s", rec.ChangeID, bundle.ChangeID)
	}
	return nil
}
