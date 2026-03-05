package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
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
	dryInputs := make([]model.DryInputRef, 0, len(detection.Generators)*3)
	wetTargets := make([]model.WetManifestTarget, 0, len(detection.Generators)*3)

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
		inversePlans = append(inversePlans, buildInversePlan(changeID, dryUnitID, detection, g, importedAt))
		dryInputs = append(dryInputs, dryInputsForGenerator(g)...)
		wetTargets = append(wetTargets, wetManifestTargetsForGenerator(detection, g)...)
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
		DryInputs:          dryInputs,
		WetManifestTargets: wetTargets,
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
		Profile:       g.Profile,
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
		SchemaVersion:    provenanceSchema,
		ProvenanceID:     "prov_" + shortID(changeID+":"+g.ID),
		ChangeID:         changeID,
		GeneratorID:      g.ID,
		GeneratorName:    g.Name,
		GeneratorProfile: g.Profile,
		Version:          "0.1.0",
		InputDigest:      inputDigest,
		Sources:          sources,
		Outputs: []model.OutputRef{{
			Role:   "rendered-manifests",
			URI:    outputURI,
			Digest: outputDigest,
		}},
		ChartPath:           chartPathForGenerator(g),
		ValuesPaths:         valuesPathsForGenerator(g),
		RenderedLineage:     renderedLineageForGenerator(detection, g),
		FieldOriginMap:      fieldOriginsForGenerator(detection, g),
		InverseEditPointers: inversePointersForGenerator(detection, g),
		RenderedAt:          renderedAt,
	}
}

func buildInversePlan(changeID, targetUnitID string, detection model.DetectionResult, g model.GeneratorDetection, createdAt string) model.InverseTransformPlan {
	return model.InverseTransformPlan{
		SchemaVersion: inversePlanSchema,
		PlanID:        "inv_" + shortID(changeID+":"+g.ID),
		ChangeID:      changeID,
		SourceKind:    string(g.Kind),
		SourceRef:     g.Root,
		TargetUnitID:  targetUnitID,
		Status:        "draft",
		Patches:       defaultPatchesForGenerator(detection, g),
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

func defaultPatchesForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.InversePatch {
	switch g.Kind {
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
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.InversePatch{{
			Operation:      "replace",
			DryPath:        fmt.Sprintf("containers.%s.variables.%s", hints.ContainerName, hints.VariableName),
			WetPath:        fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/env[name=%s]/value", hints.ContainerName, hints.VariableName),
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

func fieldOriginsForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.FieldOrigin {
	switch g.Kind {
	case model.GeneratorHelm:
		return []model.FieldOrigin{
			{
				DryPath:    "values.image.tag",
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				SourcePath: "values.yaml",
				Transform:  "helm-template",
				Confidence: 0.86,
			},
		}
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.FieldOrigin{
			{
				DryPath:    fmt.Sprintf("containers.%s.image", hints.ContainerName),
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/image", hints.ContainerName),
				SourcePath: hints.SourcePath,
				Transform:  "score-to-k8s",
				Confidence: 0.94,
			},
			{
				DryPath:    fmt.Sprintf("containers.%s.variables.%s", hints.ContainerName, hints.VariableName),
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/env[name=%s]/value", hints.ContainerName, hints.VariableName),
				SourcePath: hints.SourcePath,
				Transform:  "score-to-k8s",
				Confidence: 0.90,
			},
			{
				DryPath:    fmt.Sprintf("service.ports.%s.port", hints.ServicePortName),
				WetPath:    fmt.Sprintf("Service/spec/ports[name=%s]/port", hints.ServicePortName),
				SourcePath: hints.SourcePath,
				Transform:  "score-to-k8s",
				Confidence: 0.91,
			},
		}
	case model.GeneratorSpringBoot:
		return []model.FieldOrigin{
			{
				DryPath:    "spring.datasource.url",
				WetPath:    "ConfigMap/data/application.yaml:spring.datasource.url",
				SourcePath: "src/main/resources/application.yaml",
				Transform:  "spring-config-to-manifest",
				Confidence: 0.78,
			},
		}
	default:
		return []model.FieldOrigin{}
	}
}

func inversePointersForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.InverseEditPointer {
	switch g.Kind {
	case model.GeneratorHelm:
		return []model.InverseEditPointer{
			{
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				DryPath:    "values.image.tag",
				Owner:      "app-team",
				EditHint:   "Edit chart values file and keep chart template unchanged.",
				Confidence: 0.86,
			},
		}
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.InverseEditPointer{
			{
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/image", hints.ContainerName),
				DryPath:    fmt.Sprintf("containers.%s.image", hints.ContainerName),
				Owner:      "app-team",
				EditHint:   fmt.Sprintf("Edit the Score container image in %s.", hints.SourcePath),
				Confidence: 0.94,
			},
			{
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/env[name=%s]/value", hints.ContainerName, hints.VariableName),
				DryPath:    fmt.Sprintf("containers.%s.variables.%s", hints.ContainerName, hints.VariableName),
				Owner:      "app-team",
				EditHint:   fmt.Sprintf("Edit %s under containers.%s.variables in %s.", hints.VariableName, hints.ContainerName, hints.SourcePath),
				Confidence: 0.90,
			},
			{
				WetPath:    fmt.Sprintf("Service/spec/ports[name=%s]/port", hints.ServicePortName),
				DryPath:    fmt.Sprintf("service.ports.%s.port", hints.ServicePortName),
				Owner:      "app-team",
				EditHint:   fmt.Sprintf("Edit %s service port in %s.", hints.ServicePortName, hints.SourcePath),
				Confidence: 0.91,
			},
		}
	case model.GeneratorSpringBoot:
		return []model.InverseEditPointer{
			{
				WetPath:    "ConfigMap/data/application.yaml:spring.datasource.url",
				DryPath:    "spring.datasource.url",
				Owner:      "platform-engineer",
				EditHint:   "Edit datasource URL in application.yaml profile hierarchy.",
				Confidence: 0.78,
			},
		}
	default:
		return []model.InverseEditPointer{}
	}
}

type scoreHints struct {
	SourcePath      string
	ContainerName   string
	VariableName    string
	ServicePortName string
}

func scorePathHintsFromInputs(repo string, inputs []string) scoreHints {
	h := scoreHints{
		SourcePath:      "score.yaml",
		ContainerName:   "main",
		VariableName:    "LOG_LEVEL",
		ServicePortName: "web",
	}

	scorePath := firstScoreInputPath(inputs)
	if scorePath == "" {
		return h
	}
	h.SourcePath = filepath.ToSlash(scorePath)

	content, err := os.ReadFile(filepath.Join(repo, scorePath))
	if err != nil {
		return h
	}

	lines := strings.Split(string(content), "\n")
	inContainers := false
	inVariables := false
	inService := false
	inPorts := false
	currentContainer := ""

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if indent == 0 {
			inContainers = trimmed == "containers:"
			inService = trimmed == "service:"
			inVariables = false
			inPorts = false
			currentContainer = ""
			continue
		}

		if inContainers {
			if indent == 2 && strings.HasSuffix(trimmed, ":") {
				currentContainer = strings.TrimSuffix(trimmed, ":")
				if currentContainer != "" {
					h.ContainerName = currentContainer
				}
				inVariables = false
				continue
			}
			if indent == 4 && trimmed == "variables:" && currentContainer == h.ContainerName {
				inVariables = true
				continue
			}
			if inVariables && indent == 6 && strings.Contains(trimmed, ":") {
				name := strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[0])
				if name != "" {
					if strings.EqualFold(name, "LOG_LEVEL") || h.VariableName == "LOG_LEVEL" {
						h.VariableName = name
					}
				}
				continue
			}
		}

		if inService {
			if indent == 2 && trimmed == "ports:" {
				inPorts = true
				continue
			}
			if inPorts && indent == 4 && strings.HasSuffix(trimmed, ":") {
				name := strings.TrimSuffix(trimmed, ":")
				if name != "" {
					h.ServicePortName = name
				}
				continue
			}
		}
	}

	return h
}

func firstScoreInputPath(inputs []string) string {
	for _, in := range inputs {
		base := strings.ToLower(filepath.Base(in))
		if base == "score.yaml" || base == "score.yml" {
			return in
		}
	}
	return ""
}

func dryInputsForGenerator(g model.GeneratorDetection) []model.DryInputRef {
	out := make([]model.DryInputRef, 0, len(g.Inputs))
	for _, in := range g.Inputs {
		out = append(out, model.DryInputRef{
			GeneratorID: g.ID,
			Profile:     g.Profile,
			Role:        classifyDryInputRole(g.Kind, in),
			Path:        in,
			Required:    true,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func wetManifestTargetsForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.WetManifestTarget {
	switch g.Kind {
	case model.GeneratorHelm:
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "HelmRelease", Name: g.Name, Namespace: "apps"},
			{GeneratorID: g.ID, Kind: "Deployment", Name: g.Name, Namespace: "apps", SourceDryPath: "values.image.tag"},
			{GeneratorID: g.ID, Kind: "Service", Name: g.Name, Namespace: "apps", SourceDryPath: "values.service.port"},
		}
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "Application", Name: g.Name, Namespace: "apps"},
			{GeneratorID: g.ID, Kind: "Deployment", Name: g.Name, Namespace: "apps", SourceDryPath: fmt.Sprintf("containers.%s.image", hints.ContainerName)},
			{GeneratorID: g.ID, Kind: "Service", Name: g.Name, Namespace: "apps", SourceDryPath: fmt.Sprintf("service.ports.%s.port", hints.ServicePortName)},
		}
	case model.GeneratorSpringBoot:
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "Kustomization", Name: g.Name, Namespace: "apps"},
			{GeneratorID: g.ID, Kind: "ConfigMap", Name: g.Name + "-config", Namespace: "apps", SourceDryPath: "spring.datasource.url"},
		}
	default:
		return []model.WetManifestTarget{}
	}
}

func classifyDryInputRole(kind model.GeneratorKind, in string) string {
	base := strings.ToLower(filepath.Base(in))
	switch kind {
	case model.GeneratorHelm:
		switch {
		case base == "chart.yaml":
			return "chart"
		case strings.HasPrefix(base, "values") && (strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")):
			return "values"
		default:
			return "helm-input"
		}
	case model.GeneratorScore:
		if base == "score.yaml" || base == "score.yml" {
			return "score-spec"
		}
		return "score-input"
	case model.GeneratorSpringBoot:
		if base == "pom.xml" || base == "build.gradle" || base == "build.gradle.kts" {
			return "build-config"
		}
		if strings.HasPrefix(base, "application") && (strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")) {
			return "app-config"
		}
		return "spring-input"
	default:
		return "input"
	}
}

func chartPathForGenerator(g model.GeneratorDetection) string {
	if g.Kind != model.GeneratorHelm {
		return ""
	}
	for _, in := range g.Inputs {
		if strings.EqualFold(filepath.Base(in), "chart.yaml") {
			return in
		}
	}
	return ""
}

func valuesPathsForGenerator(g model.GeneratorDetection) []string {
	if g.Kind != model.GeneratorHelm {
		return nil
	}
	out := make([]string, 0, len(g.Inputs))
	for _, in := range g.Inputs {
		base := strings.ToLower(filepath.Base(in))
		if strings.HasPrefix(base, "values") && (strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")) {
			out = append(out, in)
		}
	}
	sort.Strings(out)
	return out
}

func renderedLineageForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.RenderedObjectLineage {
	switch g.Kind {
	case model.GeneratorHelm:
		values := valuesPathsForGenerator(g)
		sourceValues := "values.yaml"
		if len(values) > 0 {
			sourceValues = values[0]
		}
		return []model.RenderedObjectLineage{
			{Kind: "HelmRelease", Name: g.Name, Namespace: "apps", SourcePath: chartPathForGenerator(g), SourceDryPath: "Chart.yaml"},
			{Kind: "Deployment", Name: g.Name, Namespace: "apps", SourcePath: sourceValues, SourceDryPath: "values.image.tag"},
			{Kind: "Service", Name: g.Name, Namespace: "apps", SourcePath: sourceValues, SourceDryPath: "values.service.port"},
		}
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.RenderedObjectLineage{
			{Kind: "Application", Name: g.Name, Namespace: "apps", SourcePath: hints.SourcePath, SourceDryPath: "metadata.name"},
			{Kind: "Deployment", Name: g.Name, Namespace: "apps", SourcePath: hints.SourcePath, SourceDryPath: fmt.Sprintf("containers.%s.image", hints.ContainerName)},
			{Kind: "Service", Name: g.Name, Namespace: "apps", SourcePath: hints.SourcePath, SourceDryPath: fmt.Sprintf("service.ports.%s.port", hints.ServicePortName)},
		}
	case model.GeneratorSpringBoot:
		return []model.RenderedObjectLineage{
			{Kind: "Kustomization", Name: g.Name, Namespace: "apps", SourcePath: "kustomization.yaml", SourceDryPath: "resources"},
			{Kind: "ConfigMap", Name: g.Name + "-config", Namespace: "apps", SourcePath: "src/main/resources/application.yaml", SourceDryPath: "spring.datasource.url"},
		}
	default:
		return nil
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
