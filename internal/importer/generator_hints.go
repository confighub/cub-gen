package importer

import (
	"os"
	"path/filepath"
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
