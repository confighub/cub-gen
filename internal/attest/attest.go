package attest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
