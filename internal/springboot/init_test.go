package springboot

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFieldRoutesContentIncludesSourceProposalFiles(t *testing.T) {
	content := fieldRoutesContent("inventory-api")
	if strings.Contains(content, "\t") {
		t.Fatalf("generated field-routes.yaml must not contain tab indentation:\n%s", content)
	}

	var routes FieldRoutes
	if err := yaml.Unmarshal([]byte(content), &routes); err != nil {
		t.Fatalf("generated field routes should parse as YAML: %v\n%s", err, content)
	}

	cache := findFieldRoute(t, routes, "spring.cache.*")
	if cache.SourcePath != "src/main/resources/application.yaml" {
		t.Fatalf("cache sourcePath = %q", cache.SourcePath)
	}
	if !sameStringSlice(cache.ProposalFiles, []string{"pom.xml", "src/main/resources/application.yaml"}) {
		t.Fatalf("cache proposalFiles = %+v", cache.ProposalFiles)
	}

	feature := findFieldRoute(t, routes, "feature.inventoryapi.*")
	if feature.SourceField != "feature.inventoryapi.*" {
		t.Fatalf("feature sourceField = %q", feature.SourceField)
	}
}

func findFieldRoute(t *testing.T, routes FieldRoutes, match string) FieldRoute {
	t.Helper()
	for _, route := range routes.Routes {
		if route.Match == match {
			return route
		}
	}
	t.Fatalf("missing generated route %q in %+v", match, routes.Routes)
	return FieldRoute{}
}

func sameStringSlice(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
