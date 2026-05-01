package importer

import (
	"strings"

	"github.com/confighub/cub-gen/internal/appofapps"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

type appOfAppsHints struct {
	RootApplicationPath string
	ChildCatalogPath    string
	FirstChildPath      string
}

func appOfAppsPathHintsFromInputs(inputs []string) appOfAppsHints {
	h := appOfAppsHints{
		RootApplicationPath: registry.HintDefault(model.GeneratorAppOfApps, "root_application_path", "root-application.yaml"),
		ChildCatalogPath:    registry.HintDefault(model.GeneratorAppOfApps, "child_catalog_path", "apps"),
	}
	for _, in := range inputs {
		p := strings.TrimSpace(in)
		switch registry.InputRole(model.GeneratorAppOfApps, p) {
		case "root-application":
			h.RootApplicationPath = p
		case "child-application":
			if h.FirstChildPath == "" || p < h.FirstChildPath {
				h.FirstChildPath = p
			}
		}
	}
	if h.FirstChildPath != "" {
		if idx := strings.LastIndex(h.FirstChildPath, "/"); idx > 0 {
			h.ChildCatalogPath = h.FirstChildPath[:idx]
		}
	}
	return h
}

func appOfAppsAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.AppOfAppsAnalysis {
	if g.Kind != model.GeneratorAppOfApps {
		return nil
	}

	hints := appOfAppsPathHintsFromInputs(g.Inputs)
	analysis, err := appofapps.Analyze(detection.Repo, hints.RootApplicationPath)
	if err != nil {
		return &model.AppOfAppsAnalysis{
			RootApplicationPath: hints.RootApplicationPath,
			RootSourcePath:      hints.ChildCatalogPath,
			Mode:                appofapps.ModeObservedOnly,
			ModeReason:          err.Error(),
		}
	}

	children := make([]model.AppOfAppsChildApplication, 0, len(analysis.GeneratedApplications))
	for _, child := range analysis.GeneratedApplications {
		children = append(children, model.AppOfAppsChildApplication{
			Name:                 child.Name,
			Path:                 child.Path,
			SourceRepo:           child.SourceRepo,
			SourcePath:           child.SourcePath,
			DestinationNamespace: child.DestinationNamespace,
			Reason:               child.Reason,
		})
	}

	return &model.AppOfAppsAnalysis{
		RootApplicationPath:   analysis.RootApplicationPath,
		RootSourcePath:        analysis.RootSourcePath,
		ChildApplicationPaths: append([]string(nil), analysis.ChildApplicationPaths...),
		Mode:                  analysis.Mode,
		ModeReason:            analysis.ModeReason,
		GeneratedApplications: children,
	}
}
