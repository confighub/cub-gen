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
	"github.com/confighub/cub-gen/internal/registry"
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
	return registry.Capabilities(kind)
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
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "spring.application.name",
				WetPath:        "Deployment/metadata/labels[app.kubernetes.io/name]",
				EditableBy:     "app-team",
				Confidence:     0.88,
				RequiresReview: false,
				Reason:         "Application identity should be app-editable without platform escalation.",
			},
			{
				Operation:      "replace",
				DryPath:        "server.port",
				WetPath:        "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				EditableBy:     "app-team",
				Confidence:     0.91,
				RequiresReview: false,
				Reason:         "Application listener port is an app-level configuration concern.",
			},
			{
				Operation:      "replace",
				DryPath:        "spring.datasource.url",
				WetPath:        "ConfigMap/data/application.yaml:spring.datasource.url",
				EditableBy:     "platform-engineer",
				Confidence:     0.78,
				RequiresReview: true,
				Reason:         "Database connectivity impacts shared runtime dependencies.",
			},
		}
	case model.GeneratorBackstage:
		hints := backstagePathHintsFromInputs(g.Inputs)
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "metadata.name",
				WetPath:        "Application/metadata/name",
				EditableBy:     "platform-engineer",
				Confidence:     0.87,
				RequiresReview: false,
				Reason:         fmt.Sprintf("Backstage component identity is sourced from %s.", hints.CatalogPath),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.lifecycle",
				WetPath:        "Application/metadata/labels[lifecycle]",
				EditableBy:     "platform-engineer",
				Confidence:     0.82,
				RequiresReview: true,
				Reason:         "Lifecycle changes impact platform ownership and support policy.",
			},
		}
	case model.GeneratorAbly:
		hints := ablyPathHintsFromInputs(g.Inputs)
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "app.environment",
				WetPath:        "ConfigMap/data/ABLY_ENVIRONMENT",
				EditableBy:     "app-team",
				Confidence:     0.90,
				RequiresReview: false,
				Reason:         fmt.Sprintf("Environment is sourced from %s.", hints.BaseConfigPath),
			},
			{
				Operation:      "replace",
				DryPath:        "channels.inbound",
				WetPath:        "ConfigMap/data/ABLY_CHANNEL_INBOUND",
				EditableBy:     "app-team",
				Confidence:     0.88,
				RequiresReview: false,
				Reason:         "Channel mapping is app-level runtime behavior.",
			},
		}
	case model.GeneratorOpsFlow:
		hints := opsWorkflowPathHintsFromInputs(g.Inputs)
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "actions.deploy.image_tag",
				WetPath:        "Workflow/spec/templates[name=deploy]/container/image",
				EditableBy:     "platform-engineer",
				Confidence:     0.87,
				RequiresReview: true,
				Reason:         fmt.Sprintf("Deployment action image tag is sourced from %s.", hints.BaseSpecPath),
			},
			{
				Operation:      "replace",
				DryPath:        "triggers.schedule",
				WetPath:        "Workflow/spec/schedule",
				EditableBy:     "platform-engineer",
				Confidence:     0.84,
				RequiresReview: true,
				Reason:         "Schedule changes affect operational execution timing.",
			},
		}
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
		hints := springPathHintsFromInputs(g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "spring.application.name",
				WetPath:    "Deployment/metadata/labels[app.kubernetes.io/name]",
				SourcePath: hints.BaseConfigPath,
				Transform:  "spring-config-to-manifest",
				Confidence: 0.89,
			},
			{
				DryPath:    "server.port",
				WetPath:    "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				SourcePath: hints.BaseConfigPath,
				Transform:  "spring-config-to-manifest",
				Confidence: 0.92,
			},
			{
				DryPath:    "spring.datasource.url",
				WetPath:    "ConfigMap/data/application.yaml:spring.datasource.url",
				SourcePath: hints.BaseConfigPath,
				Transform:  "spring-config-to-manifest",
				Confidence: 0.78,
			},
		}
		if hints.ProfileConfigPath != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "server.port",
				WetPath:    "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				SourcePath: hints.ProfileConfigPath,
				Transform:  "spring-profile-overlay",
				Confidence: 0.88,
			})
		}
		return origins
	case model.GeneratorBackstage:
		hints := backstagePathHintsFromInputs(g.Inputs)
		return []model.FieldOrigin{
			{
				DryPath:    "metadata.name",
				WetPath:    "Application/metadata/name",
				SourcePath: hints.CatalogPath,
				Transform:  "backstage-component-to-application",
				Confidence: 0.90,
			},
			{
				DryPath:    "spec.lifecycle",
				WetPath:    "Application/metadata/labels[lifecycle]",
				SourcePath: hints.CatalogPath,
				Transform:  "backstage-component-to-application",
				Confidence: 0.82,
			},
		}
	case model.GeneratorAbly:
		hints := ablyPathHintsFromInputs(g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "app.environment",
				WetPath:    "ConfigMap/data/ABLY_ENVIRONMENT",
				SourcePath: hints.BaseConfigPath,
				Transform:  "ably-config-to-runtime",
				Confidence: 0.90,
			},
			{
				DryPath:    "channels.inbound",
				WetPath:    "ConfigMap/data/ABLY_CHANNEL_INBOUND",
				SourcePath: hints.BaseConfigPath,
				Transform:  "ably-config-to-runtime",
				Confidence: 0.88,
			},
		}
		if hints.OverlayConfigPath != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "channels.inbound",
				WetPath:    "ConfigMap/data/ABLY_CHANNEL_INBOUND",
				SourcePath: hints.OverlayConfigPath,
				Transform:  "ably-overlay-merge",
				Confidence: 0.84,
			})
		}
		return origins
	case model.GeneratorOpsFlow:
		hints := opsWorkflowPathHintsFromInputs(g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "actions.deploy.image_tag",
				WetPath:    "Workflow/spec/templates[name=deploy]/container/image",
				SourcePath: hints.BaseSpecPath,
				Transform:  "ops-workflow-to-argo-workflow",
				Confidence: 0.87,
			},
			{
				DryPath:    "triggers.schedule",
				WetPath:    "Workflow/spec/schedule",
				SourcePath: hints.BaseSpecPath,
				Transform:  "ops-workflow-to-argo-workflow",
				Confidence: 0.84,
			},
		}
		if hints.OverlaySpecPath != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "triggers.schedule",
				WetPath:    "Workflow/spec/schedule",
				SourcePath: hints.OverlaySpecPath,
				Transform:  "ops-workflow-overlay-merge",
				Confidence: 0.80,
			})
		}
		return origins
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
		hints := springPathHintsFromInputs(g.Inputs)
		return []model.InverseEditPointer{
			{
				WetPath:    "Deployment/metadata/labels[app.kubernetes.io/name]",
				DryPath:    "spring.application.name",
				Owner:      "app-team",
				EditHint:   fmt.Sprintf("Edit spring.application.name in %s.", hints.BaseConfigPath),
				Confidence: 0.89,
			},
			{
				WetPath:    "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				DryPath:    "server.port",
				Owner:      "app-team",
				EditHint:   springServerPortEditHint(hints),
				Confidence: 0.91,
			},
			{
				WetPath:    "ConfigMap/data/application.yaml:spring.datasource.url",
				DryPath:    "spring.datasource.url",
				Owner:      "platform-engineer",
				EditHint:   fmt.Sprintf("Edit spring.datasource.url in %s and coordinate with platform ownership rules.", hints.BaseConfigPath),
				Confidence: 0.78,
			},
		}
	case model.GeneratorBackstage:
		hints := backstagePathHintsFromInputs(g.Inputs)
		return []model.InverseEditPointer{
			{
				WetPath:    "Application/metadata/name",
				DryPath:    "metadata.name",
				Owner:      "platform-engineer",
				EditHint:   fmt.Sprintf("Edit metadata.name in %s.", hints.CatalogPath),
				Confidence: 0.90,
			},
			{
				WetPath:    "Application/metadata/labels[lifecycle]",
				DryPath:    "spec.lifecycle",
				Owner:      "platform-engineer",
				EditHint:   fmt.Sprintf("Edit spec.lifecycle in %s and coordinate rollout policy.", hints.CatalogPath),
				Confidence: 0.82,
			},
		}
	case model.GeneratorAbly:
		hints := ablyPathHintsFromInputs(g.Inputs)
		return []model.InverseEditPointer{
			{
				WetPath:    "ConfigMap/data/ABLY_ENVIRONMENT",
				DryPath:    "app.environment",
				Owner:      "app-team",
				EditHint:   fmt.Sprintf("Edit app.environment in %s.", hints.BaseConfigPath),
				Confidence: 0.90,
			},
			{
				WetPath:    "ConfigMap/data/ABLY_CHANNEL_INBOUND",
				DryPath:    "channels.inbound",
				Owner:      "app-team",
				EditHint:   ablyInboundChannelEditHint(hints),
				Confidence: 0.88,
			},
		}
	case model.GeneratorOpsFlow:
		hints := opsWorkflowPathHintsFromInputs(g.Inputs)
		return []model.InverseEditPointer{
			{
				WetPath:    "Workflow/spec/templates[name=deploy]/container/image",
				DryPath:    "actions.deploy.image_tag",
				Owner:      "platform-engineer",
				EditHint:   fmt.Sprintf("Edit actions.deploy.image_tag in %s.", hints.BaseSpecPath),
				Confidence: 0.87,
			},
			{
				WetPath:    "Workflow/spec/schedule",
				DryPath:    "triggers.schedule",
				Owner:      "platform-engineer",
				EditHint:   opsWorkflowScheduleEditHint(hints),
				Confidence: 0.84,
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
		role := registry.InputRole(g.Kind, in)
		out = append(out, model.DryInputRef{
			GeneratorID: g.ID,
			Profile:     g.Profile,
			Role:        role,
			Owner:       registry.OwnerForRole(g.Kind, role),
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
			{GeneratorID: g.ID, Kind: "HelmRelease", Name: g.Name, Owner: "platform-runtime", Namespace: "apps"},
			{GeneratorID: g.ID, Kind: "Deployment", Name: g.Name, Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "values.image.tag"},
			{GeneratorID: g.ID, Kind: "Service", Name: g.Name, Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "values.service.port"},
		}
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "Application", Name: g.Name, Owner: "platform-runtime", Namespace: "apps"},
			{GeneratorID: g.ID, Kind: "Deployment", Name: g.Name, Owner: "platform-runtime", Namespace: "apps", SourceDryPath: fmt.Sprintf("containers.%s.image", hints.ContainerName)},
			{GeneratorID: g.ID, Kind: "Service", Name: g.Name, Owner: "platform-runtime", Namespace: "apps", SourceDryPath: fmt.Sprintf("service.ports.%s.port", hints.ServicePortName)},
		}
	case model.GeneratorSpringBoot:
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "Kustomization", Name: g.Name, Owner: "platform-runtime", Namespace: "apps"},
			{GeneratorID: g.ID, Kind: "Deployment", Name: g.Name, Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "server.port"},
			{GeneratorID: g.ID, Kind: "ConfigMap", Name: g.Name + "-config", Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "spring.datasource.url"},
		}
	case model.GeneratorBackstage:
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "Application", Name: g.Name, Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "metadata.name"},
			{GeneratorID: g.ID, Kind: "ConfigMap", Name: g.Name + "-catalog", Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "spec.lifecycle"},
		}
	case model.GeneratorAbly:
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "ConfigMap", Name: g.Name + "-ably", Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "app.environment"},
			{GeneratorID: g.ID, Kind: "Secret", Name: g.Name + "-ably-credentials", Owner: "platform-runtime", Namespace: "apps", SourceDryPath: "credentials.api_key_ref"},
		}
	case model.GeneratorOpsFlow:
		return []model.WetManifestTarget{
			{GeneratorID: g.ID, Kind: "Workflow", Name: g.Name + "-workflow", Owner: "platform-runtime", Namespace: "ops", SourceDryPath: "actions.deploy.image_tag"},
			{GeneratorID: g.ID, Kind: "Job", Name: g.Name + "-dry-run", Owner: "platform-runtime", Namespace: "ops", SourceDryPath: "triggers.schedule"},
		}
	default:
		return []model.WetManifestTarget{}
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
		chart := chartPathForGenerator(g)
		values := valuesPathsForGenerator(g)
		if len(values) == 0 {
			// Fall back to chart path if no values files were detected.
			if chart != "" {
				values = []string{chart}
			}
		}
		lineage := []model.RenderedObjectLineage{
			{Kind: "HelmRelease", Name: g.Name, Namespace: "apps", SourcePath: chart, SourceDryPath: "Chart.yaml"},
		}
		for _, vp := range values {
			lineage = append(lineage,
				model.RenderedObjectLineage{Kind: "Deployment", Name: g.Name, Namespace: "apps", SourcePath: vp, SourceDryPath: "values.image.tag"},
				model.RenderedObjectLineage{Kind: "Service", Name: g.Name, Namespace: "apps", SourcePath: vp, SourceDryPath: "values.service.port"},
			)
		}
		return lineage
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.RenderedObjectLineage{
			{Kind: "Application", Name: g.Name, Namespace: "apps", SourcePath: hints.SourcePath, SourceDryPath: "metadata.name"},
			{Kind: "Deployment", Name: g.Name, Namespace: "apps", SourcePath: hints.SourcePath, SourceDryPath: fmt.Sprintf("containers.%s.image", hints.ContainerName)},
			{Kind: "Service", Name: g.Name, Namespace: "apps", SourcePath: hints.SourcePath, SourceDryPath: fmt.Sprintf("service.ports.%s.port", hints.ServicePortName)},
		}
	case model.GeneratorSpringBoot:
		hints := springPathHintsFromInputs(g.Inputs)
		lineage := []model.RenderedObjectLineage{
			{Kind: "Kustomization", Name: g.Name, Namespace: "apps", SourcePath: hints.BuildConfigPath, SourceDryPath: "build"},
			{Kind: "Deployment", Name: g.Name, Namespace: "apps", SourcePath: hints.BaseConfigPath, SourceDryPath: "spring.application.name"},
			{Kind: "ConfigMap", Name: g.Name + "-config", Namespace: "apps", SourcePath: hints.BaseConfigPath, SourceDryPath: "spring.datasource.url"},
		}
		if hints.ProfileConfigPath != "" {
			lineage = append(lineage, model.RenderedObjectLineage{
				Kind: "Deployment", Name: g.Name, Namespace: "apps", SourcePath: hints.ProfileConfigPath, SourceDryPath: "server.port",
			})
		} else {
			lineage = append(lineage, model.RenderedObjectLineage{
				Kind: "Deployment", Name: g.Name, Namespace: "apps", SourcePath: hints.BaseConfigPath, SourceDryPath: "server.port",
			})
		}
		return lineage
	case model.GeneratorBackstage:
		hints := backstagePathHintsFromInputs(g.Inputs)
		return []model.RenderedObjectLineage{
			{Kind: "Application", Name: g.Name, Namespace: "apps", SourcePath: hints.CatalogPath, SourceDryPath: "metadata.name"},
			{Kind: "ConfigMap", Name: g.Name + "-catalog", Namespace: "apps", SourcePath: hints.CatalogPath, SourceDryPath: "spec.lifecycle"},
		}
	case model.GeneratorAbly:
		hints := ablyPathHintsFromInputs(g.Inputs)
		lineage := []model.RenderedObjectLineage{
			{Kind: "ConfigMap", Name: g.Name + "-ably", Namespace: "apps", SourcePath: hints.BaseConfigPath, SourceDryPath: "app.environment"},
			{Kind: "Secret", Name: g.Name + "-ably-credentials", Namespace: "apps", SourcePath: hints.BaseConfigPath, SourceDryPath: "credentials.api_key_ref"},
		}
		if hints.OverlayConfigPath != "" {
			lineage = append(lineage, model.RenderedObjectLineage{
				Kind: "ConfigMap", Name: g.Name + "-ably", Namespace: "apps", SourcePath: hints.OverlayConfigPath, SourceDryPath: "channels.inbound",
			})
		}
		return lineage
	case model.GeneratorOpsFlow:
		hints := opsWorkflowPathHintsFromInputs(g.Inputs)
		lineage := []model.RenderedObjectLineage{
			{Kind: "Workflow", Name: g.Name + "-workflow", Namespace: "ops", SourcePath: hints.BaseSpecPath, SourceDryPath: "actions.deploy.image_tag"},
			{Kind: "Job", Name: g.Name + "-dry-run", Namespace: "ops", SourcePath: hints.BaseSpecPath, SourceDryPath: "triggers.schedule"},
		}
		if hints.OverlaySpecPath != "" {
			lineage = append(lineage, model.RenderedObjectLineage{
				Kind: "Workflow", Name: g.Name + "-workflow", Namespace: "ops", SourcePath: hints.OverlaySpecPath, SourceDryPath: "triggers.schedule",
			})
		}
		return lineage
	default:
		return nil
	}
}

type springHints struct {
	BuildConfigPath   string
	BaseConfigPath    string
	ProfileConfigPath string
}

func springPathHintsFromInputs(inputs []string) springHints {
	h := springHints{
		BuildConfigPath: "pom.xml",
		BaseConfigPath:  "src/main/resources/application.yaml",
	}

	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch base {
		case "pom.xml", "build.gradle", "build.gradle.kts":
			h.BuildConfigPath = p
		case "application.yaml", "application.yml":
			h.BaseConfigPath = p
		}
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		if strings.HasPrefix(base, "application-") && (strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")) {
			if h.ProfileConfigPath == "" || p < h.ProfileConfigPath {
				h.ProfileConfigPath = p
			}
		}
	}
	if h.BaseConfigPath == "" {
		if h.ProfileConfigPath != "" {
			h.BaseConfigPath = h.ProfileConfigPath
		} else {
			h.BaseConfigPath = "src/main/resources/application.yaml"
		}
	}
	return h
}

func springServerPortEditHint(h springHints) string {
	if h.ProfileConfigPath != "" {
		return fmt.Sprintf("Edit server.port in %s for environment overrides; use %s for the default.", h.ProfileConfigPath, h.BaseConfigPath)
	}
	return fmt.Sprintf("Edit server.port in %s.", h.BaseConfigPath)
}

type backstageHints struct {
	CatalogPath   string
	AppConfigPath string
}

func backstagePathHintsFromInputs(inputs []string) backstageHints {
	h := backstageHints{
		CatalogPath: "catalog-info.yaml",
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch base {
		case "catalog-info.yaml", "catalog-info.yml":
			h.CatalogPath = p
		case "app-config.yaml", "app-config.yml":
			h.AppConfigPath = p
		}
	}
	return h
}

type ablyHints struct {
	BaseConfigPath    string
	OverlayConfigPath string
}

func ablyPathHintsFromInputs(inputs []string) ablyHints {
	h := ablyHints{
		BaseConfigPath: "ably.yaml",
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch {
		case base == "ably.yaml" || base == "ably.yml" || base == "ably.json":
			h.BaseConfigPath = p
		case strings.HasPrefix(base, "ably-"):
			if h.OverlayConfigPath == "" || p < h.OverlayConfigPath {
				h.OverlayConfigPath = p
			}
		}
	}
	return h
}

func ablyInboundChannelEditHint(h ablyHints) string {
	if h.OverlayConfigPath != "" {
		return fmt.Sprintf("Edit channels.inbound in %s for environment-specific behavior; use %s for defaults.", h.OverlayConfigPath, h.BaseConfigPath)
	}
	return fmt.Sprintf("Edit channels.inbound in %s.", h.BaseConfigPath)
}

type opsWorkflowHints struct {
	BaseSpecPath    string
	OverlaySpecPath string
}

func opsWorkflowPathHintsFromInputs(inputs []string) opsWorkflowHints {
	h := opsWorkflowHints{
		BaseSpecPath: "operations.yaml",
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch {
		case base == "operations.yaml" || base == "operations.yml" || base == "workflow.yaml" || base == "workflow.yml":
			h.BaseSpecPath = p
		case strings.HasPrefix(base, "operations-") || strings.HasPrefix(base, "workflow-"):
			if h.OverlaySpecPath == "" || p < h.OverlaySpecPath {
				h.OverlaySpecPath = p
			}
		}
	}
	return h
}

func opsWorkflowScheduleEditHint(h opsWorkflowHints) string {
	if h.OverlaySpecPath != "" {
		return fmt.Sprintf("Edit triggers.schedule in %s for environment-specific cadence; use %s for defaults.", h.OverlaySpecPath, h.BaseSpecPath)
	}
	return fmt.Sprintf("Edit triggers.schedule in %s.", h.BaseSpecPath)
}

func inferInputSchema(kind model.GeneratorKind, inputPath string) string {
	return registry.SchemaRef(kind, inputPath)
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
