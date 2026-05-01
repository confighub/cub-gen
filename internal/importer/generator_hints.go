package importer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

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

type openChoreoHints struct {
	WorkloadPath         string
	ComponentTypePath    string
	ReleaseBindingPath   string
	SecretReferencePath  string
	RenderedReleasePath  string
	RenderedManifestPath string
	Variants             []string
}

func openChoreoPathHintsFromInputs(inputs []string) openChoreoHints {
	h := openChoreoHints{
		WorkloadPath:         registry.HintDefault(model.GeneratorOpenChoreo, "workload_path", "workload.yaml"),
		ComponentTypePath:    registry.HintDefault(model.GeneratorOpenChoreo, "component_type_path", "component-type.yaml"),
		ReleaseBindingPath:   registry.HintDefault(model.GeneratorOpenChoreo, "release_binding_path", "release-binding.yaml"),
		SecretReferencePath:  registry.HintDefault(model.GeneratorOpenChoreo, "secret_reference_path", "secret-reference.yaml"),
		RenderedReleasePath:  registry.HintDefault(model.GeneratorOpenChoreo, "rendered_release_path", "rendered-release.yaml"),
		RenderedManifestPath: registry.HintDefault(model.GeneratorOpenChoreo, "rendered_manifest_path", "rendered/deployment.yaml"),
	}

	var variants []string
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		role := registry.InputRole(model.GeneratorOpenChoreo, p)
		switch role {
		case "workload":
			h.WorkloadPath = p
		case "component-type":
			h.ComponentTypePath = p
		case "release-binding":
			if h.ReleaseBindingPath == "" || p < h.ReleaseBindingPath || strings.Contains(p, "/prod/") {
				h.ReleaseBindingPath = p
			}
			if variant := variantNameFromOpenChoreoPath(p); variant != "" {
				variants = append(variants, variant)
			}
		case "secret-reference":
			h.SecretReferencePath = p
		case "rendered-release":
			if h.RenderedReleasePath == "" || p < h.RenderedReleasePath || strings.Contains(p, "/prod/") {
				h.RenderedReleasePath = p
			}
			if variant := variantNameFromOpenChoreoPath(p); variant != "" {
				variants = append(variants, variant)
			}
		case "rendered-manifest":
			if strings.Contains(strings.ToLower(filepath.Base(p)), "deployment") || h.RenderedManifestPath == "" {
				h.RenderedManifestPath = p
			}
			if variant := variantNameFromOpenChoreoPath(p); variant != "" {
				variants = append(variants, variant)
			}
		}
	}
	h.Variants = uniqueSortedStringSlice(variants)
	if len(h.Variants) == 0 {
		h.Variants = []string{"prod"}
	}
	return h
}

func variantNameFromOpenChoreoPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range parts {
		switch part {
		case "envs", "environments", "rendered":
			if i+1 < len(parts) && parts[i+1] != "" {
				return parts[i+1]
			}
		}
	}
	base := strings.ToLower(filepath.Base(path))
	for _, suffix := range []string{".yaml", ".yml"} {
		base = strings.TrimSuffix(base, suffix)
	}
	for _, prefix := range []string{"release-binding-", "rendered-release-"} {
		if strings.HasPrefix(base, prefix) {
			return strings.TrimPrefix(base, prefix)
		}
	}
	return ""
}

func uniqueSortedStringSlice(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
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
