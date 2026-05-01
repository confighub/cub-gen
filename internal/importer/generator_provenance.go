package importer

import (
	"fmt"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

func chainForGenerator(detection model.DetectionResult, g model.GeneratorDetection) (model.GeneratorChain, bool) {
	for _, chain := range detection.Chains {
		if len(chain.Stages) == 0 {
			continue
		}
		downstream := chain.Stages[len(chain.Stages)-1]
		if downstream.Kind == g.Kind && strings.TrimSpace(downstream.Root) == strings.TrimSpace(g.Root) {
			return chain, true
		}
	}
	return model.GeneratorChain{}, false
}

func chainStageForMapping(chain model.GeneratorChain, mapping model.GeneratorChainMapping) model.GeneratorChainStage {
	for _, stage := range chain.Stages {
		if mapping.UpstreamKind != "" && stage.Kind != mapping.UpstreamKind {
			continue
		}
		if strings.TrimSpace(mapping.UpstreamRoot) != "" && strings.TrimSpace(stage.Root) != strings.TrimSpace(mapping.UpstreamRoot) {
			continue
		}
		return stage
	}
	if len(chain.Stages) > 0 {
		return chain.Stages[0]
	}
	return model.GeneratorChainStage{}
}

func chainMappingForOrigin(chain model.GeneratorChain, wetPath, dryPath string) (model.GeneratorChainMapping, bool) {
	for _, mapping := range chain.Mappings {
		if mapping.DownstreamWetPath != wetPath {
			continue
		}
		if strings.TrimSpace(mapping.DownstreamDryPath) != "" && mapping.DownstreamDryPath != dryPath {
			continue
		}
		return mapping, true
	}
	return model.GeneratorChainMapping{}, false
}

func chainHops(upstream model.GeneratorChainStage, mapping model.GeneratorChainMapping, downstream model.GeneratorDetection, downstreamOrigin model.FieldOrigin) []model.FieldOriginHop {
	upstreamTransform := mapping.UpstreamTransform
	if strings.TrimSpace(upstreamTransform) == "" {
		upstreamTransform = generatorChainTransform
	}
	upstreamConfidence := mapping.UpstreamConfidence
	if upstreamConfidence == 0 {
		upstreamConfidence = upstream.Confidence
	}
	return []model.FieldOriginHop{
		{
			GeneratorKind:    string(upstream.Kind),
			GeneratorProfile: upstream.Profile,
			DryPath:          mapping.UpstreamDryPath,
			SourcePath:       mapping.UpstreamSourcePath,
			Transform:        upstreamTransform,
			Confidence:       upstreamConfidence,
		},
		{
			GeneratorKind:    string(downstream.Kind),
			GeneratorProfile: downstream.Profile,
			DryPath:          downstreamOrigin.DryPath,
			SourcePath:       downstreamOrigin.SourcePath,
			Transform:        downstreamOrigin.Transform,
			Confidence:       downstreamOrigin.Confidence,
		},
	}
}

func applyGeneratorChainOrigins(detection model.DetectionResult, g model.GeneratorDetection, origins []model.FieldOrigin) []model.FieldOrigin {
	chain, ok := chainForGenerator(detection, g)
	if !ok || len(chain.Mappings) == 0 {
		return origins
	}

	updated := make([]model.FieldOrigin, 0, len(origins))
	for _, origin := range origins {
		mapping, matched := chainMappingForOrigin(chain, origin.WetPath, origin.DryPath)
		if !matched {
			updated = append(updated, origin)
			continue
		}
		downstreamOrigin := origin
		upstream := chainStageForMapping(chain, mapping)
		upstreamTransform := mapping.UpstreamTransform
		if strings.TrimSpace(upstreamTransform) == "" {
			upstreamTransform = generatorChainTransform
		}
		upstreamConfidence := mapping.UpstreamConfidence
		if upstreamConfidence == 0 {
			upstreamConfidence = upstream.Confidence
		}
		origin.DryPath = mapping.UpstreamDryPath
		origin.SourcePath = mapping.UpstreamSourcePath
		origin.Transform = upstreamTransform
		origin.Confidence = origin.Confidence * upstreamConfidence
		origin.Hops = chainHops(upstream, mapping, g, downstreamOrigin)
		updated = append(updated, origin)
	}
	return updated
}

func applyGeneratorChainPointers(detection model.DetectionResult, g model.GeneratorDetection, pointers []model.InverseEditPointer) []model.InverseEditPointer {
	chain, ok := chainForGenerator(detection, g)
	if !ok || len(chain.Mappings) == 0 {
		return pointers
	}

	updated := make([]model.InverseEditPointer, 0, len(pointers))
	for _, pointer := range pointers {
		mapping, matched := chainMappingForOrigin(chain, pointer.WetPath, pointer.DryPath)
		if !matched {
			updated = append(updated, pointer)
			continue
		}
		upstream := chainStageForMapping(chain, mapping)
		upstreamConfidence := mapping.UpstreamConfidence
		if upstreamConfidence == 0 {
			upstreamConfidence = upstream.Confidence
		}
		pointer.DryPath = mapping.UpstreamDryPath
		if strings.TrimSpace(mapping.UpstreamOwner) != "" {
			pointer.Owner = mapping.UpstreamOwner
		}
		pointer.EditHint = fmt.Sprintf(
			"Edit %s in %s. This value flows through %s -> %s before reaching the rendered manifest.",
			mapping.UpstreamDryPath,
			mapping.UpstreamSourcePath,
			string(upstream.Kind),
			string(g.Kind),
		)
		pointer.Confidence = pointer.Confidence * upstreamConfidence
		updated = append(updated, pointer)
	}
	return updated
}

func fieldOriginsForGenerator(detection model.DetectionResult, g model.GeneratorDetection, helmCLIOverrides []model.HelmCLIOverride) []model.FieldOrigin {
	switch g.Kind {
	case model.GeneratorHelm:
		hints := helmProvenancePathsForGenerator(detection.Repo, g)
		imageTagSources := helmImageTagSources(detection.Repo, hints)
		helperChain := helmImageTagUsesHelperChain(detection.Repo)
		overrideOrigins := helmCLIOverrideFieldOrigins(g, helmCLIOverrides)
		origins := make([]model.FieldOrigin, 0, len(overrideOrigins)+len(imageTagSources)+1)
		origins = append(origins, overrideOrigins...)
		if len(imageTagSources) == 0 {
			if nondeterministicOrigin, ok := helmImageTagNonDeterministicOrigin(detection.Repo); ok {
				origins = append(origins, nondeterministicOrigin)
				return origins
			}
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
			if helperChain {
				transform = helmHelperTransform
				confidenceKey = ""
				confidenceFallback = 1.0
			}
			if source.External {
				transform = helmExternalTransform
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
		return applyGeneratorChainOrigins(detection, g, origins)
	case model.GeneratorApplicationSet:
		hints := applicationSetHintsFromInputs(detection.Repo, g.Inputs)
		origins := []model.FieldOrigin{{
			DryPath:    "spec.template.metadata.name",
			WetPath:    "Application/metadata/name",
			SourcePath: hints.ApplicationSetPath,
			Transform:  registry.FieldOriginTransform(g.Kind),
			Confidence: registry.FieldOriginConfidenceFor(g.Kind, "child_name", 0.89),
		}}
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
	case model.GeneratorAppOfApps:
		hints := appOfAppsPathHintsFromInputs(g.Inputs)
		analysis := appOfAppsAnalysisForGenerator(detection, g)
		origins := []model.FieldOrigin{{
			DryPath:    "spec.source.path",
			WetPath:    "RootApplication/spec/source/path",
			SourcePath: hints.RootApplicationPath,
			Transform:  registry.FieldOriginOverlayTransform(g.Kind),
			Confidence: registry.FieldOriginConfidenceFor(g.Kind, "root_path", 0.88),
		}}
		if analysis == nil || len(analysis.GeneratedApplications) == 0 {
			return origins
		}
		for _, child := range analysis.GeneratedApplications {
			prefix := fmt.Sprintf("Application[name=%s]", child.Name)
			origins = append(origins,
				model.FieldOrigin{
					DryPath:    "metadata.name",
					WetPath:    prefix + "/metadata/name",
					SourcePath: child.Path,
					Transform:  registry.FieldOriginTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "child_name", 0.92),
				},
				model.FieldOrigin{
					DryPath:    "spec.source.repoURL",
					WetPath:    prefix + "/spec/source/repoURL",
					SourcePath: child.Path,
					Transform:  registry.FieldOriginTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "source_repo", 0.90),
				},
				model.FieldOrigin{
					DryPath:    "spec.source.path",
					WetPath:    prefix + "/spec/source/path",
					SourcePath: child.Path,
					Transform:  registry.FieldOriginTransform(g.Kind),
					Confidence: registry.FieldOriginConfidenceFor(g.Kind, "source_path", 0.90),
				},
			)
		}
		return origins
	case model.GeneratorScore:
		hints := scorePathHintsFromInputs(detection.Repo, g.Inputs)
		return applyGeneratorChainOrigins(detection, g, []model.FieldOrigin{
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
		})
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
	case model.GeneratorOpenChoreo:
		hints := openChoreoPathHintsFromInputs(g.Inputs)
		return []model.FieldOrigin{
			{
				DryPath:    "spec.containers.main.image",
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/image",
				SourcePath: hints.WorkloadPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "image", 0.88),
			},
			{
				DryPath:    "spec.environment.env.LOG_LEVEL",
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/env[name=LOG_LEVEL]/value",
				SourcePath: hints.ReleaseBindingPath,
				Transform:  registry.FieldOriginOverlayTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "env_var", 0.84),
			},
			{
				DryPath:    "spec.service.port",
				WetPath:    "Service/spec/ports[name=http]/port",
				SourcePath: hints.ComponentTypePath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "service_port", 0.82),
			},
			{
				DryPath:    "spec.secretRef",
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/env[name=DATABASE_URL]/valueFrom/secretKeyRef/name",
				SourcePath: hints.SecretReferencePath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "secret_ref", 0.86),
			},
			{
				DryPath:    "spec.resources.limits.cpu",
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/resources/limits/cpu",
				SourcePath: hints.ReleaseBindingPath,
				Transform:  registry.FieldOriginOverlayTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "resource_limit", 0.80),
			},
			{
				DryPath:    "spec.containers.main.files.LOG_FORMAT.value",
				WetPath:    "ConfigMap/data/LOG_FORMAT",
				SourcePath: hints.WorkloadPath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "mounted_file", 0.83),
			},
			{
				DryPath:    "spec.runtime.defaults.securityContext.runAsNonRoot",
				WetPath:    "Deployment/spec/template/spec/securityContext/runAsNonRoot",
				SourcePath: hints.ComponentTypePath,
				Transform:  registry.FieldOriginTransform(g.Kind),
				Confidence: registry.FieldOriginConfidenceFor(g.Kind, "platform_default", 0.78),
			},
		}
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
			if nondeterministicOrigin, ok := helmImageTagNonDeterministicOrigin(detection.Repo); ok {
				return []model.InverseEditPointer{{
					WetPath:    "Deployment/spec/template/spec/containers[0]/image",
					DryPath:    "values.image.tag",
					Owner:      "platform-engineer",
					EditHint:   "This field currently comes from Helm render-time logic like lookup/now/randAlphaNum/uuidv4. Edit the chart template or helper chain instead of values.yaml.",
					Confidence: nondeterministicOrigin.Confidence,
				}}
			}
			if _, ok := helmImageTagBuiltinOrigin(detection.Repo, hints); ok {
				return []model.InverseEditPointer{{
					WetPath:    "Deployment/spec/template/spec/containers[0]/image",
					DryPath:    "values.image.tag",
					Owner:      "platform-engineer",
					EditHint:   "This field currently comes from Helm built-in .Chart.AppVersion. Edit appVersion in Chart.yaml or pass --set image.tag=... for an invocation-specific override.",
					Confidence: 1.0,
				}}
			}
			return []model.InverseEditPointer{{
				WetPath:    "Deployment/spec/template/spec/containers[0]/image",
				DryPath:    "values.image.tag",
				Owner:      "platform-engineer",
				EditHint:   "This field is not set in the observed Helm values files. Inspect Chart.yaml, templates/, and helper defaults before editing values.yaml.",
				Confidence: 0.40,
			}}
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
		return applyGeneratorChainPointers(detection, g, []model.InverseEditPointer{{
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			DryPath:    "values.image.tag",
			Owner:      helmInversePointerOwner(preferredImageTagPath, policy.Owner),
			EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, hintKey, hintFallback), vars),
			Confidence: policy.Confidence,
		}})
	case model.GeneratorApplicationSet:
		hints := applicationSetHintsFromInputs(detection.Repo, g.Inputs)
		namePolicy := registry.InversePointerTemplateFor(g.Kind, "child_name", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.89,
		})
		sourcePathPolicy := registry.InversePointerTemplateFor(g.Kind, "source_path", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.86,
		})
		vars := map[string]string{"application_set_path": hints.ApplicationSetPath}
		out := []model.InverseEditPointer{{
			WetPath:    "Application/metadata/name",
			DryPath:    "spec.template.metadata.name",
			Owner:      namePolicy.Owner,
			EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "child_name", "Edit spec.template.metadata.name in {{application_set_path}}."), vars),
			Confidence: namePolicy.Confidence,
		}}
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
	case model.GeneratorAppOfApps:
		hints := appOfAppsPathHintsFromInputs(g.Inputs)
		analysis := appOfAppsAnalysisForGenerator(detection, g)
		rootPathPolicy := registry.InversePointerTemplateFor(g.Kind, "root_path", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.88,
		})
		vars := map[string]string{"root_application_path": hints.RootApplicationPath}
		out := []model.InverseEditPointer{{
			WetPath:    "RootApplication/spec/source/path",
			DryPath:    "spec.source.path",
			Owner:      rootPathPolicy.Owner,
			Route:      "lift-upstream",
			EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "root_path", "Route: lift-upstream. Edit spec.source.path in {{root_application_path}} to change the child app catalog."), vars),
			Confidence: rootPathPolicy.Confidence,
		}}
		if analysis == nil || len(analysis.GeneratedApplications) == 0 {
			return out
		}

		namePolicy := registry.InversePointerTemplateFor(g.Kind, "child_name", registry.InversePointerTemplate{
			Owner: "app-catalog-owner", Confidence: 0.92,
		})
		sourceRepoPolicy := registry.InversePointerTemplateFor(g.Kind, "source_repo", registry.InversePointerTemplate{
			Owner: "app-catalog-owner", Confidence: 0.90,
		})
		sourcePathPolicy := registry.InversePointerTemplateFor(g.Kind, "source_path", registry.InversePointerTemplate{
			Owner: "app-catalog-owner", Confidence: 0.90,
		})
		for _, child := range analysis.GeneratedApplications {
			childVars := map[string]string{"child_application_path": child.Path}
			prefix := fmt.Sprintf("Application[name=%s]", child.Name)
			out = append(out,
				model.InverseEditPointer{
					WetPath:    prefix + "/metadata/name",
					DryPath:    "metadata.name",
					Owner:      namePolicy.Owner,
					Route:      "apply-here",
					EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "child_name", "Route: apply-here. Edit metadata.name in {{child_application_path}}."), childVars),
					Confidence: namePolicy.Confidence,
				},
				model.InverseEditPointer{
					WetPath:    prefix + "/spec/source/repoURL",
					DryPath:    "spec.source.repoURL",
					Owner:      sourceRepoPolicy.Owner,
					Route:      "apply-here",
					EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "source_repo", "Route: apply-here. Edit spec.source.repoURL in {{child_application_path}}."), childVars),
					Confidence: sourceRepoPolicy.Confidence,
				},
				model.InverseEditPointer{
					WetPath:    prefix + "/spec/source/path",
					DryPath:    "spec.source.path",
					Owner:      sourcePathPolicy.Owner,
					Route:      "apply-here",
					EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "source_path", "Route: apply-here. Edit spec.source.path in {{child_application_path}}."), childVars),
					Confidence: sourcePathPolicy.Confidence,
				},
			)
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
		return applyGeneratorChainPointers(detection, g, []model.InverseEditPointer{
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
		})
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
	case model.GeneratorOpenChoreo:
		hints := openChoreoPathHintsFromInputs(g.Inputs)
		imagePolicy := registry.InversePointerTemplateFor(g.Kind, "image", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.88,
		})
		envPolicy := registry.InversePointerTemplateFor(g.Kind, "env_var", registry.InversePointerTemplate{
			Owner: "environment-owner", Confidence: 0.84,
		})
		portPolicy := registry.InversePointerTemplateFor(g.Kind, "service_port", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.82,
		})
		secretPolicy := registry.InversePointerTemplateFor(g.Kind, "secret_ref", registry.InversePointerTemplate{
			Owner: "security-team", Confidence: 0.86,
		})
		resourcePolicy := registry.InversePointerTemplateFor(g.Kind, "resource_limit", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.80,
		})
		mountedFilePolicy := registry.InversePointerTemplateFor(g.Kind, "mounted_file", registry.InversePointerTemplate{
			Owner: "app-team", Confidence: 0.83,
		})
		defaultPolicy := registry.InversePointerTemplateFor(g.Kind, "platform_default", registry.InversePointerTemplate{
			Owner: "platform-engineer", Confidence: 0.78,
		})
		vars := map[string]string{
			"workload_path":         hints.WorkloadPath,
			"component_type_path":   hints.ComponentTypePath,
			"release_binding_path":  hints.ReleaseBindingPath,
			"secret_reference_path": hints.SecretReferencePath,
		}
		return []model.InverseEditPointer{
			{
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/image",
				DryPath:    "spec.containers.main.image",
				Owner:      imagePolicy.Owner,
				Route:      "lift-upstream",
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "image", "Route: lift-upstream. Edit spec.containers.main.image in {{workload_path}}."), vars),
				Confidence: imagePolicy.Confidence,
			},
			{
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/env[name=LOG_LEVEL]/value",
				DryPath:    "spec.environment.env.LOG_LEVEL",
				Owner:      envPolicy.Owner,
				Route:      "apply-here",
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "env_var", "Route: apply-here. Edit environment binding data in {{release_binding_path}}."), vars),
				Confidence: envPolicy.Confidence,
			},
			{
				WetPath:    "Service/spec/ports[name=http]/port",
				DryPath:    "spec.service.port",
				Owner:      portPolicy.Owner,
				Route:      "lift-upstream",
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "service_port", "Route: lift-upstream. Edit the service port contract in {{component_type_path}}."), vars),
				Confidence: portPolicy.Confidence,
			},
			{
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/env[name=DATABASE_URL]/valueFrom/secretKeyRef/name",
				DryPath:    "spec.secretRef",
				Owner:      secretPolicy.Owner,
				Route:      "block/escalate",
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "secret_ref", "Route: block/escalate. Edit {{secret_reference_path}} through the security-owned secret flow."), vars),
				Confidence: secretPolicy.Confidence,
			},
			{
				WetPath:    "Deployment/spec/template/spec/containers[name=main]/resources/limits/cpu",
				DryPath:    "spec.resources.limits.cpu",
				Owner:      resourcePolicy.Owner,
				Route:      "overlay",
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "resource_limit", "Route: overlay. Keep this as an environment/platform overlay in {{release_binding_path}} or policy."), vars),
				Confidence: resourcePolicy.Confidence,
			},
			{
				WetPath:    "ConfigMap/data/LOG_FORMAT",
				DryPath:    "spec.containers.main.files.LOG_FORMAT.value",
				Owner:      mountedFilePolicy.Owner,
				Route:      "lift-upstream",
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "mounted_file", "Route: lift-upstream. Edit mounted file data in {{workload_path}}."), vars),
				Confidence: mountedFilePolicy.Confidence,
			},
			{
				WetPath:    "Deployment/spec/template/spec/securityContext/runAsNonRoot",
				DryPath:    "spec.runtime.defaults.securityContext.runAsNonRoot",
				Owner:      defaultPolicy.Owner,
				Route:      "block/escalate",
				EditHint:   renderTargetTemplate(registry.InverseEditHint(g.Kind, "platform_default", "Route: block/escalate. Edit the platform default in {{component_type_path}} or platform policy, not the generated Deployment."), vars),
				Confidence: defaultPolicy.Confidence,
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
