package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRuntimeMergesOverlayAndResolvesResources(t *testing.T) {
	dir := t.TempDir()
	scorePath := filepath.Join(dir, "score.yaml")
	overlayPath := filepath.Join(dir, "score-prod.yaml")

	if err := os.WriteFile(scorePath, []byte(`apiVersion: score.dev/v1b1
kind: Workload
metadata:
  name: checkout-api
containers:
  main:
    image: ghcr.io/example/checkout-api:v1.0.0
    variables:
      LOG_LEVEL: info
      DB_HOST: ${resources.db.host}
      DB_PORT: ${resources.db.port}
service:
  ports:
    web:
      port: 8080
resources:
  db:
    type: postgres
`), 0o644); err != nil {
		t.Fatalf("write score.yaml: %v", err)
	}

	if err := os.WriteFile(overlayPath, []byte(`apiVersion: score.dev/v1b1
metadata:
  name: checkout-api
containers:
  main:
    image: ghcr.io/example/checkout-api:v2.1.0
    variables:
      LOG_LEVEL: warn
      NODE_ENV: production
`), 0o644); err != nil {
		t.Fatalf("write score-prod.yaml: %v", err)
	}

	manifests, err := renderRuntime(scorePath, overlayPath, "")
	if err != nil {
		t.Fatalf("render runtime: %v", err)
	}

	if manifests.Image != "ghcr.io/example/checkout-api:v2.1.0" {
		t.Fatalf("resolved image = %q, want overlay image", manifests.Image)
	}
	text := string(manifests.Content)
	for _, want := range []string{
		"namespace: checkout-api",
		"image: ghcr.io/example/checkout-api:v2.1.0",
		"name: LOG_LEVEL",
		"value: warn",
		"name: NODE_ENV",
		"value: production",
		"value: postgres.platform.svc.cluster.local",
		"value: \"5432\"",
		"path: /healthz",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered manifests missing %q\n%s", want, text)
		}
	}
}
