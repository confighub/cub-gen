package importer

import (
	"fmt"

	"github.com/confighub/cub-gen/internal/applicationset"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

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
