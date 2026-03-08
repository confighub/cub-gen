package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/model"
)

func TestScanRepoExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		repoDir         string
		expectedKind    model.GeneratorKind
		expectedProfile string
		expectedFile    string
	}{
		{
			name:            "helm-paas",
			repoDir:         "helm-paas",
			expectedKind:    model.GeneratorHelm,
			expectedProfile: "helm-paas",
			expectedFile:    "Chart.yaml",
		},
		{
			name:            "scoredev-paas",
			repoDir:         "scoredev-paas",
			expectedKind:    model.GeneratorScore,
			expectedProfile: "scoredev-paas",
			expectedFile:    "score.yaml",
		},
		{
			name:            "springboot-paas",
			repoDir:         "springboot-paas",
			expectedKind:    model.GeneratorSpringBoot,
			expectedProfile: "springboot-paas",
			expectedFile:    "pom.xml",
		},
		{
			name:            "backstage-idp",
			repoDir:         "backstage-idp",
			expectedKind:    model.GeneratorBackstage,
			expectedProfile: "backstage-idp",
			expectedFile:    "catalog-info.yaml",
		},
		{
			name:            "ably-config",
			repoDir:         "ably-config",
			expectedKind:    model.GeneratorAbly,
			expectedProfile: "ably-config",
			expectedFile:    "ably.yaml",
		},
		{
			name:            "ops-workflow",
			repoDir:         "ops-workflow",
			expectedKind:    model.GeneratorOpsFlow,
			expectedProfile: "ops-workflow",
			expectedFile:    "operations.yaml",
		},
		{
			name:            "c3agent",
			repoDir:         "c3agent",
			expectedKind:    model.GeneratorC3Agent,
			expectedProfile: "c3agent",
			expectedFile:    "c3agent.yaml",
		},
		{
			name:            "swamp",
			repoDir:         "swamp-automation",
			expectedKind:    model.GeneratorSwamp,
			expectedProfile: "swamp",
			expectedFile:    ".swamp.yaml",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := filepath.Join("..", "..", "examples", tt.repoDir)
			result, err := ScanRepo(repo, "main")
			if err != nil {
				t.Fatalf("ScanRepo returned error: %v", err)
			}

			if len(result.Generators) != 1 {
				t.Fatalf("expected 1 generator, got %d", len(result.Generators))
			}

			g := result.Generators[0]
			if g.Kind != tt.expectedKind {
				t.Fatalf("expected kind %q, got %q", tt.expectedKind, g.Kind)
			}
			if g.Profile != tt.expectedProfile {
				t.Fatalf("expected profile %q, got %q", tt.expectedProfile, g.Profile)
			}
			if g.ID == "" {
				t.Fatal("expected non-empty generator ID")
			}
			if len(g.Inputs) == 0 {
				t.Fatal("expected at least one input")
			}
			if !contains(g.Inputs, tt.expectedFile) {
				t.Fatalf("expected inputs to contain %q; got %v", tt.expectedFile, g.Inputs)
			}
		})
	}
}

func TestScanRepoSwampIncludesNestedWorkflowInputs(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".swamp.yaml"), []byte("swamp:\n  version: \"1\"\n"), 0o644); err != nil {
		t.Fatalf("write .swamp.yaml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "workflows"), 0o755); err != nil {
		t.Fatalf("create workflows dir: %v", err)
	}
	nestedWorkflow := filepath.Join("workflows", "workflow-nightly.yaml")
	if err := os.WriteFile(filepath.Join(repo, nestedWorkflow), []byte("jobs:\n  - name: nightly\n"), 0o644); err != nil {
		t.Fatalf("write nested workflow: %v", err)
	}

	result, err := ScanRepo(repo, "main")
	if err != nil {
		t.Fatalf("ScanRepo returned error: %v", err)
	}
	if len(result.Generators) != 1 {
		t.Fatalf("expected 1 generator, got %d", len(result.Generators))
	}
	g := result.Generators[0]
	if g.Kind != model.GeneratorSwamp {
		t.Fatalf("expected kind %q, got %q", model.GeneratorSwamp, g.Kind)
	}
	if !containsSuffix(g.Inputs, filepath.ToSlash(nestedWorkflow)) {
		t.Fatalf("expected nested workflow input %q, got %v", nestedWorkflow, g.Inputs)
	}
}

func contains(v []string, suffix string) bool {
	for _, item := range v {
		if filepath.Base(item) == suffix || item == suffix {
			return true
		}
	}
	return false
}

func containsSuffix(v []string, suffix string) bool {
	for _, item := range v {
		if strings.HasSuffix(filepath.ToSlash(item), filepath.ToSlash(suffix)) {
			return true
		}
	}
	return false
}
