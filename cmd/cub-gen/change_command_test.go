package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangePreviewJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "preview",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change preview returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change preview output: %v\noutput=%s", err, out)
	}

	change, ok := got["change"].(map[string]any)
	if !ok {
		t.Fatalf("expected change object, got %T", got["change"])
	}
	if changeID, ok := change["change_id"].(string); !ok || !strings.HasPrefix(changeID, "chg_") {
		t.Fatalf("unexpected change_id: %v", change["change_id"])
	}
	if digest, ok := change["bundle_digest"].(string); !ok || !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("unexpected bundle_digest: %v", change["bundle_digest"])
	}

	verification, ok := got["verification"].(map[string]any)
	if !ok {
		t.Fatalf("expected verification object, got %T", got["verification"])
	}
	if valid, ok := verification["bundle_valid"].(bool); !ok || !valid {
		t.Fatalf("expected bundle_valid=true, got %v", verification["bundle_valid"])
	}
	if valid, ok := verification["attestation_valid"].(bool); !ok || !valid {
		t.Fatalf("expected attestation_valid=true, got %v", verification["attestation_valid"])
	}

	recommendation, ok := got["edit_recommendation"].(map[string]any)
	if !ok {
		t.Fatalf("expected edit_recommendation object, got %T", got["edit_recommendation"])
	}
	if owner, ok := recommendation["owner"].(string); !ok || strings.TrimSpace(owner) == "" {
		t.Fatalf("expected non-empty owner, got %v", recommendation["owner"])
	}
}

func TestChangeRunLocalJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "run",
		"--mode", "local",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change run returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change run output: %v\noutput=%s", err, out)
	}
	if mode, ok := got["mode"].(string); !ok || mode != "local" {
		t.Fatalf("expected mode=local, got %v", got["mode"])
	}
	decision, ok := got["decision"].(map[string]any)
	if !ok {
		t.Fatalf("expected decision object, got %T", got["decision"])
	}
	if state, ok := decision["state"].(string); !ok || state != "ALLOW" {
		t.Fatalf("expected decision state ALLOW, got %v", decision["state"])
	}
	if source, ok := decision["source"].(string); !ok || source != "local-preview" {
		t.Fatalf("expected decision source local-preview, got %v", decision["source"])
	}
}

func TestChangeRunConnectedMissingBaseURL(t *testing.T) {
	setupAliases(t)
	t.Setenv("CONFIGHUB_BASE_URL", "")

	_, _, err := runWithCapturedIO([]string{
		"change", "run",
		"--mode", "connected",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires --base-url or CONFIGHUB_BASE_URL") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestChangeExplainJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		"score",
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
	if owner, ok := explanation["owner"].(string); !ok || strings.TrimSpace(owner) == "" {
		t.Fatalf("expected non-empty owner, got %v", explanation["owner"])
	}
	if wetPath, ok := explanation["wet_path"].(string); !ok || strings.TrimSpace(wetPath) == "" {
		t.Fatalf("expected non-empty wet_path, got %v", explanation["wet_path"])
	}
	if dryPath, ok := explanation["dry_path"].(string); !ok || strings.TrimSpace(dryPath) == "" {
		t.Fatalf("expected non-empty dry_path, got %v", explanation["dry_path"])
	}
}

func TestChangeExplainWetPathFilter(t *testing.T) {
	setupAliases(t)

	previewOut, _, err := runWithCapturedIO([]string{
		"change", "preview",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change preview returned error: %v", err)
	}
	var preview map[string]any
	if err := json.Unmarshal([]byte(previewOut), &preview); err != nil {
		t.Fatalf("unmarshal change preview output: %v", err)
	}
	recommendation, ok := preview["edit_recommendation"].(map[string]any)
	if !ok {
		t.Fatalf("expected edit_recommendation object, got %T", preview["edit_recommendation"])
	}
	wetPath, ok := recommendation["wet_path"].(string)
	if !ok || strings.TrimSpace(wetPath) == "" {
		t.Fatalf("expected wet_path recommendation, got %v", recommendation["wet_path"])
	}

	explainOut, _, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		"--wet-path", wetPath,
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change explain returned error: %v", err)
	}
	var explain map[string]any
	if err := json.Unmarshal([]byte(explainOut), &explain); err != nil {
		t.Fatalf("unmarshal change explain output: %v", err)
	}
	query, ok := explain["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query object, got %T", explain["query"])
	}
	if got, ok := query["wet_path_filter"].(string); !ok || got != wetPath {
		t.Fatalf("expected wet_path_filter=%q, got %v", wetPath, query["wet_path_filter"])
	}
}

func TestChangeExplainByChangeIDFromBundle(t *testing.T) {
	setupAliases(t)

	publishOut, stderr, err := runWithCapturedIO([]string{
		"publish",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, stderr)
	}

	var bundle map[string]any
	if err := json.Unmarshal([]byte(publishOut), &bundle); err != nil {
		t.Fatalf("unmarshal publish output: %v", err)
	}
	changeID, ok := bundle["change_id"].(string)
	if !ok || strings.TrimSpace(changeID) == "" {
		t.Fatalf("missing change_id in bundle: %v", bundle["change_id"])
	}

	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(bundlePath, []byte(publishOut), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}

	out, explainErr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--change-id", changeID,
		"--bundle", bundlePath,
		"--owner", "app-team",
	})
	if err != nil {
		t.Fatalf("change explain by change-id returned error: %v\nstderr=%s", err, explainErr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change explain output: %v\noutput=%s", err, out)
	}
	change, ok := got["change"].(map[string]any)
	if !ok {
		t.Fatalf("expected change object, got %T", got["change"])
	}
	if gotID, ok := change["change_id"].(string); !ok || gotID != changeID {
		t.Fatalf("expected change_id=%q, got %v", changeID, change["change_id"])
	}
}

func TestChangeExplainByChangeIDMismatch(t *testing.T) {
	setupAliases(t)

	publishOut, stderr, err := runWithCapturedIO([]string{
		"publish",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, stderr)
	}
	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(bundlePath, []byte(publishOut), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}

	_, _, err = runWithCapturedIO([]string{
		"change", "explain",
		"--change-id", "chg_mismatch",
		"--bundle", bundlePath,
	})
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "bundle change_id mismatch") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestChangeCommandErrorModes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		sub  string
	}{
		{
			name: "missing-subcommand",
			args: []string{"change"},
			sub:  "change subcommand required",
		},
		{
			name: "unknown-subcommand",
			args: []string{"change", "unknown"},
			sub:  "unknown change subcommand",
		},
		{
			name: "preview-missing-targets",
			args: []string{"change", "preview"},
			sub:  "usage: cub-gen change preview",
		},
		{
			name: "run-missing-targets",
			args: []string{"change", "run"},
			sub:  "usage: cub-gen change run",
		},
		{
			name: "explain-missing-targets",
			args: []string{"change", "explain"},
			sub:  "usage: cub-gen change explain",
		},
		{
			name: "explain-change-id-missing-bundle",
			args: []string{"change", "explain", "--change-id", "chg_123"},
			sub:  "requires --bundle FILE",
		},
		{
			name: "api-missing-subcommand",
			args: []string{"change", "api"},
			sub:  "change api subcommand required",
		},
		{
			name: "api-unknown-subcommand",
			args: []string{"change", "api", "unknown"},
			sub:  "unknown change api subcommand",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runWithCapturedIO(tc.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("expected error containing %q, got %q", tc.sub, err.Error())
			}
		})
	}
}
