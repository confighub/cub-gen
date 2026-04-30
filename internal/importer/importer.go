package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/contracts"
	"github.com/confighub/cub-gen/internal/detect"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

const (
	generatorContractSchema       = "cub.confighub.io/generator-contract/v1"
	provenanceSchema              = "cub.confighub.io/provenance/v1"
	inversePlanSchema             = "cub.confighub.io/inverse-transform-plan/v1"
	helmCLIOverrideTransform      = "helm-cli-override"
	helmExternalTransform         = "helm-external"
	helmBuiltinTransform          = "helm-builtin"
	helmHelperTransform           = "helm-helper"
	helmNonDeterministicTransform = "helm-nondeterministic"
	helmDefaultTransform          = "helm-default"
	generatorChainTransform       = "generator-chain"
)

var (
	helmHelperDefineRe = regexp.MustCompile(`(?s)\{\{[- ]*define\s+"([^"]+)"[- ]*\}\}(.*?)\{\{[- ]*end[- ]*\}\}`)
	helmHelperRefRe    = regexp.MustCompile(`(?:include|template)\s+"([^"]+)"`)
)

// ImportRepo detects generators in a repository and produces the initial
// ConfigHub-oriented import artifacts (units, links, contracts, provenance, inverse plans).
func ImportRepo(repoPath, ref, space string) (model.ImportResult, error) {
	return ImportRepoWithOptions(repoPath, ref, space, ImportOptions{})
}

// ImportRepoWithOptions mirrors ImportRepo but accepts invocation-specific
// provenance hints such as Helm CLI overrides.
func ImportRepoWithOptions(repoPath, ref, space string, opts ImportOptions) (model.ImportResult, error) {
	detection, err := detect.ScanRepo(repoPath, ref)
	if err != nil {
		return model.ImportResult{}, err
	}

	return ImportDetectionWithOptions(detection, space, opts)
}

// ImportDetection builds import artifacts from a precomputed detection result.
// This allows a discover -> import flow that matches cub gitops command stages.
func ImportDetection(detection model.DetectionResult, space string) (model.ImportResult, error) {
	return ImportDetectionWithOptions(detection, space, ImportOptions{})
}

// ImportDetectionWithOptions builds import artifacts from a precomputed
// detection result and invocation-specific provenance hints.
func ImportDetectionWithOptions(detection model.DetectionResult, space string, opts ImportOptions) (model.ImportResult, error) {
	if detection.Repo == "" {
		return model.ImportResult{}, errors.New("detection repo is required")
	}
	importedAt := time.Now().UTC().Format(time.RFC3339)
	if space == "" {
		space = "default"
	}
	changeID := stableChangeID(detection, space, opts)

	units := make([]model.UnitRef, 0, len(detection.Generators)*3)
	links := make([]model.UnitLink, 0, len(detection.Generators))
	generatorContracts := make([]model.GeneratorContract, 0, len(detection.Generators))
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
		provenanceRecord := buildProvenance(changeID, space, detection, g, importedAt, opts)
		inversePlan := buildInversePlan(changeID, dryUnitID, detection, g, importedAt)
		if err := contracts.ValidateTriple(contract, provenanceRecord, inversePlan); err != nil {
			return model.ImportResult{}, fmt.Errorf("validate contract triple for generator %q (%s): %w", g.ID, g.Kind, err)
		}

		generatorContracts = append(generatorContracts, contract)
		provenance = append(provenance, provenanceRecord)
		inversePlans = append(inversePlans, inversePlan)
		dryInputs = append(dryInputs, dryInputsForGenerator(g)...)
		dryInputs = append(dryInputs, dryInputsForHelmCLIOverrides(g, opts.HelmCLIOverrides)...)
		wetTargets = append(wetTargets, wetManifestTargetsForGenerator(detection, g)...)
	}

	if err := contracts.ValidateGovernedImportTriples(len(detection.Generators), generatorContracts, provenance, inversePlans); err != nil {
		return model.ImportResult{}, err
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
		GeneratorContracts: generatorContracts,
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

func buildProvenance(changeID, space string, detection model.DetectionResult, g model.GeneratorDetection, renderedAt string, opts ImportOptions) model.ProvenanceRecord {
	sources := make([]model.SourceRef, 0, len(g.Inputs))
	for _, in := range g.Inputs {
		sources = append(sources, model.SourceRef{
			Role:     "generator-input",
			URI:      fmt.Sprintf("git+file://%s#%s:%s", filepath.ToSlash(detection.Repo), detection.Ref, in),
			Revision: detection.Ref,
			Path:     in,
		})
	}
	sources = append(sources, helmCLIOverrideSourcesForGenerator(detection.Ref, g, opts.HelmCLIOverrides)...)

	outputURI := fmt.Sprintf("oci://example.local/%s/%s:latest", space, sanitizeName(g.Name))
	outputDigest := digestFor(strings.Join([]string{changeID, g.ID, outputURI}, "|"))
	inputDigestParts := append([]string{}, g.Inputs...)
	inputDigestParts = append(inputDigestParts, HelmCLIOverrideDigestParts(helmCLIOverridesForGenerator(g, opts.HelmCLIOverrides))...)
	inputDigest := digestFor(strings.Join(inputDigestParts, "|"))
	helmPaths := helmProvenancePathsForGenerator(detection.Repo, g)

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
		ChartPath:           helmPaths.ChartPath,
		ValuesPaths:         helmPaths.ValuesPaths,
		HelmCLIOverrides:    helmCLIOverridesForGenerator(g, opts.HelmCLIOverrides),
		RenderedLineage:     renderedLineageForGenerator(detection, g),
		FieldOriginMap:      fieldOriginsForGenerator(detection, g, opts.HelmCLIOverrides),
		InverseEditPointers: inversePointersForGenerator(detection, g),
		HelmLayeredAnalysis: helmLayeredAnalysisForGenerator(detection, g),
		ApplicationSet:      applicationSetAnalysisForGenerator(detection, g),
		OpsWorkflow:         opsWorkflowAnalysisForGenerator(detection, g),
		SwampWorkflow:       swampWorkflowAnalysisForGenerator(detection, g),
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

func stableChangeID(detection model.DetectionResult, space string, opts ImportOptions) string {
	parts := make([]string, 0, len(detection.Generators)+3)
	parts = append(parts, "v1")
	parts = append(parts, strings.TrimSpace(strings.ToLower(space)))
	parts = append(parts, strings.TrimSpace(detection.Ref))
	parts = append(parts, HelmCLIOverrideDigestParts(opts.HelmCLIOverrides)...)

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
		policy := registry.InversePatchTemplateFor(g.Kind, "image_tag", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.86, RequiresReview: false,
		})
		return []model.InversePatch{{
			Operation:      "replace",
			DryPath:        "values.image.tag",
			WetPath:        "Deployment/spec/template/spec/containers[0]/image",
			EditableBy:     policy.EditableBy,
			Confidence:     policy.Confidence,
			RequiresReview: policy.RequiresReview,
			Reason:         registry.InversePatchReason(g.Kind, "image_tag", "Container image tag maps cleanly to helm values."),
		}}
	case model.GeneratorApplicationSet:
		hints := applicationSetHintsFromInputs(detection.Repo, g.Inputs)
		namePolicy := registry.InversePatchTemplateFor(g.Kind, "child_name", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.89, RequiresReview: false,
		})
		sourcePathPolicy := registry.InversePatchTemplateFor(g.Kind, "source_path", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.86, RequiresReview: true,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "spec.template.metadata.name",
				WetPath:        "Application/metadata/name",
				EditableBy:     namePolicy.EditableBy,
				Confidence:     namePolicy.Confidence,
				RequiresReview: namePolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "child_name", "Child Application identity is generated from the parent ApplicationSet template."),
					map[string]string{"application_set_path": hints.ApplicationSetPath},
				),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.template.spec.source.path",
				WetPath:        "Application/spec/source/path",
				EditableBy:     sourcePathPolicy.EditableBy,
				Confidence:     sourcePathPolicy.Confidence,
				RequiresReview: sourcePathPolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "source_path", "Child Application source path is generated from the parent ApplicationSet template."),
					map[string]string{"application_set_path": hints.ApplicationSetPath},
				),
			},
		}
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		policy := registry.InversePatchTemplateFor(g.Kind, "env_var", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.90, RequiresReview: false,
		})
		return []model.InversePatch{{
			Operation:      "replace",
			DryPath:        fmt.Sprintf("containers.%s.variables.%s", hints.ContainerName, hints.VariableName),
			WetPath:        fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/env[name=%s]/value", hints.ContainerName, hints.VariableName),
			EditableBy:     policy.EditableBy,
			Confidence:     policy.Confidence,
			RequiresReview: policy.RequiresReview,
			Reason:         registry.InversePatchReason(g.Kind, "env_var", "Score variable maps to a single Kubernetes env var."),
		}}
	case model.GeneratorSpringBoot:
		appNamePolicy := registry.InversePatchTemplateFor(g.Kind, "app_name", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.88, RequiresReview: false,
		})
		serverPortPolicy := registry.InversePatchTemplateFor(g.Kind, "server_port", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.91, RequiresReview: false,
		})
		datasourcePolicy := registry.InversePatchTemplateFor(g.Kind, "datasource_url", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.78, RequiresReview: true,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "spring.application.name",
				WetPath:        "Deployment/metadata/labels[app.kubernetes.io/name]",
				EditableBy:     appNamePolicy.EditableBy,
				Confidence:     appNamePolicy.Confidence,
				RequiresReview: appNamePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "app_name", "Application identity should be app-editable without platform escalation."),
			},
			{
				Operation:      "replace",
				DryPath:        "server.port",
				WetPath:        "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				EditableBy:     serverPortPolicy.EditableBy,
				Confidence:     serverPortPolicy.Confidence,
				RequiresReview: serverPortPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "server_port", "Application listener port is an app-level configuration concern."),
			},
			{
				Operation:      "replace",
				DryPath:        "spring.datasource.url",
				WetPath:        "ConfigMap/data/application.yaml:spring.datasource.url",
				EditableBy:     datasourcePolicy.EditableBy,
				Confidence:     datasourcePolicy.Confidence,
				RequiresReview: datasourcePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "datasource_url", "Database connectivity impacts shared runtime dependencies."),
			},
		}
	case model.GeneratorBackstage:
		hints := backstagePathHintsFromInputs(g.Inputs)
		identityPolicy := registry.InversePatchTemplateFor(g.Kind, "identity", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.87, RequiresReview: false,
		})
		lifecyclePolicy := registry.InversePatchTemplateFor(g.Kind, "lifecycle", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.82, RequiresReview: true,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "metadata.name",
				WetPath:        "Application/metadata/name",
				EditableBy:     identityPolicy.EditableBy,
				Confidence:     identityPolicy.Confidence,
				RequiresReview: identityPolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "identity", "Backstage component identity is sourced from {{catalog_path}}."),
					map[string]string{"catalog_path": hints.CatalogPath},
				),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.lifecycle",
				WetPath:        "Application/metadata/labels[lifecycle]",
				EditableBy:     lifecyclePolicy.EditableBy,
				Confidence:     lifecyclePolicy.Confidence,
				RequiresReview: lifecyclePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "lifecycle", "Lifecycle changes impact platform ownership and support policy."),
			},
		}
	case model.GeneratorNoConfigPlatform:
		hints := noConfigPlatformPathHintsFromInputs(g.Inputs)
		environmentPolicy := registry.InversePatchTemplateFor(g.Kind, "environment", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.90, RequiresReview: false,
		})
		channelsPolicy := registry.InversePatchTemplateFor(g.Kind, "channels", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.88, RequiresReview: false,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "app.environment",
				WetPath:        "ConfigMap/data/PROVIDER_ENVIRONMENT",
				EditableBy:     environmentPolicy.EditableBy,
				Confidence:     environmentPolicy.Confidence,
				RequiresReview: environmentPolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "environment", "Environment is sourced from {{base_config_path}}."),
					map[string]string{"base_config_path": hints.BaseConfigPath},
				),
			},
			{
				Operation:      "replace",
				DryPath:        "channels.inbound",
				WetPath:        "ConfigMap/data/PROVIDER_CHANNEL_INBOUND",
				EditableBy:     channelsPolicy.EditableBy,
				Confidence:     channelsPolicy.Confidence,
				RequiresReview: channelsPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "channels", "Channel mapping is app-level runtime behavior."),
			},
		}
	case model.GeneratorOpenChoreo:
		hints := openChoreoPathHintsFromInputs(g.Inputs)
		imagePolicy := registry.InversePatchTemplateFor(g.Kind, "image", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.88, RequiresReview: false,
		})
		envPolicy := registry.InversePatchTemplateFor(g.Kind, "env_var", registry.InversePatchTemplate{
			EditableBy: "environment-owner", Confidence: 0.84, RequiresReview: false,
		})
		portPolicy := registry.InversePatchTemplateFor(g.Kind, "service_port", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.82, RequiresReview: true,
		})
		secretPolicy := registry.InversePatchTemplateFor(g.Kind, "secret_ref", registry.InversePatchTemplate{
			EditableBy: "security-team", Confidence: 0.86, RequiresReview: true,
		})
		resourcePolicy := registry.InversePatchTemplateFor(g.Kind, "resource_limit", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.80, RequiresReview: true,
		})
		defaultPolicy := registry.InversePatchTemplateFor(g.Kind, "platform_default", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.78, RequiresReview: true,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "spec.containers.main.image",
				WetPath:        "Deployment/spec/template/spec/containers[name=main]/image",
				EditableBy:     imagePolicy.EditableBy,
				Confidence:     imagePolicy.Confidence,
				RequiresReview: imagePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "image", "Container image is app-owned Workload intent."),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.environment.env.LOG_LEVEL",
				WetPath:        "Deployment/spec/template/spec/containers[name=main]/env[name=LOG_LEVEL]/value",
				EditableBy:     envPolicy.EditableBy,
				Confidence:     envPolicy.Confidence,
				RequiresReview: envPolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "env_var", "Environment values flow through the environment/release binding."),
					map[string]string{"release_binding_path": hints.ReleaseBindingPath},
				),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.service.port",
				WetPath:        "Service/spec/ports[name=http]/port",
				EditableBy:     portPolicy.EditableBy,
				Confidence:     portPolicy.Confidence,
				RequiresReview: portPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "service_port", "Service port is constrained by the ComponentType platform contract."),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.secretRef",
				WetPath:        "Deployment/spec/template/spec/containers[name=main]/env[name=DATABASE_URL]/valueFrom/secretKeyRef/name",
				EditableBy:     secretPolicy.EditableBy,
				Confidence:     secretPolicy.Confidence,
				RequiresReview: secretPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "secret_ref", "Secret references are security-owned bindings and should not be edited on generated resources."),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.resources.limits.cpu",
				WetPath:        "Deployment/spec/template/spec/containers[name=main]/resources/limits/cpu",
				EditableBy:     resourcePolicy.EditableBy,
				Confidence:     resourcePolicy.Confidence,
				RequiresReview: resourcePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "resource_limit", "Resource limits are environment/platform-owned policy defaults."),
			},
			{
				Operation:      "replace",
				DryPath:        "spec.runtime.defaults.securityContext.runAsNonRoot",
				WetPath:        "Deployment/spec/template/spec/securityContext/runAsNonRoot",
				EditableBy:     defaultPolicy.EditableBy,
				Confidence:     defaultPolicy.Confidence,
				RequiresReview: defaultPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "platform_default", "Platform defaults are owned by the ComponentType or platform policy, not generated Deployment YAML."),
			},
		}
	case model.GeneratorOpsFlow:
		hints := opsWorkflowPathHintsFromInputs(g.Inputs)
		imageTagPolicy := registry.InversePatchTemplateFor(g.Kind, "image_tag", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.87, RequiresReview: true,
		})
		schedulePolicy := registry.InversePatchTemplateFor(g.Kind, "schedule", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.84, RequiresReview: true,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "actions.deploy.image_tag",
				WetPath:        "Workflow/spec/templates[name=deploy]/container/image",
				EditableBy:     imageTagPolicy.EditableBy,
				Confidence:     imageTagPolicy.Confidence,
				RequiresReview: imageTagPolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "image_tag", "Deployment action image tag is sourced from {{base_spec_path}}."),
					map[string]string{"base_spec_path": hints.BaseSpecPath},
				),
			},
			{
				Operation:      "replace",
				DryPath:        "triggers.schedule",
				WetPath:        "Workflow/spec/schedule",
				EditableBy:     schedulePolicy.EditableBy,
				Confidence:     schedulePolicy.Confidence,
				RequiresReview: schedulePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "schedule", "Schedule changes affect operational execution timing."),
			},
		}
	case model.GeneratorC3Agent:
		hints := c3agentPathHintsFromInputs(g.Inputs)
		fleetConfigPolicy := registry.InversePatchTemplateFor(g.Kind, "fleet_config", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.91, RequiresReview: false,
		})
		credentialsPolicy := registry.InversePatchTemplateFor(g.Kind, "credentials", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.86, RequiresReview: true,
		})
		componentPortsPolicy := registry.InversePatchTemplateFor(g.Kind, "component_ports", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.84, RequiresReview: true,
		})
		agentRuntimePolicy := registry.InversePatchTemplateFor(g.Kind, "agent_runtime", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.88, RequiresReview: true,
		})
		storagePolicy := registry.InversePatchTemplateFor(g.Kind, "storage", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.85, RequiresReview: true,
		})
		replicasPolicy := registry.InversePatchTemplateFor(g.Kind, "replicas", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.89, RequiresReview: false,
		})
		rbacPolicy := registry.InversePatchTemplateFor(g.Kind, "rbac", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.82, RequiresReview: true,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "fleet.agent_model",
				WetPath:        "ConfigMap/data/AGENT_MODEL",
				EditableBy:     fleetConfigPolicy.EditableBy,
				Confidence:     fleetConfigPolicy.Confidence,
				RequiresReview: fleetConfigPolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "fleet_config", "Fleet agent model is sourced from {{base_config_path}}."),
					map[string]string{"base_config_path": hints.BaseConfigPath},
				),
			},
			{
				Operation:      "replace",
				DryPath:        "credentials.anthropic_key_ref",
				WetPath:        "Secret/data/ANTHROPIC_API_KEY",
				EditableBy:     credentialsPolicy.EditableBy,
				Confidence:     credentialsPolicy.Confidence,
				RequiresReview: credentialsPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "credentials", "Credential references impact secret management and platform security policy."),
			},
			{
				Operation:      "replace",
				DryPath:        "components.controlplane.grpc_port",
				WetPath:        "Service/spec/ports[name=grpc]/port",
				EditableBy:     componentPortsPolicy.EditableBy,
				Confidence:     componentPortsPolicy.Confidence,
				RequiresReview: componentPortsPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "component_ports", "Control plane port changes affect service mesh connectivity."),
			},
			{
				Operation:      "replace",
				DryPath:        "agent_runtime.image",
				WetPath:        "ConfigMap/data/JOB_IMAGE",
				EditableBy:     agentRuntimePolicy.EditableBy,
				Confidence:     agentRuntimePolicy.Confidence,
				RequiresReview: agentRuntimePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "agent_runtime", "Agent runtime image and budget settings affect platform execution behavior."),
			},
			{
				Operation:      "replace",
				DryPath:        "storage.task_pvc_size",
				WetPath:        "PersistentVolumeClaim/spec/resources/requests/storage",
				EditableBy:     storagePolicy.EditableBy,
				Confidence:     storagePolicy.Confidence,
				RequiresReview: storagePolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "storage", "Storage sizing and binding affect persistent runtime state."),
			},
			{
				Operation:      "replace",
				DryPath:        "components.controlplane.replicas",
				WetPath:        "Deployment/spec/replicas",
				EditableBy:     replicasPolicy.EditableBy,
				Confidence:     replicasPolicy.Confidence,
				RequiresReview: replicasPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "replicas", "Replica tuning affects fleet concurrency and runtime cost."),
			},
			{
				Operation:      "replace",
				DryPath:        "service",
				WetPath:        "ClusterRole/rules",
				EditableBy:     rbacPolicy.EditableBy,
				Confidence:     rbacPolicy.Confidence,
				RequiresReview: rbacPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "rbac", "RBAC resources are platform-governed and must align with security policy."),
			},
		}
	case model.GeneratorSwamp:
		hints := swampPathHintsFromInputs(g.Inputs)
		workflowDefinitionPolicy := registry.InversePatchTemplateFor(g.Kind, "workflow_definition", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.90, RequiresReview: false,
		})
		vaultConfigPolicy := registry.InversePatchTemplateFor(g.Kind, "vault_config", registry.InversePatchTemplate{
			EditableBy: "platform-engineer", Confidence: 0.85, RequiresReview: true,
		})
		modelBindingPolicy := registry.InversePatchTemplateFor(g.Kind, "model_binding", registry.InversePatchTemplate{
			EditableBy: "app-team", Confidence: 0.88, RequiresReview: false,
		})
		return []model.InversePatch{
			{
				Operation:      "replace",
				DryPath:        "jobs[].steps[].task",
				WetPath:        "Workflow/spec/jobs",
				EditableBy:     workflowDefinitionPolicy.EditableBy,
				Confidence:     workflowDefinitionPolicy.Confidence,
				RequiresReview: workflowDefinitionPolicy.RequiresReview,
				Reason: renderTargetTemplate(
					registry.InversePatchReason(g.Kind, "workflow_definition", "Workflow task definitions are sourced from {{base_config_path}}."),
					map[string]string{"base_config_path": hints.BaseConfigPath},
				),
			},
			{
				Operation:      "replace",
				DryPath:        "vaults.default.type",
				WetPath:        "ConfigMap/data/VAULT_TYPE",
				EditableBy:     vaultConfigPolicy.EditableBy,
				Confidence:     vaultConfigPolicy.Confidence,
				RequiresReview: vaultConfigPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "vault_config", "Vault type changes affect secrets backend and platform security policy."),
			},
			{
				Operation:      "replace",
				DryPath:        "jobs[].steps[].task.modelIdOrName",
				WetPath:        "Workflow/spec/model_refs",
				EditableBy:     modelBindingPolicy.EditableBy,
				Confidence:     modelBindingPolicy.Confidence,
				RequiresReview: modelBindingPolicy.RequiresReview,
				Reason:         registry.InversePatchReason(g.Kind, "model_binding", "Model binding maps task steps to execution model references."),
			},
		}
	default:
		return []model.InversePatch{}
	}
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
