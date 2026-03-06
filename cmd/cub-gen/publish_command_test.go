package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishFromImportFile(t *testing.T) {
	setupAliases(t)

	importOut, importErr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "--json", "helm", "render-target"})
	if err != nil {
		t.Fatalf("gitops import returned error: %v\nstderr=%s", err, importErr)
	}
	if importErr != "" {
		t.Fatalf("expected empty stderr from import, got %q", importErr)
	}

	inPath := filepath.Join(t.TempDir(), "import.json")
	if err := os.WriteFile(inPath, []byte(importOut), 0o644); err != nil {
		t.Fatalf("write import json: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{"publish", "--in", inPath})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr from publish, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal publish output: %v\noutput=%s", err, out)
	}
	if got["schema_version"] != "cub.confighub.io/change-bundle/v1" {
		t.Fatalf("unexpected schema_version: %v", got["schema_version"])
	}
	if got["source"] != "cub-gen" {
		t.Fatalf("unexpected source: %v", got["source"])
	}
	if got["change_id"] == "" {
		t.Fatalf("expected non-empty change_id: %v", got["change_id"])
	}

	summary, ok := got["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got: %T", got["summary"])
	}
	if summary["discovered_resources"] != float64(1) {
		t.Fatalf("expected discovered_resources=1, got %v", summary["discovered_resources"])
	}
}

func TestPublishRejectsInvalidJSON(t *testing.T) {
	inPath := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(inPath, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write invalid input: %v", err)
	}

	_, _, err := runWithCapturedIO([]string{"publish", "--in", inPath})
	if err == nil {
		t.Fatal("expected publish to fail on invalid json")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "parse import flow json") {
		t.Fatalf("expected parse import flow json error, got %q", got)
	}
}
