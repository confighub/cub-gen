package importer

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

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
	if g.Kind == model.GeneratorAppOfApps {
		analysis := appOfAppsAnalysisForGenerator(detection, g)
		out := []model.WetManifestTarget{{
			GeneratorID:   g.ID,
			Kind:          "Application",
			Name:          g.Name,
			Owner:         "platform-runtime",
			Namespace:     "argocd",
			SourceDryPath: "spec.source.path",
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
				SourceDryPath: "metadata.name",
			})
		}
		return out
	}
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
	if g.Kind == model.GeneratorOpenChoreo {
		return openChoreoWetManifestTargets(g)
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

func renderedLineageForGenerator(detection model.DetectionResult, g model.GeneratorDetection) []model.RenderedObjectLineage {
	if g.Kind == model.GeneratorAppOfApps {
		analysis := appOfAppsAnalysisForGenerator(detection, g)
		hints := appOfAppsPathHintsFromInputs(g.Inputs)
		lineage := []model.RenderedObjectLineage{{
			Kind:          "Application",
			Name:          g.Name,
			Namespace:     "argocd",
			SourcePath:    hints.RootApplicationPath,
			SourceDryPath: "spec.source.path",
		}}
		if analysis == nil {
			return lineage
		}
		for _, child := range analysis.GeneratedApplications {
			lineage = append(lineage,
				model.RenderedObjectLineage{
					Kind:          "Application",
					Name:          child.Name,
					Namespace:     "argocd",
					SourcePath:    child.Path,
					SourceDryPath: "metadata.name",
				},
				model.RenderedObjectLineage{
					Kind:          "Application",
					Name:          child.Name,
					Namespace:     "argocd",
					SourcePath:    child.Path,
					SourceDryPath: "spec.source.path",
				},
			)
		}
		return lineage
	}
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
	if g.Kind == model.GeneratorOpenChoreo {
		return openChoreoRenderedLineage(g)
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

func openChoreoWetManifestTargets(g model.GeneratorDetection) []model.WetManifestTarget {
	hints := openChoreoPathHintsFromInputs(g.Inputs)
	out := make([]model.WetManifestTarget, 0, len(hints.Variants)*4)
	for _, variant := range hints.Variants {
		namespace := "apps-" + variant
		out = append(out,
			model.WetManifestTarget{
				GeneratorID:   g.ID,
				Kind:          "RenderedRelease",
				Name:          g.Name + "-" + variant,
				Owner:         "platform-runtime",
				Namespace:     namespace,
				SourceDryPath: "spec",
			},
			model.WetManifestTarget{
				GeneratorID:   g.ID,
				Kind:          "Deployment",
				Name:          g.Name,
				Owner:         "platform-runtime",
				Namespace:     namespace,
				SourceDryPath: "spec.containers.main.image",
			},
			model.WetManifestTarget{
				GeneratorID:   g.ID,
				Kind:          "Service",
				Name:          g.Name,
				Owner:         "platform-runtime",
				Namespace:     namespace,
				SourceDryPath: "spec.service.port",
			},
			model.WetManifestTarget{
				GeneratorID:   g.ID,
				Kind:          "Secret",
				Name:          g.Name + "-secret-ref",
				Owner:         "security-team",
				Namespace:     namespace,
				SourceDryPath: "spec.secretRef",
			},
		)
	}
	return out
}

func openChoreoRenderedLineage(g model.GeneratorDetection) []model.RenderedObjectLineage {
	hints := openChoreoPathHintsFromInputs(g.Inputs)
	releasePathsByVariant := openChoreoRolePathByVariant(g.Inputs, "rendered-release")
	releaseBindingPathsByVariant := openChoreoRolePathByVariant(g.Inputs, "release-binding")
	renderedManifestPathsByVariant := openChoreoRolePathByVariant(g.Inputs, "rendered-manifest")

	lineage := make([]model.RenderedObjectLineage, 0, len(hints.Variants)*6)
	for _, variant := range hints.Variants {
		namespace := "apps-" + variant
		releasePath := firstNonEmpty(releasePathsByVariant[variant], hints.RenderedReleasePath)
		bindingPath := firstNonEmpty(releaseBindingPathsByVariant[variant], hints.ReleaseBindingPath)
		renderedManifestPath := firstNonEmpty(renderedManifestPathsByVariant[variant], hints.RenderedManifestPath)
		lineage = append(lineage,
			model.RenderedObjectLineage{
				Kind:          "RenderedRelease",
				Name:          g.Name + "-" + variant,
				Namespace:     namespace,
				SourcePath:    releasePath,
				SourceDryPath: "spec",
			},
			model.RenderedObjectLineage{
				Kind:          "Deployment",
				Name:          g.Name,
				Namespace:     namespace,
				SourcePath:    hints.WorkloadPath,
				SourceDryPath: "spec.containers.main.image",
			},
			model.RenderedObjectLineage{
				Kind:          "Deployment",
				Name:          g.Name,
				Namespace:     namespace,
				SourcePath:    bindingPath,
				SourceDryPath: "spec.environment.env.LOG_LEVEL",
			},
			model.RenderedObjectLineage{
				Kind:          "Deployment",
				Name:          g.Name,
				Namespace:     namespace,
				SourcePath:    hints.SecretReferencePath,
				SourceDryPath: "spec.secretRef",
			},
			model.RenderedObjectLineage{
				Kind:          "Service",
				Name:          g.Name,
				Namespace:     namespace,
				SourcePath:    hints.ComponentTypePath,
				SourceDryPath: "spec.service.port",
			},
			model.RenderedObjectLineage{
				Kind:          "Deployment",
				Name:          g.Name,
				Namespace:     namespace,
				SourcePath:    renderedManifestPath,
				SourceDryPath: "metadata.ownerReferences",
			},
		)
	}
	return lineage
}

func openChoreoRolePathByVariant(inputs []string, role string) map[string]string {
	out := map[string]string{}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		if registry.InputRole(model.GeneratorOpenChoreo, p) != role {
			continue
		}
		variant := variantNameFromOpenChoreoPath(p)
		if variant == "" {
			continue
		}
		if current := out[variant]; current == "" || p < current {
			out[variant] = p
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
