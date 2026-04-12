package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/applicationset"
	"github.com/confighub/cub-gen/internal/contracts"
	"github.com/confighub/cub-gen/internal/detect"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

const (
	generatorContractSchema  = "cub.confighub.io/generator-contract/v1"
	provenanceSchema         = "cub.confighub.io/provenance/v1"
	inversePlanSchema        = "cub.confighub.io/inverse-transform-plan/v1"
	helmCLIOverrideTransform = "helm-cli-override"
	helmBuiltinTransform     = "helm-builtin"
	helmDefaultTransform     = "helm-default"
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

func fieldOriginsForGenerator(detection model.DetectionResult, g model.GeneratorDetection, helmCLIOverrides []model.HelmCLIOverride) []model.FieldOrigin {
	switch g.Kind {
	case model.GeneratorHelm:
		hints := helmProvenancePathsForGenerator(detection.Repo, g)
		imageTagSources := helmImageTagSources(detection.Repo, hints)
		overrideOrigins := helmCLIOverrideFieldOrigins(g, helmCLIOverrides)
		origins := make([]model.FieldOrigin, 0, len(overrideOrigins)+len(imageTagSources)+1)
		origins = append(origins, overrideOrigins...)
		if len(imageTagSources) == 0 {
			if builtinOrigin, ok := helmImageTagBuiltinOrigin(detection.Repo, hints); ok {
				origins = append(origins, builtinOrigin)
				return origins
			}
			origins = append(origins, helmImageTagDefaultOrigin())
			return origins
		}
		for _, source := range imageTagSources {
			transform := registry.FieldOriginTransform(g.Kind)
			confidenceKey := "image_tag_base"
			confidenceFallback := 0.86
			if source.Path != hints.PrimaryValuesPath {
				transform = registry.FieldOriginOverlayTransform(g.Kind)
				confidenceKey = "image_tag_overlay"
				confidenceFallback = 0.90
			}
			if strings.HasPrefix(source.Path, "charts/") {
				confidenceKey = "image_tag_subchart"
				confidenceFallback = 0.74
			}
			origins = append(origins, model.FieldOrigin{
				DryPath:     "values.image.tag",
				WetPath:     "Deployment/spec/template/spec/containers[0]/image",
				SourcePath:  source.Path,
				SourceLayer: source.Layer,
				Transform:   transform,
				Confidence:  registry.FieldOriginConfidenceFor(g.Kind, confidenceKey, confidenceFallback),
			})
		}
		return origins
	case model.GeneratorApplicationSet:
		hints := applicationSetHintsFromInputs(detection.Repo, g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "spec.template.metadata.name",
				WetPath:    "Application/metadata/name",
				SourcePath: hints.ApplicationSetPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "child_name", 0.89),
			},
		}
		if strings.TrimSpace(hints.TemplateSourcePath) != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "spec.template.spec.source.path",
				WetPath:    "Application/spec/source/path",
				SourcePath: hints.ApplicationSetPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "source_path", 0.86),
			})
		}
		return origins
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return []model.FieldOrigin{
			{
				DryPath:    fmt.Sprintf("containers.%s.image", hints.ContainerName),
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/image", hints.ContainerName),
				SourcePath: hints.SourcePath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "image", 0.94),
			},
			{
				DryPath:    fmt.Sprintf("containers.%s.variables.%s", hints.ContainerName, hints.VariableName),
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/env[name=%s]/value", hints.ContainerName, hints.VariableName),
				SourcePath: hints.SourcePath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "env_var", 0.90),
			},
			{
				DryPath:    fmt.Sprintf("service.ports.%s.port", hints.ServicePortName),
				WetPath:    fmt.Sprintf("Service/spec/ports[name=%s]/port", hints.ServicePortName),
				SourcePath: hints.SourcePath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "port", 0.91),
			},
		}
	case model.GeneratorSpringBoot:
		hints := springPathHintsFromInputs(g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "spring.application.name",
				WetPath:    "Deployment/metadata/labels[app.kubernetes.io/name]",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "app_name", 0.89),
			},
			{
				DryPath:    "server.port",
				WetPath:    "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "server_port_base", 0.92),
			},
			{
				DryPath:    "spring.datasource.url",
				WetPath:    "ConfigMap/data/application.yaml:spring.datasource.url",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "datasource_url", 0.78),
			},
		}
		if hints.ProfileConfigPath != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "server.port",
				WetPath:    "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				SourcePath: hints.ProfileConfigPath,
				Transform:  registry.FieldOriginOverlayTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "server_port_overlay", 0.88),
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
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "identity", 0.90),
			},
			{
				DryPath:    "spec.lifecycle",
				WetPath:    "Application/metadata/labels[lifecycle]",
				SourcePath: hints.CatalogPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "lifecycle", 0.82),
			},
		}
	case model.GeneratorNoConfigPlatform:
		hints := noConfigPlatformPathHintsFromInputs(g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "app.environment",
				WetPath:    "ConfigMap/data/PROVIDER_ENVIRONMENT",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "environment", 0.90),
			},
			{
				DryPath:    "channels.inbound",
				WetPath:    "ConfigMap/data/PROVIDER_CHANNEL_INBOUND",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "channels_base", 0.88),
			},
		}
		if hints.OverlayConfigPath != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "channels.inbound",
				WetPath:    "ConfigMap/data/PROVIDER_CHANNEL_INBOUND",
				SourcePath: hints.OverlayConfigPath,
				Transform:  registry.FieldOriginOverlayTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "channels_overlay", 0.84),
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
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "image_tag", 0.87),
			},
			{
				DryPath:    "triggers.schedule",
				WetPath:    "Workflow/spec/schedule",
				SourcePath: hints.BaseSpecPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "schedule_base", 0.84),
			},
		}
		if hints.OverlaySpecPath != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "triggers.schedule",
				WetPath:    "Workflow/spec/schedule",
				SourcePath: hints.OverlaySpecPath,
				Transform:  registry.FieldOriginOverlayTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "schedule_overlay", 0.80),
			})
		}
		return origins
	case model.GeneratorC3Agent:
		hints := c3agentPathHintsFromInputs(g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "fleet.agent_model",
				WetPath:    "ConfigMap/data/AGENT_MODEL",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "fleet_config", 0.91),
			},
			{
				DryPath:    "credentials.anthropic_key_ref",
				WetPath:    "Secret/data/ANTHROPIC_API_KEY",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "credentials", 0.86),
			},
			{
				DryPath:    "components.controlplane.grpc_port",
				WetPath:    "Service/controlplane/spec/ports[name=grpc]/port",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "component_ports_base", 0.84),
			},
			{
				DryPath:    "components.gateway.grpc_port",
				WetPath:    "Service/gateway/spec/ports[name=grpc]/port",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "component_ports_base", 0.84),
			},
			{
				DryPath:    "agent_runtime.image",
				WetPath:    "ConfigMap/data/JOB_IMAGE",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "agent_runtime", 0.88),
			},
			{
				DryPath:    "storage.task_pvc_size",
				WetPath:    "PersistentVolumeClaim/spec/resources/requests/storage",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "storage", 0.85),
			},
			{
				DryPath:    "components.controlplane.replicas",
				WetPath:    "Deployment/controlplane/spec/replicas",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "replicas_base", 0.89),
			},
			{
				DryPath:    "components.gateway.replicas",
				WetPath:    "Deployment/gateway/spec/replicas",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "replicas_base", 0.89),
			},
			{
				DryPath:    "service",
				WetPath:    "ClusterRole/rules",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "rbac", 0.82),
			},
		}
		if hints.OverlayConfigPath != "" {
			origins = append(origins,
				model.FieldOrigin{
					DryPath:    "components.controlplane.grpc_port",
					WetPath:    "Service/controlplane/spec/ports[name=grpc]/port",
					SourcePath: hints.OverlayConfigPath,
					Transform:  registry.FieldOriginOverlayTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "component_ports_overlay", 0.80),
				},
				model.FieldOrigin{
					DryPath:    "components.gateway.grpc_port",
					WetPath:    "Service/gateway/spec/ports[name=grpc]/port",
					SourcePath: hints.OverlayConfigPath,
					Transform:  registry.FieldOriginOverlayTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "component_ports_overlay", 0.80),
				},
				model.FieldOrigin{
					DryPath:    "components.controlplane.replicas",
					WetPath:    "Deployment/controlplane/spec/replicas",
					SourcePath: hints.OverlayConfigPath,
					Transform:  registry.FieldOriginOverlayTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "replicas_overlay", 0.85),
				},
				model.FieldOrigin{
					DryPath:    "components.gateway.replicas",
					WetPath:    "Deployment/gateway/spec/replicas",
					SourcePath: hints.OverlayConfigPath,
					Transform:  registry.FieldOriginOverlayTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "replicas_overlay", 0.85),
				},
				model.FieldOrigin{
					DryPath:    "storage.task_pvc_size",
					WetPath:    "PersistentVolumeClaim/spec/resources/requests/storage",
					SourcePath: hints.OverlayConfigPath,
					Transform:  registry.FieldOriginOverlayTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "storage", 0.85),
				},
			)
		}
		return origins
	case model.GeneratorSwamp:
		hints := swampPathHintsFromInputs(g.Inputs)
		origins := []model.FieldOrigin{
			{
				DryPath:    "vaults.default.type",
				WetPath:    "ConfigMap/data/VAULT_TYPE",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "vault_config", 0.85),
			},
			{
				DryPath:    "swamp.version",
				WetPath:    "ConfigMap/data/SWAMP_VERSION",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "version", 0.90),
			},
			{
				DryPath:    "jobs[].steps[].task",
				WetPath:    "Workflow/spec/jobs",
				SourcePath: hints.BaseConfigPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "workflow_definition", 0.88),
			},
		}
		if hints.WorkflowPath != "" {
			origins = append(origins, model.FieldOrigin{
				DryPath:    "jobs[].steps[].task.modelIdOrName",
				WetPath:    "Workflow/spec/model_refs",
				SourcePath: hints.WorkflowPath,
				Transform:  registry.FieldOriginOverlayTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "model_binding_overlay", 0.84),
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
		hints := helmProvenancePathsForGenerator(detection.Repo, g)
		imageTagSources := helmImageTagSources(detection.Repo, hints)
		policy := registry.InversePointerTemplateFor(g.Kind, "image_tag", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.86,
		})
		if len(imageTagSources) == 0 {
			if _, ok := helmImageTagBuiltinOrigin(detection.Repo, hints); ok {
				return []model.InverseEditPointer{
					{
						WetPath:    "Deployment/spec/template/spec/containers[0]/image",
						DryPath:    "values.image.tag",
						Owner:      "platform-engineer",
						EditHint:   "This field currently comes from Helm built-in .Chart.AppVersion. Edit appVersion in Chart.yaml or pass --set image.tag=... for an invocation-specific override.",
						Confidence: 1.0,
					},
				}
			}
			return []model.InverseEditPointer{
				{
					WetPath:    "Deployment/spec/template/spec/containers[0]/image",
					DryPath:    "values.image.tag",
					Owner:      "platform-engineer",
					EditHint:   "This field is not set in the observed Helm values files. Inspect Chart.yaml, templates/, and helper defaults before editing values.yaml.",
					Confidence: 0.40,
				},
			}
		}
		preferredImageTagPath := hints.PrimaryValuesPath
		if hints.OverlayValuesPath != "" {
			preferredImageTagPath = hints.OverlayValuesPath
		} else if len(imageTagSources) > 0 {
			preferredImageTagPath = imageTagSources[0].Path
		}
		vars := map[string]string{
			"base_values_path":    hints.PrimaryValuesPath,
			"overlay_values_path": preferredImageTagPath,
		}
		hintKey := "image_tag_base"
		hintFallback := "Edit values.image.tag in {{base_values_path}}."
		if preferredImageTagPath != "" && preferredImageTagPath != hints.PrimaryValuesPath {
			hintKey = "image_tag_overlay"
			hintFallback = "Edit values.image.tag in {{overlay_values_path}} for environment-specific overrides; use {{base_values_path}} for defaults."
		}
		return []model.InverseEditPointer{
			{
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				DryPath:    "values.image.tag",
				Owner:      helmInversePointerOwner(preferredImageTagPath, policy.Owner),
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, hintKey, hintFallback), vars),
				Confidence: policy.Confidence,
			},
		}
	case model.GeneratorApplicationSet:
		hints := applicationSetHintsFromInputs(detection.Repo, g.Inputs)
		namePolicy := registry.InversePointerTemplateFor(g.Kind, "child_name", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.89,
		})
		sourcePathPolicy := registry.InversePointerTemplateFor(g.Kind, "source_path", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.86,
		})
		vars := map[string]string{"application_set_path": hints.ApplicationSetPath}
		out := []model.InverseEditPointer{
			{
				WetPath:    "Application/metadata/name",
				DryPath:    "spec.template.metadata.name",
				Owner:      namePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "child_name", "Edit spec.template.metadata.name in {{application_set_path}}."), vars),
				Confidence: namePolicy.Confidence,
			},
		}
		if strings.TrimSpace(hints.TemplateSourcePath) != "" {
			out = append(out, model.InverseEditPointer{
				WetPath:    "Application/spec/source/path",
				DryPath:    "spec.template.spec.source.path",
				Owner:      sourcePathPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "source_path", "Edit spec.template.spec.source.path in {{application_set_path}}."), vars),
				Confidence: sourcePathPolicy.Confidence,
			})
		}
		return out
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		imagePolicy := registry.InversePointerTemplateFor(g.Kind, "image", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.94,
		})
		envVarPolicy := registry.InversePointerTemplateFor(g.Kind, "env_var", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.90,
		})
		portPolicy := registry.InversePointerTemplateFor(g.Kind, "port", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.91,
		})
		vars := map[string]string{
			"source_path":       hints.SourcePath,
			"container_name":    hints.ContainerName,
			"variable_name":     hints.VariableName,
			"service_port_name": hints.ServicePortName,
		}
		return []model.InverseEditPointer{
			{
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/image", hints.ContainerName),
				DryPath:    fmt.Sprintf("containers.%s.image", hints.ContainerName),
				Owner:      imagePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "image", "Edit the Score container image in {{source_path}}."), vars),
				Confidence: imagePolicy.Confidence,
			},
			{
				WetPath:    fmt.Sprintf("Deployment/spec/template/spec/containers[name=%s]/env[name=%s]/value", hints.ContainerName, hints.VariableName),
				DryPath:    fmt.Sprintf("containers.%s.variables.%s", hints.ContainerName, hints.VariableName),
				Owner:      envVarPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "env_var", "Edit {{variable_name}} under containers.{{container_name}}.variables in {{source_path}}."), vars),
				Confidence: envVarPolicy.Confidence,
			},
			{
				WetPath:    fmt.Sprintf("Service/spec/ports[name=%s]/port", hints.ServicePortName),
				DryPath:    fmt.Sprintf("service.ports.%s.port", hints.ServicePortName),
				Owner:      portPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "port", "Edit {{service_port_name}} service port in {{source_path}}."), vars),
				Confidence: portPolicy.Confidence,
			},
		}
	case model.GeneratorSpringBoot:
		hints := springPathHintsFromInputs(g.Inputs)
		appNamePolicy := registry.InversePointerTemplateFor(g.Kind, "app_name", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.89,
		})
		serverPortPolicy := registry.InversePointerTemplateFor(g.Kind, "server_port", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.91,
		})
		datasourcePolicy := registry.InversePointerTemplateFor(g.Kind, "datasource_url", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.78,
		})
		vars := map[string]string{
			"base_config_path":    hints.BaseConfigPath,
			"profile_config_path": hints.ProfileConfigPath,
		}
		serverPortHintKey := "server_port_base"
		serverPortHintFallback := "Edit server.port in {{base_config_path}}."
		if hints.ProfileConfigPath != "" {
			serverPortHintKey = "server_port_overlay"
			serverPortHintFallback = "Edit server.port in {{profile_config_path}} for environment overrides; use {{base_config_path}} for the default."
		}
		return []model.InverseEditPointer{
			{
				WetPath:    "Deployment/metadata/labels[app.kubernetes.io/name]",
				DryPath:    "spring.application.name",
				Owner:      appNamePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "app_name", "Edit spring.application.name in {{base_config_path}}."), vars),
				Confidence: appNamePolicy.Confidence,
			},
			{
				WetPath:    "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort",
				DryPath:    "server.port",
				Owner:      serverPortPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, serverPortHintKey, serverPortHintFallback), vars),
				Confidence: serverPortPolicy.Confidence,
			},
			{
				WetPath:    "ConfigMap/data/application.yaml:spring.datasource.url",
				DryPath:    "spring.datasource.url",
				Owner:      datasourcePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "datasource_url", "Edit spring.datasource.url in {{base_config_path}} and coordinate with platform ownership rules."), vars),
				Confidence: datasourcePolicy.Confidence,
			},
		}
	case model.GeneratorBackstage:
		hints := backstagePathHintsFromInputs(g.Inputs)
		namePolicy := registry.InversePointerTemplateFor(g.Kind, "name", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.90,
		})
		lifecyclePolicy := registry.InversePointerTemplateFor(g.Kind, "lifecycle", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.82,
		})
		vars := map[string]string{"catalog_path": hints.CatalogPath}
		return []model.InverseEditPointer{
			{
				WetPath:    "Application/metadata/name",
				DryPath:    "metadata.name",
				Owner:      namePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "name", "Edit metadata.name in {{catalog_path}}."), vars),
				Confidence: namePolicy.Confidence,
			},
			{
				WetPath:    "Application/metadata/labels[lifecycle]",
				DryPath:    "spec.lifecycle",
				Owner:      lifecyclePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "lifecycle", "Edit spec.lifecycle in {{catalog_path}} and coordinate rollout policy."), vars),
				Confidence: lifecyclePolicy.Confidence,
			},
		}
	case model.GeneratorNoConfigPlatform:
		hints := noConfigPlatformPathHintsFromInputs(g.Inputs)
		environmentPolicy := registry.InversePointerTemplateFor(g.Kind, "environment", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.90,
		})
		channelsPolicy := registry.InversePointerTemplateFor(g.Kind, "channels", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.88,
		})
		vars := map[string]string{
			"base_config_path":    hints.BaseConfigPath,
			"overlay_config_path": hints.OverlayConfigPath,
		}
		channelsHintKey := "channels_base"
		channelsHintFallback := "Edit channels.inbound in {{base_config_path}}."
		if hints.OverlayConfigPath != "" {
			channelsHintKey = "channels_overlay"
			channelsHintFallback = "Edit channels.inbound in {{overlay_config_path}} for environment-specific behavior; use {{base_config_path}} for defaults."
		}
		return []model.InverseEditPointer{
			{
				WetPath:    "ConfigMap/data/PROVIDER_ENVIRONMENT",
				DryPath:    "app.environment",
				Owner:      environmentPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "environment", "Edit app.environment in {{base_config_path}}."), vars),
				Confidence: environmentPolicy.Confidence,
			},
			{
				WetPath:    "ConfigMap/data/PROVIDER_CHANNEL_INBOUND",
				DryPath:    "channels.inbound",
				Owner:      channelsPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, channelsHintKey, channelsHintFallback), vars),
				Confidence: channelsPolicy.Confidence,
			},
		}
	case model.GeneratorOpsFlow:
		hints := opsWorkflowPathHintsFromInputs(g.Inputs)
		imageTagPolicy := registry.InversePointerTemplateFor(g.Kind, "image_tag", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.87,
		})
		schedulePolicy := registry.InversePointerTemplateFor(g.Kind, "schedule", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.84,
		})
		vars := map[string]string{
			"base_spec_path":    hints.BaseSpecPath,
			"overlay_spec_path": hints.OverlaySpecPath,
		}
		scheduleHintKey := "schedule_base"
		scheduleHintFallback := "Edit triggers.schedule in {{base_spec_path}}."
		if hints.OverlaySpecPath != "" {
			scheduleHintKey = "schedule_overlay"
			scheduleHintFallback = "Edit triggers.schedule in {{overlay_spec_path}} for environment-specific cadence; use {{base_spec_path}} for defaults."
		}
		return []model.InverseEditPointer{
			{
				WetPath:    "Workflow/spec/templates[name=deploy]/container/image",
				DryPath:    "actions.deploy.image_tag",
				Owner:      imageTagPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "image_tag", "Edit actions.deploy.image_tag in {{base_spec_path}}."), vars),
				Confidence: imageTagPolicy.Confidence,
			},
			{
				WetPath:    "Workflow/spec/schedule",
				DryPath:    "triggers.schedule",
				Owner:      schedulePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, scheduleHintKey, scheduleHintFallback), vars),
				Confidence: schedulePolicy.Confidence,
			},
		}
	case model.GeneratorC3Agent:
		hints := c3agentPathHintsFromInputs(g.Inputs)
		fleetConfigPolicy := registry.InversePointerTemplateFor(g.Kind, "fleet_config", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.91,
		})
		credentialsPolicy := registry.InversePointerTemplateFor(g.Kind, "credentials", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.86,
		})
		componentPortsPolicy := registry.InversePointerTemplateFor(g.Kind, "component_ports", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.84,
		})
		agentRuntimePolicy := registry.InversePointerTemplateFor(g.Kind, "agent_runtime", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.88,
		})
		storagePolicy := registry.InversePointerTemplateFor(g.Kind, "storage", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.85,
		})
		replicasPolicy := registry.InversePointerTemplateFor(g.Kind, "replicas", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.89,
		})
		rbacPolicy := registry.InversePointerTemplateFor(g.Kind, "rbac", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.82,
		})
		vars := map[string]string{
			"base_config_path":    hints.BaseConfigPath,
			"overlay_config_path": hints.OverlayConfigPath,
		}
		componentPortsHintKey := "component_ports_base"
		componentPortsHintFallback := "Edit components.controlplane.grpc_port or components.gateway.grpc_port in {{base_config_path}}."
		replicasHintKey := "replicas_base"
		replicasHintFallback := "Edit components.controlplane.replicas or components.gateway.replicas in {{base_config_path}}."
		if hints.OverlayConfigPath != "" {
			componentPortsHintKey = "component_ports_overlay"
			componentPortsHintFallback = "Edit component ports in {{overlay_config_path}} for environment-specific values; use {{base_config_path}} for defaults."
			replicasHintKey = "replicas_overlay"
			replicasHintFallback = "Edit component replica counts in {{overlay_config_path}} for environment-specific values; use {{base_config_path}} for defaults."
		}
		return []model.InverseEditPointer{
			{
				WetPath:    "ConfigMap/data/AGENT_MODEL",
				DryPath:    "fleet.agent_model",
				Owner:      fleetConfigPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "fleet_config", "Edit fleet.agent_model in {{base_config_path}}."), vars),
				Confidence: fleetConfigPolicy.Confidence,
			},
			{
				WetPath:    "Secret/data/ANTHROPIC_API_KEY",
				DryPath:    "credentials.anthropic_key_ref",
				Owner:      credentialsPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "credentials", "Edit credentials.anthropic_key_ref in {{base_config_path}} and coordinate with platform security policy."), vars),
				Confidence: credentialsPolicy.Confidence,
			},
			{
				WetPath:    "Service/spec/ports[name=grpc]/port",
				DryPath:    "components.controlplane.grpc_port",
				Owner:      componentPortsPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, componentPortsHintKey, componentPortsHintFallback), vars),
				Confidence: componentPortsPolicy.Confidence,
			},
			{
				WetPath:    "ConfigMap/data/JOB_IMAGE",
				DryPath:    "agent_runtime.image",
				Owner:      agentRuntimePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "agent_runtime", "Edit agent_runtime.image in {{base_config_path}}."), vars),
				Confidence: agentRuntimePolicy.Confidence,
			},
			{
				WetPath:    "PersistentVolumeClaim/spec/resources/requests/storage",
				DryPath:    "storage.task_pvc_size",
				Owner:      storagePolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "storage", "Edit storage.task_pvc_size in {{base_config_path}}."), vars),
				Confidence: storagePolicy.Confidence,
			},
			{
				WetPath:    "Deployment/spec/replicas",
				DryPath:    "components.controlplane.replicas",
				Owner:      replicasPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, replicasHintKey, replicasHintFallback), vars),
				Confidence: replicasPolicy.Confidence,
			},
			{
				WetPath:    "ClusterRole/rules",
				DryPath:    "service",
				Owner:      rbacPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "rbac", "Edit service identity in {{base_config_path}} and coordinate with platform security owners."), vars),
				Confidence: rbacPolicy.Confidence,
			},
		}
	case model.GeneratorSwamp:
		hints := swampPathHintsFromInputs(g.Inputs)
		workflowDefinitionPolicy := registry.InversePointerTemplateFor(g.Kind, "workflow_definition", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.90,
		})
		vaultConfigPolicy := registry.InversePointerTemplateFor(g.Kind, "vault_config", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.85,
		})
		modelBindingPolicy := registry.InversePointerTemplateFor(g.Kind, "model_binding", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.88,
		})
		vars := map[string]string{
			"base_config_path": hints.BaseConfigPath,
			"workflow_path":    hints.WorkflowPath,
		}
		modelBindingHintKey := "model_binding_base"
		modelBindingHintFallback := "Edit model references in {{base_config_path}}."
		if hints.WorkflowPath != "" {
			modelBindingHintKey = "model_binding_workflow"
			modelBindingHintFallback = "Edit model method bindings in {{workflow_path}} for task-specific overrides."
		}
		return []model.InverseEditPointer{
			{
				WetPath:    "Workflow/spec/jobs",
				DryPath:    "jobs[].steps[].task",
				Owner:      workflowDefinitionPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "workflow_definition", "Edit jobs[].steps[].task in {{base_config_path}}."), vars),
				Confidence: workflowDefinitionPolicy.Confidence,
			},
			{
				WetPath:    "ConfigMap/data/VAULT_TYPE",
				DryPath:    "vaults.default.type",
				Owner:      vaultConfigPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "vault_config", "Edit vaults.default.type in {{base_config_path}} and coordinate with platform security policy."), vars),
				Confidence: vaultConfigPolicy.Confidence,
			},
			{
				WetPath:    "Workflow/spec/model_refs",
				DryPath:    "jobs[].steps[].task.modelIdOrName",
				Owner:      modelBindingPolicy.Owner,
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, modelBindingHintKey, modelBindingHintFallback), vars),
				Confidence: modelBindingPolicy.Confidence,
			},
		}
	default:
		return []model.InverseEditPointer{}
	}
}

func helmInversePointerOwner(preferredImageTagPath, fallbackOwner string) string {
	if helmIsSubchartPath(preferredImageTagPath) {
		return "platform-engineer"
	}
	if strings.TrimSpace(fallbackOwner) == "" {
		return "platform-engineer"
	}
	return fallbackOwner
}

type applicationSetHints struct {
	ApplicationSetPath string
	TemplateSourcePath string
}

func applicationSetHintsFromInputs(repo string, inputs []string) applicationSetHints {
	h := applicationSetHints{
		ApplicationSetPath: registry.HintDefault(model.GeneratorApplicationSet, "application_set_path", "applicationset.yaml"),
	}
	appsetPath := firstInputPathForRole(model.GeneratorApplicationSet, inputs, "application-set")
	if appsetPath != "" {
		h.ApplicationSetPath = appsetPath
		if doc, err := applicationset.ParseFile(repo, appsetPath); err == nil {
			h.TemplateSourcePath = doc.TemplateSourcePath
		}
	}
	return h
}

type scoreHints struct {
	SourcePath      string
	ContainerName   string
	VariableName    string
	ServicePortName string
}

func scorePathHintsFromInputs(repo string, inputs []string) scoreHints {
	h := scoreHints{
		SourcePath:      registry.HintDefault(model.GeneratorScore, "source_path", "score.yaml"),
		ContainerName:   registry.HintDefault(model.GeneratorScore, "container_name", "main"),
		VariableName:    registry.HintDefault(model.GeneratorScore, "variable_name", "LOG_LEVEL"),
		ServicePortName: registry.HintDefault(model.GeneratorScore, "service_port_name", "web"),
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

func dryInputsForHelmCLIOverrides(g model.GeneratorDetection, overrides []model.HelmCLIOverride) []model.DryInputRef {
	if g.Kind != model.GeneratorHelm || len(overrides) == 0 {
		return nil
	}

	out := make([]model.DryInputRef, 0, len(overrides))
	for _, override := range overrides {
		out = append(out, model.DryInputRef{
			GeneratorID: g.ID,
			Profile:     g.Profile,
			Role:        "cli-override",
			Owner:       "release-automation",
			Path:        HelmCLIOverrideDisplay(override),
			Required:    false,
		})
	}
	return out
}

func wetManifestTargetsForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.WetManifestTarget {
	if g.Kind == model.GeneratorApplicationSet {
		analysis := applicationSetAnalysisForGenerator(detection, g)
		out := []model.WetManifestTarget{{
			GeneratorID:   g.ID,
			Kind:          "ApplicationSet",
			Name:          g.Name,
			Owner:         "platform-runtime",
			Namespace:     "argocd",
			SourceDryPath: "spec",
		}}
		if analysis == nil {
			return out
		}
		for _, child := range analysis.GeneratedApplications {
			out = append(out, model.WetManifestTarget{
				GeneratorID:   g.ID,
				Kind:          "Application",
				Name:          child.Name,
				Owner:         "platform-runtime",
				Namespace:     "argocd",
				SourceDryPath: "spec.template.metadata.name",
			})
		}
		return out
	}

	templates := registry.WetTargetTemplates(g.Kind)
	if len(templates) == 0 {
		return []model.WetManifestTarget{}
	}

	vars := map[string]string{"name": g.Name}
	if g.Kind == model.GeneratorScore {
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		vars["container"] = hints.ContainerName
		vars["service_port"] = hints.ServicePortName
	}

	out := make([]model.WetManifestTarget, 0, len(templates))
	for _, t := range templates {
		out = append(out, model.WetManifestTarget{
			GeneratorID:   g.ID,
			Kind:          t.Kind,
			Name:          renderTargetTemplate(t.NameTemplate, vars),
			Owner:         t.Owner,
			Namespace:     t.Namespace,
			SourceDryPath: renderTargetTemplate(t.SourceDryPathTemplate, vars),
		})
	}
	return out
}

func renderTargetTemplate(template string, vars map[string]string) string {
	if template == "" || len(vars) == 0 {
		return template
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := template
	for _, k := range keys {
		out = strings.ReplaceAll(out, "{{"+k+"}}", vars[k])
	}
	return out
}

type helmProvenancePaths struct {
	ChartPath         string
	ValuesPaths       []string
	PrimaryValuesPath string
	OverlayValuesPath string
}

type helmValueSource struct {
	Path  string
	Layer string
}

type helmDependencyDoc struct {
	Name       string `yaml:"name"`
	Alias      string `yaml:"alias"`
	Condition  string `yaml:"condition"`
	Repository string `yaml:"repository"`
	Version    string `yaml:"version"`
	ChartPath  string
	ValuesPath string
}

func helmProvenancePathsForGenerator(repoPath string, g model.GeneratorDetection) helmProvenancePaths {
	if g.Kind != model.GeneratorHelm {
		return helmProvenancePaths{}
	}

	chartRole := strings.TrimSpace(registry.HintDefault(g.Kind, "chart_role", "chart"))
	if chartRole == "" {
		chartRole = "chart"
	}
	valuesRole := strings.TrimSpace(registry.HintDefault(g.Kind, "values_role", "values"))
	if valuesRole == "" {
		valuesRole = "values"
	}
	primaryValuesBase := strings.TrimSpace(registry.HintDefault(g.Kind, "primary_values_path", "values.yaml"))
	if primaryValuesBase == "" {
		primaryValuesBase = "values.yaml"
	}

	chartPath := firstInputPathForRole(g.Kind, g.Inputs, chartRole)
	valuesPaths := helmValuesPathsForInputs(g.Kind, g.Inputs, valuesRole)

	primaryValuesPath := selectPrimaryHelmValuesPath(valuesPaths, primaryValuesBase)
	imageTagSources := helmImageTagSources(repoPath, helmProvenancePaths{
		ChartPath:         chartPath,
		ValuesPaths:       valuesPaths,
		PrimaryValuesPath: primaryValuesPath,
	})
	overlayValuesPath := ""
	for _, source := range imageTagSources {
		if source.Path != primaryValuesPath {
			overlayValuesPath = source.Path
			break
		}
	}

	return helmProvenancePaths{
		ChartPath:         chartPath,
		ValuesPaths:       valuesPaths,
		PrimaryValuesPath: primaryValuesPath,
		OverlayValuesPath: overlayValuesPath,
	}
}

func selectPrimaryHelmValuesPath(valuesPaths []string, primaryValuesBase string) string {
	primaryValuesPath := primaryValuesBase
	if selected := selectPreferredPathByBase(valuesPaths, primaryValuesBase); selected != "" {
		return selected
	}
	if len(valuesPaths) > 0 {
		return valuesPaths[0]
	}
	return primaryValuesPath
}

func helmValuesPathsForInputs(kind model.GeneratorKind, inputs []string, valuesRole string) []string {
	out := make([]string, 0, len(inputs))
	for _, in := range inputs {
		role := registry.InputRole(kind, in)
		if role == valuesRole || role == "subchart-values" {
			out = append(out, in)
		}
	}
	sort.Strings(out)
	return out
}

func firstInputPathForRole(kind model.GeneratorKind, inputs []string, role string) string {
	for _, in := range inputs {
		if registry.InputRole(kind, in) == role {
			return in
		}
	}
	return ""
}

func helmImageTagSources(repoPath string, hints helmProvenancePaths) []helmValueSource {
	imageTagPaths := make([]string, 0, len(hints.ValuesPaths))
	for _, path := range hints.ValuesPaths {
		if helmValuesPathDefinesImageTag(repoPath, path) {
			imageTagPaths = append(imageTagPaths, path)
		}
	}
	if len(imageTagPaths) == 0 {
		return nil
	}

	sort.SliceStable(imageTagPaths, func(i, j int) bool {
		leftRank := helmValuesPathPrecedence(imageTagPaths[i], hints.PrimaryValuesPath, hints.ChartPath)
		rightRank := helmValuesPathPrecedence(imageTagPaths[j], hints.PrimaryValuesPath, hints.ChartPath)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return imageTagPaths[i] < imageTagPaths[j]
	})

	umbrella := helmHasDependencyCharts(repoPath, hints.ChartPath)
	ordered := make([]helmValueSource, 0, len(imageTagPaths))
	for _, path := range imageTagPaths {
		ordered = append(ordered, helmValueSource{
			Path:  path,
			Layer: helmValuesSourceLayer(path, umbrella),
		})
	}
	return ordered
}

func helmValuesPathPrecedence(path, primaryValuesPath, chartPath string) int {
	pathDir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(path)))
	if pathDir == "." {
		pathDir = ""
	}
	switch {
	case path == primaryValuesPath:
		return 0
	case helmIsSubchartPath(path):
		return 2
	case helmChartDir(chartPath) == pathDir:
		return 1
	default:
		return 3
	}
}

func helmValuesSourceLayer(path string, umbrella bool) string {
	if helmIsSubchartPath(path) {
		parts := strings.Split(filepath.ToSlash(path), "/")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	if umbrella {
		return "umbrella"
	}
	return ""
}

func helmIsSubchartPath(path string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	return normalized == "charts" || strings.HasPrefix(normalized, "charts/")
}

func helmChartDir(chartPath string) string {
	dir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(chartPath)))
	if dir == "." {
		return ""
	}
	return dir
}

func helmHasDependencyCharts(repoPath, chartPath string) bool {
	return len(parseHelmDependencyDocs(repoPath, chartPath)) > 0
}

func helmImageTagBuiltinOrigin(repoPath string, hints helmProvenancePaths) (model.FieldOrigin, bool) {
	if strings.TrimSpace(repoPath) == "" {
		return model.FieldOrigin{}, false
	}
	if !helmTemplatesUseChartAppVersionFallback(repoPath) {
		return model.FieldOrigin{}, false
	}
	if appVersion := helmChartAppVersion(repoPath, hints.ChartPath); strings.TrimSpace(appVersion) != "" {
		return model.FieldOrigin{
			DryPath:    "values.image.tag",
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			SourcePath: "<helm-builtin>:.Chart.AppVersion",
			Transform:  helmBuiltinTransform,
			Confidence: 1.0,
		}, true
	}
	return model.FieldOrigin{}, false
}

func helmImageTagDefaultOrigin() model.FieldOrigin {
	return model.FieldOrigin{
		DryPath:    "values.image.tag",
		WetPath:    "Deployment/spec/template/spec/containers[0]/image",
		SourcePath: "<helm-default>:values.image.tag",
		Transform:  helmDefaultTransform,
		Confidence: 0.40,
	}
}

func helmTemplatesUseChartAppVersionFallback(repoPath string) bool {
	templatesDir := filepath.Join(repoPath, "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return false
	}

	patterns := []string{
		".Values.image.tag | default .Chart.AppVersion",
		"default .Chart.AppVersion .Values.image.tag",
	}
	found := false
	_ = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(content)
		for _, pattern := range patterns {
			if strings.Contains(text, pattern) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

func helmChartAppVersion(repoPath, chartPath string) string {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(chartPath) == "" {
		return ""
	}
	content, err := os.ReadFile(filepath.Join(repoPath, chartPath))
	if err != nil {
		return ""
	}

	var chartDoc struct {
		AppVersion string `yaml:"appVersion"`
	}
	if err := yaml.Unmarshal(content, &chartDoc); err != nil {
		return ""
	}
	return strings.TrimSpace(chartDoc.AppVersion)
}

func stringSliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func helmValuesPathDefinesImageTag(repoPath, relPath string) bool {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(relPath) == "" {
		return false
	}
	content, err := os.ReadFile(filepath.Join(repoPath, relPath))
	if err != nil {
		return false
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return false
	}
	return yamlPathExists(&doc, []string{"image", "tag"})
}

func yamlPathExists(node *yaml.Node, path []string) bool {
	if node == nil {
		return false
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return yamlPathExists(node.Content[0], path)
	}
	if len(path) == 0 {
		return true
	}
	if node.Kind != yaml.MappingNode {
		if node.Kind == yaml.AliasNode {
			return yamlPathExists(node.Alias, path)
		}
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != path[0] {
			continue
		}
		return yamlPathExists(node.Content[i+1], path[1:])
	}
	return false
}

func inputPathsForRole(kind model.GeneratorKind, inputs []string, role string) []string {
	out := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if registry.InputRole(kind, in) == role {
			out = append(out, in)
		}
	}
	sort.Strings(out)
	return out
}

func selectPreferredPathByBase(paths []string, preferredBase string) string {
	for _, p := range paths {
		if strings.EqualFold(filepath.Base(p), preferredBase) {
			return p
		}
	}
	return ""
}

func firstDistinctPath(paths []string, primary string) string {
	for _, p := range paths {
		if p != primary {
			return p
		}
	}
	return ""
}

func helmCLIOverridesForGenerator(g model.GeneratorDetection, overrides []model.HelmCLIOverride) []model.HelmCLIOverride {
	if g.Kind != model.GeneratorHelm || len(overrides) == 0 {
		return nil
	}
	out := make([]model.HelmCLIOverride, len(overrides))
	copy(out, overrides)
	return out
}

func helmCLIOverrideSourcesForGenerator(ref string, g model.GeneratorDetection, overrides []model.HelmCLIOverride) []model.SourceRef {
	if g.Kind != model.GeneratorHelm || len(overrides) == 0 {
		return nil
	}
	sources := make([]model.SourceRef, 0, len(overrides))
	for _, override := range overrides {
		sources = append(sources, model.SourceRef{
			Role:     "cli-override",
			URI:      fmt.Sprintf("helm-cli://%s/%s", override.Flag, override.Key),
			Revision: ref,
			Path:     HelmCLIOverrideDisplay(override),
		})
	}
	return sources
}

func helmCLIOverrideFieldOrigins(g model.GeneratorDetection, overrides []model.HelmCLIOverride) []model.FieldOrigin {
	if g.Kind != model.GeneratorHelm {
		return nil
	}

	override, ok := lastHelmCLIOverrideForKey(overrides, "image.tag")
	if !ok {
		return nil
	}

	return []model.FieldOrigin{{
		DryPath:    "values.image.tag",
		WetPath:    "Deployment/spec/template/spec/containers[0]/image",
		SourcePath: HelmCLIOverrideDisplay(override),
		Transform:  helmCLIOverrideTransform,
		Confidence: 1.0,
	}}
}

func lastHelmCLIOverrideForKey(overrides []model.HelmCLIOverride, key string) (model.HelmCLIOverride, bool) {
	for idx := len(overrides) - 1; idx >= 0; idx-- {
		if overrides[idx].Key == key {
			return overrides[idx], true
		}
	}
	return model.HelmCLIOverride{}, false
}

func applicationSetAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.ApplicationSetAnalysis {
	if g.Kind != model.GeneratorApplicationSet {
		return nil
	}

	appsetPath := firstInputPathForRole(g.Kind, g.Inputs, "application-set")
	if appsetPath == "" {
		return nil
	}

	clusterPaths := inputPathsForRole(g.Kind, g.Inputs, "cluster-inventory")
	analysis, err := applicationset.Analyze(detection.Repo, appsetPath, clusterPaths)
	if err != nil {
		return &model.ApplicationSetAnalysis{
			ApplicationSetPath:    appsetPath,
			ClusterInventoryPaths: append([]string(nil), clusterPaths...),
			Mode:                  applicationset.ModeObservedOnly,
			ModeReason:            fmt.Sprintf("ApplicationSet input %s was observed, but cub-gen could not parse it deterministically yet.", appsetPath),
		}
	}

	out := &model.ApplicationSetAnalysis{
		ApplicationSetPath:        analysis.ApplicationSetPath,
		GeneratorTypes:            append([]string(nil), analysis.GeneratorTypes...),
		UnsupportedGeneratorTypes: append([]string(nil), analysis.UnsupportedGeneratorTypes...),
		ClusterInventoryPaths:     append([]string(nil), analysis.ClusterInventoryPaths...),
		MatchedClusters:           append([]string(nil), analysis.MatchedClusters...),
		ListElementNames:          append([]string(nil), analysis.ListElementNames...),
		Mode:                      analysis.Mode,
		ModeReason:                analysis.ModeReason,
	}
	for _, child := range analysis.GeneratedApplications {
		out.GeneratedApplications = append(out.GeneratedApplications, model.ApplicationSetGeneratedApplication{
			Name:          child.Name,
			GeneratorType: child.GeneratorType,
			Reason:        child.Reason,
			InventoryPath: child.InventoryPath,
			Cluster:       child.Cluster,
			ListElement:   child.ListElement,
			SourcePath:    child.SourcePath,
		})
	}
	return out
}

type helmApplicationSetDoc struct {
	Path       string
	Selector   map[string]string
	ValueFiles []string
}

type helmClusterInventoryDoc struct {
	Path   string
	Name   string
	Labels map[string]string
}

type helmManagedCatalogDoc struct {
	Path             string
	RequiredSecurity string
}

type helmCustomerCatalogDoc struct {
	Path             string
	ValuesFile       string
	OverrideSecurity string
	Enabled          *bool
}

func helmLayeredAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.HelmLayeredAnalysis {
	if g.Kind != model.GeneratorHelm {
		return nil
	}

	appsetPath := firstInputPathForRole(g.Kind, g.Inputs, "application-set")
	clusterPaths := inputPathsForRole(g.Kind, g.Inputs, "cluster-inventory")
	managedCatalogPaths := inputPathsForRole(g.Kind, g.Inputs, "managed-service-catalog")
	customerCatalogPaths := inputPathsForRole(g.Kind, g.Inputs, "customer-service-catalog")
	helmPaths := helmProvenancePathsForGenerator(detection.Repo, g)
	dependencyCharts := helmDependencyChartsForGenerator(detection.Repo, helmPaths)
	crdPaths := helmCRDPathsForChart(detection.Repo, helmPaths.ChartPath)
	hookTemplates := helmHookTemplates(detection.Repo, helmPaths.ChartPath)
	hookTemplatePaths := make([]string, 0, len(hookTemplates))
	for _, hookTemplate := range hookTemplates {
		hookTemplatePaths = append(hookTemplatePaths, hookTemplate.Path)
	}
	valuesSchemaPath := helmValuesSchemaPath(detection.Repo, helmPaths.ChartPath)
	schemaState, schemaViolations := helmValuesSchemaValidation(detection.Repo, helmPaths, valuesSchemaPath)

	if appsetPath == "" && len(clusterPaths) == 0 && len(managedCatalogPaths) == 0 && len(customerCatalogPaths) == 0 &&
		len(dependencyCharts) == 0 && len(crdPaths) == 0 && len(hookTemplatePaths) == 0 && valuesSchemaPath == "" {
		return nil
	}

	analysis := &model.HelmLayeredAnalysis{
		ApplicationSetPath:         appsetPath,
		ClusterInventoryPaths:      append([]string(nil), clusterPaths...),
		ManagedCatalogPaths:        append([]string(nil), managedCatalogPaths...),
		CustomerCatalogPaths:       append([]string(nil), customerCatalogPaths...),
		DependencyCharts:           append([]model.HelmDependencyChart(nil), dependencyCharts...),
		CRDPaths:                   append([]string(nil), crdPaths...),
		HookTemplates:              append([]model.HelmHookTemplate(nil), hookTemplates...),
		HookTemplatePaths:          append([]string(nil), hookTemplatePaths...),
		ValuesSchemaPath:           valuesSchemaPath,
		SchemaValidationState:      schemaState,
		SchemaValidationViolations: append([]string(nil), schemaViolations...),
		SelectedValueFiles:         append([]string(nil), helmPaths.ValuesPaths...),
	}

	if appsetPath != "" {
		appsetDoc, err := parseHelmApplicationSetFile(detection.Repo, appsetPath)
		if err == nil {
			analysis.ClusterSelector = selectorString(appsetDoc.Selector)
			if len(appsetDoc.ValueFiles) > 0 {
				analysis.SelectedValueFiles = append([]string(nil), appsetDoc.ValueFiles...)
			}
			switch {
			case len(appsetDoc.Selector) == 0:
				analysis.GenerationDecisionState = "unresolved"
				analysis.GenerationDecisionReason = "ApplicationSet input is present, but cub-gen could not find a cluster matchLabels selector to attribute yet."
			case len(clusterPaths) == 0:
				analysis.GenerationDecisionState = "unresolved"
				analysis.GenerationDecisionReason = "ApplicationSet selector is present, but no cluster inventory inputs were observed to prove which clusters match it."
			default:
				matched := matchedHelmClusterInventories(detection.Repo, clusterPaths, appsetDoc.Selector)
				if len(matched) == 0 {
					analysis.GenerationDecisionState = "unresolved"
					analysis.GenerationDecisionReason = fmt.Sprintf("ApplicationSet selector %s is present, but none of the observed cluster inventory inputs match it.", selectorString(appsetDoc.Selector))
				} else {
					analysis.MatchedClusters = matched
					analysis.GenerationDecisionState = "attributed"
					analysis.GenerationDecisionReason = fmt.Sprintf("ApplicationSet selector %s matches cluster inventory %s and selects value files %s.", selectorString(appsetDoc.Selector), strings.Join(matched, ", "), strings.Join(analysis.SelectedValueFiles, ", "))
				}
			}
		} else {
			analysis.GenerationDecisionState = "unresolved"
			analysis.GenerationDecisionReason = fmt.Sprintf("ApplicationSet input %s was observed, but cub-gen could not parse it deterministically yet.", appsetPath)
		}
	}

	managedDocs := parseHelmManagedCatalogs(detection.Repo, managedCatalogPaths)
	customerDocs := parseHelmCustomerCatalogs(detection.Repo, customerCatalogPaths)
	requiredControl, requiredPath := firstRequiredHelmSecurityControl(managedDocs)
	selectedOverlay := helmPaths.OverlayValuesPath
	customerOverride := matchingCustomerCatalog(customerDocs, selectedOverlay)
	if requiredControl != "" {
		analysis.SecurityControl = requiredControl
		analysis.SecurityControlPath = requiredPath
		if customerOverride.Path != "" {
			analysis.SecurityOverridePath = customerOverride.Path
		}
		switch {
		case customerOverride.Enabled != nil && !*customerOverride.Enabled:
			analysis.SecurityDecisionState = "blocked"
			analysis.SecurityDecisionReason = fmt.Sprintf("Customer service catalog %s weakens the platform-required %s control from %s.", customerOverride.Path, requiredControl, requiredPath)
		case customerOverride.Enabled != nil:
			analysis.SecurityDecisionState = "allow"
			analysis.SecurityDecisionReason = fmt.Sprintf("Customer service catalog %s keeps the platform-required %s control enabled.", customerOverride.Path, requiredControl)
		default:
			analysis.SecurityDecisionState = "allow"
			analysis.SecurityDecisionReason = fmt.Sprintf("No customer override weakens the platform-required %s control from %s.", requiredControl, requiredPath)
		}
	} else if len(managedCatalogPaths) > 0 || len(customerCatalogPaths) > 0 {
		analysis.SecurityDecisionState = "unresolved"
		analysis.SecurityDecisionReason = "Layered service catalog inputs were observed, but cub-gen could not classify a supported security control from them yet."
	}

	return analysis
}

func helmDependencyChartsForGenerator(repoPath string, hints helmProvenancePaths) []model.HelmDependencyChart {
	deps := parseHelmDependencyDocs(repoPath, hints.ChartPath)
	out := make([]model.HelmDependencyChart, 0, len(deps))
	for _, dep := range deps {
		layer := strings.TrimSpace(dep.Alias)
		if layer == "" {
			layer = strings.TrimSpace(dep.Name)
		}
		out = append(out, model.HelmDependencyChart{
			Name:       dep.Name,
			Alias:      dep.Alias,
			Layer:      layer,
			Condition:  dep.Condition,
			Repository: dep.Repository,
			Version:    dep.Version,
			ChartPath:  dep.ChartPath,
			ValuesPath: dep.ValuesPath,
		})
	}
	return out
}

func parseHelmDependencyDocs(repoPath, chartPath string) []helmDependencyDoc {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(chartPath) == "" {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(chartPath)))
	if err != nil {
		return nil
	}

	var chartDoc struct {
		Dependencies []helmDependencyDoc `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(content, &chartDoc); err != nil {
		return nil
	}

	chartDir := helmChartDir(chartPath)
	out := make([]helmDependencyDoc, 0, len(chartDoc.Dependencies))
	for _, dep := range chartDoc.Dependencies {
		layer := strings.TrimSpace(dep.Alias)
		if layer == "" {
			layer = strings.TrimSpace(dep.Name)
		}
		chartCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", layer, "Chart.yaml"))
		valuesCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", layer, "values.yaml"))
		if !fileExists(filepath.Join(repoPath, filepath.FromSlash(chartCandidate))) && strings.TrimSpace(dep.Name) != "" && dep.Name != layer {
			fallbackChartCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", dep.Name, "Chart.yaml"))
			fallbackValuesCandidate := filepath.ToSlash(filepath.Join(chartDir, "charts", dep.Name, "values.yaml"))
			if fileExists(filepath.Join(repoPath, filepath.FromSlash(fallbackChartCandidate))) {
				chartCandidate = fallbackChartCandidate
				valuesCandidate = fallbackValuesCandidate
			}
		}
		if !fileExists(filepath.Join(repoPath, filepath.FromSlash(chartCandidate))) {
			chartCandidate = ""
		}
		if !fileExists(filepath.Join(repoPath, filepath.FromSlash(valuesCandidate))) {
			valuesCandidate = ""
		}
		dep.ChartPath = chartCandidate
		dep.ValuesPath = valuesCandidate
		out = append(out, dep)
	}

	sort.Slice(out, func(i, j int) bool {
		left := out[i].Alias
		if strings.TrimSpace(left) == "" {
			left = out[i].Name
		}
		right := out[j].Alias
		if strings.TrimSpace(right) == "" {
			right = out[j].Name
		}
		return left < right
	})
	return out
}

func helmCRDPathsForChart(repoPath, chartPath string) []string {
	chartDir := helmChartDir(chartPath)
	crdDir := filepath.Join(repoPath, filepath.FromSlash(chartDir), "crds")
	info, err := os.Stat(crdDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	out := make([]string, 0)
	_ = filepath.WalkDir(crdDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out
}

func helmHookTemplates(repoPath, chartPath string) []model.HelmHookTemplate {
	chartDir := helmChartDir(chartPath)
	templatesDir := filepath.Join(repoPath, filepath.FromSlash(chartDir), "templates")
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	out := make([]model.HelmHookTemplate, 0)
	_ = filepath.WalkDir(templatesDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		hooks := helmHookAnnotations(string(content))
		if len(hooks) == 0 {
			return nil
		}
		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		out = append(out, model.HelmHookTemplate{
			Path:  filepath.ToSlash(rel),
			Hooks: hooks,
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func helmHookAnnotations(content string) []string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "helm.sh/hook:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "helm.sh/hook:"))
		raw = strings.Trim(raw, "\"'")
		for _, hook := range strings.Split(raw, ",") {
			hook = strings.TrimSpace(hook)
			if hook == "" {
				continue
			}
			if _, ok := seen[hook]; ok {
				continue
			}
			seen[hook] = struct{}{}
			out = append(out, hook)
		}
	}
	sort.Strings(out)
	return out
}

func helmValuesSchemaPath(repoPath, chartPath string) string {
	chartDir := helmChartDir(chartPath)
	candidate := filepath.Join(repoPath, filepath.FromSlash(chartDir), "values.schema.json")
	if !fileExists(candidate) {
		return ""
	}
	rel, err := filepath.Rel(repoPath, candidate)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func helmValuesSchemaValidation(repoPath string, hints helmProvenancePaths, valuesSchemaPath string) (string, []string) {
	if strings.TrimSpace(valuesSchemaPath) == "" {
		return "", nil
	}

	schemaContent, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(valuesSchemaPath)))
	if err != nil {
		return "unresolved", []string{fmt.Sprintf("%s: %v", valuesSchemaPath, err)}
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(valuesSchemaPath, strings.NewReader(string(schemaContent))); err != nil {
		return "unresolved", []string{fmt.Sprintf("%s: %v", valuesSchemaPath, err)}
	}
	schema, err := compiler.Compile(valuesSchemaPath)
	if err != nil {
		return "unresolved", []string{fmt.Sprintf("%s: %v", valuesSchemaPath, err)}
	}

	rootValuesPaths := make([]string, 0, len(hints.ValuesPaths))
	for _, path := range hints.ValuesPaths {
		if helmIsSubchartPath(path) {
			continue
		}
		rootValuesPaths = append(rootValuesPaths, path)
	}
	sort.SliceStable(rootValuesPaths, func(i, j int) bool {
		leftRank := helmValuesPathPrecedence(rootValuesPaths[i], hints.PrimaryValuesPath, hints.ChartPath)
		rightRank := helmValuesPathPrecedence(rootValuesPaths[j], hints.PrimaryValuesPath, hints.ChartPath)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return rootValuesPaths[i] < rootValuesPaths[j]
	})

	merged := map[string]any{}
	// Merge from lowest to highest precedence so overlays win over chart defaults.
	for i := len(rootValuesPaths) - 1; i >= 0; i-- {
		path := rootValuesPaths[i]
		payload, err := yamlFileToJSONCompatible(filepath.Join(repoPath, filepath.FromSlash(path)))
		if err != nil {
			return "unresolved", []string{fmt.Sprintf("%s: %v", path, err)}
		}
		merged = mergeJSONObjects(merged, payload)
	}

	if err := schema.Validate(merged); err != nil {
		return "invalid", jsonschemaValidationMessages(err)
	}
	return "valid", nil
}

func yamlFileToJSONCompatible(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, err
	}
	converted := yamlToJSONCompatible(raw)
	if converted == nil {
		return map[string]any{}, nil
	}
	if asMap, ok := converted.(map[string]any); ok {
		return asMap, nil
	}
	return nil, fmt.Errorf("top-level values document must be a mapping")
}

func yamlToJSONCompatible(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = yamlToJSONCompatible(value)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[fmt.Sprint(key)] = yamlToJSONCompatible(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, value := range typed {
			out = append(out, yamlToJSONCompatible(value))
		}
		return out
	default:
		return typed
	}
}

func mergeJSONObjects(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		if baseMap, ok := out[key].(map[string]any); ok {
			if overlayMap, ok := value.(map[string]any); ok {
				out[key] = mergeJSONObjects(baseMap, overlayMap)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func jsonschemaValidationMessages(err error) []string {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			return nil
		}
		return []string{msg}
	}
	leaves := flattenJSONSchemaValidationErrors(validationErr)
	out := make([]string, 0, len(leaves))
	for _, leaf := range leaves {
		location := strings.TrimSpace(leaf.InstanceLocation)
		if location == "" {
			location = "/"
		}
		out = append(out, fmt.Sprintf("%s: %s", location, strings.TrimSpace(leaf.Message)))
	}
	sort.Strings(out)
	return out
}

func flattenJSONSchemaValidationErrors(err *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if err == nil {
		return nil
	}
	if len(err.Causes) == 0 {
		return []*jsonschema.ValidationError{err}
	}
	out := make([]*jsonschema.ValidationError, 0, len(err.Causes))
	for _, cause := range err.Causes {
		out = append(out, flattenJSONSchemaValidationErrors(cause)...)
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func parseHelmApplicationSetFile(repo, path string) (helmApplicationSetDoc, error) {
	type applicationSet struct {
		Spec struct {
			Generators []struct {
				Clusters struct {
					Selector struct {
						MatchLabels map[string]string `yaml:"matchLabels"`
					} `yaml:"selector"`
				} `yaml:"clusters"`
			} `yaml:"generators"`
			Template struct {
				Spec struct {
					Source struct {
						Helm struct {
							ValueFiles []string `yaml:"valueFiles"`
						} `yaml:"helm"`
					} `yaml:"source"`
				} `yaml:"spec"`
			} `yaml:"template"`
		} `yaml:"spec"`
	}

	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return helmApplicationSetDoc{}, err
	}

	var doc applicationSet
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return helmApplicationSetDoc{}, err
	}

	out := helmApplicationSetDoc{Path: filepath.ToSlash(path)}
	for _, generator := range doc.Spec.Generators {
		if len(generator.Clusters.Selector.MatchLabels) == 0 {
			continue
		}
		out.Selector = map[string]string{}
		for k, v := range generator.Clusters.Selector.MatchLabels {
			out.Selector[k] = v
		}
		break
	}
	out.ValueFiles = uniqueStringsInOrder(doc.Spec.Template.Spec.Source.Helm.ValueFiles)
	return out, nil
}

func matchedHelmClusterInventories(repo string, paths []string, selector map[string]string) []string {
	matched := make([]string, 0, len(paths))
	for _, path := range paths {
		doc, err := parseHelmClusterInventoryFile(repo, path)
		if err != nil || doc.Name == "" {
			continue
		}
		if clusterLabelsMatch(doc.Labels, selector) {
			matched = append(matched, doc.Name)
		}
	}
	return uniqueSortedStrings(matched)
}

func parseHelmClusterInventoryFile(repo, path string) (helmClusterInventoryDoc, error) {
	type clusterInventory struct {
		Metadata struct {
			Name   string            `yaml:"name"`
			Labels map[string]string `yaml:"labels"`
		} `yaml:"metadata"`
	}

	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return helmClusterInventoryDoc{}, err
	}

	var doc clusterInventory
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return helmClusterInventoryDoc{}, err
	}
	return helmClusterInventoryDoc{
		Path:   filepath.ToSlash(path),
		Name:   strings.TrimSpace(doc.Metadata.Name),
		Labels: doc.Metadata.Labels,
	}, nil
}

func parseHelmManagedCatalogs(repo string, paths []string) []helmManagedCatalogDoc {
	type managedCatalog struct {
		Spec struct {
			SecurityControls struct {
				OAuth2Proxy struct {
					Required bool `yaml:"required"`
				} `yaml:"oauth2Proxy"`
			} `yaml:"securityControls"`
		} `yaml:"spec"`
	}

	out := make([]helmManagedCatalogDoc, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			continue
		}
		var doc managedCatalog
		if err := yaml.Unmarshal(content, &doc); err != nil {
			continue
		}
		entry := helmManagedCatalogDoc{Path: filepath.ToSlash(path)}
		if doc.Spec.SecurityControls.OAuth2Proxy.Required {
			entry.RequiredSecurity = "oauth2Proxy"
		}
		out = append(out, entry)
	}
	return out
}

func parseHelmCustomerCatalogs(repo string, paths []string) []helmCustomerCatalogDoc {
	type customerCatalog struct {
		Spec struct {
			Overlay struct {
				ValuesFile string `yaml:"valuesFile"`
			} `yaml:"overlay"`
			SecurityOverrides struct {
				OAuth2Proxy struct {
					Enabled *bool `yaml:"enabled"`
				} `yaml:"oauth2Proxy"`
			} `yaml:"securityOverrides"`
		} `yaml:"spec"`
	}

	out := make([]helmCustomerCatalogDoc, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			continue
		}
		var doc customerCatalog
		if err := yaml.Unmarshal(content, &doc); err != nil {
			continue
		}
		entry := helmCustomerCatalogDoc{
			Path:       filepath.ToSlash(path),
			ValuesFile: strings.TrimSpace(doc.Spec.Overlay.ValuesFile),
			Enabled:    doc.Spec.SecurityOverrides.OAuth2Proxy.Enabled,
		}
		if doc.Spec.SecurityOverrides.OAuth2Proxy.Enabled != nil {
			entry.OverrideSecurity = "oauth2Proxy"
		}
		out = append(out, entry)
	}
	return out
}

func firstRequiredHelmSecurityControl(docs []helmManagedCatalogDoc) (string, string) {
	for _, doc := range docs {
		if strings.TrimSpace(doc.RequiredSecurity) == "" {
			continue
		}
		return doc.RequiredSecurity, doc.Path
	}
	return "", ""
}

func matchingCustomerCatalog(docs []helmCustomerCatalogDoc, selectedOverlay string) helmCustomerCatalogDoc {
	for _, doc := range docs {
		if selectedOverlay != "" && doc.ValuesFile == selectedOverlay {
			return doc
		}
	}
	if len(docs) > 0 {
		return docs[0]
	}
	return helmCustomerCatalogDoc{}
}

func clusterLabelsMatch(labels, selector map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return true
}

func selectorString(selector map[string]string) string {
	if len(selector) == 0 {
		return ""
	}
	keys := make([]string, 0, len(selector))
	for key := range selector {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+selector[key])
	}
	return strings.Join(parts, ", ")
}

func uniqueStringsInOrder(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func renderedLineageForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.RenderedObjectLineage {
	if g.Kind == model.GeneratorApplicationSet {
		analysis := applicationSetAnalysisForGenerator(detection, g)
		hints := applicationSetHintsFromInputs(detection.Repo, g.Inputs)
		lineage := []model.RenderedObjectLineage{{
			Kind:          "ApplicationSet",
			Name:          g.Name,
			Namespace:     "argocd",
			SourcePath:    hints.ApplicationSetPath,
			SourceDryPath: "spec",
		}}
		if analysis == nil {
			return lineage
		}
		for _, child := range analysis.GeneratedApplications {
			lineage = append(lineage, model.RenderedObjectLineage{
				Kind:          "Application",
				Name:          child.Name,
				Namespace:     "argocd",
				SourcePath:    hints.ApplicationSetPath,
				SourceDryPath: "spec.template.metadata.name",
			})
		}
		return lineage
	}

	templates := registry.RenderedLineageTemplates(g.Kind)
	if len(templates) == 0 {
		return nil
	}
	vars, singleHints, multiHints := lineageTemplateContext(detection, g)

	lineage := make([]model.RenderedObjectLineage, 0, len(templates))
	for i := 0; i < len(templates); i++ {
		tpl := templates[i]
		if tpl.SourcePathHintMulti {
			// Preserve legacy order for repeated-source templates:
			// emit all contiguous templates per source path before moving on.
			j := i + 1
			for j < len(templates) {
				next := templates[j]
				if !next.SourcePathHintMulti ||
					next.SourcePathHint != tpl.SourcePathHint ||
					next.SourcePathHintFallback != tpl.SourcePathHintFallback {
					break
				}
				j++
			}

			sourcePaths := lineageSourcePathsForTemplate(tpl, singleHints, multiHints)
			if len(sourcePaths) > 0 {
				for _, sourcePath := range sourcePaths {
					for k := i; k < j; k++ {
						groupTpl := templates[k]
						if strings.TrimSpace(sourcePath) == "" && groupTpl.Optional {
							continue
						}
						lineage = append(lineage, model.RenderedObjectLineage{
							Kind:          groupTpl.Kind,
							Name:          renderTargetTemplate(groupTpl.NameTemplate, vars),
							Namespace:     groupTpl.Namespace,
							SourcePath:    sourcePath,
							SourceDryPath: renderTargetTemplate(groupTpl.SourceDryPathTemplate, vars),
						})
					}
				}
			}

			i = j - 1
			continue
		}

		sourcePaths := lineageSourcePathsForTemplate(tpl, singleHints, multiHints)
		if len(sourcePaths) == 0 {
			continue
		}

		name := renderTargetTemplate(tpl.NameTemplate, vars)
		sourceDryPath := renderTargetTemplate(tpl.SourceDryPathTemplate, vars)
		for _, sourcePath := range sourcePaths {
			if strings.TrimSpace(sourcePath) == "" && tpl.Optional {
				continue
			}
			lineage = append(lineage, model.RenderedObjectLineage{
				Kind:          tpl.Kind,
				Name:          name,
				Namespace:     tpl.Namespace,
				SourcePath:    sourcePath,
				SourceDryPath: sourceDryPath,
			})
		}
	}
	return lineage
}

func lineageTemplateContext(detection model.DetectionResult, g model.GeneratorDetection) (map[string]string, map[string]string, map[string][]string) {
	vars := map[string]string{"name": g.Name}
	singleHints := map[string]string{}
	multiHints := map[string][]string{}

	switch g.Kind {
	case model.GeneratorHelm:
		helmPaths := helmProvenancePathsForGenerator(detection.Repo, g)
		singleHints["chart_path"] = helmPaths.ChartPath
		singleHints["primary_values_path"] = helmPaths.PrimaryValuesPath
		multiHints["values_paths"] = helmPaths.ValuesPaths
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		singleHints["source_path"] = hints.SourcePath
		vars["container_name"] = hints.ContainerName
		vars["service_port_name"] = hints.ServicePortName
	case model.GeneratorSpringBoot:
		hints := springPathHintsFromInputs(g.Inputs)
		singleHints["build_config_path"] = hints.BuildConfigPath
		singleHints["base_config_path"] = hints.BaseConfigPath
		singleHints["profile_config_path"] = hints.ProfileConfigPath
	case model.GeneratorBackstage:
		hints := backstagePathHintsFromInputs(g.Inputs)
		singleHints["catalog_path"] = hints.CatalogPath
	case model.GeneratorNoConfigPlatform:
		hints := noConfigPlatformPathHintsFromInputs(g.Inputs)
		singleHints["base_config_path"] = hints.BaseConfigPath
		singleHints["overlay_config_path"] = hints.OverlayConfigPath
	case model.GeneratorOpsFlow:
		hints := opsWorkflowPathHintsFromInputs(g.Inputs)
		singleHints["base_spec_path"] = hints.BaseSpecPath
		singleHints["overlay_spec_path"] = hints.OverlaySpecPath
	case model.GeneratorC3Agent:
		hints := c3agentPathHintsFromInputs(g.Inputs)
		singleHints["base_config_path"] = hints.BaseConfigPath
		singleHints["overlay_config_path"] = hints.OverlayConfigPath
	case model.GeneratorSwamp:
		hints := swampPathHintsFromInputs(g.Inputs)
		singleHints["base_config_path"] = hints.BaseConfigPath
		singleHints["workflow_path"] = hints.WorkflowPath
	}

	return vars, singleHints, multiHints
}

func lineageSourcePathsForTemplate(tpl registry.RenderedLineageTemplate, singleHints map[string]string, multiHints map[string][]string) []string {
	if tpl.SourcePathHintMulti {
		sourcePaths := append([]string(nil), multiHints[tpl.SourcePathHint]...)
		if len(sourcePaths) == 0 && tpl.SourcePathHintFallback != "" {
			if fallback := strings.TrimSpace(singleHints[tpl.SourcePathHintFallback]); fallback != "" {
				sourcePaths = []string{fallback}
			}
		}
		return sourcePaths
	}

	sourcePath := strings.TrimSpace(singleHints[tpl.SourcePathHint])
	if sourcePath == "" && tpl.SourcePathHintFallback != "" {
		sourcePath = strings.TrimSpace(singleHints[tpl.SourcePathHintFallback])
	}
	if sourcePath == "" && tpl.Optional {
		return nil
	}
	return []string{sourcePath}
}

type springHints struct {
	BuildConfigPath   string
	BaseConfigPath    string
	ProfileConfigPath string
}

func springPathHintsFromInputs(inputs []string) springHints {
	h := springHints{
		BuildConfigPath: registry.HintDefault(model.GeneratorSpringBoot, "build_config_path", "pom.xml"),
		BaseConfigPath:  registry.HintDefault(model.GeneratorSpringBoot, "base_config_path", "src/main/resources/application.yaml"),
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
			h.BaseConfigPath = registry.HintDefault(model.GeneratorSpringBoot, "base_config_path", "src/main/resources/application.yaml")
		}
	}
	return h
}

type backstageHints struct {
	CatalogPath   string
	AppConfigPath string
}

func backstagePathHintsFromInputs(inputs []string) backstageHints {
	h := backstageHints{
		CatalogPath: registry.HintDefault(model.GeneratorBackstage, "catalog_path", "catalog-info.yaml"),
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

type noConfigPlatformHints struct {
	BaseConfigPath    string
	OverlayConfigPath string
}

func noConfigPlatformPathHintsFromInputs(inputs []string) noConfigPlatformHints {
	h := noConfigPlatformHints{
		BaseConfigPath: registry.HintDefault(model.GeneratorNoConfigPlatform, "base_config_path", "no-config-platform.yaml"),
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch {
		case base == "no-config-platform.yaml" || base == "no-config-platform.yml" || base == "no-config-platform.json":
			h.BaseConfigPath = p
		case strings.HasPrefix(base, "no-config-platform-"):
			if h.OverlayConfigPath == "" || p < h.OverlayConfigPath {
				h.OverlayConfigPath = p
			}
		}
	}
	return h
}

type opsWorkflowHints struct {
	BaseSpecPath    string
	OverlaySpecPath string
}

func opsWorkflowPathHintsFromInputs(inputs []string) opsWorkflowHints {
	h := opsWorkflowHints{
		BaseSpecPath: registry.HintDefault(model.GeneratorOpsFlow, "base_spec_path", "operations.yaml"),
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

type c3agentHints struct {
	BaseConfigPath    string
	OverlayConfigPath string
}

func c3agentPathHintsFromInputs(inputs []string) c3agentHints {
	h := c3agentHints{
		BaseConfigPath: registry.HintDefault(model.GeneratorC3Agent, "base_config_path", "c3agent.yaml"),
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch {
		case base == "c3agent.yaml" || base == "c3agent.yml" || base == "c3agent.json":
			h.BaseConfigPath = p
		case strings.HasPrefix(base, "c3agent-"):
			if h.OverlayConfigPath == "" || p < h.OverlayConfigPath {
				h.OverlayConfigPath = p
			}
		}
	}
	return h
}

type opsWorkflowDoc struct {
	Path         string
	WorkflowName string
	Schedule     string
	ActionNames  []string
}

type opsExecutionPolicy struct {
	Path           string
	AllowedActions []string
	BlockedActions []string
	ApprovalGates  []string
}

func opsWorkflowAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.OpsWorkflowAnalysis {
	if g.Kind != model.GeneratorOpsFlow {
		return nil
	}

	workflowPaths := opsWorkflowPathsFromInputs(g.Inputs)
	if len(workflowPaths) == 0 {
		return nil
	}

	docs := make([]opsWorkflowDoc, 0, len(workflowPaths))
	for _, path := range workflowPaths {
		doc, err := parseOpsWorkflowFile(detection.Repo, path)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil
	}

	baseDoc := opsBaseWorkflowDoc(docs)
	if baseDoc.Path == "" {
		baseDoc = docs[0]
	}

	policyPath := opsExecutionPolicyPathFromRepo(detection.Repo, g.Inputs)
	policy := opsExecutionPolicy{}
	if policyPath != "" {
		if parsed, err := parseOpsExecutionPolicyFile(detection.Repo, policyPath); err == nil {
			policy = parsed
		}
	}

	workflowPathValues := make([]string, 0, len(docs))
	overlayPaths := make([]string, 0, len(docs))
	workflowNames := make([]string, 0, len(docs))
	schedules := make([]string, 0, len(docs))
	scheduleOverrides := make([]string, 0)
	actionSet := map[string]struct{}{}
	baseActionSet := map[string]struct{}{}
	addedActions := make([]string, 0)
	removedActions := make([]string, 0)

	for _, action := range baseDoc.ActionNames {
		baseActionSet[action] = struct{}{}
	}

	for _, doc := range docs {
		workflowPathValues = append(workflowPathValues, doc.Path)
		if doc.Path != baseDoc.Path {
			overlayPaths = append(overlayPaths, doc.Path)
		}
		if doc.WorkflowName != "" {
			workflowNames = append(workflowNames, doc.WorkflowName)
		}
		if doc.Schedule != "" {
			schedules = append(schedules, doc.Schedule)
		}
		if doc.Path != baseDoc.Path && doc.Schedule != "" && doc.Schedule != baseDoc.Schedule {
			scheduleOverrides = append(scheduleOverrides, doc.Path+":"+doc.Schedule)
		}
		for _, action := range doc.ActionNames {
			actionSet[action] = struct{}{}
		}
		if doc.Path != baseDoc.Path {
			addedActions = append(addedActions, differenceStrings(doc.ActionNames, baseDoc.ActionNames)...)
			removedActions = append(removedActions, differenceStrings(baseDoc.ActionNames, doc.ActionNames)...)
		}
	}

	actionNames := sortedStringSet(actionSet)
	allowedActions := uniqueSortedStrings(policy.AllowedActions)
	blockedActions := uniqueSortedStrings(policy.BlockedActions)
	unapprovedActions := differenceAgainstAllowList(actionNames, allowedActions)
	blockedActionsUsed := intersectionStrings(actionNames, blockedActions)

	return &model.OpsWorkflowAnalysis{
		WorkflowPaths:        uniqueSortedStrings(workflowPathValues),
		BaseWorkflowPath:     baseDoc.Path,
		OverlayWorkflowPaths: uniqueSortedStrings(overlayPaths),
		PolicyPath:           policy.Path,
		WorkflowNames:        uniqueSortedStrings(workflowNames),
		Schedules:            uniqueSortedStrings(schedules),
		ScheduleOverrides:    uniqueSortedStrings(scheduleOverrides),
		ActionNames:          actionNames,
		AllowedActions:       allowedActions,
		BlockedActions:       blockedActions,
		ApprovalGates:        uniqueSortedStrings(policy.ApprovalGates),
		UnapprovedActions:    unapprovedActions,
		BlockedActionsUsed:   blockedActionsUsed,
		AddedActions:         uniqueSortedStrings(addedActions),
		RemovedActions:       uniqueSortedStrings(removedActions),
	}
}

func opsWorkflowPathsFromInputs(inputs []string) []string {
	paths := make([]string, 0, len(inputs))
	for _, in := range inputs {
		base := strings.ToLower(filepath.Base(in))
		ext := strings.ToLower(filepath.Ext(base))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if !strings.HasPrefix(base, "operations") && !strings.HasPrefix(base, "workflow") {
			continue
		}
		paths = append(paths, filepath.ToSlash(in))
	}
	return uniqueSortedStrings(paths)
}

func opsBaseWorkflowDoc(docs []opsWorkflowDoc) opsWorkflowDoc {
	for _, doc := range docs {
		base := strings.ToLower(filepath.Base(doc.Path))
		if base == "operations.yaml" || base == "operations.yml" || base == "workflow.yaml" || base == "workflow.yml" {
			return doc
		}
	}
	return opsWorkflowDoc{}
}

func opsExecutionPolicyPathFromRepo(repo string, inputs []string) string {
	knownBasenames := map[string]struct{}{
		"execution-policy.yaml": {},
		"execution-policy.yml":  {},
		"workflow-policy.yaml":  {},
		"workflow-policy.yml":   {},
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		if _, ok := knownBasenames[base]; !ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(p))); err == nil {
			return p
		}
	}

	candidates := []string{
		"platform/execution-policy.yaml",
		"platform/execution-policy.yml",
		"execution-policy.yaml",
		"execution-policy.yml",
		"platform/workflow-policy.yaml",
		"platform/workflow-policy.yml",
		"workflow-policy.yaml",
		"workflow-policy.yml",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(candidate))); err == nil {
			return candidate
		}
	}
	return ""
}

func parseOpsWorkflowFile(repo, path string) (opsWorkflowDoc, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return opsWorkflowDoc{}, err
	}

	doc := opsWorkflowDoc{Path: filepath.ToSlash(path)}
	actionSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inWorkflow := false
	workflowIndent := 0
	inTriggers := false
	triggersIndent := 0
	inActions := false
	actionsIndent := 0

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if inWorkflow && indent <= workflowIndent && !strings.HasPrefix(trimmed, "workflow:") {
			inWorkflow = false
		}
		if inTriggers && indent <= triggersIndent && !strings.HasPrefix(trimmed, "triggers:") {
			inTriggers = false
		}
		if inActions && indent <= actionsIndent && !strings.HasPrefix(trimmed, "actions:") {
			inActions = false
		}

		if strings.HasPrefix(trimmed, "workflow:") {
			inWorkflow = true
			workflowIndent = indent
			continue
		}
		if strings.HasPrefix(trimmed, "triggers:") {
			inTriggers = true
			triggersIndent = indent
			continue
		}
		if strings.HasPrefix(trimmed, "actions:") {
			inActions = true
			actionsIndent = indent
			continue
		}

		if inWorkflow && strings.HasPrefix(trimmed, "name:") && indent >= workflowIndent+2 {
			doc.WorkflowName = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")))
			continue
		}
		if inTriggers && strings.HasPrefix(trimmed, "schedule:") && indent >= triggersIndent+2 {
			doc.Schedule = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "schedule:")))
			continue
		}
		if inActions && indent == actionsIndent+2 && strings.HasSuffix(trimmed, ":") {
			action := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			if action != "" {
				actionSet[action] = struct{}{}
			}
			continue
		}
	}

	doc.ActionNames = sortedStringSet(actionSet)
	return doc, nil
}

func parseOpsExecutionPolicyFile(repo, path string) (opsExecutionPolicy, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return opsExecutionPolicy{}, err
	}

	policy := opsExecutionPolicy{Path: filepath.ToSlash(path)}
	allowedSet := map[string]struct{}{}
	blockedSet := map[string]struct{}{}
	approvalSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inSpec := false
	specIndent := 0
	mode := ""
	currentApprovalEnv := ""
	currentApprovalCount := ""

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if strings.HasPrefix(trimmed, "spec:") {
			inSpec = true
			specIndent = indent
			mode = ""
			currentApprovalEnv = ""
			currentApprovalCount = ""
			continue
		}
		if !inSpec {
			continue
		}
		if indent <= specIndent {
			inSpec = false
			mode = ""
			currentApprovalEnv = ""
			currentApprovalCount = ""
			continue
		}

		if indent == specIndent+2 && strings.Contains(trimmed, ":") {
			if mode == "approval_gates" && currentApprovalEnv != "" && currentApprovalCount != "" {
				approvalSet[currentApprovalEnv+":"+currentApprovalCount] = struct{}{}
			}
			parts := strings.SplitN(trimmed, ":", 2)
			mode = strings.TrimSpace(parts[0])
			value := ""
			if len(parts) == 2 {
				value = strings.TrimSpace(parts[1])
			}
			currentApprovalEnv = ""
			currentApprovalCount = ""

			switch mode {
			case "allowed_actions":
				for _, item := range parseYAMLInlineList(value) {
					allowedSet[item] = struct{}{}
				}
			case "blocked_actions":
				for _, item := range parseYAMLInlineList(value) {
					blockedSet[item] = struct{}{}
				}
			}
			continue
		}

		switch mode {
		case "allowed_actions":
			if strings.HasPrefix(trimmed, "- ") {
				value := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if value != "" {
					allowedSet[value] = struct{}{}
				}
			}
		case "blocked_actions":
			if strings.HasPrefix(trimmed, "- ") {
				value := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if value != "" {
					blockedSet[value] = struct{}{}
				}
			}
		case "approval_gates":
			if indent == specIndent+4 && strings.HasSuffix(trimmed, ":") {
				if currentApprovalEnv != "" && currentApprovalCount != "" {
					approvalSet[currentApprovalEnv+":"+currentApprovalCount] = struct{}{}
				}
				currentApprovalEnv = parseYAMLScalar(strings.TrimSuffix(trimmed, ":"))
				currentApprovalCount = ""
				continue
			}
			if currentApprovalEnv != "" && strings.HasPrefix(trimmed, "required_approvals:") {
				currentApprovalCount = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "required_approvals:")))
			}
		}
	}
	if currentApprovalEnv != "" && currentApprovalCount != "" {
		approvalSet[currentApprovalEnv+":"+currentApprovalCount] = struct{}{}
	}

	policy.AllowedActions = sortedStringSet(allowedSet)
	policy.BlockedActions = sortedStringSet(blockedSet)
	policy.ApprovalGates = sortedStringSet(approvalSet)
	return policy, nil
}

type swampHints struct {
	BaseConfigPath string
	WorkflowPath   string
}

func swampPathHintsFromInputs(inputs []string) swampHints {
	h := swampHints{
		BaseConfigPath: registry.HintDefault(model.GeneratorSwamp, "base_config_path", ".swamp.yaml"),
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch {
		case base == ".swamp.yaml" || base == ".swamp.yml" || base == ".swamp.json":
			h.BaseConfigPath = p
		case strings.HasPrefix(base, "workflow-"):
			if h.WorkflowPath == "" || p < h.WorkflowPath {
				h.WorkflowPath = p
			}
		}
	}
	return h
}

type swampWorkflowDoc struct {
	Path            string
	StepNames       []string
	ModelRefs       []string
	MethodRefs      []string
	ModelMethodRefs []string
	JobStepCounts   map[string]int
}

type swampPolicy struct {
	Path                 string
	ApprovedModels       []string
	ApprovedModelMethods []string
	RequiredSteps        []string
	ForbiddenStepNames   []string
	MaxStepsPerJob       int
	MaxParallelJobs      int
}

func swampWorkflowAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.SwampWorkflowAnalysis {
	if g.Kind != model.GeneratorSwamp {
		return nil
	}

	workflowPaths := swampWorkflowPathsFromInputs(g.Inputs)
	if len(workflowPaths) == 0 {
		return nil
	}

	docs := make([]swampWorkflowDoc, 0, len(workflowPaths))
	for _, path := range workflowPaths {
		doc, err := parseSwampWorkflowFile(detection.Repo, path)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil
	}

	policyPath := swampPolicyPathFromRepo(detection.Repo, g.Inputs)
	policy := swampPolicy{}
	if policyPath != "" {
		if parsed, err := parseSwampPolicyFile(detection.Repo, policyPath); err == nil {
			policy = parsed
		}
	}

	stepSet := map[string]struct{}{}
	modelSet := map[string]struct{}{}
	methodSet := map[string]struct{}{}
	modelMethodSet := map[string]struct{}{}
	jobsExceedingMax := make([]string, 0)

	totalJobs := 0
	totalSteps := 0
	maxJobsInWorkflow := 0

	for _, doc := range docs {
		totalJobs += len(doc.JobStepCounts)
		totalSteps += len(doc.StepNames)
		if len(doc.JobStepCounts) > maxJobsInWorkflow {
			maxJobsInWorkflow = len(doc.JobStepCounts)
		}

		for _, step := range doc.StepNames {
			stepSet[step] = struct{}{}
		}
		for _, modelRef := range doc.ModelRefs {
			modelSet[modelRef] = struct{}{}
		}
		for _, methodRef := range doc.MethodRefs {
			methodSet[methodRef] = struct{}{}
		}
		for _, modelMethodRef := range doc.ModelMethodRefs {
			modelMethodSet[modelMethodRef] = struct{}{}
		}

		if policy.MaxStepsPerJob > 0 {
			jobNames := make([]string, 0, len(doc.JobStepCounts))
			for jobName := range doc.JobStepCounts {
				jobNames = append(jobNames, jobName)
			}
			sort.Strings(jobNames)
			for _, jobName := range jobNames {
				stepCount := doc.JobStepCounts[jobName]
				if stepCount <= policy.MaxStepsPerJob {
					continue
				}
				jobsExceedingMax = append(
					jobsExceedingMax,
					fmt.Sprintf("%s:%s(%d)", doc.Path, jobName, stepCount),
				)
			}
		}
	}

	stepNames := sortedStringSet(stepSet)
	modelRefs := sortedStringSet(modelSet)
	methodRefs := sortedStringSet(methodSet)
	modelMethodRefs := sortedStringSet(modelMethodSet)

	requiredSteps := uniqueSortedStrings(policy.RequiredSteps)
	approvedModels := uniqueSortedStrings(policy.ApprovedModels)
	approvedModelMethods := uniqueSortedStrings(policy.ApprovedModelMethods)
	forbiddenStepNames := uniqueSortedStrings(policy.ForbiddenStepNames)
	jobsExceedingMax = uniqueSortedStrings(jobsExceedingMax)

	missingRequiredSteps := differenceStrings(requiredSteps, stepNames)
	forbiddenStepsPresent := intersectionStrings(stepNames, forbiddenStepNames)
	unapprovedModels := differenceAgainstAllowList(modelRefs, approvedModels)
	unapprovedModelMethods := differenceAgainstAllowList(modelMethodRefs, approvedModelMethods)

	baseDoc := docs[0]
	addedSteps := make([]string, 0)
	removedSteps := make([]string, 0)
	addedModelMethods := make([]string, 0)
	removedModelMethods := make([]string, 0)
	for _, doc := range docs[1:] {
		addedSteps = append(addedSteps, differenceStrings(doc.StepNames, baseDoc.StepNames)...)
		removedSteps = append(removedSteps, differenceStrings(baseDoc.StepNames, doc.StepNames)...)
		addedModelMethods = append(addedModelMethods, differenceStrings(doc.ModelMethodRefs, baseDoc.ModelMethodRefs)...)
		removedModelMethods = append(removedModelMethods, differenceStrings(baseDoc.ModelMethodRefs, doc.ModelMethodRefs)...)
	}

	workflowPathValues := make([]string, 0, len(docs))
	for _, doc := range docs {
		workflowPathValues = append(workflowPathValues, doc.Path)
	}

	return &model.SwampWorkflowAnalysis{
		WorkflowPaths:          uniqueSortedStrings(workflowPathValues),
		BaseWorkflowPath:       baseDoc.Path,
		PolicyPath:             policy.Path,
		StepNames:              stepNames,
		ModelRefs:              modelRefs,
		MethodRefs:             methodRefs,
		ModelMethodRefs:        modelMethodRefs,
		ApprovedModels:         approvedModels,
		ApprovedModelMethods:   approvedModelMethods,
		RequiredSteps:          requiredSteps,
		MissingRequiredSteps:   missingRequiredSteps,
		UnapprovedModels:       unapprovedModels,
		UnapprovedModelMethods: unapprovedModelMethods,
		ForbiddenStepNames:     forbiddenStepNames,
		ForbiddenStepsPresent:  forbiddenStepsPresent,
		MaxStepsPerJob:         policy.MaxStepsPerJob,
		MaxParallelJobs:        policy.MaxParallelJobs,
		TotalJobs:              totalJobs,
		TotalSteps:             totalSteps,
		JobsExceedingMaxSteps:  jobsExceedingMax,
		ExceedsMaxParallelJobs: policy.MaxParallelJobs > 0 && maxJobsInWorkflow > policy.MaxParallelJobs,
		AddedSteps:             uniqueSortedStrings(addedSteps),
		RemovedSteps:           uniqueSortedStrings(removedSteps),
		AddedModelMethodRefs:   uniqueSortedStrings(addedModelMethods),
		RemovedModelMethodRefs: uniqueSortedStrings(removedModelMethods),
	}
}

func swampWorkflowPathsFromInputs(inputs []string) []string {
	paths := make([]string, 0, len(inputs))
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		ext := strings.ToLower(filepath.Ext(base))
		if !strings.HasPrefix(base, "workflow-") {
			continue
		}
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		paths = append(paths, p)
	}
	return uniqueSortedStrings(paths)
}

func swampPolicyPathFromRepo(repo string, inputs []string) string {
	knownBasenames := map[string]struct{}{
		"swamp-constraints.yaml": {},
		"swamp-constraints.yml":  {},
		"workflow-policy.yaml":   {},
		"workflow-policy.yml":    {},
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		if _, ok := knownBasenames[base]; !ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(p))); err == nil {
			return p
		}
	}

	candidates := []string{
		"platform/swamp-constraints.yaml",
		"platform/swamp-constraints.yml",
		"swamp-constraints.yaml",
		"swamp-constraints.yml",
		"platform/workflow-policy.yaml",
		"platform/workflow-policy.yml",
		"workflow-policy.yaml",
		"workflow-policy.yml",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(candidate))); err == nil {
			return candidate
		}
	}
	return ""
}

func parseSwampWorkflowFile(repo, path string) (swampWorkflowDoc, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return swampWorkflowDoc{}, err
	}

	doc := swampWorkflowDoc{
		Path:          filepath.ToSlash(path),
		JobStepCounts: map[string]int{},
	}
	stepSet := map[string]struct{}{}
	modelSet := map[string]struct{}{}
	methodSet := map[string]struct{}{}
	modelMethodSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inJobs := false
	jobsIndent := 0
	inSteps := false
	stepsIndent := 0
	currentJob := ""
	inTask := false
	taskIndent := 0
	taskModel := ""
	taskMethod := ""

	finalizeTask := func() {
		if taskModel != "" && taskMethod != "" {
			modelMethodSet[taskModel+"."+taskMethod] = struct{}{}
		}
		taskModel = ""
		taskMethod = ""
	}

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if inTask && indent <= taskIndent && !strings.HasPrefix(trimmed, "task:") {
			finalizeTask()
			inTask = false
		}
		if inSteps && indent <= stepsIndent && !strings.HasPrefix(trimmed, "steps:") {
			inSteps = false
		}
		if inJobs && indent <= jobsIndent && !strings.HasPrefix(trimmed, "jobs:") {
			inJobs = false
			currentJob = ""
		}

		if strings.HasPrefix(trimmed, "jobs:") {
			inJobs = true
			jobsIndent = indent
			continue
		}

		if inJobs && indent == jobsIndent+2 && strings.HasPrefix(trimmed, "- name:") {
			currentJob = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))
			if currentJob == "" {
				currentJob = fmt.Sprintf("job-%d", len(doc.JobStepCounts)+1)
			}
			if _, ok := doc.JobStepCounts[currentJob]; !ok {
				doc.JobStepCounts[currentJob] = 0
			}
			continue
		}

		if inJobs && strings.HasPrefix(trimmed, "steps:") {
			inSteps = true
			stepsIndent = indent
			continue
		}

		if inSteps {
			if strings.HasPrefix(trimmed, "- name:") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))
				if step != "" {
					stepSet[step] = struct{}{}
					if currentJob != "" {
						doc.JobStepCounts[currentJob]++
					}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "name:") && indent >= stepsIndent+2 {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")))
				if step != "" {
					stepSet[step] = struct{}{}
					if currentJob != "" {
						doc.JobStepCounts[currentJob]++
					}
				}
				continue
			}
		}

		if strings.HasPrefix(trimmed, "task:") {
			inTask = true
			taskIndent = indent
			taskModel = ""
			taskMethod = ""
			continue
		}

		if inTask && strings.HasPrefix(trimmed, "modelIdOrName:") {
			modelRef := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "modelIdOrName:")))
			if modelRef != "" {
				modelSet[modelRef] = struct{}{}
				taskModel = modelRef
			}
			continue
		}
		if inTask && strings.HasPrefix(trimmed, "methodName:") {
			methodRef := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "methodName:")))
			if methodRef != "" {
				methodSet[methodRef] = struct{}{}
				taskMethod = methodRef
			}
			continue
		}
	}

	if inTask {
		finalizeTask()
	}

	doc.StepNames = sortedStringSet(stepSet)
	doc.ModelRefs = sortedStringSet(modelSet)
	doc.MethodRefs = sortedStringSet(methodSet)
	doc.ModelMethodRefs = sortedStringSet(modelMethodSet)
	return doc, nil
}

func parseSwampPolicyFile(repo, path string) (swampPolicy, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return swampPolicy{}, err
	}

	policy := swampPolicy{
		Path: filepath.ToSlash(path),
	}
	approvedModelSet := map[string]struct{}{}
	approvedModelMethodSet := map[string]struct{}{}
	requiredStepSet := map[string]struct{}{}
	forbiddenStepSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inSpec := false
	specIndent := 0
	mode := ""
	currentModelMethodsKey := ""

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if strings.HasPrefix(trimmed, "spec:") {
			inSpec = true
			specIndent = indent
			mode = ""
			currentModelMethodsKey = ""
			continue
		}
		if !inSpec {
			continue
		}
		if indent <= specIndent {
			inSpec = false
			mode = ""
			currentModelMethodsKey = ""
			continue
		}

		if indent == specIndent+2 && strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			mode = strings.TrimSpace(parts[0])
			value := ""
			if len(parts) == 2 {
				value = strings.TrimSpace(parts[1])
			}
			currentModelMethodsKey = ""

			switch mode {
			case "max_steps_per_job":
				if parsed, parseErr := strconv.Atoi(parseYAMLScalar(value)); parseErr == nil {
					policy.MaxStepsPerJob = parsed
				}
			case "max_parallel_jobs":
				if parsed, parseErr := strconv.Atoi(parseYAMLScalar(value)); parseErr == nil {
					policy.MaxParallelJobs = parsed
				}
			case "approved_models":
				for _, item := range parseYAMLInlineList(value) {
					if item == "" {
						continue
					}
					approvedModelSet[item] = struct{}{}
				}
			case "forbidden_step_names":
				for _, item := range parseYAMLInlineList(value) {
					if item == "" {
						continue
					}
					forbiddenStepSet[item] = struct{}{}
				}
			case "required_steps":
				for _, item := range parseYAMLInlineList(value) {
					if item == "" {
						continue
					}
					requiredStepSet[item] = struct{}{}
				}
			}
			continue
		}

		switch mode {
		case "approved_models":
			if strings.HasPrefix(trimmed, "- ") {
				modelRef := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if modelRef != "" {
					approvedModelSet[modelRef] = struct{}{}
				}
			}
		case "approved_model_methods":
			if indent == specIndent+4 && strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				currentModelMethodsKey = parseYAMLScalar(parts[0])
				inline := ""
				if len(parts) == 2 {
					inline = strings.TrimSpace(parts[1])
				}
				for _, method := range parseYAMLInlineList(inline) {
					if currentModelMethodsKey == "" || method == "" {
						continue
					}
					approvedModelMethodSet[currentModelMethodsKey+"."+method] = struct{}{}
				}
				continue
			}
			if indent >= specIndent+6 && strings.HasPrefix(trimmed, "- ") && currentModelMethodsKey != "" {
				method := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if method != "" {
					approvedModelMethodSet[currentModelMethodsKey+"."+method] = struct{}{}
				}
			}
		case "required_steps":
			if strings.HasPrefix(trimmed, "- name:") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))
				if step != "" {
					requiredStepSet[step] = struct{}{}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "name:") && indent >= specIndent+4 {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")))
				if step != "" {
					requiredStepSet[step] = struct{}{}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "- ") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if step != "" {
					requiredStepSet[step] = struct{}{}
				}
			}
		case "forbidden_step_names":
			if strings.HasPrefix(trimmed, "- ") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if step != "" {
					forbiddenStepSet[step] = struct{}{}
				}
			}
		}
	}

	policy.ApprovedModels = sortedStringSet(approvedModelSet)
	policy.ApprovedModelMethods = sortedStringSet(approvedModelMethodSet)
	policy.RequiredSteps = sortedStringSet(requiredStepSet)
	policy.ForbiddenStepNames = sortedStringSet(forbiddenStepSet)
	return policy, nil
}

func parseYAMLScalar(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return strings.TrimSpace(value)
}

func parseYAMLInlineList(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil
	}
	body := strings.TrimSpace(value[1 : len(value)-1])
	if body == "" {
		return nil
	}
	parts := strings.Split(body, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := parseYAMLScalar(part)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return uniqueSortedStrings(items)
}

func sortedStringSet(in map[string]struct{}) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for value := range in {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return sortedStringSet(set)
}

func differenceStrings(values, baseline []string) []string {
	if len(values) == 0 {
		return nil
	}
	baselineSet := map[string]struct{}{}
	for _, v := range baseline {
		baselineSet[strings.TrimSpace(v)] = struct{}{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := baselineSet[key]; ok {
			continue
		}
		out = append(out, key)
	}
	return uniqueSortedStrings(out)
}

func intersectionStrings(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	setB := map[string]struct{}{}
	for _, value := range b {
		setB[strings.TrimSpace(value)] = struct{}{}
	}
	out := make([]string, 0, len(a))
	for _, value := range a {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := setB[key]; ok {
			out = append(out, key)
		}
	}
	return uniqueSortedStrings(out)
}

func differenceAgainstAllowList(observed, allowList []string) []string {
	if len(observed) == 0 || len(allowList) == 0 {
		return nil
	}
	allowedSet := map[string]struct{}{}
	for _, value := range allowList {
		allowedSet[strings.TrimSpace(value)] = struct{}{}
	}
	out := make([]string, 0, len(observed))
	for _, value := range observed {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := allowedSet[key]; ok {
			continue
		}
		out = append(out, key)
	}
	return uniqueSortedStrings(out)
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
