package springboot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetEmbeddedConfigMutatesExampleFixture(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "examples", "springboot-paas", "confighub", "inventory-api-prod.yaml")
	routesPath := filepath.Join("..", "..", "examples", "springboot-paas", "operational", "field-routes.yaml")
	tempPath := filepath.Join(t.TempDir(), "inventory-api-prod.yaml")

	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source fixture: %v", err)
	}
	if err := os.WriteFile(tempPath, raw, 0o644); err != nil {
		t.Fatalf("write temp fixture: %v", err)
	}

	result, err := SetEmbeddedConfig(SetEmbeddedConfigOptions{
		FilePath:        tempPath,
		ConfigMapName:   "inventory-api-config",
		ConfigKey:       "application.yaml",
		FieldRoutesPath: routesPath,
		FieldPath:       "feature.inventory.reservationMode",
		Value:           "optimistic",
	})
	if err != nil {
		t.Fatalf("SetEmbeddedConfig() error = %v", err)
	}
	if !result.Allowed {
		t.Fatalf("expected allowed result, got %+v", result)
	}
	if !result.Updated {
		t.Fatalf("expected updated=true, got %+v", result)
	}
	if got, want := result.OldValue, "strict"; got != want {
		t.Fatalf("old value = %q, want %q", got, want)
	}
	if got, want := result.NewValue, "optimistic"; got != want {
		t.Fatalf("new value = %q, want %q", got, want)
	}

	mutated, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("read mutated fixture: %v", err)
	}
	text := string(mutated)
	if !strings.Contains(text, "reservationMode: optimistic") {
		t.Fatalf("mutated fixture missing optimistic reservation mode:\n%s", text)
	}
	if strings.Contains(text, "reservationMode: strict") {
		t.Fatalf("mutated fixture still contains strict reservation mode:\n%s", text)
	}
}

func TestSetEmbeddedConfigRespectsBlockedRoutes(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "examples", "springboot-paas", "confighub", "inventory-api-prod.yaml")
	routesPath := filepath.Join("..", "..", "examples", "springboot-paas", "operational", "field-routes.yaml")
	tempPath := filepath.Join(t.TempDir(), "inventory-api-prod.yaml")

	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source fixture: %v", err)
	}
	if err := os.WriteFile(tempPath, raw, 0o644); err != nil {
		t.Fatalf("write temp fixture: %v", err)
	}

	result, err := SetEmbeddedConfig(SetEmbeddedConfigOptions{
		FilePath:        tempPath,
		ConfigMapName:   "inventory-api-config",
		ConfigKey:       "application.yaml",
		FieldRoutesPath: routesPath,
		FieldPath:       "spring.datasource.url",
		Value:           "jdbc:postgresql://shadow.platform.svc:5432/inventory",
	})
	if err != nil {
		t.Fatalf("SetEmbeddedConfig() error = %v", err)
	}
	if result.Allowed {
		t.Fatalf("expected blocked result, got %+v", result)
	}
	if result.Updated {
		t.Fatalf("expected updated=false for blocked mutation, got %+v", result)
	}

	mutated, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("read fixture after blocked mutation: %v", err)
	}
	text := string(mutated)
	if !strings.Contains(text, "jdbc:postgresql://postgres.platform.svc:5432/inventory") {
		t.Fatalf("blocked mutation changed datasource fixture:\n%s", text)
	}
	if strings.Contains(text, "shadow.platform.svc") {
		t.Fatalf("blocked mutation wrote unexpected datasource URL:\n%s", text)
	}
}
