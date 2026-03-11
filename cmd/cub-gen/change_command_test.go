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
			args: []string{"change", "run"},
			sub:  "unknown change subcommand",
		},
		{
			name: "preview-missing-targets",
			args: []string{"change", "preview"},
			sub:  "usage: cub-gen change preview",
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
