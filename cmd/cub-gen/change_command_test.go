package main

import (
	"encoding/json"
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
			args: []string{"change", "explain"},
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
