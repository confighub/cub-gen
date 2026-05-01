package appofapps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootsDetectsRootAndChildCatalog(t *testing.T) {
	repo := t.TempDir()
	mustWriteAppOfAppsFile(t, filepath.Join(repo, "root-application.yaml"), `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: platform-root
spec:
  source:
    repoURL: https://github.com/acme/platform-apps
    path: apps
  destination:
    namespace: argocd
`)
	mustWriteAppOfAppsFile(t, filepath.Join(repo, "apps", "payments.yaml"), `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: payments-api
spec:
  source:
    repoURL: https://github.com/acme/payments-api
    path: deploy/helm
  destination:
    namespace: payments
`)

	roots, err := FindRoots(repo)
	if err != nil {
		t.Fatalf("FindRoots returned error: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected one root, got %+v", roots)
	}
	if roots[0].Name != "platform-root" {
		t.Fatalf("expected platform-root, got %q", roots[0].Name)
	}
	if len(roots[0].ChildApplications) != 1 || roots[0].ChildApplications[0].Name != "payments-api" {
		t.Fatalf("expected payments-api child, got %+v", roots[0].ChildApplications)
	}
}

func TestAnalyzeAppOfAppsAuthoritative(t *testing.T) {
	repo := filepath.Join("..", "..", "testdata", "app-of-apps-standalone")

	analysis, err := Analyze(repo, "root-application.yaml")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if analysis.Mode != ModeAuthoritative {
		t.Fatalf("expected authoritative mode, got %+v", analysis)
	}
	if analysis.RootSourcePath != "apps" {
		t.Fatalf("expected root source path apps, got %+v", analysis)
	}
	if len(analysis.GeneratedApplications) != 3 {
		t.Fatalf("expected 3 child apps, got %+v", analysis.GeneratedApplications)
	}
}

func mustWriteAppOfAppsFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
