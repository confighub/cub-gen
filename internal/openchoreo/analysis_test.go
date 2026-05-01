package openchoreo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeRepoHardGate(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "openchoreo-hardgate"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	analysis, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("analyze repo: %v", err)
	}
	if !analysis.HasWorkload() {
		t.Fatalf("expected workload in analysis: %+v", analysis)
	}
	if got := len(analysis.BlockingDiagnostics()); got != 0 {
		t.Fatalf("expected no blocking diagnostics, got %+v", analysis.BlockingDiagnostics())
	}
	for _, role := range []string{"workload", "component-type", "release-binding", "secret-reference", "rendered-release", "rendered-manifest"} {
		if !hasRole(analysis, role) {
			t.Fatalf("expected role %q in %+v", role, analysis.Artifacts)
		}
	}
	if !contains(analysis.Variants, "dev") || !contains(analysis.Variants, "prod") {
		t.Fatalf("expected dev/prod variants, got %+v", analysis.Variants)
	}
}

func TestAnalyzeRepoRejectsUnsupportedVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workload.yaml"), "apiVersion: openchoreo.dev/v9alpha1\nkind: Workload\nmetadata:\n  name: bad\n")

	analysis, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("analyze repo: %v", err)
	}
	assertDiagnostic(t, analysis, "unsupported_openchoreo_api_version")
}

func TestAnalyzeRepoReportsMissingComponentType(t *testing.T) {
	root := t.TempDir()
	writeMinimalOpenChoreoRepo(t, root)
	if err := os.Remove(filepath.Join(root, "component-type-web-service.yaml")); err != nil {
		t.Fatalf("remove component type: %v", err)
	}

	analysis, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("analyze repo: %v", err)
	}
	assertDiagnostic(t, analysis, "missing_component_type")
}

func TestAnalyzeRepoReportsUnresolvedSecretReference(t *testing.T) {
	root := t.TempDir()
	writeMinimalOpenChoreoRepo(t, root)
	if err := os.Remove(filepath.Join(root, "secret-reference-payments-db.yaml")); err != nil {
		t.Fatalf("remove secret reference: %v", err)
	}

	analysis, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("analyze repo: %v", err)
	}
	assertDiagnostic(t, analysis, "unresolved_secret_reference")
}

func TestAnalyzeRepoReportsUnknownRenderedOwner(t *testing.T) {
	root := t.TempDir()
	writeMinimalOpenChoreoRepo(t, root)
	writeFile(t, filepath.Join(root, "rendered", "prod", "service.yaml"), `apiVersion: v1
kind: Service
metadata:
  name: payments-api
spec:
  ports:
    - name: http
      port: 8080
`)

	analysis, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("analyze repo: %v", err)
	}
	assertDiagnostic(t, analysis, "unknown_rendered_owner")
}

func TestAnalyzeRepoDoesNotBlockOnRenderedOnlyRepo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "rendered", "prod", "service.yaml"), `apiVersion: v1
kind: Service
metadata:
  name: unrelated
spec:
  ports:
    - name: http
      port: 8080
`)

	analysis, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("analyze repo: %v", err)
	}
	if analysis.HasWorkload() {
		t.Fatalf("did not expect OpenChoreo workload in rendered-only repo: %+v", analysis)
	}
	if got := len(analysis.BlockingDiagnostics()); got != 0 {
		t.Fatalf("expected rendered-only repo to have no blocking diagnostics, got %+v", analysis.BlockingDiagnostics())
	}
}

func writeMinimalOpenChoreoRepo(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "workload-payments-api.yaml"), `apiVersion: openchoreo.dev/v1alpha1
kind: Workload
metadata:
  name: payments-api
spec:
  secretRef: payments-db
`)
	writeFile(t, filepath.Join(root, "component-type-web-service.yaml"), `apiVersion: openchoreo.dev/v1alpha1
kind: ComponentType
metadata:
  name: web-service
`)
	writeFile(t, filepath.Join(root, "secret-reference-payments-db.yaml"), `apiVersion: openchoreo.dev/v1alpha1
kind: SecretReference
metadata:
  name: payments-db
`)
	writeFile(t, filepath.Join(root, "envs", "prod", "release-binding-prod.yaml"), `apiVersion: openchoreo.dev/v1alpha1
kind: ReleaseBinding
metadata:
  name: payments-api-prod
`)
	writeFile(t, filepath.Join(root, "rendered", "prod", "rendered-release-prod.yaml"), `apiVersion: openchoreo.dev/v1alpha1
kind: RenderedRelease
metadata:
  name: payments-api-prod
`)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}

func hasRole(analysis Analysis, role string) bool {
	for _, artifact := range analysis.Artifacts {
		if artifact.Role == role {
			return true
		}
	}
	return false
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func assertDiagnostic(t *testing.T, analysis Analysis, code string) {
	t.Helper()
	for _, diagnostic := range analysis.BlockingDiagnostics() {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic %q, got %+v", code, analysis.Diagnostics)
}
