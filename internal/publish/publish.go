package publish

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/proof"
)

const (
	changeBundleSchema = "cub.confighub.io/change-bundle/v1"
	changeBundleSource = "cub-gen"
	digestAlgorithm    = "sha256"
)

// Summary captures high-signal counts for bridge ingestion and audit logs.
type Summary struct {
	DiscoveredResources int      `json:"discovered_resources"`
	DryUnits            int      `json:"dry_units"`
	WetUnits            int      `json:"wet_units"`
	GeneratorUnits      int      `json:"generator_units"`
	Links               int      `json:"links"`
	Contracts           int      `json:"contracts"`
	ProvenanceRecords   int      `json:"provenance_records"`
	InversePlans        int      `json:"inverse_transform_plans"`
	DryInputs           int      `json:"dry_inputs"`
	WetManifestTargets  int      `json:"wet_manifest_targets"`
	GeneratorProfiles   []string `json:"generator_profiles"`
}

// Counts returns log-friendly summary counts for proof events.
func (s Summary) Counts() map[string]int {
	return map[string]int{
		"discovered_resources":    s.DiscoveredResources,
		"dry_units":               s.DryUnits,
		"wet_units":               s.WetUnits,
		"generator_units":         s.GeneratorUnits,
		"links":                   s.Links,
		"contracts":               s.Contracts,
		"provenance_records":      s.ProvenanceRecords,
		"inverse_transform_plans": s.InversePlans,
		"dry_inputs":              s.DryInputs,
		"wet_manifest_targets":    s.WetManifestTargets,
	}
}

// ChangeBundle is a local bridge artifact that can be uploaded to ConfigHub
// later without changing cub-gen's local-first behavior.
type ChangeBundle struct {
	SchemaVersion      string                          `json:"schema_version"`
	Source             string                          `json:"source"`
	GeneratedAt        string                          `json:"generated_at"`
	DigestAlgorithm    string                          `json:"digest_algorithm"`
	BundleDigest       string                          `json:"bundle_digest"`
	Space              string                          `json:"space"`
	TargetSlug         string                          `json:"target_slug"`
	TargetPath         string                          `json:"target_path"`
	RenderTargetSlug   string                          `json:"render_target_slug"`
	RenderTargetPath   string                          `json:"render_target_path,omitempty"`
	Ref                string                          `json:"ref"`
	ChangeID           string                          `json:"change_id,omitempty"`
	TraceID            string                          `json:"trace_id"`
	Summary            Summary                         `json:"summary"`
	ProofEvents        []proof.Event                   `json:"proof_events"`
	Discovered         []gitopsflow.DiscoveredResource `json:"discovered"`
	DryUnits           []model.UnitRef                 `json:"dry_units"`
	WetUnits           []model.UnitRef                 `json:"wet_units"`
	GeneratorUnits     []model.UnitRef                 `json:"generator_units"`
	Links              []model.UnitLink                `json:"links"`
	Contracts          []model.GeneratorContract       `json:"contracts"`
	Provenance         []model.ProvenanceRecord        `json:"provenance"`
	InversePlans       []model.InverseTransformPlan    `json:"inverse_transform_plans"`
	DryInputs          []model.DryInputRef             `json:"dry_inputs"`
	WetManifestTargets []model.WetManifestTarget       `json:"wet_manifest_targets"`
}

// BuildBundleAt builds a deterministic bridge artifact from import output.
func BuildBundleAt(imported gitopsflow.ImportFlowResult, at time.Time) ChangeBundle {
	profilesSet := map[string]struct{}{}
	for _, item := range imported.Discovered {
		if item.GeneratorProfile != "" {
			profilesSet[item.GeneratorProfile] = struct{}{}
		}
	}
	profiles := make([]string, 0, len(profilesSet))
	for p := range profilesSet {
		profiles = append(profiles, p)
	}
	sort.Strings(profiles)

	bundle := ChangeBundle{
		SchemaVersion:    changeBundleSchema,
		Source:           changeBundleSource,
		GeneratedAt:      at.UTC().Format(time.RFC3339),
		DigestAlgorithm:  digestAlgorithm,
		Space:            imported.Space,
		TargetSlug:       imported.TargetSlug,
		TargetPath:       imported.TargetPath,
		RenderTargetSlug: imported.RenderTargetSlug,
		RenderTargetPath: imported.RenderTargetPath,
		Ref:              imported.Ref,
		ChangeID:         extractChangeID(imported),
		Summary: Summary{
			DiscoveredResources: len(imported.Discovered),
			DryUnits:            len(imported.DryUnits),
			WetUnits:            len(imported.WetUnits),
			GeneratorUnits:      len(imported.GeneratorUnits),
			Links:               len(imported.Links),
			Contracts:           len(imported.Contracts),
			ProvenanceRecords:   len(imported.Provenance),
			InversePlans:        len(imported.InversePlans),
			DryInputs:           len(imported.DryInputs),
			WetManifestTargets:  len(imported.WetManifestTargets),
			GeneratorProfiles:   profiles,
		},
		Discovered:         imported.Discovered,
		DryUnits:           imported.DryUnits,
		WetUnits:           imported.WetUnits,
		GeneratorUnits:     imported.GeneratorUnits,
		Links:              imported.Links,
		Contracts:          imported.Contracts,
		Provenance:         imported.Provenance,
		InversePlans:       imported.InversePlans,
		DryInputs:          imported.DryInputs,
		WetManifestTargets: imported.WetManifestTargets,
	}
	bundle.TraceID = proof.TraceID(bundle.ChangeID, bundle.Space, bundle.TargetSlug, bundle.RenderTargetSlug, bundle.Ref)
	bundle.ProofEvents = []proof.Event{proof.NewEvent(proof.Input{
		EventType:         proof.EventTypeChangeBundlePublished,
		EventTime:         at,
		Source:            changeBundleSource,
		ChangeID:          bundle.ChangeID,
		Space:             bundle.Space,
		TargetSlug:        bundle.TargetSlug,
		TargetPath:        bundle.TargetPath,
		RenderTargetSlug:  bundle.RenderTargetSlug,
		RenderTargetPath:  bundle.RenderTargetPath,
		Ref:               bundle.Ref,
		ArtifactKind:      proof.ArtifactKindChangeBundle,
		SummaryCounts:     bundle.Summary.Counts(),
		GeneratorProfiles: bundle.Summary.GeneratorProfiles,
	})}
	bundle.BundleDigest = computeBundleDigest(bundle)
	bundle.ProofEvents = proof.SetArtifactDigest(bundle.ProofEvents, proof.ArtifactKindChangeBundle, bundle.BundleDigest)
	return bundle
}

// BuildBundle uses current UTC time for bundle generation.
func BuildBundle(imported gitopsflow.ImportFlowResult) ChangeBundle {
	return BuildBundleAt(imported, time.Now().UTC())
}

func extractChangeID(imported gitopsflow.ImportFlowResult) string {
	for _, p := range imported.Provenance {
		if p.ChangeID != "" {
			return p.ChangeID
		}
	}
	for _, p := range imported.InversePlans {
		if p.ChangeID != "" {
			return p.ChangeID
		}
	}
	return ""
}

func computeBundleDigest(bundle ChangeBundle) string {
	// Digest excludes digest fields so re-hashing published output is stable.
	digestInput := bundle
	digestInput.BundleDigest = ""
	digestInput.DigestAlgorithm = ""
	digestInput.ProofEvents = proof.BlankArtifactDigests(digestInput.ProofEvents)

	b, err := json.Marshal(digestInput)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return digestAlgorithm + ":" + hex.EncodeToString(sum[:])
}

// VerifyBundle validates bundle schema and digest integrity.
func VerifyBundle(bundle ChangeBundle) error {
	if bundle.SchemaVersion != changeBundleSchema {
		return fmt.Errorf("unsupported schema_version %q", bundle.SchemaVersion)
	}
	if bundle.DigestAlgorithm == "" {
		return fmt.Errorf("missing digest_algorithm")
	}
	if bundle.DigestAlgorithm != digestAlgorithm {
		return fmt.Errorf("unsupported digest_algorithm %q", bundle.DigestAlgorithm)
	}
	if bundle.BundleDigest == "" {
		return fmt.Errorf("missing bundle_digest")
	}
	expected := computeBundleDigest(bundle)
	if bundle.BundleDigest != expected {
		return fmt.Errorf("bundle digest mismatch: expected %s, got %s", expected, bundle.BundleDigest)
	}
	if err := proof.ValidateArtifactEvents(bundle.ProofEvents, proof.Expected{
		EventType:        proof.EventTypeChangeBundlePublished,
		EventTime:        bundle.GeneratedAt,
		Source:           bundle.Source,
		TraceID:          bundle.TraceID,
		ChangeID:         bundle.ChangeID,
		Space:            bundle.Space,
		TargetSlug:       bundle.TargetSlug,
		TargetPath:       bundle.TargetPath,
		RenderTargetSlug: bundle.RenderTargetSlug,
		RenderTargetPath: bundle.RenderTargetPath,
		ArtifactKind:     proof.ArtifactKindChangeBundle,
		ArtifactDigest:   bundle.BundleDigest,
	}); err != nil {
		return err
	}
	return nil
}
