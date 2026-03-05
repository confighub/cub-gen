package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/detect"
	"github.com/confighub/cub-gen/internal/model"
)

const (
	generatorContractSchema = "cub.confighub.io/generator-contract/v1"
	provenanceSchema        = "cub.confighub.io/provenance/v1"
	inversePlanSchema       = "cub.confighub.io/inverse-transform-plan/v1"
)

// ImportRepo detects generators in a repository and produces the initial
// ConfigHub-oriented import artifacts (units, links, contracts, provenance, inverse plans).
func ImportRepo(repoPath, ref, space string) (model.ImportResult, error) {
	detection, err := detect.ScanRepo(repoPath, ref)
	if err != nil {
		return model.ImportResult{}, err
	}

	return ImportDetection(detection, space)
}

// ImportDetection builds import artifacts from a precomputed detection result.
// This allows a discover -> import flow that matches cub gitops command stages.
func ImportDetection(detection model.DetectionResult, space string) (model.ImportResult, error) {
	if detection.Repo == "" {
		return model.ImportResult{}, errors.New("detection repo is required")
	}
	importedAt := time.Now().UTC().Format(time.RFC3339)
	if space == "" {
		space = "default"
	}
	changeID := stableChangeID(detection, space)

	units := make([]model.UnitRef, 0, len(detection.Generators)*3)
	links := make([]model.UnitLink, 0, len(detection.Generators))
	contracts := make([]model.GeneratorContract, 0, len(detection.Generators))
	provenance := make([]model.ProvenanceRecord, 0, len(detection.Generators))
	inversePlans := make([]model.InverseTransformPlan, 0, len(detection.Generators))

	for _, g := range detection.Generators {
		dryUnitID := "dry_" + shortID(g.ID+":dry")
		wetUnitID := "wet_" + shortID(g.ID+":wet")
		generatorUnitID := "gen_" + shortID(g.ID+":generator")

		units = append(units,
			model.UnitRef{ID: dryUnitID, Kind: "dry-unit", Name: fmt.Sprintf("%s-dry", g.Name), Layer: "dry"},
			model.UnitRef{ID: wetUnitID, Kind: "wet-unit", Name: fmt.Sprintf("%s-wet", g.Name), Layer: "wet"},
			model.UnitRef{ID: generatorUnitID, Kind: "generator-unit", Name: fmt.Sprintf("%s-generator", g.Name), Layer: "generator"},
		)
		links = append(links, model.UnitLink{
			DryUnitID:       dryUnitID,
			WetUnitID:       wetUnitID,
			GeneratorUnitID: generatorUnitID,
		})

		contract := buildContract(detection, g)
		contracts = append(contracts, contract)
		provenance = append(provenance, buildProvenance(changeID, space, detection, g, importedAt))
		inversePlans = append(inversePlans, buildInversePlan(changeID, dryUnitID, g, importedAt))
	}

	return model.ImportResult{
		Repo:               detection.Repo,
		Ref:                detection.Ref,
		Space:              space,
		ChangeID:           changeID,
		ImportedAt:         importedAt,
		Detection:          detection,
		Units:              units,
		Links:              links,
		GeneratorContracts: contracts,
		Provenance:         provenance,
		InversePlans:       inversePlans,
	}, nil
}

func buildContract(detection model.DetectionResult, g model.GeneratorDetection) model.GeneratorContract {
	inputs := make([]model.GeneratorInput, 0, len(g.Inputs))
	for i, in := range g.Inputs {
		inputs = append(inputs, model.GeneratorInput{
			Name:      fmt.Sprintf("input_%02d", i+1),
			SchemaRef: inferInputSchema(g.Kind, in),
			Required:  true,
		})
	}

	return model.GeneratorContract{
		SchemaVersion: generatorContractSchema,
		GeneratorID:   g.ID,
		Name:          g.Name,
		Kind:          string(g.Kind),
		Version:       "0.1.0",
		SourceRepo:    detection.Repo,
		SourceRef:     detection.Ref,
		SourcePath:    g.Root,
		Inputs:        inputs,
		OutputFormat:  "kubernetes/yaml",
		Transport:     "oci+git",
		Capabilities:  capabilitiesForKind(g.Kind),
		Deterministic: true,
	}
}

func buildProvenance(changeID, space string, detection model.DetectionResult, g model.GeneratorDetection, renderedAt string) model.ProvenanceRecord {
	sources := make([]model.SourceRef, 0, len(g.Inputs))
	for _, in := range g.Inputs {
		sources = append(sources, model.SourceRef{
			Role:     "generator-input",
			URI:      fmt.Sprintf("git+file://%s#%s:%s", filepath.ToSlash(detection.Repo), detection.Ref, in),
			Revision: detection.Ref,
			Path:     in,
		})
	}

	outputURI := fmt.Sprintf("oci://example.local/%s/%s:latest", space, sanitizeName(g.Name))
	outputDigest := digestFor(strings.Join([]string{changeID, g.ID, outputURI}, "|"))
	inputDigest := digestFor(strings.Join(g.Inputs, "|"))

	return model.ProvenanceRecord{
		SchemaVersion: provenanceSchema,
		ProvenanceID:  "prov_" + shortID(changeID+":"+g.ID),
		ChangeID:      changeID,
		GeneratorID:   g.ID,
		GeneratorName: g.Name,
		Version:       "0.1.0",
		InputDigest:   inputDigest,
		Sources:       sources,
		Outputs: []model.OutputRef{{
			Role:   "rendered-manifests",
			URI:    outputURI,
			Digest: outputDigest,
		}},
		RenderedAt: renderedAt,
	}
}

func buildInversePlan(changeID, targetUnitID string, g model.GeneratorDetection, createdAt string) model.InverseTransformPlan {
	return model.InverseTransformPlan{
		SchemaVersion: inversePlanSchema,
		PlanID:        "inv_" + shortID(changeID+":"+g.ID),
		ChangeID:      changeID,
		SourceKind:    string(g.Kind),
		SourceRef:     g.Root,
		TargetUnitID:  targetUnitID,
		Status:        "draft",
		Patches:       defaultPatchesForKind(g.Kind),
		CreatedAt:     createdAt,
	}
}

func stableChangeID(detection model.DetectionResult, space string) string {
	parts := make([]string, 0, len(detection.Generators)+3)
	parts = append(parts, "v1")
	parts = append(parts, strings.TrimSpace(strings.ToLower(space)))
	parts = append(parts, strings.TrimSpace(detection.Ref))

	entries := make([]string, 0, len(detection.Generators))
	for _, g := range detection.Generators {
		entries = append(entries, strings.Join([]string{
			string(g.Kind),
			g.ID,
			g.Name,
			g.Root,
			strings.Join(g.Inputs, ","),
		}, ":"))
	}
	sort.Strings(entries)
	parts = append(parts, entries...)

	return "chg_" + shortID(strings.Join(parts, "|"))
}

func capabilitiesForKind(kind model.GeneratorKind) []string {
	switch kind {
	case model.GeneratorHelm:
		return []string{"render-manifests", "values-overrides", "inverse-values-patch"}
	case model.GeneratorScore:
		return []string{"render-manifests", "workload-spec", "inverse-score-patch"}
	case model.GeneratorSpringBoot:
		return []string{"render-app-config", "profile-overrides", "inverse-app-config-patch"}
	default:
		return []string{"render-manifests"}
	}
}

func defaultPatchesForKind(kind model.GeneratorKind) []model.InversePatch {
	switch kind {
	case model.GeneratorHelm:
		return []model.InversePatch{{
			Operation:      "replace",
			DryPath:        "values.image.tag",
			WetPath:        "Deployment/spec/template/spec/containers[0]/image",
			EditableBy:     "app-team",
			Confidence:     0.86,
			RequiresReview: false,
			Reason:         "Container image tag maps cleanly to helm values.",
		}}
	case model.GeneratorScore:
		return []model.InversePatch{{
			Operation:      "replace",
			DryPath:        "containers.main.variables.LOG_LEVEL",
			WetPath:        "Deployment/spec/template/spec/containers[0]/env[name=LOG_LEVEL]/value",
			EditableBy:     "app-team",
			Confidence:     0.90,
			RequiresReview: false,
			Reason:         "Score variable maps to a single Kubernetes env var.",
		}}
	case model.GeneratorSpringBoot:
		return []model.InversePatch{{
			Operation:      "replace",
			DryPath:        "spring.datasource.url",
			WetPath:        "ConfigMap/data/application.yaml:spring.datasource.url",
			EditableBy:     "platform-engineer",
			Confidence:     0.78,
			RequiresReview: true,
			Reason:         "Database connectivity impacts shared runtime dependencies.",
		}}
	default:
		return []model.InversePatch{}
	}
}

func inferInputSchema(kind model.GeneratorKind, inputPath string) string {
	ext := strings.ToLower(filepath.Ext(inputPath))
	switch {
	case ext == ".yaml" || ext == ".yml":
		if kind == model.GeneratorHelm && strings.Contains(strings.ToLower(filepath.Base(inputPath)), "chart") {
			return "https://json.schemastore.org/chart"
		}
		if kind == model.GeneratorScore {
			return "https://docs.score.dev/schemas/score-v1b1.json"
		}
		if kind == model.GeneratorSpringBoot {
			return "https://json.schemastore.org/spring-configuration-metadata"
		}
		return "https://json-schema.org/draft/2020-12/schema"
	case ext == ".xml":
		return "https://maven.apache.org/xsd/maven-4.0.0.xsd"
	default:
		return "https://json-schema.org/draft/2020-12/schema"
	}
}

func sanitizeName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, " ", "-")
	n = strings.ReplaceAll(n, "_", "-")
	if n == "" {
		return "generator"
	}
	return n
}

func shortID(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

func digestFor(s string) string {
	h := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(h[:])
}
