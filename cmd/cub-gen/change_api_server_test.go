package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangeAPIHTTPRunLifecycleGolden(t *testing.T) {
	setupAliases(t)

	srv := httptest.NewServer(newChangeAPIHandler("platform", "HEAD", "ci-bot"))
	defer srv.Close()

	postReq := map[string]any{
		"action": "run",
		"mode":   "local",
		"input": map[string]any{
			"target_slug":        "score",
			"render_target_slug": "render-target",
			"space":              "platform",
			"ref":                "HEAD",
		},
	}
	status, postResp := mustJSONRequest(t, http.MethodPost, srv.URL+"/v1/changes", postReq)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from POST /v1/changes, got %d body=%v", status, postResp)
	}

	changeID := nestedString(t, postResp, "change", "change_id")
	if !strings.HasPrefix(changeID, "chg_") {
		t.Fatalf("expected change_id prefix chg_, got %q", changeID)
	}

	status, getResp := mustJSONRequest(t, http.MethodGet, srv.URL+"/v1/changes/"+changeID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from GET /v1/changes/{id}, got %d body=%v", status, getResp)
	}

	status, explainResp := mustJSONRequest(t, http.MethodGet, srv.URL+"/v1/changes/"+changeID+"/explanations?owner=app-team", nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from GET /v1/changes/{id}/explanations, got %d body=%v", status, explainResp)
	}

	snapshot := map[string]any{
		"post":    postResp,
		"get":     getResp,
		"explain": explainResp,
	}
	normalizeChangeAPIHTTPRunSnapshot(snapshot)
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "change-api-http-run.golden.json"), snapshot)
}

func TestChangeAPIHTTPErrorGolden(t *testing.T) {
	setupAliases(t)

	srv := httptest.NewServer(newChangeAPIHandler("platform", "HEAD", "ci-bot"))
	defer srv.Close()

	badReq := map[string]any{
		"action": "run",
		"input": map[string]any{
			"target_slug":        "score",
			"render_target_slug": "render-target",
		},
	}
	status, body := mustJSONRequest(t, http.MethodPost, srv.URL+"/v1/changes", badReq)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 from invalid run request, got %d body=%v", status, body)
	}

	assertGoldenJSON(t, filepath.Join("testdata", "parity", "change-api-http-error.golden.json"), body)
}

func mustJSONRequest(t *testing.T, method, url string, payload any) (int, map[string]any) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request payload: %v", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode response JSON: %v\nbody=%s", err, string(data))
	}
	return res.StatusCode, decoded
}

func nestedString(t *testing.T, root map[string]any, path ...string) string {
	t.Helper()

	var cur any = root
	for _, segment := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("expected map at %q, got %T", segment, cur)
		}
		cur = m[segment]
	}
	s, ok := cur.(string)
	if !ok {
		t.Fatalf("expected string at %v, got %T", path, cur)
	}
	return s
}

func normalizeChangeAPIHTTPRunSnapshot(snapshot map[string]any) {
	for _, key := range []string{"post", "get", "explain"} {
		obj, ok := snapshot[key].(map[string]any)
		if !ok {
			continue
		}

		if change, ok := obj["change"].(map[string]any); ok {
			if id, ok := change["change_id"].(string); ok && strings.HasPrefix(id, "chg_") {
				change["change_id"] = "chg_REDACTED"
			}
			if digest, ok := change["bundle_digest"].(string); ok && strings.HasPrefix(digest, "sha256:") {
				change["bundle_digest"] = "sha256:REDACTED"
			}
			if digest, ok := change["attestation_digest"].(string); ok && strings.HasPrefix(digest, "sha256:") {
				change["attestation_digest"] = "sha256:REDACTED"
			}
		}

		if artifacts, ok := obj["artifacts"].(map[string]any); ok {
			for artifactKey := range artifacts {
				artifacts[artifactKey] = "change://chg_REDACTED/" + artifactKey
			}
		}
	}
}
