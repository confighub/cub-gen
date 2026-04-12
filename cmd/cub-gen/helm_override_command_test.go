package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitOpsImportHelmCLIOverridesJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"gitops", "import",
		"--space", "platform",
		"--json",
		"--set", "image.tag=v9.9.9",
		"helm",
		"render-target",
	})
	if err != nil {
		t.Fatalf("gitops import returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal import output: %v\noutput=%s", err, out)
	}

	overrides, ok := got["helm_cli_overrides"].([]any)
	if !ok || len(overrides) != 1 {
		t.Fatalf("expected top-level helm_cli_overrides, got %T %+v", got["helm_cli_overrides"], got["helm_cli_overrides"])
	}

	provenance, ok := got["provenance"].([]any)
	if !ok || len(provenance) == 0 {
		t.Fatalf("expected provenance records, got %T %+v", got["provenance"], got["provenance"])
	}
	record, ok := provenance[0].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance record object, got %T", provenance[0])
	}
	fieldOrigins, ok := record["field_origin_map"].([]any)
	if !ok || len(fieldOrigins) == 0 {
		t.Fatalf("expected field_origin_map, got %T %+v", record["field_origin_map"], record["field_origin_map"])
	}
	firstOrigin, ok := fieldOrigins[0].(map[string]any)
	if !ok {
		t.Fatalf("expected field origin object, got %T", fieldOrigins[0])
	}
	if firstOrigin["transform"] != "helm-cli-override" {
		t.Fatalf("expected helm-cli-override transform, got %v", firstOrigin["transform"])
	}
	if firstOrigin["source_path"] != "--set image.tag=v9.9.9" {
		t.Fatalf("expected override source_path, got %v", firstOrigin["source_path"])
	}
}

func TestChangeExplainHelmCLIOverrideWins(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		"--set", "image.tag=v9.9.9",
		"helm",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change explain returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change explain output: %v\noutput=%s", err, out)
	}
	explanation, ok := got["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("expected explanation object, got %T", got["explanation"])
	}
	if explanation["source_transform"] != "helm-cli-override" {
		t.Fatalf("expected source_transform=helm-cli-override, got %v", explanation["source_transform"])
	}
	if explanation["source_path"] != "--set image.tag=v9.9.9" {
		t.Fatalf("expected source_path override, got %v", explanation["source_path"])
	}
	if explanation["owner"] != "release-automation" {
		t.Fatalf("expected owner release-automation, got %v", explanation["owner"])
	}
	editHint, _ := explanation["edit_hint"].(string)
	if !strings.Contains(editHint, "Helm CLI override") {
		t.Fatalf("expected override-aware edit hint, got %q", editHint)
	}
	warning, _ := explanation["warning"].(string)
	if !strings.Contains(warning, "editing values files alone will not change") {
		t.Fatalf("expected override warning, got %q", warning)
	}
}

func TestChangeAPIHTTPPreviewWithHelmCLIOverride(t *testing.T) {
	setupAliases(t)

	srv := httptest.NewServer(newChangeAPIHandler("platform", "HEAD", "ci-bot"))
	defer srv.Close()

	req := map[string]any{
		"action": "preview",
		"input": map[string]any{
			"target_slug": "helm",
			"helm_set":    []string{"image.tag=v9.9.9"},
		},
	}
	status, body := mustJSONRequest(t, http.MethodPost, srv.URL+"/v1/changes", req)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from preview request, got %d body=%v", status, body)
	}

	input, ok := body["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input object, got %T", body["input"])
	}
	overrides, ok := input["helm_cli_overrides"].([]any)
	if !ok || len(overrides) != 1 {
		t.Fatalf("expected echoed helm_cli_overrides, got %T %+v", input["helm_cli_overrides"], input["helm_cli_overrides"])
	}
}

func TestChangeExplainHelmDefaultOriginWarning(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "Chart.yaml"), "apiVersion: v2\nname: no-values\nversion: 0.1.0\n")

	out, stderr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		repo,
	})
	if err != nil {
		t.Fatalf("change explain returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change explain output: %v\noutput=%s", err, out)
	}
	explanation, ok := got["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("expected explanation object, got %T", got["explanation"])
	}
	if explanation["origin_type"] != "default" {
		t.Fatalf("expected origin_type=default, got %v", explanation["origin_type"])
	}
	if explanation["source_transform"] != "helm-default" {
		t.Fatalf("expected source_transform=helm-default, got %v", explanation["source_transform"])
	}
	if explanation["source_path"] != "<helm-default>:values.image.tag" {
		t.Fatalf("expected default synthetic source, got %v", explanation["source_path"])
	}
	warning, _ := explanation["warning"].(string)
	if !strings.Contains(warning, "not set in the observed values files") {
		t.Fatalf("expected default warning, got %q", warning)
	}
}

func TestChangeExplainHelmBuiltinOriginWarning(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "Chart.yaml"), "apiVersion: v2\nname: builtin-demo\nversion: 0.1.0\nappVersion: 2.3.4\n")
	mustWriteFile(t, filepath.Join(repo, "values.yaml"), "image:\n  repository: ghcr.io/example/builtin-demo\n")
	mustWriteFile(t, filepath.Join(repo, "templates", "deployment.yaml"), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: builtin-demo
spec:
  template:
    spec:
      containers:
        - name: builtin-demo
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
`)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		repo,
	})
	if err != nil {
		t.Fatalf("change explain returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change explain output: %v\noutput=%s", err, out)
	}
	explanation, ok := got["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("expected explanation object, got %T", got["explanation"])
	}
	if explanation["origin_type"] != "builtin" {
		t.Fatalf("expected origin_type=builtin, got %v", explanation["origin_type"])
	}
	if explanation["source_transform"] != "helm-builtin" {
		t.Fatalf("expected source_transform=helm-builtin, got %v", explanation["source_transform"])
	}
	if explanation["source_path"] != "<helm-builtin>:.Chart.AppVersion" {
		t.Fatalf("expected builtin synthetic source, got %v", explanation["source_path"])
	}
	warning, _ := explanation["warning"].(string)
	if !strings.Contains(warning, "Helm built-in .Chart.AppVersion") {
		t.Fatalf("expected builtin warning, got %q", warning)
	}
	editHint, _ := explanation["edit_hint"].(string)
	if !strings.Contains(editHint, "Edit appVersion in Chart.yaml") {
		t.Fatalf("expected builtin edit hint, got %q", editHint)
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
