package publish

import (
	"sort"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
)

const (
	changeBundleSchema = "cub.confighub.io/change-bundle/v1"
	changeBundleSource = "cub-gen"
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

// ChangeBundle is a local bridge artifact that can be uploaded to ConfigHub
// later without changing cub-gen's local-first behavior.
type ChangeBundle struct {
	SchemaVersion      string                          `json:"schema_version"`
	Source             string                          `json:"source"`
	GeneratedAt        string                          `json:"generated_at"`
	Space              string                          `json:"space"`
	TargetSlug         string                          `json:"target_slug"`
	TargetPath         string                          `json:"target_path"`
	RenderTargetSlug   string                          `json:"render_target_slug"`
	Ref                string                          `json:"ref"`
	ChangeID           string                          `json:"change_id,omitempty"`
	Summary            Summary                         `json:"summary"`
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

	return ChangeBundle{
		SchemaVersion:    changeBundleSchema,
		Source:           changeBundleSource,
		GeneratedAt:      at.UTC().Format(time.RFC3339),
		Space:            imported.Space,
		TargetSlug:       imported.TargetSlug,
		TargetPath:       imported.TargetPath,
		RenderTargetSlug: imported.RenderTargetSlug,
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
