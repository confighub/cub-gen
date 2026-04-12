package applicationset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeListGeneratorAuthoritative(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "applicationset.yaml"), `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: list-demo
spec:
  generators:
    - list:
        elements:
          - name: dev
            server: https://dev.example.invalid
          - name: prod
            server: https://prod.example.invalid
  template:
    metadata:
      name: "{{name}}-inventory"
    spec:
      source:
        path: apps/inventory
`)

	analysis, err := Analyze(repo, "applicationset.yaml", nil)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if analysis.Mode != ModeAuthoritative {
		t.Fatalf("expected authoritative mode, got %+v", analysis)
	}
	if len(analysis.GeneratedApplications) != 2 {
		t.Fatalf("expected two generated applications, got %+v", analysis.GeneratedApplications)
	}
	if analysis.GeneratedApplications[0].Name != "dev-inventory" || analysis.GeneratedApplications[1].Name != "prod-inventory" {
		t.Fatalf("unexpected generated applications: %+v", analysis.GeneratedApplications)
	}
}

func TestAnalyzeUnsupportedGeneratorObservedOnly(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "applicationset.yaml"), `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: git-demo
spec:
  generators:
    - git:
        repoURL: https://github.com/example/platform-apps.git
        revision: HEAD
        directories:
          - path: apps/*
  template:
    metadata:
      name: "{{path.basename}}"
`)

	analysis, err := Analyze(repo, "applicationset.yaml", nil)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if analysis.Mode != ModeObservedOnly {
		t.Fatalf("expected observed-only mode, got %+v", analysis)
	}
	if !strings.Contains(analysis.ModeReason, "unsupported generator types git") {
		t.Fatalf("expected explicit unsupported reason, got %+v", analysis)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
