package gitops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverImportCleanupFlow(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "Chart.yaml"), "apiVersion: v2\nname: payments\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(repo, "values.yaml"), "image:\n  tag: v1\n")

	discovered, err := Discover(repo, "main", "platform", "")
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if discovered.DiscoverUnitSlug == "" {
		t.Fatal("expected discover unit slug")
	}
	if len(discovered.Resources) != 1 {
		t.Fatalf("expected 1 discovered resource, got %d", len(discovered.Resources))
	}
	if _, err := os.Stat(discovered.DiscoverFile); err != nil {
		t.Fatalf("expected discover file to exist: %v", err)
	}

	imported, err := Import(repo, repo, "main", "platform", "")
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if len(imported.Discovered) != 1 {
		t.Fatalf("expected 1 discovered resource in import, got %d", len(imported.Discovered))
	}
	if len(imported.DryUnits) != 1 {
		t.Fatalf("expected 1 dry unit, got %d", len(imported.DryUnits))
	}
	if len(imported.WetUnits) != 1 {
		t.Fatalf("expected 1 wet unit, got %d", len(imported.WetUnits))
	}
	if len(imported.GeneratorUnits) != 1 {
		t.Fatalf("expected 1 generator unit, got %d", len(imported.GeneratorUnits))
	}
	if len(imported.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(imported.Links))
	}

	deleted, _, err := Cleanup(repo, "platform")
	if err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	if !deleted {
		t.Fatal("expected discover state to be deleted")
	}

	deleted, _, err = Cleanup(repo, "platform")
	if err != nil {
		t.Fatalf("Cleanup second call returned error: %v", err)
	}
	if deleted {
		t.Fatal("expected no discover state to delete on second cleanup")
	}
}

func TestWhereResourceFiltering(t *testing.T) {
	repo := t.TempDir()
	// Helm
	writeFile(t, filepath.Join(repo, "helm", "Chart.yaml"), "apiVersion: v2\nname: payments\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(repo, "helm", "values.yaml"), "image:\n  tag: v1\n")
	// Score
	writeFile(t, filepath.Join(repo, "score", "score.yaml"), "apiVersion: score.dev/v1b1\nkind: Workload\nmetadata:\n  name: checkout\n")
	// Spring Boot
	writeFile(t, filepath.Join(repo, "spring", "pom.xml"), "<project></project>\n")
	writeFile(t, filepath.Join(repo, "spring", "src", "main", "resources", "application.yaml"), "spring:\n  application:\n    name: inventory\n")

	filtered, err := Discover(repo, "main", "platform", "kind IN ('HelmRelease','Application')")
	if err != nil {
		t.Fatalf("Discover filter IN returned error: %v", err)
	}
	if len(filtered.Resources) != 2 {
		t.Fatalf("expected 2 resources for IN filter, got %d", len(filtered.Resources))
	}

	springOnly, err := Discover(repo, "main", "platform", "kind = 'Kustomization'")
	if err != nil {
		t.Fatalf("Discover filter kind= returned error: %v", err)
	}
	if len(springOnly.Resources) != 1 {
		t.Fatalf("expected 1 spring resource, got %d", len(springOnly.Resources))
	}
	if springOnly.Resources[0].GeneratorKind != "springboot" {
		t.Fatalf("expected springboot generator, got %q", springOnly.Resources[0].GeneratorKind)
	}

	_, err = Discover(repo, "main", "platform", "metadata.namespace = 'argocd'")
	if err == nil {
		t.Fatal("expected unsupported where-resource clause to fail")
	}
}

func TestTargetAliasResolution(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "Chart.yaml"), "apiVersion: v2\nname: payments\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(repo, "values.yaml"), "image:\n  tag: v1\n")

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "targets.json")
	writeFile(t, cfgPath, "{\n  \"targets\": {\n    \"helm-dev\": \""+repo+"\"\n  }\n}\n")
	t.Setenv("CUB_GEN_TARGETS_FILE", cfgPath)

	discovered, err := Discover("helm-dev", "main", "platform", "")
	if err != nil {
		t.Fatalf("Discover alias returned error: %v", err)
	}
	if discovered.TargetSlug != "helm-dev" {
		t.Fatalf("expected target slug helm-dev, got %q", discovered.TargetSlug)
	}
	if discovered.TargetPath != repo {
		t.Fatalf("expected target path %q, got %q", repo, discovered.TargetPath)
	}

	deleted, _, err := Cleanup("helm-dev", "platform")
	if err != nil {
		t.Fatalf("Cleanup alias returned error: %v", err)
	}
	if !deleted {
		t.Fatal("expected alias cleanup to delete discover state")
	}
}

func TestTargetCapabilityValidation(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "score.yaml"), "apiVersion: score.dev/v1b1\nkind: Workload\nmetadata:\n  name: checkout\n")

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "targets.json")
	writeFile(t, cfgPath, "{\n  \"targets\": {\n    \"discover\": {\n      \"path\": \""+repo+"\",\n      \"toolchain\": \"kubernetes/yaml\",\n      \"providers\": [\"kubernetes\"]\n    },\n    \"render-flux-only\": {\n      \"toolchain\": \"kubernetes/yaml\",\n      \"providers\": [\"fluxrenderer\"]\n    },\n    \"discover-no-k8s\": {\n      \"path\": \""+repo+"\",\n      \"toolchain\": \"kubernetes/yaml\",\n      \"providers\": [\"fluxrenderer\"]\n    }\n  }\n}\n")
	t.Setenv("CUB_GEN_TARGETS_FILE", cfgPath)

	if _, err := Discover("discover-no-k8s", "main", "platform", ""); err == nil {
		t.Fatal("expected discover capability check to fail")
	}

	_, err := Import("discover", "render-flux-only", "main", "platform", "")
	if err == nil {
		t.Fatal("expected import capability check to fail for missing argocd renderer")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "missing providers") {
		t.Fatalf("expected missing providers error, got %q", got)
	}
}

func TestUnknownTargetReturnsError(t *testing.T) {
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "targets.json")
	writeFile(t, cfgPath, "{\n  \"targets\": {\n    \"known\": \"./missing\"\n  }\n}\n")
	t.Setenv("CUB_GEN_TARGETS_FILE", cfgPath)

	if _, err := Discover("unknown-target", "main", "platform", ""); err == nil {
		t.Fatal("expected unknown target error")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
