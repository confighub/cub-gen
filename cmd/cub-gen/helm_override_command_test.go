package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
