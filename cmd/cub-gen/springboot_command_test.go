package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpringBootSetEmbeddedConfigJSON(t *testing.T) {
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

	out, stderr, err := runWithCapturedIO([]string{
		"springboot", "set-embedded-config",
		"--file", tempPath,
		"--configmap", "inventory-api-config",
		"--routes", routesPath,
		"--json",
		"feature.inventory.reservationMode",
		"optimistic",
	})
	if err != nil {
		t.Fatalf("set-embedded-config returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput=%s", err, out)
	}
	if allowed, ok := got["allowed"].(bool); !ok || !allowed {
		t.Fatalf("expected allowed=true, got %v", got["allowed"])
	}
	if action, ok := got["action"].(string); !ok || action != "mutable-in-ch" {
		t.Fatalf("expected action mutable-in-ch, got %v", got["action"])
	}
	if oldValue, ok := got["old_value"].(string); !ok || oldValue != "strict" {
		t.Fatalf("expected old_value=strict, got %v", got["old_value"])
	}
	if newValue, ok := got["new_value"].(string); !ok || newValue != "optimistic" {
		t.Fatalf("expected new_value=optimistic, got %v", got["new_value"])
	}
}

func TestSpringBootSetEmbeddedConfigBlocked(t *testing.T) {
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

	out, stderr, err := runWithCapturedIO([]string{
		"springboot", "set-embedded-config",
		"--file", tempPath,
		"--configmap", "inventory-api-config",
		"--routes", routesPath,
		"spring.datasource.url",
		"jdbc:postgresql://shadow.platform.svc:5432/inventory",
	})
	if err == nil {
		t.Fatal("expected blocked mutation error")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(out, "BLOCKED") {
		t.Fatalf("expected BLOCKED output, got %q", out)
	}
	if !strings.Contains(err.Error(), "blocked by field routes") {
		t.Fatalf("unexpected error: %v", err)
	}
}
